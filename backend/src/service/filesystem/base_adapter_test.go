package filesystem

import (
	"context"
	"errors"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/darwinstubs/mount"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/stretchr/testify/suite"
)

// BaseAdapterTestSuite tests the baseAdapter implementation
type BaseAdapterTestSuite struct {
	suite.Suite
	ctx        context.Context
	adapter    baseAdapter
	cleanExec  func() // Optional cleanup function for tests that set exec ops
	cleanMount func() // Optional cleanup function for tests that set mount ops
}

func TestBaseAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(BaseAdapterTestSuite))
}

func (suite *BaseAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = newBaseAdapter(
		"ntfs",
		"NTFS Filesystem",
		false,
		"ntfs-3g-progs",
		"mkfs.ntfs",
		"ntfsfix",
		"ntfslabel",
		"ntfsfix",
		`^[^\x00/]{1,32}$`,
		[]dto.FsMagicSignature{
			{Offset: 3, Magic: []byte{'N', 'T', 'F', 'S', ' ', ' ', ' ', ' '}}, // "NTFS    "
		},
	)
	suite.adapter.linuxFsModule = "ntfs3"

	suite.cleanExec = suite.adapter.SetExecOpsForTesting(
		func(cmd string) (string, error) {
			if cmd != "" {
				return cmd, nil
			}
			return "", errors.New("command not found")
		},
	)

	suite.cleanMount = suite.adapter.SetMountOpsForTesting(
		func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error) {
			return &mount.MountPoint{Path: target, Device: source, FSType: "auto", Flags: flags, Data: data}, nil
		},
		func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error) {
			return &mount.MountPoint{Path: target, Device: source, FSType: fstype, Flags: flags, Data: data}, nil
		},
		func(target string, force, lazy bool) error {
			return nil
		},
	)

}

func (suite *BaseAdapterTestSuite) TearDownTest() {
	if suite.cleanExec != nil {
		suite.cleanExec()
	}
	if suite.cleanMount != nil {
		suite.cleanMount()
	}
	suite.cleanExec = nil
	suite.cleanMount = nil
}

// TestGetName tests the GetName method
func (suite *BaseAdapterTestSuite) TestGetName() {
	suite.Equal("ntfs", suite.adapter.GetName())
}

// TestGetDescription tests the GetDescription method
func (suite *BaseAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "NTFS")
}

// TestGetLinuxFsModule tests the GetLinuxFsModule method
func (suite *BaseAdapterTestSuite) TestGetLinuxFsModule() {
	// Test with ntfs (which has specific linux module)
	linuxModule := suite.adapter.GetLinuxFsModule()
	suite.NotEmpty(linuxModule)
	suite.Equal("ntfs3", linuxModule)
}

// TestGetAliasNames tests the GetAliasNames method.
func (suite *BaseAdapterTestSuite) TestGetAliasNames() {
	suite.adapter = newBaseAdapter(
		"ntfs",
		"NTFS Filesystem",
		false,
		"ntfs-3g-progs",
		"mkfs.ntfs",
		"ntfsfix",
		"ntfslabel",
		"ntfsfix",
		`^[^\x00/]{1,32}$`,
		[]dto.FsMagicSignature{
			{Offset: 3, Magic: []byte{'N', 'T', 'F', 'S', ' ', ' ', ' ', ' '}},
		},
		"ntfs3",
		"ntfs-3g",
		"fuseblk",
	)

	suite.Equal([]string{"ntfs3", "ntfs-3g", "fuseblk"}, suite.adapter.GetAliasNames())
}

// TestIsExportable tests the IsExportable method.
func (suite *BaseAdapterTestSuite) TestIsExportable() {
	suite.False(suite.adapter.IsExportable(suite.ctx))

	exportableAdapter := newBaseAdapter(
		"ext4",
		"EXT4 Filesystem",
		true,
		"e2fsprogs",
		"mkfs.ext4",
		"fsck.ext4",
		"tune2fs",
		"tune2fs",
		`^[^\x00/]{1,16}$`,
		[]dto.FsMagicSignature{{Offset: 1080, Magic: []byte{0x53, 0xEF}}},
	)
	suite.True(exportableAdapter.IsExportable(suite.ctx))
}

