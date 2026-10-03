package service

import (
	"context"
	"strings"

	"github.com/dianlight/srat/dbom"
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

// forceUserGroupProber is the narrow capability needed to resolve the Samba
// force user/group declared by the filesystem adapter. FilesystemService
// implements it; tests can stub it without pulling the full service.
type forceUserGroupProber interface {
	GetSambaForceUserGroup(fsType string) (string, string)
}

// enrichExportedShareForceUserGroup resolves the Samba "force user" and
// "force group" for a share from its backing filesystem adapter and stores
// them on the ephemeral ExportedShare fields. When no adapter is available
// (nil prober, unknown fs, missing mount data) it falls back to the legacy
// "root"/"root" values so the generated smb.conf is unchanged.
func enrichExportedShareForceUserGroup(dbs *dbom.ExportedShare, md *dto.MountPointData, prober forceUserGroupProber) {
	if dbs == nil {
		return
	}
	forceUser, forceGroup := "root", "root"
	if prober != nil && md != nil {
		fsType := ""
		if md.FSType != nil {
			fsType = strings.TrimSpace(*md.FSType)
		}
		if fsType == "" && md.Partition != nil && md.Partition.FsType != nil {
			fsType = strings.TrimSpace(*md.Partition.FsType)
		}
		if fsType != "" {
			forceUser, forceGroup = prober.GetSambaForceUserGroup(fsType)
		}
	}
	dbs.ForceUser = forceUser
	dbs.ForceGroup = forceGroup
}
