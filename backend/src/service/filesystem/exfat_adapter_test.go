package filesystem_test

import (
	"context"
	"testing"

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
	suite.Equal("exfatprogs", support.AlpinePackage)
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
