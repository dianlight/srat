package dto_test

import (
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestDiskMap_AddAndGet(t *testing.T) {
	id := "disk-1"
	d := dto.Disk{Id: &id}

	m := dto.DiskMap{}
	err := (&m).AddOrUpdate(&d)
	assert.NoError(t, err)

	got, ok := (&m).Get(id)
	assert.True(t, ok)
	assert.Equal(t, id, *got.Id)
}

func TestDiskMap_AddInvalidID(t *testing.T) {
	m := dto.DiskMap{}
	d := dto.Disk{}
	err := (&m).AddOrUpdate(&d)
	assert.Error(t, err)
}

func TestDiskMap_Remove(t *testing.T) {
	id := "disk-2"
	d := dto.Disk{Id: &id}
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&d)

	removed := (&m).Remove(id)
	assert.True(t, removed)

	_, ok := (&m).Get(id)
	assert.False(t, ok)

	// Removing again should return false
	removed = (&m).Remove(id)
	assert.False(t, removed)
}

func TestDiskMap_AddPartitionAndRemovePartition(t *testing.T) {
	diskID := "disk-3"
	d := dto.Disk{Id: &diskID}
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&d)

	partID := "part-1"
	p := dto.Partition{Id: &partID}

	// Add partition
	err := (&m).AddPartition(diskID, p)
	assert.NoError(t, err)

	// Verify partition is present
	gotDisk, ok := (&m).Get(diskID)
	assert.True(t, ok)
	if assert.NotNil(t, gotDisk.Partitions) {
		_, present := (*gotDisk.Partitions)[partID]
		assert.True(t, present)
	}

	// Remove partition
	removed := (&m).RemovePartition(diskID, partID)
	assert.True(t, removed)

	// Verify removed
	gotDisk, ok = (&m).Get(diskID)
	assert.True(t, ok)
	if assert.NotNil(t, gotDisk.Partitions) {
		_, present := (*gotDisk.Partitions)[partID]
		assert.False(t, present)
	}
}

func TestDiskMap_AddPartition_Errors(t *testing.T) {
	m := dto.DiskMap{}
	// No disk present
	partID := "p0"
	p := dto.Partition{Id: &partID}
	err := (&m).AddPartition("missing", p)
	assert.Error(t, err)

	// Empty partition id
	diskID := "disk-4"
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID})
	err = (&m).AddPartition(diskID, dto.Partition{})
	assert.Error(t, err)
}

func TestDiskMap_AddAndRemoveMountPoint(t *testing.T) {
	// Prepare disk and partition
	diskID := "d1"
	partID := "p1"
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID, Partitions: &map[string]dto.Partition{partID: {Id: &partID}}})

	// Add mount point
	mp := dto.MountPointData{Path: "/mnt/data"}
	err := (&m).AddOrUpdateMountPoint(diskID, partID, mp)
	assert.NoError(t, err)

	// Verify presence
	d, ok := (&m).Get(diskID)
	assert.True(t, ok)
	if assert.NotNil(t, d.Partitions) {
		part := (*d.Partitions)[partID]
		if assert.NotNil(t, part.MountPointData) {
			v, present := (*part.MountPointData)["/mnt/data"]
			assert.True(t, present)
			assert.Equal(t, "/mnt/data", v.Path)
		}
	}

	// Remove mount point
	removed := (&m).RemoveMountPoint(diskID, partID, "/mnt/data")
	assert.True(t, removed)

	// Verify removal
	d, ok = (&m).Get(diskID)
	assert.True(t, ok)
	if assert.NotNil(t, d.Partitions) {
		part := (*d.Partitions)[partID]
		if assert.NotNil(t, part.MountPointData) {
			_, present := (*part.MountPointData)["/mnt/data"]
			assert.False(t, present)
		}
	}
}

