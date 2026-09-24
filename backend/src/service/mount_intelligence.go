package service

import (
	"context"
	"strings"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service/filesystem"
	"github.com/dianlight/srat/service/volume"
)

func enrichSharePartitionFromCache(share *dto.SharedResource, disks *dto.DiskMap) {
	if share == nil || share.MountPointData == nil || disks == nil {
		return
	}
	if share.MountPointData.Partition != nil && share.MountPointData.Partition.FsType != nil && *share.MountPointData.Partition.FsType != "" {
		if share.MountPointData.FSType == nil || *share.MountPointData.FSType == "" {
			share.MountPointData.FSType = share.MountPointData.Partition.FsType
		}
		return
	}

	partitionID := strings.TrimSpace(share.MountPointData.DeviceId)
	if partitionID == "" {
		return
	}

	partition, _, found := disks.GetPartitionByID(partitionID)
	if !found || partition == nil {
		return
	}

	share.MountPointData.Partition = partition
	if share.MountPointData.FSType == nil || *share.MountPointData.FSType == "" {
		share.MountPointData.FSType = partition.FsType
	}
}

func isShareNFSExportable(ctx context.Context, share dto.SharedResource) bool {
	if share.MountPointData == nil || share.MountPointData.Partition == nil {
		return false
	}

	partition := share.MountPointData.Partition
	if partition.FsType != nil && *partition.FsType != "" {
		registry := filesystem.NewRegistry()
		adapter, err := registry.Get(*partition.FsType)
		if err == nil {
			return adapter.IsExportable(ctx)
		}
	}

	if partition.FilesystemInfo != nil && partition.FilesystemInfo.Support != nil {
		return partition.FilesystemInfo.Support.IsExportable
	}

	return false
}

func resolveActualMountPointPath(share dto.SharedResource) string {
	if share.MountPointData == nil {
		return ""
	}

	if share.MountPointData.Path != "" {
		return share.MountPointData.Path
	}

	partition := share.MountPointData.Partition
	if partition == nil || partition.MountPointData == nil {
		return ""
	}

	for _, mountPoint := range *partition.MountPointData {
		if mountPoint.Path == "" {
			continue
		}
		if mountPoint.IsMounted {
			return mountPoint.Path
		}
	}

	for _, mountPoint := range *partition.MountPointData {
		if mountPoint.Path != "" {
			return mountPoint.Path
		}
	}

	return ""
}

func matchPartitionWithDevName(partition *dto.Partition, devName string) bool {
	return volume.MatchPartitionWithDevName(partition, devName)
}
