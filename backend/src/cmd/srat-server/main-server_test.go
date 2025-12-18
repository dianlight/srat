package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServerRequiresSambaConfigFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping server integration tests in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cwd := packageDir(t)

	cmd := exec.CommandContext(ctx, "go", "run", ".", "-single-instance", "-port=0")
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "SRAT_MOCK=true")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected srat-server to fail without -out flag; output: %s", string(output))
	}

	if !strings.Contains(string(output), "Missing samba config!") {
		t.Fatalf("expected missing samba config message, got: %s", string(output))
	}
}

func TestValidateSambaConfig(t *testing.T) {
	if err := validateSambaConfig("/tmp/smb.conf"); err != nil {
		t.Fatalf("expected valid path, got %v", err)
	}
}

func TestValidateSambaConfigEmpty(t *testing.T) {
	if err := validateSambaConfig("   "); err == nil {
		t.Fatalf("expected error for empty path")
	}
}

func packageDir(t *testing.T) string {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to resolve working directory: %v", err)
	}

	return filepath.Clean(cwd)
}
