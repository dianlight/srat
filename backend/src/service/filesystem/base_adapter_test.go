package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBaseAdapterStateCommandAvailability(t *testing.T) {
	t.Run("state command present", func(t *testing.T) {
		tempDir := t.TempDir()
		createFakeCommand(t, tempDir, "statecmd")
		t.Setenv("PATH", tempDir)

		adapter := baseAdapter{
			stateCommand: "statecmd",
		}

		support := adapter.checkCommandAvailability()
		if !support.CanGetState {
			t.Fatalf("expected CanGetState to be true when state command exists")
		}
		if len(support.MissingTools) != 0 {
			t.Fatalf("expected no missing tools, got %v", support.MissingTools)
		}
	})

	t.Run("state command missing", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("PATH", tempDir)

		adapter := baseAdapter{
			stateCommand: "statecmd",
		}

		support := adapter.checkCommandAvailability()
		if support.CanGetState {
			t.Fatalf("expected CanGetState to be false when state command is missing")
		}
		if len(support.MissingTools) != 1 || support.MissingTools[0] != "statecmd" {
			t.Fatalf("expected missing tools to include statecmd, got %v", support.MissingTools)
		}
	})
}

func createFakeCommand(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to create fake command %s: %v", name, err)
	}
}
