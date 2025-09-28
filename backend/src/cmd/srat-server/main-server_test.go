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

func TestBuildServerContextState(t *testing.T) {
	now := time.Unix(1700000000, 0)
	opts := serverContextOptions{
		AddonIPAddress:  "192.168.1.2",
		ReadOnlyMode:    true,
		ProtectedMode:   true,
		SecureMode:      true,
		UpdateFilePath:  "/tmp/update.bin",
		SambaConfigFile: "/etc/samba/smb.conf",
		Template:        []byte("template"),
		DockerInterface: "eth0",
		DockerNetwork:   "bridge",
		DatabasePath:    ":memory:",
		SupervisorToken: "token",
		SupervisorURL:   "http://supervisor.local",
		Heartbeat:       10,
		StartTime:       now,
	}

	state := buildServerContextState(opts)

	if state.AddonIpAddress != opts.AddonIPAddress {
		t.Fatalf("unexpected AddonIpAddress: %q", state.AddonIpAddress)
	}
	if state.ReadOnlyMode != opts.ReadOnlyMode {
		t.Fatalf("unexpected ReadOnlyMode: %v", state.ReadOnlyMode)
	}
	if state.ProtectedMode != opts.ProtectedMode {
		t.Fatalf("unexpected ProtectedMode: %v", state.ProtectedMode)
	}
	if state.SecureMode != opts.SecureMode {
		t.Fatalf("unexpected SecureMode: %v", state.SecureMode)
	}
	if state.UpdateFilePath != opts.UpdateFilePath {
		t.Fatalf("unexpected UpdateFilePath: %q", state.UpdateFilePath)
	}
	if state.SambaConfigFile != opts.SambaConfigFile {
		t.Fatalf("unexpected SambaConfigFile: %q", state.SambaConfigFile)
	}
	if string(state.Template) != string(opts.Template) {
		t.Fatalf("unexpected template data")
	}
	if state.DockerInterface != opts.DockerInterface {
		t.Fatalf("unexpected DockerInterface: %q", state.DockerInterface)
	}
	if state.DockerNet != opts.DockerNetwork {
		t.Fatalf("unexpected DockerNet: %q", state.DockerNet)
	}
	if state.DatabasePath != opts.DatabasePath {
		t.Fatalf("unexpected DatabasePath: %q", state.DatabasePath)
	}
	if state.SupervisorToken != opts.SupervisorToken {
		t.Fatalf("unexpected SupervisorToken: %q", state.SupervisorToken)
	}
	if state.SupervisorURL != opts.SupervisorURL {
		t.Fatalf("unexpected SupervisorURL: %q", state.SupervisorURL)
	}
	if state.Heartbeat != opts.Heartbeat {
		t.Fatalf("unexpected Heartbeat: %d", state.Heartbeat)
	}
	if !state.StartTime.Equal(opts.StartTime) {
		t.Fatalf("unexpected StartTime: %v", state.StartTime)
	}
}
