package filesystem_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

// VfatAdapterTestSuite tests the VfatAdapter
type VfatAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestVfatAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(VfatAdapterTestSuite))
}

func (suite *VfatAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewVfatAdapter()
	suite.Require().NotNil(suite.adapter)
}

func (suite *VfatAdapterTestSuite) TestGetName() {
	suite.Equal("vfat", suite.adapter.GetName())
}

func (suite *VfatAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "FAT")
}

func (suite *VfatAdapterTestSuite) TestGetMountFlags() {
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

func (suite *VfatAdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.True(support.CanFormat)
	suite.True(support.CanMount)
	suite.True(support.CanCheck)
	suite.True(support.CanSetLabel)
	suite.True(support.CanGetState)
	suite.Equal("dosfstools", support.AlpinePackage)
	suite.Empty(support.MissingTools)
}
