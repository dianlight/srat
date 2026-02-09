package filesystem_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

type ReiserfsAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestReiserfsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(ReiserfsAdapterTestSuite))
}

func (suite *ReiserfsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewReiserfsAdapter()
	suite.Require().NotNil(suite.adapter)
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
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.Equal("reiserfsprogs", support.AlpinePackage)
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
