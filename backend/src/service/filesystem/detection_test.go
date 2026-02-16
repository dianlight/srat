package filesystem_test

import (
	"context"
	"os"
	"testing"

	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
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

func (suite *DetectionTestSuite) TestDetectFilesystemType_NonExistent() {
	adapters := []filesystem.FilesystemAdapter{
		filesystem.NewExt4Adapter(),
		filesystem.NewVfatAdapter(),
		filesystem.NewNtfsAdapter(),
		filesystem.NewBtrfsAdapter(),
		filesystem.NewXfsAdapter(),
	}

	fsType, err := filesystem.DetectFilesystemType("/dev/nonexistent", adapters)
	suite.Error(err)
	suite.Empty(fsType)
	suite.True(errors.Is(err, filesystem.ErrorDeviceNotFound))
}

func (suite *DetectionTestSuite) TestDetectFilesystemType_EmptyFile() {
	adapters := []filesystem.FilesystemAdapter{
		filesystem.NewExt4Adapter(),
		filesystem.NewVfatAdapter(),
		filesystem.NewNtfsAdapter(),
		filesystem.NewBtrfsAdapter(),
		filesystem.NewXfsAdapter(),
	}

	// Create a temporary empty file
	tmpFile, err := os.CreateTemp("", "empty-fs-*")
	suite.Require().NoError(err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	fsType, err := filesystem.DetectFilesystemType(tmpFile.Name(), adapters)
	suite.Error(err)
	suite.Empty(fsType)
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

func (suite *DetectionTestSuite) TestDetectFilesystemType_Ext4Signature() {
	adapters := []filesystem.FilesystemAdapter{
		filesystem.NewExt4Adapter(),
		filesystem.NewVfatAdapter(),
		filesystem.NewNtfsAdapter(),
		filesystem.NewBtrfsAdapter(),
		filesystem.NewXfsAdapter(),
	}

	// Create a temp file with ext4 signature
	tmpFile, err := os.CreateTemp("", "ext4-device-*")
	suite.Require().NoError(err)
	defer os.Remove(tmpFile.Name())

	// Write enough zeros to cover the signature location
	// ext4 magic at offset 1080: 0x53 0xEF
	buffer := make([]byte, 1082)
	buffer[1080] = 0x53
	buffer[1081] = 0xEF

	_, err = tmpFile.Write(buffer)
	suite.Require().NoError(err)
	tmpFile.Close()

	fsType, err := filesystem.DetectFilesystemType(tmpFile.Name(), adapters)
	suite.NoError(err)
	suite.Equal("ext4", fsType)
}

func (suite *DetectionTestSuite) TestDetectFilesystemType_VfatSignature() {
	adapters := []filesystem.FilesystemAdapter{
		filesystem.NewExt4Adapter(),
		filesystem.NewVfatAdapter(),
		filesystem.NewNtfsAdapter(),
		filesystem.NewBtrfsAdapter(),
		filesystem.NewXfsAdapter(),
	}

	// Create a temp file with FAT32 signature
	tmpFile, err := os.CreateTemp("", "vfat-device-*")
	suite.Require().NoError(err)
	defer os.Remove(tmpFile.Name())

	// Write enough zeros
	buffer := make([]byte, 100)
	// FAT32 signature at offset 82: "FAT32   "
	copy(buffer[82:], []byte("FAT32   "))

	_, err = tmpFile.Write(buffer)
	suite.Require().NoError(err)
	tmpFile.Close()

	fsType, err := filesystem.DetectFilesystemType(tmpFile.Name(), adapters)
	suite.NoError(err)
	suite.Equal("vfat", fsType)
}

func (suite *DetectionTestSuite) TestDetectFilesystemType_NtfsSignature() {
	adapters := []filesystem.FilesystemAdapter{
		filesystem.NewExt4Adapter(),
		filesystem.NewVfatAdapter(),
		filesystem.NewNtfsAdapter(),
		filesystem.NewBtrfsAdapter(),
		filesystem.NewXfsAdapter(),
	}

	// Create a temp file with NTFS signature
	tmpFile, err := os.CreateTemp("", "ntfs-device-*")
	suite.Require().NoError(err)
	defer os.Remove(tmpFile.Name())

	// Write enough zeros
	buffer := make([]byte, 20)
	// NTFS signature at offset 3: "NTFS    "
	copy(buffer[3:], []byte("NTFS    "))

	_, err = tmpFile.Write(buffer)
	suite.Require().NoError(err)
	tmpFile.Close()

	fsType, err := filesystem.DetectFilesystemType(tmpFile.Name(), adapters)
	suite.NoError(err)
	suite.Equal("ntfs", fsType)
}

func (suite *DetectionTestSuite) TestDetectFilesystemType_XfsSignature() {
	adapters := []filesystem.FilesystemAdapter{
		filesystem.NewExt4Adapter(),
		filesystem.NewVfatAdapter(),
		filesystem.NewNtfsAdapter(),
		filesystem.NewBtrfsAdapter(),
		filesystem.NewXfsAdapter(),
	}

	// Create a temp file with XFS signature
	tmpFile, err := os.CreateTemp("", "xfs-device-*")
	suite.Require().NoError(err)
	defer os.Remove(tmpFile.Name())

	// Write enough zeros
	buffer := make([]byte, 10)
	// XFS signature at offset 0: "XFSB"
	copy(buffer[0:], []byte("XFSB"))

	_, err = tmpFile.Write(buffer)
	suite.Require().NoError(err)
	tmpFile.Close()

	fsType, err := filesystem.DetectFilesystemType(tmpFile.Name(), adapters)
	suite.NoError(err)
	suite.Equal("xfs", fsType)
}

func (suite *DetectionTestSuite) TestDetectFilesystemType_UnknownFilesystem() {
	adapters := []filesystem.FilesystemAdapter{
		filesystem.NewExt4Adapter(),
		filesystem.NewVfatAdapter(),
	}

	// Create a temp file with no recognized signature
	tmpFile, err := os.CreateTemp("", "unknown-device-*")
	suite.Require().NoError(err)
	defer os.Remove(tmpFile.Name())

	// Write random data that doesn't match any signature
	buffer := make([]byte, 2000)
	for i := range buffer {
		buffer[i] = byte(i % 256)
	}

	_, err = tmpFile.Write(buffer)
	suite.Require().NoError(err)
	tmpFile.Close()

	fsType, err := filesystem.DetectFilesystemType(tmpFile.Name(), adapters)
	suite.Error(err)
	suite.Empty(fsType)
	suite.True(errors.Is(err, filesystem.ErrorUnknownFilesystem))
}