func TestDiskMap_AddMountPoint_Errors(t *testing.T) {
	m := dto.DiskMap{}

	// No disk
	err := (&m).AddOrUpdateMountPoint("", "p1", dto.MountPointData{Path: "/mnt/x"})
	assert.Error(t, err)

	// Disk missing
	err = (&m).AddOrUpdateMountPoint("missing", "p1", dto.MountPointData{Path: "/mnt/x"})
	assert.Error(t, err)

	// Disk present, partition missing
	diskID := "d2"
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID})
	err = (&m).AddOrUpdateMountPoint(diskID, "p1", dto.MountPointData{Path: "/mnt/x"})
	assert.Error(t, err)

	// Empty path
	parts := map[string]dto.Partition{"p1": {Id: ptr("p1")}}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID, Partitions: &parts})
	err = (&m).AddOrUpdateMountPoint(diskID, "p1", dto.MountPointData{})
	assert.Error(t, err)
}

func ptr(s string) *string { return &s }

func TestDiskMap_GetPartition(t *testing.T) {
	diskID := "disk-get"
	partID := "partition-get"
	parts := map[string]dto.Partition{partID: {Id: ptr(partID)}}
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID, Partitions: &parts})

	part, ok := (&m).GetPartition(diskID, partID)
	assert.True(t, ok)
	assert.Equal(t, partID, *part.Id)

	_, missing := (&m).GetPartition("nope", partID)
	assert.False(t, missing)
}

func TestDiskMap_GetMountPoint(t *testing.T) {
	diskID := "disk-mp"
	partID := "part-mp"
	mount := dto.MountPointData{Path: "/mnt/mp"}
	parts := map[string]dto.Partition{partID: {Id: ptr(partID), MountPointData: &map[string]dto.MountPointData{"/mnt/mp": mount}}}
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID, Partitions: &parts})

	mp, ok := (&m).GetMountPoint(diskID, partID, "/mnt/mp")
	assert.True(t, ok)
	assert.Equal(t, "/mnt/mp", mp.Path)

	_, missing := (&m).GetMountPoint(diskID, partID, "")
	assert.False(t, missing)
	_, missing = (&m).GetMountPoint(diskID, "unknown", "/mnt/mp")
	assert.False(t, missing)
	_, missing = (&m).GetMountPoint("unknown", partID, "/mnt/mp")
	assert.False(t, missing)
}

func TestDiskMap_GetMountPointByPath(t *testing.T) {
	diskID := "disk-mp-path"
	partID := "part-mp-path"
	mount := dto.MountPointData{Path: "/mnt/mp-path"}
	parts := map[string]dto.Partition{partID: {Id: ptr(partID), MountPointData: &map[string]dto.MountPointData{"/mnt/mp-path": mount}}}
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID, Partitions: &parts})

	mp, ok := (&m).GetMountPointByPath("/mnt/mp-path")
	assert.True(t, ok)
	assert.Equal(t, "/mnt/mp-path", mp.Path)

	_, missing := (&m).GetMountPointByPath("/missing")
	assert.False(t, missing)
}

func TestDiskMap_GetAllMountPoints(t *testing.T) {
	diskID := "disk-all-mp"
	partID := "part-all-mp"
	mount1 := dto.MountPointData{Path: "/mnt/mp1"}
	mount2 := dto.MountPointData{Path: "/mnt/mp2"}
	parts := map[string]dto.Partition{partID: {Id: ptr(partID), MountPointData: &map[string]dto.MountPointData{"/mnt/mp1": mount1, "/mnt/mp2": mount2}}}
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID, Partitions: &parts})

	allMPs := (&m).GetAllMountPoints()
	assert.Len(t, allMPs, 2)
}

