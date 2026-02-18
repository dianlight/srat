package filesystem_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

type Ext4AdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestExt4AdapterTestSuite(t *testing.T) {
	suite.Run(t, new(Ext4AdapterTestSuite))
}

func (suite *Ext4AdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewExt4Adapter()
	suite.Require().NotNil(suite.adapter, "Ext4Adapter should be initialized")
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

	suite.NotEmpty(support.AlpinePackage)
	suite.Equal("e2fsprogs", support.AlpinePackage)
	// Note: Actual availability depends on system configuration
}

func (suite *Ext4AdapterTestSuite) TestGetFsSignatureMagic() {
	signatures := suite.adapter.GetFsSignatureMagic()
	suite.NotEmpty(signatures)
	suite.Len(signatures, 1)
	suite.Equal(int64(1080), signatures[0].Offset)
	suite.Equal([]byte{0x53, 0xEF}, signatures[0].Magic)
}

func (suite *Ext4AdapterTestSuite) TestFormat_NonExistentDevice() {
	// Test formatting a non-existent device
	err := suite.adapter.Format(suite.ctx, "/dev/nonexistent-device-12345", dto.FormatOptions{}, nil)
	suite.Error(err)
}

func (suite *Ext4AdapterTestSuite) TestCheck_NonExistentDevice() {
	// Test checking a non-existent device
	result, err := suite.adapter.Check(suite.ctx, "/dev/nonexistent-device-12345", dto.CheckOptions{}, nil)
	suite.Error(err)
	// Result may be zero-initialized or have error info
	_ = result
}

func (suite *Ext4AdapterTestSuite) TestGetLabel_NonExistentDevice() {
	// Test getting label from a non-existent device
	label, err := suite.adapter.GetLabel(suite.ctx, "/dev/nonexistent-device-12345")
	suite.Error(err)
	suite.Empty(label)
}

func (suite *Ext4AdapterTestSuite) TestSetLabel_NonExistentDevice() {
	// Test setting label on a non-existent device
	err := suite.adapter.SetLabel(suite.ctx, "/dev/nonexistent-device-12345", "testlabel")
	suite.Error(err)
}

func (suite *Ext4AdapterTestSuite) TestGetState_NonExistentDevice() {
	// Test getting state from a non-existent device
	state, err := suite.adapter.GetState(suite.ctx, "/dev/nonexistent-device-12345")
	suite.Error(err)
	// State may be zero-initialized or have error info
	_ = state
}

func (suite *Ext4AdapterTestSuite) TestFormat_WithCancelledContext() {
	// Test formatting with a cancelled context
	cancelledCtx, cancel := context.WithCancel(suite.ctx)
	cancel() // Cancel immediately

	err := suite.adapter.Format(cancelledCtx, "/tmp/test-device", dto.FormatOptions{}, nil)
	suite.Error(err)
}
