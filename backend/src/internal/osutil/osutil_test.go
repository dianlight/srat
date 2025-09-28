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