func TestDiskMap_AddMountPointShare(t *testing.T) {
	diskID := "disk-share"
	partID := "part-share"
	path := "/mnt/share"
	mount := dto.MountPointData{Path: path}
	parts := map[string]dto.Partition{partID: {Id: ptr(partID), DiskId: ptr(diskID), MountPointData: &map[string]dto.MountPointData{path: mount}}}
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID, Partitions: &parts})

	partition := parts[partID]
	share := &dto.SharedResource{Name: "testshare", MountPointData: &dto.MountPointData{Path: path, Partition: &partition}}

	// Add share to mount point
	disk, err := (&m).AddMountPointShare(share)
	assert.NoError(t, err)
	assert.Equal(t, diskID, *disk.Id)

	// Verify share is present
	mp, ok := (&m).GetMountPoint(diskID, partID, path)
	assert.True(t, ok)
	assert.NotNil(t, mp.Share)
	assert.Equal(t, "testshare", mp.Share.Name)

	// Update share
	newShare := &dto.SharedResource{Name: "updatedshare", MountPointData: &dto.MountPointData{Path: path, Partition: &partition}}
	disk, err = (&m).AddMountPointShare(newShare)
	assert.NoError(t, err)
	assert.Equal(t, diskID, *disk.Id)

	mp, ok = (&m).GetMountPoint(diskID, partID, path)
	assert.True(t, ok)
	assert.NotNil(t, mp.Share)
	assert.Equal(t, "updatedshare", mp.Share.Name)
}

func TestDiskMap_AddMountPointShare_WithoutPartitionInfo(t *testing.T) {
	diskID := "disk-search"
	partID := "part-search"
	path := "/mnt/search"
	mount := dto.MountPointData{Path: path}
	parts := map[string]dto.Partition{partID: {Id: ptr(partID), DiskId: ptr(diskID), MountPointData: &map[string]dto.MountPointData{path: mount}}}
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID, Partitions: &parts})

	// Add share without partition info - should search and find it
	share := &dto.SharedResource{Name: "testshare", MountPointData: &dto.MountPointData{Path: path}}
	disk, err := (&m).AddMountPointShare(share)
	assert.NoError(t, err)
	assert.Equal(t, diskID, *disk.Id)

	// Verify share is present
	mp, ok := (&m).GetMountPoint(diskID, partID, path)
	assert.True(t, ok)
	assert.NotNil(t, mp.Share)
	assert.Equal(t, "testshare", mp.Share.Name)
}

func TestDiskMap_AddMountPointShare_Errors(t *testing.T) {
	m := dto.DiskMap{}

	// Nil share
	_, err := (&m).AddMountPointShare(nil)
	assert.Error(t, err)

	// Share with nil mount point data
	shareNoMPD := &dto.SharedResource{Name: "testshare"}
	_, err = (&m).AddMountPointShare(shareNoMPD)
	assert.Error(t, err)

	// Share with empty path and no partition to search
	shareNoPath := &dto.SharedResource{Name: "testshare", MountPointData: &dto.MountPointData{Path: ""}}
	_, err = (&m).AddMountPointShare(shareNoPath)
	assert.Error(t, err)

	// Share with nil partition and mount point not found
	shareNoPart := &dto.SharedResource{Name: "testshare", MountPointData: &dto.MountPointData{Path: "/mnt/nonexistent"}}
	_, err = (&m).AddMountPointShare(shareNoPart)
	assert.Error(t, err)

	// Share with nil partition disk id
	partNoDiskId := &dto.Partition{Id: ptr("p1")}
	sharePartNoDiskId := &dto.SharedResource{Name: "testshare", MountPointData: &dto.MountPointData{Path: "/mnt/x", Partition: partNoDiskId}}
	_, err = (&m).AddMountPointShare(sharePartNoDiskId)
	assert.Error(t, err)

	// Share with nil partition id
	partNoId := &dto.Partition{DiskId: ptr("d1")}
	sharePartNoId := &dto.SharedResource{Name: "testshare", MountPointData: &dto.MountPointData{Path: "/mnt/x", Partition: partNoId}}
	_, err = (&m).AddMountPointShare(sharePartNoId)
	assert.Error(t, err)

	// Disk not found
	shareValidButNoDisk := &dto.SharedResource{Name: "testshare", MountPointData: &dto.MountPointData{Path: "/mnt/x", Partition: &dto.Partition{Id: ptr("p1"), DiskId: ptr("missing")}}}
	_, err = (&m).AddMountPointShare(shareValidButNoDisk)
	assert.Error(t, err)

	// Disk present but no partitions
	diskID := "disk-no-parts"
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID})
	shareNoParts := &dto.SharedResource{Name: "testshare", MountPointData: &dto.MountPointData{Path: "/mnt/x", Partition: &dto.Partition{Id: ptr("p1"), DiskId: ptr(diskID)}}}
	_, err = (&m).AddMountPointShare(shareNoParts)
	assert.Error(t, err)

	// Partition not found
	diskID2 := "disk-with-parts"
	parts := map[string]dto.Partition{"p1": {Id: ptr("p1")}}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID2, Partitions: &parts})
	shareMissingPart := &dto.SharedResource{Name: "testshare", MountPointData: &dto.MountPointData{Path: "/mnt/x", Partition: &dto.Partition{Id: ptr("missing-part"), DiskId: ptr(diskID2)}}}
	_, err = (&m).AddMountPointShare(shareMissingPart)
	assert.Error(t, err)

	// Partition has no mount points
	sharePtNoMP := &dto.SharedResource{Name: "testshare", MountPointData: &dto.MountPointData{Path: "/mnt/x", Partition: &dto.Partition{Id: ptr("p1"), DiskId: ptr(diskID2)}}}
	_, err = (&m).AddMountPointShare(sharePtNoMP)
	assert.Error(t, err)

	// Mount point not found
	diskID3 := "disk-with-mp"
	mount := dto.MountPointData{Path: "/mnt/real"}
	parts3 := map[string]dto.Partition{"p1": {Id: ptr("p1"), MountPointData: &map[string]dto.MountPointData{"/mnt/real": mount}}}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID3, Partitions: &parts3})
	shareMissingMP := &dto.SharedResource{Name: "testshare", MountPointData: &dto.MountPointData{Path: "/mnt/missing", Partition: &dto.Partition{Id: ptr("p1"), DiskId: ptr(diskID3)}}}
	_, err = (&m).AddMountPointShare(shareMissingMP)
	assert.Error(t, err)
}

