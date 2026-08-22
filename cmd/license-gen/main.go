package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
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
	hwid := flag.String("hwid", "", "Hardware ID from the customer machine")
	features := flag.String("features", "ha-mode,audit-log", "Comma-separated list of features")
	maxInstances := flag.Int("max-instances", 1, "Maximum allowed instances")
	keyFile := flag.String("key-file", "", "Path to private key file (alternative to SIGNING_PRIVATE_KEY env var)")
	output := flag.String("out", "license.lic", "Output license file path")
	flag.Parse()

	if *customer == "" {
		fmt.Println("Error: -customer flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse feature list
	var featureList []string
	for _, f := range splitComma(*features) {
		if f != "" {
			featureList = append(featureList, f)
		}
	}

	// Build the license payload
	payload := map[string]interface{}{
		"customer":      *customer,
		"expires_at":    time.Now().AddDate(0, 0, *expiryDays).Unix(),
		"hwid":          *hwid,
		"features":      featureList,
		"max_instances": *maxInstances,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Load private key: prefer --key-file flag, fall back to SIGNING_PRIVATE_KEY env var
	var privPEM string
	if *keyFile != "" {
		data, err := os.ReadFile(*keyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading key file: %v\n", err)
			os.Exit(1)
		}
		privPEM = string(data)
	} else {
		privPEM = os.Getenv("SIGNING_PRIVATE_KEY")
		if privPEM == "" {
			fmt.Fprintln(os.Stderr, "Error: --key-file or SIGNING_PRIVATE_KEY environment variable required")
			fmt.Fprintln(os.Stderr, "Example: --key-file=private.pem")
			os.Exit(1)
		}
	}

	// Find the actual private key block (skip EC PARAMETERS, etc.)
	remaining := []byte(privPEM)
	var block *pem.Block
	for {
		block, remaining = pem.Decode(remaining)
		if block == nil {
			fmt.Fprintln(os.Stderr, "Error: no PEM block found in private key")
			os.Exit(1)
		}
		if block.Type == "EC PRIVATE KEY" || block.Type == "PRIVATE KEY" || block.Type == "ECDSA PRIVATE KEY" {
			break
		}
		if len(remaining) == 0 {
			fmt.Fprintln(os.Stderr, "Error: no private key PEM block found")
			os.Exit(1)
		}
	}

	var ecdsaPriv *ecdsa.PrivateKey

	// Try EC PRIVATE KEY format first
	if priv, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		ecdsaPriv = priv
	} else if genericPriv, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		// Try PKCS8 format as fallback
		var ok bool
		ecdsaPriv, ok = genericPriv.(*ecdsa.PrivateKey)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error: PKCS8 key is not ECDSA")
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: cannot parse private key (tried EC and PKCS8): %v\n", err)
		os.Exit(1)
	}

	// Sign the JSON payload with ECDSA-P256
	hash := sha256.Sum256(jsonData)
	signature, err := ecdsa.SignASN1(rand.Reader, ecdsaPriv, hash[:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error signing license: %v\n", err)
		os.Exit(1)
	}

	// Combine into format: base64( jsonData + "|" + base64(signature) )
	// The validator decodes the full base64, splits by "|", then decodes the signature part
	signatureB64 := base64.StdEncoding.EncodeToString(signature)
	inner := concat(jsonData, []byte{'|'}, []byte(signatureB64))
	combined := base64.StdEncoding.EncodeToString(inner)

	if err := os.WriteFile(*output, []byte(combined+"\n"), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing license file: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("✅ License generated: %s\n", *output)
	fmt.Printf("   Customer:    %s\n", *customer)
	fmt.Printf("   Expires:     %s (%d days)\n", time.Now().AddDate(0, 0, *expiryDays).Format("2006-01-02"), *expiryDays)
	fmt.Printf("   HWID:       %s\n", *hwid)
	fmt.Printf("   Features:    %v\n", featureList)
	fmt.Printf("   Max Nodes:  %d\n", *maxInstances)
}

// generateKeyPair generates a new ECDSA P-256 key pair for testing purposes.
// Run with: go run cmd/license-gen/main.go --generate-key
func generateKeyPair() error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	// Encode private key
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privBytes,
	})

	// Encode public key
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return err
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	if err := os.WriteFile("private.pem", []byte(privPEM), 0600); err != nil {
		return err
	}
	if err := os.WriteFile("public.pem", []byte(pubPEM), 0644); err != nil {
		return err
	}

	fmt.Println("✅ Key pair generated: private.pem, public.pem")
	fmt.Println("   ⚠️  Keep private.pem SECRET - never commit to Git")
	fmt.Println("   Embed public.pem content into pkg/license/license.go")
	return nil
}

func concat(parts ...[]byte) []byte {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	result := make([]byte, total)
	offset := 0
	for _, p := range parts {
		offset += copy(result[offset:], p)
	}
	return result
}

func splitComma(s string) []string {
	result := []string{}
	for _, part := range split(s, ',') {
		result = append(result, part)
	}
	return result
}

func split(s string, sep rune) []string {
	var parts []string
	var current string
	for _, r := range s {
		if r == sep {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	parts = append(parts, current)
	return parts
}
