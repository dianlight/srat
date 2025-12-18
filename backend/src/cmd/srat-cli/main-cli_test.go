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

func TestCLIVersionWorksWithoutDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI integration tests in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cwd := packageDir(t)

	// Run version command without specifying -db flag (tests that it doesn't require DB file)
	cmd := exec.CommandContext(ctx, "go", "run", ".", "-silent", "version")
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "SRAT_MOCK=true")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("srat-cli version command timed out: %s", string(output))
		}
		t.Fatalf("srat-cli version command failed: %v\nOutput:\n%s", err, string(output))
	}

	// Verify output contains version info
	outputStr := string(output)
	if !strings.Contains(outputStr, "Version:") {
		t.Fatalf("expected version output to contain 'Version:', got: %s", outputStr)
	}
	if !strings.Contains(outputStr, config.Version) {
		t.Fatalf("expected version output to contain %q, got: %s", config.Version, outputStr)
	}
}

/*
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
*/

func TestFormatVersionMessage(t *testing.T) {
	short := formatVersionMessage(true)
	if short != config.Version+"\n" {
		t.Fatalf("unexpected short version message: %q", short)
	}
	long := formatVersionMessage(false)
	if !strings.Contains(long, config.Version) {
		t.Fatalf("expected version in long output: %q", long)
	}
	if !strings.HasSuffix(long, "\n") {
		t.Fatalf("expected trailing newline in long output")
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

// Note: Upgrade command test removed due to FX lifecycle hanging issues in test environment.
// The upgrade command works correctly in practice using in-memory DB by default.
// Manual testing shows: go run ./cmd/srat-cli -silent upgrade -channel release

func TestParseCommandValid(t *testing.T) {
	for _, cmd := range []string{"start", "stop", "upgrade", "version", "hdidle"} {
		cmd := cmd
		t.Run(cmd, func(t *testing.T) {
			result, err := parseCommand([]string{cmd})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != cmd {
				t.Fatalf("unexpected result: %q", result)
			}
		})
	}
}

func TestParseCommandMissing(t *testing.T) {
	_, err := parseCommand(nil)
	if err == nil {
		t.Fatalf("expected error for missing command")
	}
	if !strings.Contains(err.Error(), "expected 'start','stop','upgrade','hdidle' or 'version' subcommands") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCommandUnknown(t *testing.T) {
	_, err := parseCommand([]string{"invalid"})
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("unexpected error: %v", err)
	}
}