func TestDiskMap_RemoveMountPointShare(t *testing.T) {
	diskID := "disk-remove-share"
	partID := "part-remove-share"
	path := "/mnt/remove-share"
	shareName := "todelete"
	share := &dto.SharedResource{Name: shareName}
	mount := dto.MountPointData{Path: path, Share: share}
	parts := map[string]dto.Partition{partID: {Id: ptr(partID), MountPointData: &map[string]dto.MountPointData{path: mount}}}
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID, Partitions: &parts})

	// Verify share exists initially
	mp, ok := (&m).GetMountPoint(diskID, partID, path)
	assert.True(t, ok)
	assert.NotNil(t, mp.Share)
	assert.Equal(t, shareName, mp.Share.Name)

	// Remove share by share name
	removed, disk := (&m).RemoveMountPointShare(shareName)
	assert.True(t, removed)
	assert.NotNil(t, disk)
	assert.Equal(t, diskID, *disk.Id)

	// Verify share is nil
	mp, ok = (&m).GetMountPoint(diskID, partID, path)
	assert.True(t, ok)
	assert.Nil(t, mp.Share)

	// Removing again should return false (no share with that name exists anymore)
	removed, disk = (&m).RemoveMountPointShare(shareName)
	assert.False(t, removed)
	assert.Nil(t, disk)
}

