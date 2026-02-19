package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func TestBaseAdapterGetName(t *testing.T) {
	adapter := baseAdapter{
		name: "testfs",
	}
	if got := adapter.GetName(); got != "testfs" {
		t.Errorf("GetName() = %v, want %v", got, "testfs")
	}
}

func TestBaseAdapterGetDescription(t *testing.T) {
	adapter := baseAdapter{
		description: "Test Filesystem",
	}
	if got := adapter.GetDescription(); got != "Test Filesystem" {
		t.Errorf("GetDescription() = %v, want %v", got, "Test Filesystem")
	}
}

func TestBaseAdapterGetLinuxFsModule(t *testing.T) {
	tests := []struct {
		name          string
		adapterName   string
		linuxFsModule string
		want          string
	}{
		{
			name:          "with explicit module",
			adapterName:   "ntfs",
			linuxFsModule: "ntfs3",
			want:          "ntfs3",
		},
		{
			name:          "without explicit module",
			adapterName:   "ext4",
			linuxFsModule: "",
			want:          "ext4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := baseAdapter{
				name:          tt.adapterName,
				linuxFsModule: tt.linuxFsModule,
			}
			if got := adapter.GetLinuxFsModule(); got != tt.want {
				t.Errorf("GetLinuxFsModule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseAdapterCheckCommandAvailability(t *testing.T) {
	tests := []struct {
		name          string
		mkfsCommand   string
		fsckCommand   string
		labelCommand  string
		stateCommand  string
		setupCommands []string
		wantFormat    bool
		wantCheck     bool
		wantLabel     bool
		wantState     bool
	}{
		{
			name:          "all commands available",
			mkfsCommand:   "mkfscmd",
			fsckCommand:   "fsckcmd",
			labelCommand:  "labelcmd",
			stateCommand:  "statecmd",
			setupCommands: []string{"mkfscmd", "fsckcmd", "labelcmd", "statecmd"},
			wantFormat:    true,
			wantCheck:     true,
			wantLabel:     true,
			wantState:     true,
		},
		{
			name:          "no commands available",
			mkfsCommand:   "mkfscmd",
			fsckCommand:   "fsckcmd",
			labelCommand:  "labelcmd",
			stateCommand:  "statecmd",
			setupCommands: []string{},
			wantFormat:    false,
			wantCheck:     false,
			wantLabel:     false,
			wantState:     false,
		},
		{
			name:          "partial commands available",
			mkfsCommand:   "mkfscmd",
			fsckCommand:   "fsckcmd",
			labelCommand:  "labelcmd",
			stateCommand:  "statecmd",
			setupCommands: []string{"mkfscmd", "statecmd"},
			wantFormat:    true,
			wantCheck:     false,
			wantLabel:     false,
			wantState:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			for _, cmd := range tt.setupCommands {
				createFakeCommand(t, tempDir, cmd)
			}
			t.Setenv("PATH", tempDir)

			adapter := baseAdapter{
				mkfsCommand:   tt.mkfsCommand,
				fsckCommand:   tt.fsckCommand,
				labelCommand:  tt.labelCommand,
				stateCommand:  tt.stateCommand,
				alpinePackage: "testpkg",
			}

			support := adapter.checkCommandAvailability()

			if support.CanFormat != tt.wantFormat {
				t.Errorf("CanFormat = %v, want %v", support.CanFormat, tt.wantFormat)
			}
			if support.CanCheck != tt.wantCheck {
				t.Errorf("CanCheck = %v, want %v", support.CanCheck, tt.wantCheck)
			}
			if support.CanSetLabel != tt.wantLabel {
				t.Errorf("CanSetLabel = %v, want %v", support.CanSetLabel, tt.wantLabel)
			}
			if support.CanGetState != tt.wantState {
				t.Errorf("CanGetState = %v, want %v", support.CanGetState, tt.wantState)
			}
			if support.AlpinePackage != "testpkg" {
				t.Errorf("AlpinePackage = %v, want %v", support.AlpinePackage, "testpkg")
			}
			if !support.CanMount {
				t.Errorf("CanMount should always be true")
			}
		})
	}
}

func TestCommandExists(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		setup      func(t *testing.T) string
		wantExists bool
	}{
		{
			name:    "command exists",
			command: "testcmd",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				createFakeCommand(t, tempDir, "testcmd")
				return tempDir
			},
			wantExists: true,
		},
		{
			name:    "command does not exist",
			command: "nonexistent",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := tt.setup(t)
			t.Setenv("PATH", tempDir)

			if got := commandExists(tt.command); got != tt.wantExists {
				t.Errorf("commandExists() = %v, want %v", got, tt.wantExists)
			}
		})
	}
}

