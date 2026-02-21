package filesystem_test

import (
	"context"
	"os"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
)

type DetectionTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestDetectionTestSuite(t *testing.T) {
	suite.Run(t, new(DetectionTestSuite))
}

func (suite *DetectionTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *DetectionTestSuite) TestIsDeviceSupported_NonExistentDevice() {
	// Test with ext4 adapter
	ext4Adapter := filesystem.NewExt4Adapter()

	supported, err := ext4Adapter.IsDeviceSupported(suite.ctx, "/dev/nonexistent")
	suite.Error(err)
	suite.False(supported)
}

func (suite *DetectionTestSuite) TestIsDeviceSupported_EmptyDevice() {
	// Create a temporary empty file
	tmpFile, err := os.CreateTemp("", "empty-device-*")
	suite.Require().NoError(err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Test with ext4 adapter
	ext4Adapter := filesystem.NewExt4Adapter()

	supported, err := ext4Adapter.IsDeviceSupported(suite.ctx, tmpFile.Name())
	// Empty file should return false (no error, but not supported)
	suite.NoError(err)
	suite.False(supported)
}

// Test that all adapters have IsDeviceSupported implemented
func (suite *DetectionTestSuite) TestAllAdaptersHaveIsDeviceSupported() {
	adapters := []filesystem.FilesystemAdapter{
		filesystem.NewExt4Adapter(),
		filesystem.NewVfatAdapter(),
		filesystem.NewNtfsAdapter(),
		filesystem.NewBtrfsAdapter(),
		filesystem.NewXfsAdapter(),
	}

	// Create a temp file that won't match any signatures
	tmpFile, err := os.CreateTemp("", "test-device-*")
	suite.Require().NoError(err)
	defer os.Remove(tmpFile.Name())

	// Write some random data
	_, err = tmpFile.Write([]byte("This is not a valid filesystem"))
	suite.Require().NoError(err)
	tmpFile.Close()

	for _, adapter := range adapters {
		// Each adapter should be able to check the device
		supported, err := adapter.IsDeviceSupported(suite.ctx, tmpFile.Name())
		suite.NoError(err, "Adapter %s should not error on invalid device", adapter.GetName())
		suite.False(supported, "Adapter %s should not support invalid device", adapter.GetName())
	}
}
