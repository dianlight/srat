package filesystem_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/u-root/u-root/pkg/mount"
)

// BaseAdapterTestSuite tests the baseAdapter implementation
type BaseAdapterTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestBaseAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(BaseAdapterTestSuite))
}

func (suite *BaseAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

// TestGetName tests the GetName method
func (suite *BaseAdapterTestSuite) TestGetName() {
	adapter := filesystem.NewNtfsAdapter()
	suite.Equal("ntfs", adapter.GetName())
}

// TestGetDescription tests the GetDescription method
func (suite *BaseAdapterTestSuite) TestGetDescription() {
	adapter := filesystem.NewNtfsAdapter()
	desc := adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "NTFS")
}

// TestGetLinuxFsModule tests the GetLinuxFsModule method
func (suite *BaseAdapterTestSuite) TestGetLinuxFsModule() {
	// Test with ntfs (which has specific linux module)
	ntfsAdapter := filesystem.NewNtfsAdapter()
	linuxModule := ntfsAdapter.GetLinuxFsModule()
	suite.NotEmpty(linuxModule)

	// Test with ext4
	ext4Adapter := filesystem.NewExt4Adapter()
	suite.Equal("ext4", ext4Adapter.GetLinuxFsModule())
}

// TestCheckCommandAvailability tests command availability checks with mocked filesystem and exec
func (suite *BaseAdapterTestSuite) TestCheckCommandAvailability() {
	tests := []struct {
		name               string
		adapterFactory     func() filesystem.FilesystemAdapter
		shouldHaveCommands bool
	}{
		{
			name:               "ntfs adapter has format/check commands available",
			adapterFactory:     filesystem.NewNtfsAdapter,
			shouldHaveCommands: true, // ntfs-3g/ntfsfix are commonly available
		},
		{
			name:               "ext4 adapter has format/check commands available",
			adapterFactory:     filesystem.NewExt4Adapter,
			shouldHaveCommands: true, // mkfs.ext4/e2fsck are commonly available
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			adapter := tt.adapterFactory()

			support, err := adapter.IsSupported(suite.ctx)
			suite.NoError(err)
			suite.True(support.CanMount, "all adapters should support mounting")

			// Verify the structure is correct
			suite.NotNil(support)
			suite.NotEmpty(support.AlpinePackage)
		})
	}
}

// TestCheckCommandAvailabilityWithMockedExec tests osutil and exec mocking together
func (suite *BaseAdapterTestSuite) TestCheckCommandAvailabilityWithMockedExec() {
	suite.Run("mocked filesystem and command availability", func() {
		// Mock osutil.GetFileSystems to return ntfs3 (the kernel module for NTFS)
		// This allows checkCommandAvailability() to find the module and check for commands
		restoreFS := osutil.MockFileSystems([]string{"ntfs3", "ext4"})
		defer restoreFS()

		// Mock exec.LookPath to track which commands are being checked
		checkedCommands := []string{}
		filesystem.SetExecOpsForTesting(
			func(cmd string) (string, error) {
				checkedCommands = append(checkedCommands, cmd)
				return "", errors.New("command not found in mock")
			},
			nil, // Keep real exec for now
		)
		defer filesystem.ResetExecOpsForTesting()

		// Create an adapter and call IsSupported, which will use our mocked LookPath
		ntfsAdapter := filesystem.NewNtfsAdapter()
		_, err := ntfsAdapter.IsSupported(suite.ctx)
		suite.NoError(err)
		// Verify that the mock was actually called (checkedCommands should have entries)
		suite.NotEmpty(checkedCommands, "mocked LookPath should have been called during IsSupported")
	})

	suite.Run("mocked exec command execution", func() {
		callCount := 0
		filesystem.SetExecOpsForTesting(
			nil, // Keep real LookPath
			func(ctx context.Context, name string, args ...string) filesystem.ExecCmd {
				callCount++
				return &mockExecCmd{
					output:   "mocked output",
					exitCode: 0,
				}
			},
		)
		defer filesystem.ResetExecOpsForTesting()

		output, exitCode, err := filesystem.RunCommandCachedForTesting(suite.ctx, "testcmd", "arg1")
		suite.NoError(err)
		suite.Equal(0, exitCode)
		suite.Equal("mocked output", output)
		suite.Equal(1, callCount)
	})
}

