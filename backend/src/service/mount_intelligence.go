package service

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service/filesystem"
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
	if partition == nil || devName == "" {
		return false
	}

	fullDevName := devName
	if !strings.HasPrefix(devName, "/dev/") {
		fullDevName = filepath.Join("/dev", devName)
	}

	candidates := []string{}
	if partition.DevicePath != nil {
		candidates = append(candidates, *partition.DevicePath)
	}
	if partition.LegacyDevicePath != nil {
		candidates = append(candidates, *partition.LegacyDevicePath)
	}
	if partition.LegacyDeviceName != nil {
		candidates = append(candidates, *partition.LegacyDeviceName)
	}
	if partition.Id != nil {
		candidates = append(candidates, *partition.Id)
	}

	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		if trimmed == devName || trimmed == fullDevName {
			return true
		}
		if filepath.Base(trimmed) == devName {
			return true
		}
	}

	return false
}
