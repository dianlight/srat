package filesystem_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

// BtrfsAdapterTestSuite tests the BtrfsAdapter
type BtrfsAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestBtrfsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(BtrfsAdapterTestSuite))
}

func (suite *BtrfsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewBtrfsAdapter()
	suite.Require().NotNil(suite.adapter)
}

func (suite *BtrfsAdapterTestSuite) TestGetName() {
	suite.Equal("btrfs", suite.adapter.GetName())
}

func (suite *BtrfsAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "tree")
}

func (suite *BtrfsAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundCompress := false
	foundSubvol := false
	for _, flag := range flags {
		if flag.Name == "compress" {
			foundCompress = true
			suite.True(flag.NeedsValue)
		}
		if flag.Name == "subvol" {
			foundSubvol = true
			suite.True(flag.NeedsValue)
		}
	}

	suite.True(foundCompress)
	suite.True(foundSubvol)
}

func (suite *BtrfsAdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.Equal("btrfs-progs", support.AlpinePackage)
}
