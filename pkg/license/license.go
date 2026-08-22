package license

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// LicensePayload represents the signed license data structure.
type LicensePayload struct {
	Customer     string   `json:"customer"`
	ExpiresAt    int64    `json:"expires_at"`
	HWID         string   `json:"hwid"`
	Features     []string `json:"features"`
	MaxInstances int      `json:"max_instances"`
}

// Embed the PUBLIC key directly into the binary.
// This is the ECDSA P-256 public key that corresponds to the private key used by license-gen.
// NEVER commit the private key to version control.
const publicKeyPEM = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEXyUeeXkEJe+LI/toYQ6nanMXXxtm
ldV9N1QbWSvJP3LWXJNWVpFUAWm7n5in9QebSWCr633i3LPPZXUurEDuig==
-----END PUBLIC KEY-----`

// buildTimestamp is injected at build time via ldflags.
// See Makefile for injection: -X api-first/pkg/license.buildTimestamp=<timestamp>
var buildTimestamp string

// GetHardwareID returns a stable hardware identifier for node-locking.
// It combines the primary network interface MAC address with the hostname for stability.
func GetHardwareID() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			host, _ := os.Hostname()
			return strings.ToUpper(iface.HardwareAddr.String()) + "_" + host, nil
		}
	}
	return "", errors.New("no active network interface found")
}

// parsePublickey parses the embedded PEM public key into an *ecdsa.PublicKey.
func parsePublicKey() (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse embedded public key PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
	}
	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("embedded key is not an ECDSA public key")
	}
	return ecdsaPub, nil
}

// ValidateLicenseFile performs full offline validation of a license file.
//
// Expected file format (base64-encoded): <JSON_PAYLOAD>|<ECDSA_SIGNATURE>
//
// Validation steps:
//  1. Read and base64-decode the license file
//  2. Split into JSON payload and signature
//  3. Verify ECDSA signature against embedded public key
//  4. Parse JSON into LicensePayload
//  5. Check expiration timestamp
//  6. Check hardware ID binding (anti-piracy)
//  7. Check clock tampering (Phase 5 hardening)
func ValidateLicenseFile(filePath string) (*LicensePayload, error) {
	// 1. Read the license file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("license file not found: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encoding: %w", err)
	}

	// Expected format: [JSON_PAYLOAD]|[SIGNATURE]
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid license format: expected json|signature")
	}
	jsonPayload := []byte(parts[0])
	signature, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}

	// 2. Verify Signature using embedded Public Key
	ecdsaPub, err := parsePublicKey()
	if err != nil {
		return nil, fmt.Errorf("public key error: %w", err)
	}

	hash := sha256.Sum256(jsonPayload)
	if !ecdsa.VerifyASN1(ecdsaPub, hash[:], signature) {
		return nil, errors.New("invalid signature: license has been tampered with")
	}

	// 3. Parse JSON
	var payload LicensePayload
	if err := json.Unmarshal(jsonPayload, &payload); err != nil {
		return nil, fmt.Errorf("invalid license JSON: %w", err)
	}

	// 4. Check Expiration
	if time.Now().Unix() > payload.ExpiresAt {
		return nil, errors.New("license has expired")
	}

	// 5. Check Hardware ID (Anti-piracy node-locking)
	currentHWID, err := GetHardwareID()
	if err == nil && payload.HWID != "" && payload.HWID != currentHWID {
		return nil, errors.New("hardware ID mismatch: license is not valid for this machine")
	}

	// 6. Clock tampering detection (Phase 5)
	if err := checkClockTampering(payload.ExpiresAt); err != nil {
		return nil, err
	}

	return &payload, nil
}

// checkClockTampering detects if the system clock has been rolled back.
// buildTimestamp is injected at compile time via ldflags.
func checkClockTampering(licenseExpiry int64) error {
	if buildTimestamp == "" {
		// No build timestamp injected; skip clock tampering check
		return nil
	}

	buildTime, err := strconv.ParseInt(buildTimestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid build timestamp in binary: %w", err)
	}

	// If system time is before the build time, someone rolled back the clock
	if time.Now().Unix() < buildTime {
		return errors.New("system clock tampering detected: system time is before binary build time")
	}

	// If license expires BEFORE the binary was built, it's definitely expired
	if licenseExpiry < buildTime {
		return errors.New("license expired before this binary was released")
	}

	return nil
}

// HasFeature checks if a specific feature is enabled in the license.
func (l *LicensePayload) HasFeature(feature string) bool {
	for _, f := range l.Features {
		if f == feature {
			return true
		}
	}
	return false
}
