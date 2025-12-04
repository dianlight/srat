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
	err := (&m).Add(&d)
	assert.NoError(t, err)

	got, ok := (&m).Get(id)
	assert.True(t, ok)
	assert.Equal(t, id, *got.Id)
}

func TestDiskMap_AddInvalidID(t *testing.T) {
	m := dto.DiskMap{}
	d := dto.Disk{}
	err := (&m).Add(&d)
	assert.Error(t, err)
}

func TestDiskMap_Remove(t *testing.T) {
	id := "disk-2"
	d := dto.Disk{Id: &id}
	m := dto.DiskMap{}
	_ = (&m).Add(&d)

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
	_ = (&m).Add(&d)

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
	_ = (&m).Add(&dto.Disk{Id: &diskID})
	err = (&m).AddPartition(diskID, dto.Partition{})
	assert.Error(t, err)
}

func TestDiskMap_AddAndRemoveMountPoint(t *testing.T) {
	// Prepare disk and partition
	diskID := "d1"
	partID := "p1"
	m := dto.DiskMap{}
	_ = (&m).Add(&dto.Disk{Id: &diskID, Partitions: &map[string]dto.Partition{partID: {Id: &partID}}})

	// Add mount point
	mp := dto.MountPointData{Path: "/mnt/data"}
	err := (&m).AddMountPoint(diskID, partID, mp)
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
	err := (&m).AddMountPoint("", "p1", dto.MountPointData{Path: "/mnt/x"})
	assert.Error(t, err)

	// Disk missing
	err = (&m).AddMountPoint("missing", "p1", dto.MountPointData{Path: "/mnt/x"})
	assert.Error(t, err)

	// Disk present, partition missing
	diskID := "d2"
	_ = (&m).Add(&dto.Disk{Id: &diskID})
	err = (&m).AddMountPoint(diskID, "p1", dto.MountPointData{Path: "/mnt/x"})
	assert.Error(t, err)

	// Empty path
	parts := map[string]dto.Partition{"p1": {Id: ptr("p1")}}
	_ = (&m).Add(&dto.Disk{Id: &diskID, Partitions: &parts})
	err = (&m).AddMountPoint(diskID, "p1", dto.MountPointData{})
	assert.Error(t, err)
}

func ptr(s string) *string { return &s }

func TestDiskMap_GetPartition(t *testing.T) {
	diskID := "disk-get"
	partID := "partition-get"
	parts := map[string]dto.Partition{partID: {Id: ptr(partID)}}
	m := dto.DiskMap{}
	_ = (&m).Add(&dto.Disk{Id: &diskID, Partitions: &parts})

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
	_ = (&m).Add(&dto.Disk{Id: &diskID, Partitions: &parts})

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
	_ = (&m).Add(&dto.Disk{Id: &diskID, Partitions: &parts})

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
	_ = (&m).Add(&dto.Disk{Id: &diskID, Partitions: &parts})

	allMPs := (&m).GetAllMountPoints()
	assert.Len(t, allMPs, 2)
}