func TestDiskMap_RemoveMountPointShare_MultipleDisks(t *testing.T) {
	// Setup multiple disks with multiple partitions
	diskID1 := "disk-1"
	diskID2 := "disk-2"
	partID1 := "part-1"
	partID2 := "part-2"
	partID3 := "part-3"
	path1 := "/mnt/share1"
	path2 := "/mnt/share2"
	path3 := "/mnt/share3"

	shareName1 := "share1"
	shareName2 := "share2"
	shareName3 := "share3"

	share1 := &dto.SharedResource{Name: shareName1}
	share2 := &dto.SharedResource{Name: shareName2}
	share3 := &dto.SharedResource{Name: shareName3}

	mount1 := dto.MountPointData{Path: path1, Share: share1}
	mount2 := dto.MountPointData{Path: path2, Share: share2}
	mount3 := dto.MountPointData{Path: path3, Share: share3}

	parts1 := map[string]dto.Partition{
		partID1: {Id: ptr(partID1), MountPointData: &map[string]dto.MountPointData{path1: mount1}},
		partID2: {Id: ptr(partID2), MountPointData: &map[string]dto.MountPointData{path2: mount2}},
	}
	parts2 := map[string]dto.Partition{
		partID3: {Id: ptr(partID3), MountPointData: &map[string]dto.MountPointData{path3: mount3}},
	}

	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID1, Partitions: &parts1})
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID2, Partitions: &parts2})

	// Remove share from disk 2
	removed, disk := (&m).RemoveMountPointShare(shareName3)
	assert.True(t, removed)
	assert.NotNil(t, disk)
	assert.Equal(t, diskID2, *disk.Id)

	// Verify share3 is removed
	mp, ok := (&m).GetMountPoint(diskID2, partID3, path3)
	assert.True(t, ok)
	assert.Nil(t, mp.Share)

	// Verify share1 and share2 are still present
	mp1, ok1 := (&m).GetMountPoint(diskID1, partID1, path1)
	assert.True(t, ok1)
	assert.NotNil(t, mp1.Share)
	assert.Equal(t, shareName1, mp1.Share.Name)

	mp2, ok2 := (&m).GetMountPoint(diskID1, partID2, path2)
	assert.True(t, ok2)
	assert.NotNil(t, mp2.Share)
	assert.Equal(t, shareName2, mp2.Share.Name)
}

func TestDiskMap_RemoveMountPointShare_MountPointWithoutShare(t *testing.T) {
	diskID := "disk-no-share"
	partID := "part-no-share"
	path := "/mnt/no-share"
	mount := dto.MountPointData{Path: path, Share: nil} // No share attached
	parts := map[string]dto.Partition{partID: {Id: ptr(partID), MountPointData: &map[string]dto.MountPointData{path: mount}}}
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID, Partitions: &parts})

	// Try to remove non-existent share
	removed, disk := (&m).RemoveMountPointShare("nonexistent")
	assert.False(t, removed)
	assert.Nil(t, disk)
}

func TestDiskMap_RemoveMountPointShare_MultipleMountPointsSameDisk(t *testing.T) {
	diskID := "disk-multi-mp"
	partID := "part-multi-mp"
	path1 := "/mnt/mp1"
	path2 := "/mnt/mp2"
	shareName := "target-share"
	share1 := &dto.SharedResource{Name: "other-share"}
	share2 := &dto.SharedResource{Name: shareName}

	mount1 := dto.MountPointData{Path: path1, Share: share1}
	mount2 := dto.MountPointData{Path: path2, Share: share2}

	parts := map[string]dto.Partition{
		partID: {Id: ptr(partID), MountPointData: &map[string]dto.MountPointData{
			path1: mount1,
			path2: mount2,
		}},
	}
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID, Partitions: &parts})

	// Remove target-share
	removed, disk := (&m).RemoveMountPointShare(shareName)
	assert.True(t, removed)
	assert.NotNil(t, disk)

	// Verify only share2 is removed
	mp1, ok1 := (&m).GetMountPoint(diskID, partID, path1)
	assert.True(t, ok1)
	assert.NotNil(t, mp1.Share)
	assert.Equal(t, "other-share", mp1.Share.Name)

	mp2, ok2 := (&m).GetMountPoint(diskID, partID, path2)
	assert.True(t, ok2)
	assert.Nil(t, mp2.Share)
}