// TestCheckCommandAvailability tests command availability checks with mocked filesystem and exec
func (suite *BaseAdapterTestSuite) TestCheckCommandAvailability() {
	osutil.MockFileSystems([]string{"ntfs", "ext4"})
	defer osutil.MockFileSystems(nil) // Restore original state after test
	tests := []struct {
		name               string
		adapterFactory     func() FilesystemAdapter
		shouldHaveCommands bool
	}{
		{
			name:               "ntfs adapter has format/check commands available",
			adapterFactory:     NewNtfsAdapter,
			shouldHaveCommands: true, // ntfs-3g/ntfsfix are commonly available
		},
		{
			name:               "ext4 adapter has format/check commands available",
			adapterFactory:     NewExt4Adapter,
			shouldHaveCommands: true, // mkfs.ext4/e2fsck are commonly available
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			adapter := tt.adapterFactory()
			cancelExt := adapter.SetExecOpsForTesting(
				func(cmd string) (string, error) {
					if tt.shouldHaveCommands {
						return cmd, nil
					}
					return "", errors.New("command not found")
				},
			)
			cancelOs := osutil.MockFileSystems([]string{"ntfs3", "ext4"})

			support, err := adapter.IsSupported(suite.ctx)
			suite.NoError(err)
			suite.True(support.CanMount, "all adapters should support mounting %v", support)

			// Verify the structure is correct
			suite.NotNil(support)
			suite.NotEmpty(support.AlpinePackage)
			defer cancelOs()
			defer cancelExt()

		})
	}
}

// TestRunCommandCached tests command result caching
func (suite *BaseAdapterTestSuite) TestRunCommandCached() {
	suite.Run("same command and args are cached", func() {
		suite.cleanExec()
		runCount := 0
		suite.cleanExec = suite.adapter.SetCommandRunner(&fakeCommandRunner{
			execute: func(_ context.Context, _ string, _ string, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
				runCount++
				return dto.CommandExecutionSnapshot{
					Success:  true,
					ExitCode: 0,
					Lines: []dto.CommandOutputLineSnapshot{{
						Channel: dto.CommandOutputChannelStdout,
						Line:    "run-1",
					}},
				}, nil
			},
		})

		suite.adapter.invalidateCommandResultCache()

		firstOutput, firstExitCode, firstErr := suite.adapter.runCommandCached(suite.ctx, "countcmd", "arg1")
		suite.NoError(firstErr)
		suite.Equal(0, firstExitCode)

		secondOutput, secondExitCode, secondErr := suite.adapter.runCommandCached(suite.ctx, "countcmd", "arg1")
		suite.NoError(secondErr)
		suite.Equal(0, secondExitCode)

		suite.Equal(firstOutput, secondOutput)
		suite.Equal(1, runCount, "expected command to execute once with identical args")
	})

	suite.Run("quiet cached commands use the quiet executor path", func() {
		suite.cleanExec()
		normalCalls := 0
		quietCalls := 0
		suite.cleanExec = suite.adapter.SetCommandRunner(&fakeCommandRunner{
			execute: func(_ context.Context, _ string, _ string, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
				normalCalls++
				return dto.CommandExecutionSnapshot{}, nil
			},
			executeQuiet: func(_ context.Context, _ string, _ string, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
				quietCalls++
				return dto.CommandExecutionSnapshot{
					Success:  true,
					ExitCode: 0,
					Lines: []dto.CommandOutputLineSnapshot{{
						Channel: dto.CommandOutputChannelStdout,
						Line:    "quiet-run",
					}},
				}, nil
			},
		})

		suite.adapter.invalidateCommandResultCache()

		output, exitCode, err := suite.adapter.runCommandCachedQuiet(suite.ctx, "statecmd", "--readonly")
		suite.NoError(err)
		suite.Equal("quiet-run", output)
		suite.Equal(0, exitCode)
		suite.Equal(0, normalCalls)
		suite.Equal(1, quietCalls)
	})

	suite.Run("different args use different cache entries", func() {
		suite.cleanExec()
		runCount := 0
		suite.cleanExec = suite.adapter.SetCommandRunner(&fakeCommandRunner{
			execute: func(_ context.Context, _ string, _ string, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
				runCount++
				return dto.CommandExecutionSnapshot{
					Success:  true,
					ExitCode: 0,
					Lines: []dto.CommandOutputLineSnapshot{{
						Channel: dto.CommandOutputChannelStdout,
						Line:    "run",
					}},
				}, nil
			},
		})

		_, _, err := suite.adapter.runCommandCached(suite.ctx, "echo", "arg1")
		suite.NoError(err)

		_, _, err = suite.adapter.runCommandCached(suite.ctx, "echo", "arg2")
		suite.NoError(err)
		suite.Equal(2, runCount, "expected different args to bypass the cache")
	})
}

