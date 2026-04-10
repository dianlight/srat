package unixsamba

import (
	"context"
	"errors"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/require"
)

func TestValidateUnixSambaCommand_AllowsKnownCommands(t *testing.T) {
	err := validateUnixSambaCommand("pdbedit", "-L", "-v", "-u", "testuser")
	require.NoError(t, err)
}

func TestValidateUnixSambaCommand_RejectsUnknownCommand(t *testing.T) {
	err := validateUnixSambaCommand("sh", "-c", "echo unsafe")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestValidateUnixSambaCommand_RejectsNewlineInArguments(t *testing.T) {
	err := validateUnixSambaCommand("useradd", "valid", "arg-with\nnewline")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid newline")
}

func TestDefaultCommandExecutor_UsesSharedRunner(t *testing.T) {
	restore := SetCommandRunner(&testCommandRunner{
		execute: func(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
			require.Equal(t, "unixsamba:pdbedit", commandID)
			require.Equal(t, "Unix Samba pdbedit", label)
			require.Equal(t, "pdbedit", command)
			require.Equal(t, []string{"-L"}, args)
			return dto.CommandExecutionSnapshot{
				Success: true,
				Lines: []dto.CommandOutputLineSnapshot{{
					Channel: dto.CommandOutputChannelStdout,
					Line:    "runner-output",
				}},
			}, nil
		},
	})
	defer restore()

	output, err := (&defaultCommandExecutor{}).RunCommand(context.Background(), "pdbedit", "-L")
	require.NoError(t, err)
	require.Equal(t, "runner-output", output)
}

func TestDefaultCommandExecutor_UsesSharedRunnerWithInput(t *testing.T) {
	restore := SetCommandRunner(&testCommandRunner{
		executeWithInput: func(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
			require.Equal(t, "unixsamba:smbpasswd", commandID)
			require.Equal(t, "Unix Samba smbpasswd", label)
			require.Equal(t, "secret\nsecret\n", stdinContent)
			require.Equal(t, "smbpasswd", command)
			require.Equal(t, []string{"-s", "testuser"}, args)
			return dto.CommandExecutionSnapshot{
				Success: true,
				Lines: []dto.CommandOutputLineSnapshot{{
					Channel: dto.CommandOutputChannelStdout,
					Line:    "password updated",
				}},
			}, nil
		},
	})
	defer restore()

	output, err := (&defaultCommandExecutor{}).RunCommandWithInput(context.Background(), "secret\nsecret\n", "smbpasswd", "-s", "testuser")
	require.NoError(t, err)
	require.Equal(t, "password updated", output)
}

func TestGetSambaVersion(t *testing.T) {
	restoreExec := MockSambaVersionExec(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		require.Equal(t, "smbd", name)
		require.Equal(t, []string{"--version"}, args)
		return []byte("Version 4.23.2\n"), nil
	})
	defer restoreExec()

	version, err := GetSambaVersion()
	require.NoError(t, err)
	require.Equal(t, "4.23.2", version)
}

func TestGetSambaVersion_UsesConfiguredCommandRunner(t *testing.T) {
	restore := SetCommandRunner(&testCommandRunner{
		execute: func(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
			require.Equal(t, "unixsamba:smbd", commandID)
			require.Equal(t, "Unix Samba smbd", label)
			require.Equal(t, "smbd", command)
			require.Equal(t, []string{"--version"}, args)
			return dto.CommandExecutionSnapshot{
				Success: true,
				Lines: []dto.CommandOutputLineSnapshot{{
					Channel: dto.CommandOutputChannelStdout,
					Line:    "Version 4.24.1",
				}},
			}, nil
		},
	})
	defer restore()

	version, err := GetSambaVersion()
	require.NoError(t, err)
	require.Equal(t, "4.24.1", version)
}

func TestGetSambaVersion_CommandError(t *testing.T) {
	restoreExec := MockSambaVersionExec(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, errors.New("command failed")
	})
	defer restoreExec()

	version, err := GetSambaVersion()
	require.Error(t, err)
	require.Empty(t, version)
}

func TestGetSambaVersion_InvalidOutput(t *testing.T) {
	restoreExec := MockSambaVersionExec(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("not-a-version-line"), nil
	})
	defer restoreExec()

	version, err := GetSambaVersion()
	require.NoError(t, err)
	require.Empty(t, version)
}

func TestIsSambaVersionSufficient(t *testing.T) {
	restoreVersion := MockSambaVersion("4.23.1")
	defer restoreVersion()

	sufficient, err := IsSambaVersionSufficient()
	require.NoError(t, err)
	require.True(t, sufficient)
}

type testCommandRunner struct {
	start                 func(context.Context, string, string, string, ...string) (string, error)
	startQuiet            func(context.Context, string, string, string, ...string) (string, error)
	execute               func(context.Context, string, string, string, ...string) (dto.CommandExecutionSnapshot, error)
	executeQuiet          func(context.Context, string, string, string, ...string) (dto.CommandExecutionSnapshot, error)
	executeWithInput      func(context.Context, string, string, string, string, ...string) (dto.CommandExecutionSnapshot, error)
	executeWithInputQuiet func(context.Context, string, string, string, string, ...string) (dto.CommandExecutionSnapshot, error)
	getSnapshot           func(string) (dto.CommandExecutionSnapshot, bool)
}

func (t *testCommandRunner) LookPath(command string) (string, error) {
	if command == "" {
		return "", errors.New("command is empty")
	}
	return command, nil
}

func (t *testCommandRunner) Start(ctx context.Context, commandID, label, command string, args ...string) (string, error) {
	if t.start != nil {
		return t.start(ctx, commandID, label, command, args...)
	}
	return "", nil
}

func (t *testCommandRunner) StartQuiet(ctx context.Context, commandID, label, command string, args ...string) (string, error) {
	if t.startQuiet != nil {
		return t.startQuiet(ctx, commandID, label, command, args...)
	}
	return t.Start(ctx, commandID, label, command, args...)
}

func (t *testCommandRunner) StartWithInput(ctx context.Context, commandID, label, _ string, command string, args ...string) (string, error) {
	return t.Start(ctx, commandID, label, command, args...)
}

func (t *testCommandRunner) StartWithInputQuiet(ctx context.Context, commandID, label, _ string, command string, args ...string) (string, error) {
	return t.StartQuiet(ctx, commandID, label, command, args...)
}

func (t *testCommandRunner) Execute(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	if t.execute != nil {
		return t.execute(ctx, commandID, label, command, args...)
	}
	return dto.CommandExecutionSnapshot{}, nil
}

func (t *testCommandRunner) ExecuteQuiet(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	if t.executeQuiet != nil {
		return t.executeQuiet(ctx, commandID, label, command, args...)
	}
	return t.Execute(ctx, commandID, label, command, args...)
}

func (t *testCommandRunner) ExecuteWithInput(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	if t.executeWithInput != nil {
		return t.executeWithInput(ctx, commandID, label, stdinContent, command, args...)
	}
	return t.Execute(ctx, commandID, label, command, args...)
}

func (t *testCommandRunner) ExecuteWithInputQuiet(ctx context.Context, commandID, label, stdinContent, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	if t.executeWithInputQuiet != nil {
		return t.executeWithInputQuiet(ctx, commandID, label, stdinContent, command, args...)
	}
	return t.ExecuteQuiet(ctx, commandID, label, command, args...)
}

func (t *testCommandRunner) GetSnapshot(executionID string) (dto.CommandExecutionSnapshot, bool) {
	if t.getSnapshot != nil {
		return t.getSnapshot(executionID)
	}
	return dto.CommandExecutionSnapshot{}, false
}
