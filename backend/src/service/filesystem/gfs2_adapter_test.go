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

type Gfs2AdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestGfs2AdapterTestSuite(t *testing.T) {
	suite.Run(t, new(Gfs2AdapterTestSuite))
}

func (suite *Gfs2AdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewGfs2Adapter()
	suite.Require().NotNil(suite.adapter)

	controller := mock.NewMockController(suite.T())
	execMock := mock.Mock[filesystem.ExecCmd](controller)
	mock.When(execMock.StdoutPipe()).ThenReturn(io.NopCloser(strings.NewReader("")), nil)
	mock.When(execMock.StderrPipe()).ThenReturn(io.NopCloser(strings.NewReader("")), nil)
	mock.When(execMock.Start()).ThenReturn(nil)
	mock.When(execMock.Wait()).ThenReturn(nil)
	filesystem.SetExecOpsForTesting(
		func(cmd string) (string, error) {
			return "", errors.New("command not found")
		},
		func(ctx context.Context, cmd string, args ...string) filesystem.ExecCmd {
			return execMock
		},
	)
}

func (suite *Gfs2AdapterTestSuite) TearDownTest() {
	filesystem.ResetExecOpsForTesting()
}

func (suite *Gfs2AdapterTestSuite) TestGetName() {
	suite.Equal("gfs2", suite.adapter.GetName())
}

func (suite *Gfs2AdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "Global")
}

func (suite *Gfs2AdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundLockProto := false
	foundSpectator := false
	for _, flag := range flags {
		if flag.Name == "lockproto" {
			foundLockProto = true
			suite.True(flag.NeedsValue)
		}
		if flag.Name == "spectator" {
			foundSpectator = true
			suite.False(flag.NeedsValue)
		}
	}

	suite.True(foundLockProto)
	suite.True(foundSpectator)
}

func (suite *Gfs2AdapterTestSuite) TestIsSupported() {
	restore := osutil.MockFileSystems([]string{"gfs2", "ext4"})
	defer restore()

	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.False(support.CanFormat)
	suite.True(support.CanMount)
	suite.False(support.CanCheck)
	suite.False(support.CanSetLabel)
	suite.False(support.CanGetState)
	suite.Equal("gfs2-utils", support.AlpinePackage)
	suite.NotEmpty(support.MissingTools)
}

func (suite *Gfs2AdapterTestSuite) TestIsDeviceSupportedWithSignature() {
	signatures := suite.adapter.GetFsSignatureMagic()
	suite.NotEmpty(signatures)

	signature := signatures[0]
	path := createTempDeviceWithMagic(suite.T(), signature.Offset, signature.Magic)

	supported, err := suite.adapter.IsDeviceSupported(suite.ctx, path)
	suite.NoError(err)
	suite.True(supported)
}
