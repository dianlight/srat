package filesystem

import (
	"context"
	"errors"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type XfsMissingToolTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestXfsMissingToolTestSuite(t *testing.T) {
	suite.Run(t, new(XfsMissingToolTestSuite))
}

func (suite *XfsMissingToolTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func TestIsMissingFilesystemToolError(t *testing.T) {
	assert.False(t, IsMissingFilesystemToolError(nil))
	assert.True(t, IsMissingFilesystemToolError(errors.New(`exec: "xfs_admin": executable file not found in $PATH`)))
	assert.True(t, IsMissingFilesystemToolError(errors.New(`command not found: xfs_admin`)))
	assert.True(t, IsMissingFilesystemToolError(errors.New(`EXEC: "XFS_ADMIN": EXECUTABLE FILE NOT FOUND IN $PATH`)))
	assert.False(t, IsMissingFilesystemToolError(errors.New("permission denied")))
	assert.False(t, IsMissingFilesystemToolError(errors.New("no such file or directory")))
	assert.False(t, IsMissingFilesystemToolError(errors.New("tune2fs failed with exit code 1")))
}

func (suite *XfsMissingToolTestSuite) TestGetLabel_MissingBinaryReturnsUnknown() {
	adapter := NewXfsAdapter().(*XfsAdapter)
	resetExec := adapter.SetExecOpsForTesting(func(cmd string) (string, error) {
		return "", errors.New(`exec: "xfs_admin": executable file not found in $PATH`)
	})
	defer resetExec()
	adapter.invalidateCommandResultCache()
	defer adapter.invalidateCommandResultCache()

	label, err := adapter.GetLabel(suite.ctx, "/dev/disk/by-id/usb-Crucial__CT480M500SSD_ABCDEFA74566-0:0-part2")
	suite.NoError(err)
	suite.Empty(label)
}

func (suite *XfsMissingToolTestSuite) TestGetLabel_MissingBinaryRaceReturnsUnknown() {
	adapter := NewXfsAdapter().(*XfsAdapter)
	resetRunner := adapter.SetCommandRunner(&fakeCommandRunner{
		lookPath: func(command string) (string, error) {
			return command, nil
		},
		execute: func(_ context.Context, _ string, _ string, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
			return dto.CommandExecutionSnapshot{ExitCode: -1},
				errors.New(`exec: "xfs_admin": executable file not found in $PATH`)
		},
	})
	defer resetRunner()
	adapter.invalidateCommandResultCache()
	defer adapter.invalidateCommandResultCache()

	label, err := adapter.GetLabel(suite.ctx, "/dev/race-missing-xfs-label")
	suite.NoError(err)
	suite.Empty(label)
}

func (suite *XfsMissingToolTestSuite) TestGetLabel_GenuineFailureStillReports() {
	adapter := NewXfsAdapter().(*XfsAdapter)
	resetRunner := adapter.SetCommandRunner(&fakeCommandRunner{
		lookPath: func(command string) (string, error) {
			return command, nil
		},
		execute: func(_ context.Context, _ string, _ string, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
			return dto.CommandExecutionSnapshot{
				Success:  true,
				ExitCode: 1,
				Lines: []dto.CommandOutputLineSnapshot{{
					Channel: dto.CommandOutputChannelStdout,
					Line:    "xfs_admin: /dev/fake is not a valid XFS filesystem",
				}},
			}, nil
		},
	})
	defer resetRunner()
	adapter.invalidateCommandResultCache()
	defer adapter.invalidateCommandResultCache()

	label, err := adapter.GetLabel(suite.ctx, "/dev/fake-xfs-genuine-failure")
	require.Error(suite.T(), err)
	assert.Empty(suite.T(), label)
	assert.Contains(suite.T(), err.Error(), "xfs_admin")
}

func (suite *XfsMissingToolTestSuite) TestGetLabel_SuccessParsesLabel() {
	adapter := NewXfsAdapter().(*XfsAdapter)
	resetRunner := adapter.SetCommandRunner(&fakeCommandRunner{
		lookPath: func(command string) (string, error) {
			return command, nil
		},
		execute: func(_ context.Context, _ string, _ string, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
			return dto.CommandExecutionSnapshot{
				Success:  true,
				ExitCode: 0,
				Lines: []dto.CommandOutputLineSnapshot{{
					Channel: dto.CommandOutputChannelStdout,
					Line:    `label = "mylabel"`,
				}},
			}, nil
		},
	})
	defer resetRunner()
	adapter.invalidateCommandResultCache()
	defer adapter.invalidateCommandResultCache()

	label, err := adapter.GetLabel(suite.ctx, "/dev/xfs-label-success-1353")
	suite.NoError(err)
	suite.Equal("mylabel", label)
}

func (suite *XfsMissingToolTestSuite) TestGetLabel_EmptyOutputReturnsEmpty() {
	adapter := NewXfsAdapter().(*XfsAdapter)
	resetRunner := adapter.SetCommandRunner(&fakeCommandRunner{
		lookPath: func(command string) (string, error) {
			return command, nil
		},
		execute: func(_ context.Context, _ string, _ string, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
			return dto.CommandExecutionSnapshot{
				Success:  true,
				ExitCode: 0,
			}, nil
		},
	})
	defer resetRunner()
	adapter.invalidateCommandResultCache()
	defer adapter.invalidateCommandResultCache()

	label, err := adapter.GetLabel(suite.ctx, "/dev/xfs-label-empty-1353")
	suite.NoError(err)
	suite.Empty(label)
}

func (suite *XfsMissingToolTestSuite) TestGetState_MissingBinaryReturnsUnknown() {
	adapter := NewXfsAdapter().(*XfsAdapter)
	resetExec := adapter.SetExecOpsForTesting(func(cmd string) (string, error) {
		return "", errors.New(`exec: "xfs_repair": executable file not found in $PATH`)
	})
	defer resetExec()

	state, err := adapter.GetState(suite.ctx, "/dev/missing-xfs-state")
	suite.NoError(err)
	suite.Equal("Unknown", state.StateDescription)
}

func (suite *XfsMissingToolTestSuite) TestGetState_MissingBinaryRaceReturnsUnknown() {
	adapter := NewXfsAdapter().(*XfsAdapter)
	resetRunner := adapter.SetCommandRunner(&fakeCommandRunner{
		lookPath: func(command string) (string, error) {
			return command, nil
		},
		execute: func(_ context.Context, _ string, _ string, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
			return dto.CommandExecutionSnapshot{ExitCode: -1},
				errors.New(`exec: "xfs_repair": executable file not found in $PATH`)
		},
	})
	defer resetRunner()
	adapter.invalidateCommandResultCache()
	defer adapter.invalidateCommandResultCache()
	resetMounted := adapter.SetIsDeviceMountedForTesting(func(device string) bool { return false })
	defer resetMounted()

	state, err := adapter.GetState(suite.ctx, "/dev/race-missing-xfs-state")
	suite.NoError(err)
	suite.Equal("Unknown", state.StateDescription)
}

func (suite *XfsMissingToolTestSuite) TestGetState_SuccessClean() {
	adapter := NewXfsAdapter().(*XfsAdapter)
	resetRunner := adapter.SetCommandRunner(&fakeCommandRunner{
		lookPath: func(command string) (string, error) {
			return command, nil
		},
		execute: func(_ context.Context, _ string, _ string, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
			return dto.CommandExecutionSnapshot{Success: true, ExitCode: 0}, nil
		},
	})
	defer resetRunner()
	adapter.invalidateCommandResultCache()
	defer adapter.invalidateCommandResultCache()
	resetMounted := adapter.SetIsDeviceMountedForTesting(func(device string) bool { return false })
	defer resetMounted()

	state, err := adapter.GetState(suite.ctx, "/dev/xfs-state-clean-1353")
	suite.NoError(err)
	suite.Equal("Clean", state.StateDescription)
	suite.True(state.IsClean)
	suite.False(state.HasErrors)
}

func (suite *XfsMissingToolTestSuite) TestGetState_CorrectableErrors() {
	adapter := NewXfsAdapter().(*XfsAdapter)
	resetRunner := adapter.SetCommandRunner(&fakeCommandRunner{
		lookPath: func(command string) (string, error) {
			return command, nil
		},
		execute: func(_ context.Context, _ string, _ string, _ string, _ ...string) (dto.CommandExecutionSnapshot, error) {
			return dto.CommandExecutionSnapshot{Success: true, ExitCode: 1}, nil
		},
	})
	defer resetRunner()
	adapter.invalidateCommandResultCache()
	defer adapter.invalidateCommandResultCache()
	resetMounted := adapter.SetIsDeviceMountedForTesting(func(device string) bool { return false })
	defer resetMounted()

	state, err := adapter.GetState(suite.ctx, "/dev/xfs-state-correctable-1353")
	suite.NoError(err)
	suite.Equal("Has correctable errors", state.StateDescription)
	suite.False(state.IsClean)
	suite.True(state.HasErrors)
}
