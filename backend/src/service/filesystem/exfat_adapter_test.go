package filesystem_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

type ExfatAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestExfatAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(ExfatAdapterTestSuite))
}

func (suite *ExfatAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewExfatAdapter()
	suite.Require().NotNil(suite.adapter)
}

func (suite *ExfatAdapterTestSuite) TestGetName() {
	suite.Equal("exfat", suite.adapter.GetName())
}

func (suite *ExfatAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "Extended")
}

func (suite *ExfatAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundUID := false
	foundIOCharset := false
	for _, flag := range flags {
		if flag.Name == "uid" {
			foundUID = true
			suite.True(flag.NeedsValue)
		}
		if flag.Name == "iocharset" {
			foundIOCharset = true
			suite.True(flag.NeedsValue)
		}
	}

	suite.True(foundUID)
	suite.True(foundIOCharset)
}

func (suite *ExfatAdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.True(support.CanFormat)
	suite.True(support.CanMount)
	suite.True(support.CanCheck)
	suite.True(support.CanSetLabel)
	suite.True(support.CanGetState)
	suite.Equal("exfatprogs", support.AlpinePackage)
	suite.Empty(support.MissingTools)
}

func (suite *ExfatAdapterTestSuite) TestIsDeviceSupportedWithSignature() {
	signatures := suite.adapter.GetFsSignatureMagic()
	suite.NotEmpty(signatures)

	signature := signatures[0]
	path := createTempDeviceWithMagic(suite.T(), signature.Offset, signature.Magic)

	supported, err := suite.adapter.IsDeviceSupported(suite.ctx, path)
	suite.NoError(err)
	suite.True(supported)
}

func (suite *ExfatAdapterTestSuite) TestGetLinuxFsModule() {
	module := suite.adapter.GetLinuxFsModule()
	suite.Equal("exfat", module)
}

func (suite *ExfatAdapterTestSuite) TestFormat_NonExistentDevice() {
	err := suite.adapter.Format(suite.ctx, "/dev/nonexistent-exfat-12345", dto.FormatOptions{}, nil)
	suite.Error(err)
}

func (suite *ExfatAdapterTestSuite) TestCheck_NonExistentDevice() {
	result, err := suite.adapter.Check(suite.ctx, "/dev/nonexistent-exfat-12345", dto.CheckOptions{}, nil)
	suite.Error(err)
	_ = result
}

func (suite *ExfatAdapterTestSuite) TestGetLabel_NonExistentDevice() {
	label, err := suite.adapter.GetLabel(suite.ctx, "/dev/nonexistent-exfat-12345")
	suite.Error(err)
	suite.Empty(label)
}

func (suite *ExfatAdapterTestSuite) TestSetLabel_NonExistentDevice() {
	err := suite.adapter.SetLabel(suite.ctx, "/dev/nonexistent-exfat-12345", "testlabel")
	suite.Error(err)
}

func (suite *ExfatAdapterTestSuite) TestGetState_NonExistentDevice() {
	state, err := suite.adapter.GetState(suite.ctx, "/dev/nonexistent-exfat-12345")
	suite.Error(err)
	_ = state
}
