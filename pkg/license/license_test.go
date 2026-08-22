package license

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestValidateLicenseFile(t *testing.T) {
	licPath := "../../test.lic"
	if _, err := os.Stat(licPath); os.IsNotExist(err) {
		t.Skip("test.lic not found, run license-gen first")
	}

	lic, err := ValidateLicenseFile(licPath)
	if err != nil {
		t.Fatalf("ValidateLicenseFile failed: %v", err)
	}

	if lic.Customer != "TestCorp" {
		t.Errorf("expected customer TestCorp, got %s", lic.Customer)
	}

	if len(lic.Features) != 2 {
		t.Errorf("expected 2 features, got %d", len(lic.Features))
	}

	if lic.MaxInstances != 5 {
		t.Errorf("expected maxInstances 5, got %d", lic.MaxInstances)
	}

	t.Logf("License validated: customer=%s, features=%v, maxInstances=%d", lic.Customer, lic.Features, lic.MaxInstances)
}

func TestGetHardwareID(t *testing.T) {
	hwid, err := GetHardwareID()
	if err != nil {
		t.Fatalf("GetHardwareID failed: %v", err)
	}
	if hwid == "" {
		t.Error("GetHardwareID returned empty string")
	}
	t.Logf("Hardware ID: %s", hwid)
}

func TestValidateLicenseFile_NotFound(t *testing.T) {
	_, err := ValidateLicenseFile("/nonexistent/path/license.lic")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestValidateLicenseFile_InvalidSignature(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "bad_*.lic")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("aW52YWxkZGF0YXxub3R2YWxpZHNpZ25hdHVyZQ==\n")
	tmpFile.Close()

	_, err = ValidateLicenseFile(tmpFile.Name())
	if err == nil {
		t.Error("expected error for invalid signature, got nil")
	}
	t.Logf("Got expected error for tampered license: %v", err)
}

func TestLicenseGenEndToEnd(t *testing.T) {
	// Build license-gen if not already built
	if _, err := os.Stat("../../out/license-gen"); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "../../out/license-gen", "../../cmd/license-gen")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("could not build license-gen: %v: %s", err, string(out))
		}
	}

	// Generate a license
	tmpLic, err := os.CreateTemp("", "e2e_*.lic")
	if err != nil {
		t.Fatal(err)
	}
	tmpLicPath := tmpLic.Name()
	tmpLic.Close()
	defer os.Remove(tmpLicPath)

	cmd := exec.Command("../../out/license-gen",
		"-customer", "E2ETest",
		"-expiry", "30",
		"-hwid", "",
		"-features", "feature-a,feature-b",
		"-max-instances", "3",
		"-key-file", "../../cmd/license-gen/private.pem",
		"-out", tmpLicPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("license-gen failed: %v: %s", err, string(out))
	}

	// Validate the generated license
	lic, err := ValidateLicenseFile(tmpLicPath)
	if err != nil {
		t.Fatalf("ValidateLicenseFile failed: %v", err)
	}

	if lic.Customer != "E2ETest" {
		t.Errorf("expected customer E2ETest, got %s", lic.Customer)
	}

	if len(lic.Features) != 2 || !strings.Contains(strings.Join(lic.Features, ","), "feature-a") {
		t.Errorf("expected features [feature-a, feature-b], got %v", lic.Features)
	}

	if lic.MaxInstances != 3 {
		t.Errorf("expected maxInstances 3, got %d", lic.MaxInstances)
	}

	t.Logf("End-to-end test passed: customer=%s, features=%v, maxInstances=%d", lic.Customer, lic.Features, lic.MaxInstances)
}