func TestDiskMap_RemoveMountPointShare_Errors(t *testing.T) {
	m := dto.DiskMap{}

	// Empty share name
	removed, disk := (&m).RemoveMountPointShare("")
	assert.False(t, removed)
	assert.Nil(t, disk)

	// Share not found (empty map)
	removed, disk = (&m).RemoveMountPointShare("nonexistent")
	assert.False(t, removed)
	assert.Nil(t, disk)

	// Disk present but no partitions
	diskID := "disk-no-parts"
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID})
	removed, disk = (&m).RemoveMountPointShare("someshare")
	assert.False(t, removed)
	assert.Nil(t, disk)

	// Partition has no mount points
	diskID2 := "disk-with-parts"
	parts := map[string]dto.Partition{"p1": {Id: ptr("p1")}}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID2, Partitions: &parts})
	removed, disk = (&m).RemoveMountPointShare("someshare")
	assert.False(t, removed)
	assert.Nil(t, disk)

	// Mount point exists but has no share
	diskID3 := "disk-with-mp"
	mount := dto.MountPointData{Path: "/mnt/real", Share: nil}
	parts3 := map[string]dto.Partition{"p1": {Id: ptr("p1"), MountPointData: &map[string]dto.MountPointData{"/mnt/real": mount}}}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID3, Partitions: &parts3})
	removed, disk = (&m).RemoveMountPointShare("someshare")
	assert.False(t, removed)
	assert.Nil(t, disk)

	// Mount point with different share name
	diskID4 := "disk-with-different-share"
	shareOther := &dto.SharedResource{Name: "different-share"}
	mountWithShare := dto.MountPointData{Path: "/mnt/shared", Share: shareOther}
	parts4 := map[string]dto.Partition{"p1": {Id: ptr("p1"), MountPointData: &map[string]dto.MountPointData{"/mnt/shared": mountWithShare}}}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID4, Partitions: &parts4})
	removed, disk = (&m).RemoveMountPointShare("nonexistent-share")
	assert.False(t, removed)
	assert.Nil(t, disk)
}

func TestDiskMap_RemoveMountPointShare_NilMap(t *testing.T) {
	var m *dto.DiskMap = nil
	removed, disk := m.RemoveMountPointShare("anyshare")
	assert.False(t, removed)
	assert.Nil(t, disk)
}

func TestDiskMap_AddHDIdleDevice(t *testing.T) {
	diskID := "hdidle-disk-1"
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID})

	// Create an HDIdleDevice
	hdIdle := &dto.HDIdleDevice{DiskId: diskID, Enabled: dto.HdidleEnableds.YESENABLED}

	// Add HDIdleDevice
	err := (&m).AddHDIdleDevice(hdIdle)
	assert.NoError(t, err)

	// Verify it was set
	d, ok := (&m).Get(diskID)
	assert.True(t, ok)
	assert.NotNil(t, d.HDIdleDevice)
	// Device field is not present on HDIdleDevice; verify DiskId and Enabled instead
	assert.Equal(t, diskID, d.HDIdleDevice.DiskId)
	assert.Equal(t, dto.HdidleEnableds.YESENABLED, d.HDIdleDevice.Enabled)
}

func TestDiskMap_AddHDIdleDevice_Update(t *testing.T) {
	diskID := "hdidle-disk-2"
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID})

	// Add initial HDIdleDevice
	hdIdle1 := &dto.HDIdleDevice{DiskId: diskID, Enabled: dto.HdidleEnableds.YESENABLED}
	err := (&m).AddHDIdleDevice(hdIdle1)
	assert.NoError(t, err)

	// Update with new HDIdleDevice
	hdIdle2 := &dto.HDIdleDevice{DiskId: diskID, Enabled: dto.HdidleEnableds.NOENABLED}
	err = (&m).AddHDIdleDevice(hdIdle2)
	assert.NoError(t, err)

	// Verify it was updated
	d, ok := (&m).Get(diskID)
	assert.True(t, ok)
	assert.NotNil(t, d.HDIdleDevice)
	assert.Equal(t, diskID, d.HDIdleDevice.DiskId)
	assert.Equal(t, dto.HdidleEnableds.NOENABLED, d.HDIdleDevice.Enabled)
}

