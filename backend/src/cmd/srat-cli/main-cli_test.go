package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dianlight/srat/config"
)

func TestCLIVersionShortOutputsVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI integration tests in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cwd := packageDir(t)

	cmd := exec.CommandContext(ctx, "go", "run", ".", "-silent", "version", "-short")
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "SRAT_MOCK=true")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("srat-cli version command timed out: %s", string(output))
		}
		t.Fatalf("srat-cli version command failed: %v\nOutput:\n%s", err, string(output))
	}

	got := strings.TrimSpace(string(output))
	if got != config.Version {
		t.Fatalf("unexpected version output: got %q want %q", got, config.Version)
	}
}

func TestCLIStartRequiresOutputFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI integration tests in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cwd := packageDir(t)

	cmd := exec.CommandContext(ctx, "go", "run", ".", "-silent", "start")
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "SRAT_MOCK=true")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failure when missing -out flag; output: %s", string(output))
	}

	if !strings.Contains(string(output), "Missing samba config!") {
		t.Fatalf("expected missing samba config message, got: %s", string(output))
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
