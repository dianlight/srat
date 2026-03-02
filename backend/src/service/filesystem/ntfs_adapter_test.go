package filesystem_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
)

// NtfsAdapterTestSuite tests the NtfsAdapter
type NtfsAdapterTestSuite struct {
	suite.Suite
	adapter     filesystem.FilesystemAdapter
	ntfsAdapter *filesystem.NtfsAdapter
	ctx         context.Context
	cleanExec   func() // Optional cleanup function for tests that set exec ops
	cleanGetFs  func() // Optional cleanup function for tests that set GetFs ops
	cleanMount  func() // Optional cleanup function for tests that set mounted state ops
}

func TestNtfsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(NtfsAdapterTestSuite))
}

func (suite *NtfsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewNtfsAdapter()
	suite.Require().NotNil(suite.adapter)

	ntfsAdapter, ok := suite.adapter.(*filesystem.NtfsAdapter)
	suite.Require().True(ok)
	suite.ntfsAdapter = ntfsAdapter

	controller := mock.NewMockController(suite.T())
	execMock := mock.Mock[filesystem.ExecCmd](controller)
	mock.When(execMock.StdoutPipe()).ThenReturn(io.NopCloser(strings.NewReader("")), nil)
	mock.When(execMock.StderrPipe()).ThenReturn(io.NopCloser(strings.NewReader("")), nil)
	mock.When(execMock.Start()).ThenReturn(nil)
	mock.When(execMock.Wait()).ThenReturn(nil)
	suite.cleanExec = suite.adapter.SetExecOpsForTesting(
		func(cmd string) (string, error) {
			if cmd != "" {
				return cmd, nil
			}
			return "", errors.New("command not found")
		},
		func(ctx context.Context, cmd string, args ...string) filesystem.ExecCmd {
			return execMock
		},
	)
	suite.cleanGetFs = suite.adapter.SetGetFilesystemsForTesting(func() ([]string, error) {
		return []string{"ntfs3", "exfat"}, nil
	})
}

func (suite *NtfsAdapterTestSuite) TearDownTest() {
	if suite.cleanExec != nil {
		suite.cleanExec()
	}
	if suite.cleanGetFs != nil {
		suite.cleanGetFs()
	}
	if suite.cleanMount != nil {
		suite.cleanMount()
	}
}

func (suite *NtfsAdapterTestSuite) TestGetName() {
	suite.Equal("ntfs", suite.adapter.GetName())
}

func (suite *NtfsAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "NTFS")
}

func (suite *NtfsAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundPermissions := false
	for _, flag := range flags {
		if flag.Name == "permissions" {
			foundPermissions = true
			suite.False(flag.NeedsValue)
		}
	}

	suite.True(foundPermissions)
}

func (suite *NtfsAdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.True(support.CanFormat)
	suite.True(support.CanMount)
	suite.True(support.CanCheck)
	suite.True(support.CanSetLabel)
	suite.True(support.CanGetState)
	suite.Equal("ntfs-3g-progs", support.AlpinePackage)
	suite.Empty(support.MissingTools)
}

func (suite *NtfsAdapterTestSuite) TestGetState_MountedWithoutCachedState_AssumesClean() {
	device := "/dev/sda1"

	suite.cleanMount = suite.ntfsAdapter.SetIsDeviceMountedForTesting(func(mountedDevice string) bool {
		return mountedDevice == device
	})

	state, err := suite.adapter.GetState(suite.ctx, device)
	suite.NoError(err)
	suite.True(state.IsMounted)
	suite.True(state.IsClean)
	suite.False(state.HasErrors)
	suite.Equal("Mounted (no previous unmounted state; assuming clean)", state.StateDescription)
	suite.Equal("assumed_clean_mounted", state.AdditionalInfo["stateSource"])
}

func (suite *NtfsAdapterTestSuite) TestGetState_MountedWithCachedState_ReturnsLastUnmountedState() {
	device := "/dev/sdb1"
	mounted := false

	suite.cleanMount = suite.ntfsAdapter.SetIsDeviceMountedForTesting(func(mountedDevice string) bool {
		return mounted && mountedDevice == device
	})

	stateWhenUnmounted, err := suite.adapter.GetState(suite.ctx, device)
	suite.NoError(err)
	suite.False(stateWhenUnmounted.IsMounted)

	mounted = true
	stateWhenMounted, err := suite.adapter.GetState(suite.ctx, device)
	suite.NoError(err)

	suite.True(stateWhenMounted.IsMounted)
	suite.Equal(stateWhenUnmounted.IsClean, stateWhenMounted.IsClean)
	suite.Equal(stateWhenUnmounted.HasErrors, stateWhenMounted.HasErrors)
	suite.Equal("cached_unmounted_state", stateWhenMounted.AdditionalInfo["stateSource"])
	suite.Contains(stateWhenMounted.StateDescription, "Mounted (last known:")
}
