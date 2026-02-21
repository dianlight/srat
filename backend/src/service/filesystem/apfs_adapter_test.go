package filesystem_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service/filesystem"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
)

type ApfsAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	clean   func()
	ctx     context.Context
}

func TestApfsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(ApfsAdapterTestSuite))
}

func (suite *ApfsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewApfsAdapter()
	suite.Require().NotNil(suite.adapter)
	controller := mock.NewMockController(suite.T())
	execMock := mock.Mock[filesystem.ExecCmd](controller)
	suite.clean = suite.adapter.SetExecOpsForTesting(
		func(cmd string) (string, error) {
			if cmd == "apfs-fuse" || cmd == "apfsutil" {
				return cmd, nil
			}
			return "", errors.New("command not found")
		}, func(ctx context.Context, cmd string, args ...string) filesystem.ExecCmd {
			return execMock
		})
}

func (suite *ApfsAdapterTestSuite) TearDownTest() {
	if suite.clean != nil {
		suite.clean()
	}
}

func (suite *ApfsAdapterTestSuite) TestGetName() {
	suite.Equal("apfs", suite.adapter.GetName())
}

func (suite *ApfsAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "Apple")
}

func (suite *ApfsAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundUID := false
	foundVol := false
	for _, flag := range flags {
		if flag.Name == "uid" {
			foundUID = true
			suite.True(flag.NeedsValue)
		}
		if flag.Name == "vol" {
			foundVol = true
			suite.True(flag.NeedsValue)
		}
	}

	suite.True(foundUID)
	suite.True(foundVol)
}

func (suite *ApfsAdapterTestSuite) TestIsSupported() {
	restore := osutil.MockFileSystems([]string{"apfs", "ext4"})
	defer restore()

	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.False(support.CanFormat)
	suite.True(support.CanMount)
	suite.False(support.CanCheck)
	suite.False(support.CanSetLabel)
	suite.True(support.CanGetState)
	suite.Equal("apfs-fuse", support.AlpinePackage)
	suite.Empty(support.MissingTools)
}

func (suite *ApfsAdapterTestSuite) TestIsDeviceSupportedWithSignature() {
	signatures := suite.adapter.GetFsSignatureMagic()
	suite.NotEmpty(signatures)

	signature := signatures[0]
	path := createTempDeviceWithMagic(suite.T(), signature.Offset, signature.Magic)

	supported, err := suite.adapter.IsDeviceSupported(suite.ctx, path)
	suite.NoError(err)
	suite.True(supported)
}