func createFakeCommand(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to create fake command %s: %v", name, err)
	}
}

func TestRunCommandCached(t *testing.T) {
	t.Run("same command and args are cached", func(t *testing.T) {
		tempDir := t.TempDir()
		counterFile := filepath.Join(tempDir, "counter.txt")
		createCountingCommand(t, tempDir, "countcmd", counterFile)
		t.Setenv("PATH", tempDir)

		commandResultCache.Flush()
		ctx := context.Background()

		firstOutput, firstExitCode, firstErr := runCommandCached(ctx, "countcmd", "arg1")
		if firstErr != nil {
			t.Fatalf("expected no error on first cached command execution, got %v", firstErr)
		}
		if firstExitCode != 0 {
			t.Fatalf("expected first exit code 0, got %d", firstExitCode)
		}

		secondOutput, secondExitCode, secondErr := runCommandCached(ctx, "countcmd", "arg1")
		if secondErr != nil {
			t.Fatalf("expected no error on second cached command execution, got %v", secondErr)
		}
		if secondExitCode != 0 {
			t.Fatalf("expected second exit code 0, got %d", secondExitCode)
		}

		if firstOutput != secondOutput {
			t.Fatalf("expected cached output to match, got first=%q second=%q", firstOutput, secondOutput)
		}

		count := readCounter(t, counterFile)
		if count != 1 {
			t.Fatalf("expected command to execute once with identical args, got %d executions", count)
		}
	})

	t.Run("different args use different cache entries", func(t *testing.T) {
		tempDir := t.TempDir()
		createFakeCommand(t, tempDir, "countcmd")
		t.Setenv("PATH", tempDir)

		commandResultCache.Flush()
		ctx := context.Background()

		_, _, err := runCommandCached(ctx, "countcmd", "arg1")
		if err != nil {
			t.Fatalf("expected no error on first command execution, got %v", err)
		}

		_, _, err = runCommandCached(ctx, "countcmd", "arg2")
		if err != nil {
			t.Fatalf("expected no error on second command execution with different args, got %v", err)
		}

		keyArg1 := commandCacheKey("countcmd", "arg1")
		keyArg2 := commandCacheKey("countcmd", "arg2")
		if keyArg1 == keyArg2 {
			t.Fatalf("expected cache keys to differ for different args")
		}

		if _, found := commandResultCache.Get(keyArg1); !found {
			t.Fatalf("expected cache entry for arg1")
		}
		if _, found := commandResultCache.Get(keyArg2); !found {
			t.Fatalf("expected cache entry for arg2")
		}

		if len(commandResultCache.Items()) != 2 {
			t.Fatalf("expected 2 cache entries for different args, got %d", len(commandResultCache.Items()))
		}
	})
}

func createCountingCommand(t *testing.T, dir, name, counterFile string) {
	t.Helper()
	path := filepath.Join(dir, name)
	script := "#!/bin/sh\n" +
		"count=$(cat '" + counterFile + "' 2>/dev/null || echo 0)\n" +
		"count=$((count + 1))\n" +
		"echo \"$count\" > '" + counterFile + "'\n" +
		"echo \"run-$count\"\n" +
		"exit 0\n"

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to create counting command %s: %v", name, err)
	}
}

func readCounter(t *testing.T, counterFile string) int {
	t.Helper()
	content, err := os.ReadFile(counterFile)
	if err != nil {
		t.Fatalf("failed to read counter file %s: %v", counterFile, err)
	}

	counter := strings.TrimSpace(string(content))
	value, err := strconv.Atoi(counter)
	if err != nil {
		t.Fatalf("unexpected counter value %q: %v", counter, err)
	}

	return value
}
