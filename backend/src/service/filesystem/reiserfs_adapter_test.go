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

type ReiserfsAdapterTestSuite struct {
	suite.Suite
	adapter    filesystem.FilesystemAdapter
	ctx        context.Context
	cleanExec  func() // Optional cleanup function for tests that set exec ops
	cleanGetFs func() // Optional cleanup function for tests that set GetFs ops
}

func TestReiserfsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(ReiserfsAdapterTestSuite))
}

func (suite *ReiserfsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewReiserfsAdapter()
	suite.Require().NotNil(suite.adapter)

	controller := mock.NewMockController(suite.T())
	execMock := mock.Mock[filesystem.ExecCmd](controller)
	mock.When(execMock.StdoutPipe()).ThenReturn(io.NopCloser(strings.NewReader("")), nil)
	mock.When(execMock.StderrPipe()).ThenReturn(io.NopCloser(strings.NewReader("")), nil)
	mock.When(execMock.Start()).ThenReturn(nil)
	mock.When(execMock.Wait()).ThenReturn(nil)
	suite.cleanExec = suite.adapter.SetExecOpsForTesting(
		func(cmd string) (string, error) {
			return "", errors.New("command not found")
		},
		func(ctx context.Context, cmd string, args ...string) filesystem.ExecCmd {
			return execMock
		},
	)
	suite.cleanGetFs = suite.adapter.SetGetFilesystemsForTesting(func() ([]string, error) {
		return []string{"reiserfs", "ext4"}, nil
	})
}

func (suite *ReiserfsAdapterTestSuite) TearDownTest() {
	if suite.cleanExec != nil {
		suite.cleanExec()
	}
	if suite.cleanGetFs != nil {
		suite.cleanGetFs()
	}
}

func (suite *ReiserfsAdapterTestSuite) TestGetName() {
	suite.Equal("reiserfs", suite.adapter.GetName())
}

func (suite *ReiserfsAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "Reiser")
}

func (suite *ReiserfsAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundHash := false
	foundNoLog := false
	for _, flag := range flags {
		if flag.Name == "hash" {
			foundHash = true
			suite.True(flag.NeedsValue)
		}
		if flag.Name == "nolog" {
			foundNoLog = true
			suite.False(flag.NeedsValue)
		}
	}

	suite.True(foundHash)
	suite.True(foundNoLog)
}

func (suite *ReiserfsAdapterTestSuite) TestIsSupported() {
	restore := osutil.MockFileSystems([]string{"reiserfs", "ext4"})
	defer restore()

	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.False(support.CanFormat)
	suite.True(support.CanMount)
	suite.False(support.CanCheck)
	suite.False(support.CanSetLabel)
	suite.False(support.CanGetState)
	suite.Equal("reiserfsprogs", support.AlpinePackage)
	suite.NotEmpty(support.MissingTools)
}

func (suite *ReiserfsAdapterTestSuite) TestIsDeviceSupportedWithSignature() {
	signatures := suite.adapter.GetFsSignatureMagic()
	suite.NotEmpty(signatures)

	signature := signatures[0]
	path := createTempDeviceWithMagic(suite.T(), signature.Offset, signature.Magic)

	supported, err := suite.adapter.IsDeviceSupported(suite.ctx, path)
	suite.NoError(err)
	suite.True(supported)
}
