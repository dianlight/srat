package service

import (
	"context"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsShareNFSExportable_UsesAdapterWhenFsTypeAvailable(t *testing.T) {
	ctx := context.Background()

	share := dto.SharedResource{
		MountPointData: &dto.MountPointData{
			Partition: &dto.Partition{
				FsType: new("ext4"),
				FilesystemInfo: &dto.FilesystemInfo{
					Support: &dto.FilesystemSupport{IsExportable: false},
				},
			},
		},
	}

	assert.True(t, isShareNFSExportable(ctx, share), "ext4 should be exportable via adapter decision")
}

func TestIsShareNFSExportable_ApfsStaysNonExportable(t *testing.T) {
	ctx := context.Background()

	share := dto.SharedResource{
		MountPointData: &dto.MountPointData{
			Partition: &dto.Partition{
				FsType: new("apfs"),
				FilesystemInfo: &dto.FilesystemInfo{
					Support: &dto.FilesystemSupport{IsExportable: true},
				},
			},
		},
	}

	assert.False(t, isShareNFSExportable(ctx, share), "apfs should not be exportable via adapter decision")
}

func TestResolveActualMountPointPath_PrefersMountedPartitionPath(t *testing.T) {
	mountedPath := "/mnt/live"
	fallbackPath := "/mnt/fallback"
	sharePath := ""

	partition := &dto.Partition{
		MountPointData: &map[string]dto.MountPointData{
			"fallback": {Path: fallbackPath, IsMounted: false},
			"live":     {Path: mountedPath, IsMounted: true},
		},
	}

	share := dto.SharedResource{
		MountPointData: &dto.MountPointData{
			Path:      sharePath,
			Partition: partition,
		},
	}

	assert.Equal(t, mountedPath, resolveActualMountPointPath(share))
}

func TestMatchPartitionWithDevName(t *testing.T) {
	partition := &dto.Partition{
		Id:               new("ata-test-part1"),
		LegacyDeviceName: new("sda1"),
		LegacyDevicePath: new("/dev/sda1"),
		DevicePath:       new("/dev/disk/by-id/ata-test-part1"),
	}

	assert.True(t, matchPartitionWithDevName(partition, "sda1"))
	assert.True(t, matchPartitionWithDevName(partition, "/dev/sda1"))
	assert.True(t, matchPartitionWithDevName(partition, "ata-test-part1"))
	assert.False(t, matchPartitionWithDevName(partition, "sdb1"))
}

func TestEnrichSharePartitionFromCache_PopulatesPartitionAndFsType(t *testing.T) {
	partitionID := "part-001"
	diskID := "disk-001"
	fsType := "ext4"
	devPath := "/dev/disk/by-id/test-part-001"

	disks := dto.DiskMap{
		diskID: {
			Id: &diskID,
			Partitions: &map[string]dto.Partition{
				partitionID: {
					Id:         &partitionID,
					DiskId:     &diskID,
					FsType:     &fsType,
					DevicePath: &devPath,
				},
			},
		},
	}

	share := dto.SharedResource{MountPointData: &dto.MountPointData{DeviceId: partitionID}}
	enrichSharePartitionFromCache(&share, &disks)

	require.NotNil(t, share.MountPointData.Partition)
	require.NotNil(t, share.MountPointData.FSType)
	assert.Equal(t, fsType, *share.MountPointData.FSType)
	assert.Equal(t, partitionID, *share.MountPointData.Partition.Id)
}

func TestHashDirtyTracker_IsStableAndDistinct(t *testing.T) {
	trackerA := dto.DataDirtyTracker{Users: true, Shares: false, Settings: true}
	trackerB := dto.DataDirtyTracker{Users: true, Shares: true, Settings: true}

	hashA1 := hashDirtyTracker(trackerA)
	hashA2 := hashDirtyTracker(trackerA)
	hashB := hashDirtyTracker(trackerB)

	assert.Equal(t, hashA1, hashA2)
	assert.NotEqual(t, hashA1, hashB)
}
