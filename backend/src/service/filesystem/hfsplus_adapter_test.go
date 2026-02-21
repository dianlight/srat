package filesystem_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service/filesystem"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
)

type HfsplusAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
	cleanExec   func() // Optional cleanup function for tests that set exec ops
}

func TestHfsplusAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(HfsplusAdapterTestSuite))
}

func (suite *HfsplusAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewHfsplusAdapter()
	suite.Require().NotNil(suite.adapter)

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
}

func (suite *HfsplusAdapterTestSuite) TearDownTest() {
	if suite.cleanExec != nil {
		suite.cleanExec()
	}
}

func (suite *HfsplusAdapterTestSuite) TestGetName() {
	suite.Equal("hfsplus", suite.adapter.GetName())
}

func (suite *HfsplusAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "Hierarchical")
}

func (suite *HfsplusAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundUID := false
	foundForce := false
	for _, flag := range flags {
		if flag.Name == "uid" {
			foundUID = true
			suite.True(flag.NeedsValue)
		}
		if flag.Name == "force" {
			foundForce = true
			suite.False(flag.NeedsValue)
		}
	}

	suite.True(foundUID)
	suite.True(foundForce)
}

func (suite *HfsplusAdapterTestSuite) TestIsSupported() {
	restore := osutil.MockFileSystems([]string{"hfsplus", "ext4"})
	defer restore()

	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.True(support.CanFormat)
	suite.True(support.CanMount)
	suite.True(support.CanCheck)
	suite.False(support.CanSetLabel)
	suite.True(support.CanGetState)
	suite.Equal("hfsprogs", support.AlpinePackage)
	suite.Empty(support.MissingTools)
}

func (suite *HfsplusAdapterTestSuite) TestIsDeviceSupportedWithSignature() {
	signatures := suite.adapter.GetFsSignatureMagic()
	suite.NotEmpty(signatures)

	signature := signatures[0]
	path := createTempDeviceWithMagic(suite.T(), signature.Offset, signature.Magic)

	supported, err := suite.adapter.IsDeviceSupported(suite.ctx, path)
	suite.NoError(err)
	suite.True(supported)
}