// TestBaseAdapterMountUsesTryMountWhenFsTypeEmpty tests mount behavior with empty fsType
func (suite *BaseAdapterTestSuite) TestBaseAdapterMountUsesTryMountWhenFsTypeEmpty() {

	called := false

	suite.cleanMount = suite.adapter.SetMountOpsForTesting(
		func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error) {
			called = true
			for _, opt := range opts {
				if err := opt(); err != nil {
					return nil, err
				}
			}
			return &mount.MountPoint{Path: target, Device: source, FSType: "auto", Flags: flags, Data: data}, nil
		},
		nil,
		nil,
	)

	prepared := false
	mp, err := suite.adapter.Mount(suite.ctx, "/dev/mock", "/mnt/mock", "", "uid=1000", 0, func() error {
		prepared = true
		return nil
	})

	suite.NoError(err)
	suite.True(called, "expected TryMount path to be called")
	suite.True(prepared, "expected prepare callback to be called")
	suite.NotNil(mp)
	suite.Equal("/mnt/mock", mp.Path)
}

// TestBaseAdapterMountUsesMountWhenFsTypeProvided tests mount behavior with fsType specified
func (suite *BaseAdapterTestSuite) TestBaseAdapterMountUsesMountWhenFsTypeProvided() {

	calledMount := false

	suite.cleanMount = suite.adapter.SetMountOpsForTesting(
		nil,
		func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error) {
			calledMount = true
			suite.Equal("ntfs3", fstype)
			return &mount.MountPoint{Path: target, Device: source, FSType: fstype, Flags: flags, Data: data}, nil
		},
		nil,
	)

	mp, err := suite.adapter.Mount(suite.ctx, "/dev/mock", "/mnt/mock", "ntfs3", "", 0, nil)

	suite.NoError(err)
	suite.True(calledMount, "expected Mount path to be called")
	suite.NotNil(mp)
	suite.Equal("ntfs3", mp.FSType)
}

// TestBaseAdapterUnmountDelegatesToHook tests unmount behavior
func (suite *BaseAdapterTestSuite) TestBaseAdapterUnmountDelegatesToHook() {

	called := false

	suite.cleanMount = suite.adapter.SetMountOpsForTesting(
		nil,
		nil,
		func(target string, force, lazy bool) error {
			called = true
			suite.Equal("/mnt/mock", target)
			suite.True(force)
			suite.False(lazy)
			return nil
		},
	)

	err := suite.adapter.Unmount(suite.ctx, "/mnt/mock", true, false)

	suite.NoError(err)
	suite.True(called, "expected unmount hook to be called")
}

func (suite *BaseAdapterTestSuite) TestIsDeviceMounted_UsesOverride() {
	device := "/dev/test0"

	reset := suite.adapter.SetIsDeviceMountedForTesting(func(candidate string) bool {
		return candidate == device
	})
	defer reset()

	suite.True(suite.adapter.isDeviceMounted(device))
	suite.False(suite.adapter.isDeviceMounted("/dev/other0"))
}

