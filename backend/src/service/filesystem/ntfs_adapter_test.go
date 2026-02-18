package filesystem_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

// NtfsAdapterTestSuite tests the NtfsAdapter
type NtfsAdapterTestSuite struct {
	suite.Suite
	adapter filesystem.FilesystemAdapter
	ctx     context.Context
}

func TestNtfsAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(NtfsAdapterTestSuite))
}

func (suite *NtfsAdapterTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.adapter = filesystem.NewNtfsAdapter()
	suite.Require().NotNil(suite.adapter)
}

func (suite *NtfsAdapterTestSuite) TestGetName() {
	suite.Equal("ntfs", suite.adapter.GetName())
}

func (suite *NtfsAdapterTestSuite) TestGetDescription() {
	desc := suite.adapter.GetDescription()
	suite.NotEmpty(desc)
	suite.Contains(desc, "NTFS")
}

func (suite *NtfsAdapterTestSuite) TestGetMountFlags() {
	flags := suite.adapter.GetMountFlags()
	suite.NotEmpty(flags)

	foundPermissions := false
	for _, flag := range flags {
		if flag.Name == "permissions" {
			foundPermissions = true
			suite.False(flag.NeedsValue)
		}
	}

	suite.True(foundPermissions)
}

func (suite *NtfsAdapterTestSuite) TestIsSupported() {
	support, err := suite.adapter.IsSupported(suite.ctx)
	suite.NoError(err)
	suite.Equal("ntfs-3g-progs", support.AlpinePackage)
}
