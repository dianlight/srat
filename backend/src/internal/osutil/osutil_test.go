package osutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleMountInfo = `36 35 98:0 / /mnt/root rw,nosuid - ext4 /dev/root rw,relatime
37 36 98:1 /subdir /mnt/data rw,noatime shared:402 - xfs /dev/sdb1 rw`

func TestLoadMountInfoWithMock(t *testing.T) {
	restore := MockMountInfo(sampleMountInfo)
	t.Cleanup(restore)

	entries, err := LoadMountInfo()
	require.NoError(t, err)
	require.Len(t, entries, 2)

	root := entries[0]
	assert.Equal(t, "/mnt/root", root.MountDir)
	assert.Equal(t, "ext4", root.FsType)
	assert.Equal(t, "/dev/root", root.MountSource)
	require.Contains(t, root.MountOptions, "rw")
	assert.Equal(t, "", root.MountOptions["rw"])
	require.Contains(t, root.MountOptions, "nosuid")

	data := entries[1]
	assert.Equal(t, 37, data.MountID)
	assert.Equal(t, 36, data.ParentID)
	assert.Equal(t, "xfs", data.FsType)
	assert.Equal(t, "/mnt/data", data.MountDir)
	require.Len(t, data.OptionalFields, 1)
	assert.Equal(t, "shared:402", data.OptionalFields[0])
}

func TestIsMounted(t *testing.T) {
	restore := MockMountInfo(sampleMountInfo)
	t.Cleanup(restore)

	mounted, err := IsMounted("/mnt/root")
	require.NoError(t, err)
	assert.True(t, mounted)

	mountedWithTrailingSlash, err := IsMounted("/mnt/data/")
	require.NoError(t, err)
	assert.True(t, mountedWithTrailingSlash)

	missing, err := IsMounted("/mnt/missing")
	require.NoError(t, err)
	assert.False(t, missing)
}

func TestParseHelpers(t *testing.T) {
	opts := parseOptions("rw,noatime,uid=1000")
	assert.Equal(t, "", opts["rw"])
	assert.Equal(t, "", opts["noatime"])
	assert.Equal(t, "1000", opts["uid"])

	optional := parseOptional("shared:5 master:1")
	require.Len(t, optional, 2)
	assert.Equal(t, "shared:5", optional[0])
	assert.Equal(t, "master:1", optional[1])
}

func TestMountInfoEntry(t *testing.T) {
	entry := &MountInfoEntry{
		MountID:      1,
		ParentID:     0,
		DevMajor:     8,
		DevMinor:     1,
		Root:         "/",
		MountDir:     "/mnt/test",
		MountOptions: map[string]string{"rw": "", "noatime": ""},
		FsType:       "ext4",
		MountSource:  "/dev/sda1",
	}

	assert.Equal(t, 1, entry.MountID)
	assert.Equal(t, 0, entry.ParentID)
	assert.Equal(t, 8, entry.DevMajor)
	assert.Equal(t, 1, entry.DevMinor)
	assert.Equal(t, "/mnt/test", entry.MountDir)
	assert.Equal(t, "ext4", entry.FsType)
	assert.Equal(t, "/dev/sda1", entry.MountSource)
}

func TestParseOptionsEmpty(t *testing.T) {
	opts := parseOptions("")
	assert.NotNil(t, opts)
	assert.Empty(t, opts)
}

func TestParseOptionsSingleValue(t *testing.T) {
	opts := parseOptions("rw")
	assert.Len(t, opts, 1)
	assert.Equal(t, "", opts["rw"])
}

func TestParseOptionsMultipleValues(t *testing.T) {
	opts := parseOptions("rw,nosuid,nodev,uid=1000,gid=100")
	assert.Len(t, opts, 5)
	assert.Equal(t, "", opts["rw"])
	assert.Equal(t, "", opts["nosuid"])
	assert.Equal(t, "", opts["nodev"])
	assert.Equal(t, "1000", opts["uid"])
	assert.Equal(t, "100", opts["gid"])
}

func TestParseOptionalEmpty(t *testing.T) {
	optional := parseOptional("")
	assert.Nil(t, optional)
}

func TestParseOptionalWhitespace(t *testing.T) {
	optional := parseOptional("   ")
	assert.Nil(t, optional)
}

func TestParseOptionalSingleField(t *testing.T) {
	optional := parseOptional("shared:402")
	require.Len(t, optional, 1)
	assert.Equal(t, "shared:402", optional[0])
}

func TestParseOptionalMultipleFields(t *testing.T) {
	optional := parseOptional("shared:5 master:1 propagate_from:2")
	require.Len(t, optional, 3)
	assert.Equal(t, "shared:5", optional[0])
	assert.Equal(t, "master:1", optional[1])
	assert.Equal(t, "propagate_from:2", optional[2])
}

func TestConvertInfosEmpty(t *testing.T) {
	entries := convertInfos(nil)
	assert.NotNil(t, entries)
	assert.Empty(t, entries)
}

func TestMockMountInfoRestore(t *testing.T) {
	// Set initial mock
	restore1 := MockMountInfo("first")
	
	// Set second mock
	restore2 := MockMountInfo("second")
	
	// Restore to first
	restore2()
	
	// Restore to original
	restore1()
	
	assert.True(t, true) // Test completes without panic
}

func TestIsMountedMultipleMounts(t *testing.T) {
	mountInfo := `36 35 98:0 / /mnt/a rw - ext4 /dev/sda1 rw
37 35 98:1 / /mnt/b rw - ext4 /dev/sda2 rw
38 35 98:2 / /mnt/c rw - xfs /dev/sdb1 rw`
	
	restore := MockMountInfo(mountInfo)
	t.Cleanup(restore)

	mountedA, err := IsMounted("/mnt/a")
	require.NoError(t, err)
	assert.True(t, mountedA)

	mountedB, err := IsMounted("/mnt/b")
	require.NoError(t, err)
	assert.True(t, mountedB)

	mountedC, err := IsMounted("/mnt/c")
	require.NoError(t, err)
	assert.True(t, mountedC)

	notMounted, err := IsMounted("/mnt/d")
	require.NoError(t, err)
	assert.False(t, notMounted)
}

func TestMountInfoSuperOptions(t *testing.T) {
	restore := MockMountInfo(sampleMountInfo)
	t.Cleanup(restore)

	entries, err := LoadMountInfo()
	require.NoError(t, err)
	require.Len(t, entries, 2)

	// Check super options
	assert.NotNil(t, entries[0].SuperOptions)
	assert.NotNil(t, entries[1].SuperOptions)
}

func TestMountInfoDeviceNumbers(t *testing.T) {
	restore := MockMountInfo(sampleMountInfo)
	t.Cleanup(restore)

	entries, err := LoadMountInfo()
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.Equal(t, 98, entries[0].DevMajor)
	assert.Equal(t, 0, entries[0].DevMinor)
	assert.Equal(t, 98, entries[1].DevMajor)
	assert.Equal(t, 1, entries[1].DevMinor)
}

func TestIsKernelModuleLoaded(t *testing.T) {
	// Note: This test will check actual kernel modules on the system.
	// We test with a module that is very likely to be loaded (like 'loop')
	// and one that is unlikely to exist.
	
	// Test with a common module (may not be loaded on all systems)
	// We just verify the function doesn't error
	_, err := IsKernelModuleLoaded("loop")
	assert.NoError(t, err)
	
	// Test with a non-existent module
	loaded, err := IsKernelModuleLoaded("definitely_not_a_real_module_xyz123")
	assert.NoError(t, err)
	assert.False(t, loaded)
}
