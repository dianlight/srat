package filesystem_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

type F2fsAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestF2fsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(F2fsAdapterTestSuite))
}

func (suite *F2fsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewF2fsAdapter()
	suite.Require().NotNil(suite.adapter)
}

func (suite *F2fsAdapterTestSuite) TestGetName() {
	suite.Equal("f2fs", suite.adapter.GetName())
}

func (suite *F2fsAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "Flash")
}

func (suite *F2fsAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundBackgroundGC := false
	foundDiscard := false
	for _, flag := range flags {
		if flag.Name == "background_gc" {
			foundBackgroundGC = true
			suite.True(flag.NeedsValue)
		}
		if flag.Name == "discard" {
			foundDiscard = true
			suite.False(flag.NeedsValue)
		}
	}

	suite.True(foundBackgroundGC)
	suite.True(foundDiscard)
}

func (suite *F2fsAdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.True(support.CanFormat)
	suite.True(support.CanMount)
	suite.True(support.CanCheck)
	suite.False(support.CanSetLabel)
	suite.True(support.CanGetState)
	suite.Equal("f2fs-tools", support.AlpinePackage)
	suite.Empty(support.MissingTools)
}

func (suite *F2fsAdapterTestSuite) TestIsDeviceSupportedWithSignature() {
	signatures := suite.adapter.GetFsSignatureMagic()
	suite.NotEmpty(signatures)

	signature := signatures[0]
	path := createTempDeviceWithMagic(suite.T(), signature.Offset, signature.Magic)

	supported, err := suite.adapter.IsDeviceSupported(suite.ctx, path)
	suite.NoError(err)
	suite.True(supported)
}
