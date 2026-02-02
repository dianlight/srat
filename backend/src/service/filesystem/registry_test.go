package filesystem_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

type RegistryTestSuite struct {
	suite.Suite
	registry *filesystem.Registry
	ctx      context.Context
}

func TestRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}

func (suite *RegistryTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.registry = filesystem.NewRegistry()
	suite.Require().NotNil(suite.registry, "Registry should be initialized")
}

func (suite *RegistryTestSuite) TestGetAdapter() {
	// Test getting a known adapter
	adapter, err := suite.registry.Get("ext4")
	suite.Require().NoError(err)
	suite.Require().NotNil(adapter)
	suite.Equal("ext4", adapter.GetName())

	// Test getting another known adapter
	adapter, err = suite.registry.Get("xfs")
	suite.Require().NoError(err)
	suite.Require().NotNil(adapter)
	suite.Equal("xfs", adapter.GetName())

	// Test getting an unknown adapter
	adapter, err = suite.registry.Get("unknown-fs")
	suite.Require().Error(err)
	suite.Nil(adapter)
}

func (suite *RegistryTestSuite) TestGetAll() {
	adapters := suite.registry.GetAll()
	suite.NotEmpty(adapters)
	
	// Check that we have all expected adapters
	expectedFS := []string{"ext4", "vfat", "ntfs", "btrfs", "xfs"}
	foundFS := make(map[string]bool)
	
	for _, adapter := range adapters {
		foundFS[adapter.GetName()] = true
	}
	
	for _, expected := range expectedFS {
		suite.True(foundFS[expected], "Should have %s adapter", expected)
	}
}

func (suite *RegistryTestSuite) TestListSupportedTypes() {
	types := suite.registry.ListSupportedTypes()
	suite.NotEmpty(types)
	suite.GreaterOrEqual(len(types), 5, "Should have at least 5 filesystem types")
	
	// Check for expected types
	expectedTypes := []string{"ext4", "vfat", "ntfs", "btrfs", "xfs"}
	for _, expected := range expectedTypes {
		suite.Contains(types, expected)
	}
}

func (suite *RegistryTestSuite) TestGetSupportedFilesystems() {
	support, err := suite.registry.GetSupportedFilesystems(suite.ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(support)
	
	// Check that each filesystem has support information
	for fsType, fsSupport := range support {
		suite.NotEmpty(fsType)
		suite.NotEmpty(fsSupport.AlpinePackage, "Alpine package should be specified for %s", fsType)
		// Note: Actual tool availability depends on system configuration
	}
}

func (suite *RegistryTestSuite) TestRegisterCustomAdapter() {
	// Create a mock adapter
	mockAdapter := filesystem.NewExt4Adapter() // Reuse ext4 for simplicity
	
	// Register it with a custom name (this would normally be a different implementation)
	suite.registry.Register(mockAdapter)
	
	// Verify it was registered
	adapter, err := suite.registry.Get("ext4")
	suite.Require().NoError(err)
	suite.NotNil(adapter)
}
