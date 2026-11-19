package service

import (
	"context"
	"os"
	"strings"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service/filesystem"
	"github.com/dianlight/srat/service/volume"
)

func enrichSharePartitionFromCache(share *dto.SharedResource, disks *dto.DiskMap) {
	if share == nil || share.MountPointData == nil || disks == nil {
		return
	}

	// Reflect the live mount-point state from the volume cache: HA ownership,
	// mounted/invalid flags and write support are derived at runtime and are
	// never persisted on the share's mount row.
	if share.MountPointData.Path != "" {
		// Resolve the live mount point from the volume cache, tolerating path
		// form variants (spaces vs. underscores/dashes) that have caused stale
		// "invalid" flags in the past (#1585, #1073).
		path := share.MountPointData.Path
		mp, ok := disks.GetMountPointByPath(path)
		if !ok {
			if norm := volume.NormalizeMountPath(path); norm != path {
				if m, ok2 := disks.GetMountPointByPath(norm); ok2 && (share.MountPointData.DeviceId == "" || m.DeviceId == share.MountPointData.DeviceId) {
					mp, ok = m, true
					path = norm
				}
			}
		}
		if !ok {
			// Only accept a normalized candidate when it belongs to the same
			// device as the share, and only when there is exactly one such
			// candidate, so we never pick an arbitrary row by iteration order.
			var compatible []*dto.MountPointData
			for _, candidate := range disks.GetAllMountPoints() {
				if candidate == nil || candidate.Path == "" {
					continue
				}
				if volume.NormalizeMountPath(candidate.Path) != volume.NormalizeMountPath(share.MountPointData.Path) {
					continue
				}
				if share.MountPointData.DeviceId != "" && candidate.DeviceId != share.MountPointData.DeviceId {
					continue
				}
				compatible = append(compatible, candidate)
			}
			if len(compatible) == 1 {
				mp, ok = compatible[0], true
				path = compatible[0].Path
			}
		}
		if !ok && share.MountPointData.DeviceId != "" {
			// Ownership transfer: the same device can be remounted under an
			// HA path (/media, /share, /backup) instead of /mnt, or back
			// again. Path matching cannot find it, so fall back to a unique
			// live mount row for the device. Ambiguous matches are ignored
			// rather than guessed.
			var live []*dto.MountPointData
			for _, candidate := range disks.GetAllMountPoints() {
				if candidate == nil || candidate.Path == "" {
					continue
				}
				if candidate.DeviceId != share.MountPointData.DeviceId {
					continue
				}
				if !candidate.IsMounted || candidate.IsInvalid {
					continue
				}
				live = append(live, candidate)
			}
			if len(live) == 1 {
				mp, ok = live[0], true
				path = live[0].Path
			}
		}
		if ok && mp != nil {
			share.MountPointData.Path = path
			share.MountPointData.IsHAMounted = mp.IsHAMounted || volume.IsHAManagedMountPath(path)
			share.MountPointData.IsMounted = mp.IsMounted
			share.MountPointData.IsInvalid = mp.IsInvalid
			if mp.IsWriteSupported != nil {
				share.MountPointData.IsWriteSupported = mp.IsWriteSupported
			}
			// Keep the mount type in sync with the new owner so HA guards and
			// automount decisions apply to the re-associated share.
			if share.MountPointData.IsHAMounted {
				if share.MountPointData.Type == "" || share.MountPointData.Type == "ADDON" {
					share.MountPointData.Type = "HOST"
				}
			} else if share.MountPointData.Type == "HOST" {
				share.MountPointData.Type = "ADDON"
			}
		} else {
			// Cache miss: derive mount state from the OS so a mounted/existing
			// volume is not falsely marked invalid (#1585).
			if share.MountPointData.IsMounted {
				share.MountPointData.IsInvalid = false
			} else if info, err := os.Stat(share.MountPointData.Path); err == nil {
				if info.IsDir() {
					share.MountPointData.IsInvalid = false
				} else {
					share.MountPointData.IsInvalid = true
				}
			} else {
				share.MountPointData.IsInvalid = true
			}
			if !share.MountPointData.IsMounted {
				if mounted, err := osutil.IsMounted(share.MountPointData.Path); err == nil && mounted {
					share.MountPointData.IsMounted = true
					share.MountPointData.IsInvalid = false
				}
			} else {
				share.MountPointData.IsInvalid = false
			}
			// Cache miss: still derive HA ownership from the path so a share
			// on an HA-managed mount is never misclassified.
			if volume.IsHAManagedMountPath(share.MountPointData.Path) {
				share.MountPointData.IsHAMounted = true
				if share.MountPointData.Type == "" || share.MountPointData.Type == "ADDON" {
					share.MountPointData.Type = "HOST"
				}
			}
		}
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