func TestDiskMap_AddHDIdleDevice_Errors(t *testing.T) {
	m := dto.DiskMap{}

	// Nil disk map (empty but not nil)
	diskID := "hdidle-test"

	// Empty diskID
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID})
	err := (&m).AddHDIdleDevice(&dto.HDIdleDevice{DiskId: ""})
	assert.Error(t, err)

	// Disk not found
	err = (&m).AddHDIdleDevice(&dto.HDIdleDevice{DiskId: "nonexistent"})
	assert.Error(t, err)
}

func TestDiskMap_AddSmartInfo(t *testing.T) {
	diskID := "smart-disk-1"
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID})

	// Create a SmartInfo
	smartInfo := &dto.SmartInfo{
		DiskId:       diskID,
		Supported:    true,
		DiskType:     "SATA",
		ModelName:    "Samsung SSD 860 EVO",
		SerialNumber: "S3Z9NB0K123456",
		Firmware:     "RVT01B6Q",
		RotationRate: 0,
	}

	// Add SmartInfo
	err := (&m).AddSmartInfo(smartInfo)
	assert.NoError(t, err)

	// Verify it was set
	d, ok := (&m).Get(diskID)
	assert.True(t, ok)
	assert.NotNil(t, d.SmartInfo)
	assert.True(t, d.SmartInfo.Supported)
	assert.Equal(t, "SATA", d.SmartInfo.DiskType)
	assert.Equal(t, "Samsung SSD 860 EVO", d.SmartInfo.ModelName)
	assert.Equal(t, "S3Z9NB0K123456", d.SmartInfo.SerialNumber)
}

func TestDiskMap_AddSmartInfo_Update(t *testing.T) {
	diskID := "smart-disk-2"
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID})

	// Add initial SmartInfo
	smartInfo1 := &dto.SmartInfo{
		DiskId:       diskID,
		Supported:    true,
		DiskType:     "SATA",
		ModelName:    "Old Model",
		SerialNumber: "OLD123",
	}
	err := (&m).AddSmartInfo(smartInfo1)
	assert.NoError(t, err)

	// Update with new SmartInfo
	smartInfo2 := &dto.SmartInfo{
		DiskId:       diskID,
		Supported:    true,
		DiskType:     "NVMe",
		ModelName:    "New Model",
		SerialNumber: "NEW456",
		RotationRate: 7200,
	}
	err = (&m).AddSmartInfo(smartInfo2)
	assert.NoError(t, err)

	// Verify it was updated
	d, ok := (&m).Get(diskID)
	assert.True(t, ok)
	assert.NotNil(t, d.SmartInfo)
	assert.Equal(t, "NVMe", d.SmartInfo.DiskType)
	assert.Equal(t, "New Model", d.SmartInfo.ModelName)
	assert.Equal(t, "NEW456", d.SmartInfo.SerialNumber)
	assert.Equal(t, 7200, d.SmartInfo.RotationRate)
}

func TestDiskMap_AddSmartInfo_UnsupportedDevice(t *testing.T) {
	diskID := "smart-disk-4"
	m := dto.DiskMap{}
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID})

	// Create SmartInfo for unsupported device
	smartInfo := &dto.SmartInfo{
		DiskId:    diskID,
		Supported: false,
		DiskType:  "Unknown",
	}

	// Add SmartInfo
	err := (&m).AddSmartInfo(smartInfo)
	assert.NoError(t, err)

	// Verify it was set
	d, ok := (&m).Get(diskID)
	assert.True(t, ok)
	assert.NotNil(t, d.SmartInfo)
	assert.False(t, d.SmartInfo.Supported)
	assert.Equal(t, "Unknown", d.SmartInfo.DiskType)
}

func TestDiskMap_AddSmartInfo_Errors(t *testing.T) {
	m := dto.DiskMap{}

	// Test data
	diskID := "smart-test"

	// Empty diskID
	_ = (&m).AddOrUpdate(&dto.Disk{Id: &diskID})
	err := (&m).AddSmartInfo(&dto.SmartInfo{DiskId: ""})
	assert.Error(t, err)

	// Disk not found
	err = (&m).AddSmartInfo(&dto.SmartInfo{DiskId: "nonexistent"})
	assert.Error(t, err)
}
