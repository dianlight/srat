package filesystem_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service/filesystem"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

type Ext4AdapterTestSuite struct {
	suite.Suite
	adapter    filesystem.FilesystemAdapter
	ctx        context.Context
	cleanExec  func() // Optional cleanup function for tests that set exec ops
	cleanGetFs func() // Optional cleanup function for tests that set GetFs ops
}

func TestExt4AdapterTestSuite(t *testing.T) {
	suite.Run(t, new(Ext4AdapterTestSuite))
}

func (suite *Ext4AdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewExt4Adapter()
	suite.Require().NotNil(suite.adapter, "Ext4Adapter should be initialized")

	// Mock exec operations for testing
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

	// Mock GetFilesystems for testing
	suite.cleanGetFs = suite.adapter.SetGetFilesystemsForTesting(func() ([]string, error) {
		return []string{"ext4", "xfs"}, nil
	})
}

func (suite *Ext4AdapterTestSuite) TearDownTest() {
	if suite.cleanExec != nil {
		suite.cleanExec()
	}
	if suite.cleanGetFs != nil {
		suite.cleanGetFs()
	}
}

func (suite *Ext4AdapterTestSuite) TestGetName() {
	name := suite.adapter.GetName()
	suite.Equal("ext4", name)
}

func (suite *Ext4AdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "EXT4")
}

func (suite *Ext4AdapterTestSuite) TestGetLinuxFsModule() {
	module := suite.adapter.GetLinuxFsModule()
	suite.Equal("ext4", module)
}

func (suite *Ext4AdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	// Check for some expected flags
	foundData := false
	foundDiscard := false
	for _, flag := range flags {
		if flag.Name == "data" {
			foundData = true
			suite.True(flag.NeedsValue, "data flag should need a value")
		}
		if flag.Name == "discard" {
			foundDiscard = true
			suite.False(flag.NeedsValue, "discard flag should not need a value")
		}
	}

	suite.True(foundData, "Should have 'data' mount flag")
	suite.True(foundDiscard, "Should have 'discard' mount flag")
}

func (suite *Ext4AdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.Require().NoError(err)
	suite.True(support.CanFormat, "ext4 should support formatting")
	suite.True(support.CanMount, "ext4 should support mounting")
	suite.True(support.CanCheck, "ext4 should support checking")
	suite.True(support.CanSetLabel, "ext4 should support setting label")
	suite.True(support.CanGetState, "ext4 should support getting state")

	suite.Equal("e2fsprogs", support.AlpinePackage)
	suite.Empty(support.MissingTools)
}

func (suite *Ext4AdapterTestSuite) TestGetFsSignatureMagic() {
	signatures := suite.adapter.GetFsSignatureMagic()
	suite.NotEmpty(signatures)
	suite.Len(signatures, 1)
	suite.Equal(int64(1080), signatures[0].Offset)
	suite.Equal([]byte{0x53, 0xEF}, signatures[0].Magic)
}

func (suite *Ext4AdapterTestSuite) TestFormat_WithCancelledContext() {
	// Test formatting with a cancelled context
	cancelledCtx, cancel := context.WithCancel(suite.ctx)
	cancel() // Cancel immediately

	err := suite.adapter.Format(cancelledCtx, "/tmp/test-device", dto.FormatOptions{}, nil)
	suite.Error(err)
}