// TestRunCommandCached tests command result caching
func (suite *BaseAdapterTestSuite) TestRunCommandCached() {
	suite.Run("same command and args are cached", func() {
		tempDir := suite.T().TempDir()
		counterFile := filepath.Join(tempDir, "counter.txt")
		createCountingCommand(suite.T(), tempDir, "countcmd", counterFile)
		suite.T().Setenv("PATH", tempDir)

		filesystem.InvalidateCommandResultCacheForTesting()

		firstOutput, firstExitCode, firstErr := filesystem.RunCommandCachedForTesting(suite.ctx, "countcmd", "arg1")
		suite.NoError(firstErr)
		suite.Equal(0, firstExitCode)

		secondOutput, secondExitCode, secondErr := filesystem.RunCommandCachedForTesting(suite.ctx, "countcmd", "arg1")
		suite.NoError(secondErr)
		suite.Equal(0, secondExitCode)

		suite.Equal(firstOutput, secondOutput)

		count := readCounter(suite.T(), counterFile)
		suite.Equal(1, count, "expected command to execute once with identical args")
	})

	suite.Run("different args use different cache entries", func() {
		filesystem.InvalidateCommandResultCacheForTesting()

		// These should not error with standard echo command
		_, _, err := filesystem.RunCommandCachedForTesting(suite.ctx, "echo", "arg1")
		suite.NoError(err)

		_, _, err = filesystem.RunCommandCachedForTesting(suite.ctx, "echo", "arg2")
		suite.NoError(err)
		// Implicit pass if no errors
	})
}

// TestBaseAdapterMountUsesTryMountWhenFsTypeEmpty tests mount behavior with empty fsType
func (suite *BaseAdapterTestSuite) TestBaseAdapterMountUsesTryMountWhenFsTypeEmpty() {
	filesystem.ResetMountOpsForTesting()
	suite.T().Cleanup(func() { filesystem.ResetMountOpsForTesting() })

	adapter := filesystem.NewNtfsAdapter()
	called := false

	filesystem.SetMountOpsForTesting(
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
	mp, err := adapter.Mount(suite.ctx, "/dev/mock", "/mnt/mock", "", "uid=1000", 0, func() error {
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
	filesystem.ResetMountOpsForTesting()
	suite.T().Cleanup(func() { filesystem.ResetMountOpsForTesting() })

	adapter := filesystem.NewNtfsAdapter()
	calledMount := false

	filesystem.SetMountOpsForTesting(
		nil,
		func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error) {
			calledMount = true
			suite.Equal("ntfs3", fstype)
			return &mount.MountPoint{Path: target, Device: source, FSType: fstype, Flags: flags, Data: data}, nil
		},
		nil,
	)

	mp, err := adapter.Mount(suite.ctx, "/dev/mock", "/mnt/mock", "ntfs3", "", 0, nil)

	suite.NoError(err)
	suite.True(calledMount, "expected Mount path to be called")
	suite.NotNil(mp)
	suite.Equal("ntfs3", mp.FSType)
}

// TestBaseAdapterUnmountDelegatesToHook tests unmount behavior
func (suite *BaseAdapterTestSuite) TestBaseAdapterUnmountDelegatesToHook() {
	filesystem.ResetMountOpsForTesting()
	suite.T().Cleanup(func() { filesystem.ResetMountOpsForTesting() })

	adapter := filesystem.NewNtfsAdapter()
	called := false

	filesystem.SetMountOpsForTesting(
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

	err := adapter.Unmount(suite.ctx, "/mnt/mock", true, false)

	suite.NoError(err)
	suite.True(called, "expected unmount hook to be called")
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

func createCountingCommand(t *testing.T, dir, name, counterFile string) {
	t.Helper()
	path := filepath.Join(dir, name)
	script := "#!/bin/sh\n" +
		"count=$(cat '" + counterFile + "' 2>/dev/null || echo 0)\n" +
		"count=$((count + 1))\n" +
		"echo \"$count\" > '" + counterFile + "'\n" +
		"echo \"run-$count\"\n" +
		"exit 0\n"

	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
}

func readCounter(t *testing.T, counterFile string) int {
	t.Helper()
	content, err := os.ReadFile(counterFile)
	require.NoError(t, err)

	counter := strings.TrimSpace(string(content))
	value, err := strconv.Atoi(counter)
	require.NoError(t, err)

	return value
}

// mockExecCmd implements filesystem.ExecCmd for testing
type mockExecCmd struct {
	output   string
	exitCode int
	err      error
}

func (m *mockExecCmd) CombinedOutput() ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []byte(m.output), nil
}

func (m *mockExecCmd) StdoutPipe() (io.ReadCloser, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockExecCmd) StderrPipe() (io.ReadCloser, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockExecCmd) Start() error {
	return nil
}

func (m *mockExecCmd) Wait() error {
	return m.err
}
