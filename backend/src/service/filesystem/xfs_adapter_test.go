package filesystem_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

// XfsAdapterTestSuite tests the XfsAdapter
type XfsAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestXfsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(XfsAdapterTestSuite))
}

func (suite *XfsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewXfsAdapter()
	suite.Require().NotNil(suite.adapter)
}

func (suite *XfsAdapterTestSuite) TestGetName() {
	suite.Equal("xfs", suite.adapter.GetName())
}

func (suite *XfsAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "XFS")
}

func (suite *XfsAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundInode64 := false
	foundAllocsize := false
	for _, flag := range flags {
		if flag.Name == "inode64" {
			foundInode64 = true
			suite.False(flag.NeedsValue)
		}
		if flag.Name == "allocsize" {
			foundAllocsize = true
			suite.True(flag.NeedsValue)
		}
	}

	suite.True(foundInode64)
	suite.True(foundAllocsize)
}

func (suite *XfsAdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.True(support.CanFormat)
	suite.True(support.CanMount)
	suite.True(support.CanCheck)
	suite.True(support.CanSetLabel)
	suite.True(support.CanGetState)
	suite.Equal("xfsprogs-extra", support.AlpinePackage)
	suite.Empty(support.MissingTools)
}
