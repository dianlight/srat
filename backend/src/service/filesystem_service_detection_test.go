package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeBootSector writes a 512-byte boot sector carrying magic at offset so the
// adapter signature probe can be exercised without a real block device.
func writeBootSector(t *testing.T, name string, offset int, magic string) string {
	t.Helper()

	sector := make([]byte, 512)
	copy(sector[offset:], magic)

	devicePath := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(devicePath, sector, 0o600))
	return devicePath
}

// TestFsTypeFromDevice_NTFS covers issue #1513: the adapter signature table must
// recognise an NTFS boot sector, which the vendored u-root magic table
// (mount.FSFromBlock) does not list among its detectable filesystems.
func TestFsTypeFromDevice_NTFS(t *testing.T) {
	fsService := service.NewFilesystemService(context.Background(), func() {}, nil)

	// NTFS boot sector: jump instruction 0xEB 0x52 0x90 followed by the
	// "NTFS    " OEM id at offset 3.
	devicePath := writeBootSector(t, "sdc1", 3, "NTFS    ")

	fsType, err := fsService.FsTypeFromDevice(devicePath)

	require.Nil(t, err)
	assert.Equal(t, "ntfs", fsType)
}

// TestFsTypeFromDevice_Unknown verifies a device with no known magic still
// reports an unknown-filesystem error rather than a bogus type.
func TestFsTypeFromDevice_Unknown(t *testing.T) {
	fsService := service.NewFilesystemService(context.Background(), func() {}, nil)

	devicePath := filepath.Join(t.TempDir(), "garbage")
	require.NoError(t, os.WriteFile(devicePath, []byte("not a filesystem"), 0o600))

	fsType, err := fsService.FsTypeFromDevice(devicePath)

	assert.NotNil(t, err)
	assert.Empty(t, fsType)
}

// TestFsTypeFromDevice_MissingDevice verifies an unreadable path reports an
// error instead of panicking.
func TestFsTypeFromDevice_MissingDevice(t *testing.T) {
	fsService := service.NewFilesystemService(context.Background(), func() {}, nil)

	fsType, err := fsService.FsTypeFromDevice(filepath.Join(t.TempDir(), "missing"))

	assert.NotNil(t, err)
	assert.Empty(t, fsType)
}
