package filesystem_test

import (
	"context"
	"testing"

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
	suite.Contains(desc, "Extended")
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
