Phase 2: The Go Implementation (The Core Engine)
Create pkg/license/license.go. This is the heart of your air-gapped system.

Step 2.1: Define the License Structure & Embed the Public Key

go
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
	"strings"
	"time"
)

// LicensePayload matches the signed file
type LicensePayload struct {
	Customer   string   `json:"customer"`
	ExpiresAt  int64    `json:"expires_at"`
	HWID       string   `json:"hwid"`
	Features   []string `json:"features"`
	MaxInstances int    `json:"max_instances"`
}

// Embed the PUBLIC key directly into the binary.
// NEVER commit the private key.
const publicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAu1SU1LfVLPHCYZM6...
-----END PUBLIC KEY-----`
Step 2.2: Hardware ID Generator (Node-Locking)

go
func GetHardwareID() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	// Combine primary MAC with hostname for stability
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			host, _ := os.Hostname()
			return strings.ToUpper(iface.HardwareAddr.String()) + "_" + host, nil
		}
	}
	return "", errors.New("no active network interface found")
}
Step 2.3: The Offline Signature Validator

go
func ValidateLicenseFile(filePath string) (*LicensePayload, error) {
	// 1. Read the license file (Base64 encoded JSON + Signature)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("license file not found: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encoding: %w", err)
	}

	// Expected format: [JSON_PAYLOAD]|[SIGNATURE]
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 2 {
		return nil, errors.New("invalid license format")
	}
	jsonPayload := []byte(parts[0])
	signature, _ := base64.StdEncoding.DecodeString(parts[1])

	// 2. Verify Signature using embedded Public Key
	block, _ := pem.Decode([]byte(publicKeyPEM))
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("invalid public key type")
	}

	hash := sha256.Sum256(jsonPayload)
	if !ecdsa.VerifyASN1(ecdsaPub, hash[:], signature) {
		return nil, errors.New("invalid signature: license has been tampered with")
	}

	// 3. Parse JSON
	var payload LicensePayload
	if err := json.Unmarshal(jsonPayload, &payload); err != nil {
		return nil, err
	}

	// 4. Check Expiration
	if time.Now().Unix() > payload.ExpiresAt {
		return nil, errors.New("license has expired")
	}

	// 5. Check Hardware ID (Anti-piracy)
	currentHWID, _ := GetHardwareID()
	if payload.HWID != currentHWID {
		return nil, errors.New("hardware ID mismatch: license is not valid for this machine")
	}

	return &payload, nil
}
Step 2.4: Integrate into main.go

go
package main

import (
	"log"
	"your-module/pkg/license"
)

func main() {
	// Enforce license on startup
	lic, err := license.ValidateLicenseFile("/etc/myapp/license.lic")
	if err != nil {
		log.Fatalf("🚨 LICENSE ERROR: %v", err)
	}

	log.Printf("✅ License validated for: %s", lic.Customer)
	log.Printf("✅ Enabled Features: %v", lic.Features)

	// Feature gating example:
	if contains(lic.Features, "ha-mode") {
		enableHighAvailability()
	}

	// Start your API router...
}
Phase 3: The License Generator CLI (Kept in Git)
Create cmd/license-gen/main.go. This tool is run internally by your DevOps team to generate .lic files for customers.

Important: The Private Key is never stored in Git. Your CI/CD or local machine injects it via an environment variable (SIGNING_PRIVATE_KEY).

go
package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	customer := flag.String("customer", "", "Customer name")
	expiryDays := flag.Int("expiry", 365, "Days until expiry")
	hwid := flag.String("hwid", "", "Hardware ID from the customer")
	flag.Parse()

	payload := map[string]interface{}{
		"customer":    *customer,
		"expires_at":  time.Now().AddDate(0, 0, *expiryDays).Unix(),
		"hwid":        *hwid,
		"features":    []string{"ha-mode", "audit-log"},
		"max_instances": 1,
	}

	jsonData, _ := json.Marshal(payload)

	// Load private key from environment (NEVER hardcoded)
	privPEM := os.Getenv("SIGNING_PRIVATE_KEY")
	block, _ := pem.Decode([]byte(privPEM))
	priv, _ := x509.ParsePKCS8PrivateKey(block.Bytes)
	ecdsaPriv := priv.(*ecdsa.PrivateKey)

	// Sign
	hash := sha256.Sum256(jsonData)
	signature, _ := ecdsa.SignASN1(rand.Reader, ecdsaPriv, hash[:])

	// Combine and encode
	final := base64.StdEncoding.EncodeToString(append(jsonData, []byte("|")...))
	final += base64.StdEncoding.EncodeToString(signature)

	os.WriteFile("license.lic", []byte(final), 0644)
	fmt.Println("✅ License generated: license.lic")
}
Phase 4: Git Security (The .gitignore)
To ensure the private key and customer licenses never leak:

gitignore
# Private keys - NEVER COMMIT
*.pem
*.key
private.pem

# Generated license files
*.lic
/licenses/output/

# Environment files containing secrets
.env
.env.local
How to inject the Private Key in your Build Pipeline:

Local Dev: Set export SIGNING_PRIVATE_KEY="$(cat private.pem)"

GitHub Actions/GitLab CI: Store private.pem as a Protected Secret Variable and pass it to the license-gen tool.

Phase 5: Air-Gapped Clock Tampering (Extra Enterprise Hardening)
Since they can't call home, advanced implementations embed the build timestamp into the binary. Add this to your Makefile:

makefile
BUILD_TIME = $(shell date +%s)
LDFLAGS = -ldflags "-X main.buildTimestamp=$(BUILD_TIME)"

build:
	go build $(LDFLAGS) -o myapp ./cmd/server
Then in license.go:

go
var buildTimestamp string // Injected at build time

func checkClockTampering(licenseExpiry int64) error {
	buildTime, _ := strconv.ParseInt(buildTimestamp, 10, 64)
	// If system time is before the build time, someone rolled back the clock
	if time.Now().Unix() < buildTime {
		return errors.New("system clock tampering detected")
	}
	// If license expires BEFORE the binary was built, it's definitely expired
	if licenseExpiry < buildTime {
		return errors.New("license expired before this binary was released")
	}
	return nil
}