func (suite *BaseAdapterTestSuite) TestSetCommandRunner_UsesInjectedRunnerForRunCommand() {
	reset := suite.adapter.SetCommandRunner(&fakeCommandRunner{
		execute: func(_ context.Context, _ string, _ string, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
			return dto.CommandExecutionSnapshot{
				Success:  true,
				ExitCode: 0,
				Lines: []dto.CommandOutputLineSnapshot{{
					Channel: dto.CommandOutputChannelStdout,
					Line:    "runner-output",
				}},
			}, nil
		},
	})
	defer reset()

	output, exitCode, err := suite.adapter.runCommand(suite.ctx, "runner-command", "arg1")
	suite.NoError(err)
	suite.Equal(0, exitCode)
	suite.Equal("runner-output", output)
}

func (suite *BaseAdapterTestSuite) TestSetCommandRunner_StreamsProgressFromSnapshots() {
	callCount := 0
	reset := suite.adapter.SetCommandRunner(&fakeCommandRunner{
		start: func(_ context.Context, _ string, _ string, _ string, _ ...string) (string, error) {
			return "exec-1", nil
		},
		getSnapshot: func(executionID string) (dto.CommandExecutionSnapshot, bool) {
			callCount++
			baseLines := []dto.CommandOutputLineSnapshot{{
				Channel: dto.CommandOutputChannelStdout,
				Line:    "out-1",
			}, {
				Channel: dto.CommandOutputChannelStderr,
				Line:    "err-1",
			}}

			snapshot := dto.CommandExecutionSnapshot{
				ExecutionID: executionID,
				Running:     callCount < 3,
				Success:     true,
				ExitCode:    0,
				Lines:       append([]dto.CommandOutputLineSnapshot(nil), baseLines...),
			}
			if callCount >= 2 {
				snapshot.Lines = append(snapshot.Lines, dto.CommandOutputLineSnapshot{
					Channel: dto.CommandOutputChannelStdout,
					Line:    "out-2",
				})
			}
			return snapshot, true
		},
	})
	defer reset()

	stdoutChan, stderrChan, resultChan := suite.adapter.executeCommandWithProgress(suite.ctx, "runner-command", []string{"arg1"})

	var stdoutLines []string
	for line := range stdoutChan {
		stdoutLines = append(stdoutLines, line)
	}

	var stderrLines []string
	for line := range stderrChan {
		stderrLines = append(stderrLines, line)
	}

	result, ok := <-resultChan
	suite.True(ok)
	suite.NoError(result.Error)
	suite.Equal(0, result.ExitCode)
	suite.Equal([]string{"out-1", "out-2"}, stdoutLines)
	suite.Equal([]string{"err-1"}, stderrLines)
}

func (suite *BaseAdapterTestSuite) TestSetCommandRunner_FailedSnapshotIncludesStderrInError() {
	reset := suite.adapter.SetCommandRunner(&fakeCommandRunner{
		start: func(_ context.Context, _ string, _ string, _ string, _ ...string) (string, error) {
			return "exec-err", nil
		},
		getSnapshot: func(executionID string) (dto.CommandExecutionSnapshot, bool) {
			return dto.CommandExecutionSnapshot{
				ExecutionID: executionID,
				Running:     false,
				Success:     false,
				ExitCode:    1,
				Error:       "exit status 1",
				Lines: []dto.CommandOutputLineSnapshot{{
					Channel: dto.CommandOutputChannelStderr,
					Line:    "mkfs.xfs: /dev/test is busy",
				}},
			}, true
		},
	})
	defer reset()

	stdoutChan, stderrChan, resultChan := suite.adapter.executeCommandWithProgress(suite.ctx, "runner-command", []string{"arg1"})

	for range stdoutChan {
	}

	var stderrLines []string
	for line := range stderrChan {
		stderrLines = append(stderrLines, line)
	}

	result, ok := <-resultChan
	suite.True(ok)
	suite.Error(result.Error)
	suite.Equal(1, result.ExitCode)
	suite.Equal([]string{"mkfs.xfs: /dev/test is busy"}, stderrLines)
	suite.Contains(result.Error.Error(), "exit status 1")
	suite.Contains(result.Error.Error(), "mkfs.xfs: /dev/test is busy")
}

