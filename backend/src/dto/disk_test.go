package dto_test

import (
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestDisk_Fields(t *testing.T) {
	deviceName := "sda"
	devicePath := "/dev/sda"
	byIDPath := "/dev/disk/by-id/disk-123"
	bus := "USB"
	ejectable := true
	id := "disk-123"
	model := "Samsung SSD"
	removable := true
	revision := "1.0"
	seat := "seat0"
	serial := "S123456"
	size := 1000000000
	vendor := "Samsung"

	disk := dto.Disk{
		LegacyDeviceName: &deviceName,
		LegacyDevicePath: &devicePath,
		DevicePath:       &byIDPath,
		ConnectionBus:    &bus,
		Ejectable:        &ejectable,
		Id:               &id,
		Model:            &model,
		Removable:        &removable,
		Revision:         &revision,
		Seat:             &seat,
		Serial:           &serial,
		Size:             &size,
		Vendor:           &vendor,
	}

	assert.Equal(t, deviceName, *disk.LegacyDeviceName)
	assert.Equal(t, devicePath, *disk.LegacyDevicePath)
	assert.Equal(t, byIDPath, *disk.DevicePath)
	assert.Equal(t, bus, *disk.ConnectionBus)
	assert.Equal(t, ejectable, *disk.Ejectable)
	assert.Equal(t, id, *disk.Id)
	assert.Equal(t, model, *disk.Model)
	assert.Equal(t, removable, *disk.Removable)
	assert.Equal(t, revision, *disk.Revision)
	assert.Equal(t, seat, *disk.Seat)
	assert.Equal(t, serial, *disk.Serial)
	assert.Equal(t, size, *disk.Size)
	assert.Equal(t, vendor, *disk.Vendor)
}

func TestDisk_WithPartitions(t *testing.T) {
	partitionName := "sda1"
	partitionPath := "/dev/sda1"
	partitionID := "part-123"
	fsType := "ext4"
	name := "Data"
	size := 500000000
	system := false

	partition := dto.Partition{
		LegacyDeviceName: &partitionName,
		LegacyDevicePath: &partitionPath,
		Id:               &partitionID,
		FsType:           &fsType,
		Name:             &name,
		Size:             &size,
		System:           &system,
	}

	partitions := []dto.Partition{partition}

	disk := dto.Disk{
		Partitions: &partitions,
	}

	assert.NotNil(t, disk.Partitions)
	assert.Len(t, *disk.Partitions, 1)
	assert.Equal(t, partitionName, *(*disk.Partitions)[0].LegacyDeviceName)
	assert.Equal(t, fsType, *(*disk.Partitions)[0].FsType)
}

func TestDisk_WithSmartInfo(t *testing.T) {
	smartInfo := dto.SmartInfo{
		DiskType: "SATA",
		Temperature: dto.SmartTempValue{
			Value: 45,
			Min:   20,
			Max:   80,
		},
		PowerOnHours: dto.SmartRangeValue{
			Value: 1000,
			Code:  9,
		},
	}

	disk := dto.Disk{
		SmartInfo: &smartInfo,
	}

	assert.NotNil(t, disk.SmartInfo)
	assert.Equal(t, "SATA", disk.SmartInfo.DiskType)
	assert.Equal(t, 45, disk.SmartInfo.Temperature.Value)
	assert.Equal(t, 1000, disk.SmartInfo.PowerOnHours.Value)
}

func TestPartition_Fields(t *testing.T) {
	legacyPath := "/dev/sda1"
	legacyName := "sda1"
	devicePath := "/dev/disk/by-id/part-123"
	id := "part-123"
	fsType := "ntfs"
	name := "Windows"
	size := 1000000000
	system := true

	partition := dto.Partition{
		LegacyDevicePath: &legacyPath,
		LegacyDeviceName: &legacyName,
		DevicePath:       &devicePath,
		Id:               &id,
		FsType:           &fsType,
		Name:             &name,
		Size:             &size,
		System:           &system,
	}

	assert.Equal(t, legacyPath, *partition.LegacyDevicePath)
	assert.Equal(t, legacyName, *partition.LegacyDeviceName)
	assert.Equal(t, devicePath, *partition.DevicePath)
	assert.Equal(t, id, *partition.Id)
	assert.Equal(t, fsType, *partition.FsType)
	assert.Equal(t, name, *partition.Name)
	assert.Equal(t, size, *partition.Size)
	assert.True(t, *partition.System)
}

func TestPartition_WithMountPoints(t *testing.T) {
	mountPath := "/mnt/data"
	hostMount := dto.MountPointData{
		Path: mountPath,
		Type: "HOST",
	}
	hostMountPoints := []dto.MountPointData{hostMount}

	addonMount := dto.MountPointData{
		Path: mountPath,
		Type: "ADDON",
	}
	addonMountPoints := []dto.MountPointData{addonMount}

	partition := dto.Partition{
		HostMountPointData: &hostMountPoints,
		MountPointData:     &addonMountPoints,
	}

	assert.NotNil(t, partition.HostMountPointData)
	assert.Len(t, *partition.HostMountPointData, 1)
	assert.Equal(t, mountPath, (*partition.HostMountPointData)[0].Path)
	assert.NotNil(t, partition.MountPointData)
	assert.Len(t, *partition.MountPointData, 1)
	assert.Equal(t, mountPath, (*partition.MountPointData)[0].Path)
}

func TestDisk_NilFields(t *testing.T) {
	disk := dto.Disk{}

	assert.Nil(t, disk.LegacyDeviceName)
	assert.Nil(t, disk.LegacyDevicePath)
	assert.Nil(t, disk.DevicePath)
	assert.Nil(t, disk.ConnectionBus)
	assert.Nil(t, disk.Ejectable)
	assert.Nil(t, disk.Partitions)
	assert.Nil(t, disk.Id)
	assert.Nil(t, disk.Model)
	assert.Nil(t, disk.Removable)
	assert.Nil(t, disk.Revision)
	assert.Nil(t, disk.Seat)
	assert.Nil(t, disk.Serial)
	assert.Nil(t, disk.Size)
	assert.Nil(t, disk.Vendor)
	assert.Nil(t, disk.SmartInfo)
}

func TestPartition_NilFields(t *testing.T) {
	partition := dto.Partition{}

	assert.Nil(t, partition.LegacyDevicePath)
	assert.Nil(t, partition.LegacyDeviceName)
	assert.Nil(t, partition.DevicePath)
	assert.Nil(t, partition.Id)
	assert.Nil(t, partition.FsType)
	assert.Nil(t, partition.Name)
	assert.Nil(t, partition.Size)
	assert.Nil(t, partition.System)
	assert.Nil(t, partition.HostMountPointData)
	assert.Nil(t, partition.MountPointData)
}