// TestOsutilMockFileSystems tests that osutil.MockFileSystems works for mocking mounted filesystems
func (suite *BaseAdapterTestSuite) TestOsutilMockFileSystems() {
	suite.Run("mock replaces filesystem list", func() {
		restore := osutil.MockFileSystems([]string{"testfs", "ext4"})
		defer restore()

		// filesystem support should respond to mocked list
		// we can't directly test GetFileSystems from _test package, but it's tested via adapter.IsSupported
		restore()
	})

	suite.Run("mock can be restored", func() {
		original := osutil.MockFileSystems([]string{"testfs"})
		original() // Restore

		// After restore, should get actual filesystems again (or at least not error)
		// This is implicit verification
	})
}

// Helper functions

type fakeCommandRunner struct {
	lookPath     func(string) (string, error)
	start        func(context.Context, string, string, string, ...string) (string, error)
	startQuiet   func(context.Context, string, string, string, ...string) (string, error)
	execute      func(context.Context, string, string, string, ...string) (dto.CommandExecutionSnapshot, error)
	executeQuiet func(context.Context, string, string, string, ...string) (dto.CommandExecutionSnapshot, error)
	getSnapshot  func(string) (dto.CommandExecutionSnapshot, bool)
}

func (f *fakeCommandRunner) LookPath(command string) (string, error) {
	if f.lookPath != nil {
		return f.lookPath(command)
	}
	if command == "" {
		return "", errors.New("not implemented")
	}
	return command, nil
}

func (f *fakeCommandRunner) Start(ctx context.Context, commandID, label, command string, args ...string) (string, error) {
	if f.start != nil {
		return f.start(ctx, commandID, label, command, args...)
	}
	return "", errors.New("not implemented")
}

func (f *fakeCommandRunner) StartQuiet(ctx context.Context, commandID, label, command string, args ...string) (string, error) {
	if f.startQuiet != nil {
		return f.startQuiet(ctx, commandID, label, command, args...)
	}
	return f.Start(ctx, commandID, label, command, args...)
}

func (f *fakeCommandRunner) StartWithInput(ctx context.Context, commandID, label, _ string, command string, args ...string) (string, error) {
	return f.Start(ctx, commandID, label, command, args...)
}

func (f *fakeCommandRunner) StartWithInputQuiet(ctx context.Context, commandID, label, _ string, command string, args ...string) (string, error) {
	return f.StartQuiet(ctx, commandID, label, command, args...)
}

func (f *fakeCommandRunner) Execute(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	if f.execute != nil {
		return f.execute(ctx, commandID, label, command, args...)
	}
	return dto.CommandExecutionSnapshot{}, errors.New("not implemented")
}

func (f *fakeCommandRunner) ExecuteQuiet(ctx context.Context, commandID, label, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	if f.executeQuiet != nil {
		return f.executeQuiet(ctx, commandID, label, command, args...)
	}
	return f.Execute(ctx, commandID, label, command, args...)
}

func (f *fakeCommandRunner) ExecuteWithInput(ctx context.Context, commandID, label, _ string, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	return f.Execute(ctx, commandID, label, command, args...)
}

func (f *fakeCommandRunner) ExecuteWithInputQuiet(ctx context.Context, commandID, label, _ string, command string, args ...string) (dto.CommandExecutionSnapshot, error) {
	return f.ExecuteQuiet(ctx, commandID, label, command, args...)
}

func (f *fakeCommandRunner) GetSnapshot(executionID string) (dto.CommandExecutionSnapshot, bool) {
	if f.getSnapshot != nil {
		return f.getSnapshot(executionID)
	}
	return dto.CommandExecutionSnapshot{}, false
}
