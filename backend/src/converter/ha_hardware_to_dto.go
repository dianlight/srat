package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/u-root/u-root/pkg/mount"
)

// goverter:converter
// goverter:output:file ./ha_hardware_to_dto_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:default:update
type HaHardwareToDto interface {
	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:map Filesystems Partitions | filesystemsToPartitionsMap
	// goverter:useUnderlyingTypeMethods
	// goverter:skipCopySameType
	// goverter:map . LegacyDeviceName | extractDevice
	// goverter:ignore SmartInfo HDIdleDevice DevicePath LegacyDevicePath RefreshVersion
	DriveToDisk(source hardware.Drive, target *dto.Disk) error

	// goverter:useZeroValueOnPointerInconsistency
	// goverter:useUnderlyingTypeMethods
	// goverter:ignore MountPointData DevicePath FsType RefreshVersion DiskId FilesystemInfo
	// goverter:map Device LegacyDevicePath
	// goverter:map Device LegacyDeviceName | trimDevPrefix
	// goverter:map . HostMountPointData | mountPointsToMountPointDatas
	// goverter:map Id Id | filesystemUUIDToPartitionID
	// goverter:map Id Uuid | partitionIDToFilesystemUUID
	filesystemToPartition(source hardware.Filesystem) (dto.Partition, error)

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:useUnderlyingTypeMethods
	// goverter:ignore MountPointData DevicePath FsType RefreshVersion DiskId FilesystemInfo
	// goverter:map Device LegacyDevicePath
	// goverter:map Device LegacyDeviceName | trimDevPrefix
	// goverter:map . HostMountPointData | mountPointsToMountPointDatas
	// goverter:map Id Id | filesystemUUIDToPartitionID
	// goverter:map Id Uuid | partitionIDToFilesystemUUID
	FilesystemToPartition(source hardware.Filesystem, target *dto.Partition) error
}

func mountPointsToMountPointDatas(source hardware.Filesystem) *map[string]dto.MountPointData {
	mountPointDatas := make(map[string]dto.MountPointData)

	if source.MountPoints == nil || len(*source.MountPoints) == 0 {
		return &mountPointDatas
	}

	fstype, _, _ := mount.FSFromBlock(*source.Device)

	for _, s := range *source.MountPoints {
		mountPointDatas[s] = dto.MountPointData{
			Path:        s,
			DeviceId:    *source.Id,
			FSType:      &fstype,
			Flags:       nil,
			CustomFlags: nil,
			IsMounted:   true,
			Type:        "HOST",
		}
	}

	return &mountPointDatas
}

var deviceRegexp = regexp.MustCompile(`p?\d+$`)

func extractDevice(source hardware.Drive) *string {
	if source.Filesystems == nil || len(*source.Filesystems) == 0 || (*source.Filesystems)[0].Device == nil {
		return nil
	}
	// Trim trailing digits to get the disk device from a partition device (e.g., /dev/sda1 -> /dev/sda).
	originalDevice := *(*source.Filesystems)[0].Device
	trimmedDevice := deviceRegexp.ReplaceAllString(originalDevice, "")
	trimmedDevice = strings.TrimPrefix(trimmedDevice, "/dev/")
	return &trimmedDevice
}

func trimDevPrefix(source *string) *string {
	if source == nil {
		return nil
	}
	trimmedDevice := strings.TrimPrefix(*source, "/dev/")
	return &trimmedDevice
}

// filesystemsToPartitionsMap converts a list of HA filesystems to a map of Partitions keyed by Id.
// This is used by the goverter mapping for Filesystems -> Partitions.
// goverter:helper
func filesystemsToPartitionsMap(source *[]hardware.Filesystem) (*map[string]dto.Partition, error) {
	m := make(map[string]dto.Partition)
	if source == nil || len(*source) == 0 {
		return &m, nil
	}
	for _, fs := range *source {
		var p dto.Partition
		if fs.Device != nil {
			x := *fs.Device
			p.LegacyDevicePath = &x
		}
		p.LegacyDeviceName = trimDevPrefix(fs.Device)
		if fs.Id != nil {
			x, err := filesystemUUIDToPartitionID(fs.Id)
			if err != nil {
				return nil, fmt.Errorf("error converting filesystem ID to partition ID: %w", err)
			}
			p.Id = x
		}
		if fs.Id != nil {
			x := *fs.Id
			p.Uuid = &x
		}
		if fs.Name != nil {
			x := *fs.Name
			p.Name = &x
		}
		if fs.Size != nil {
			x := *fs.Size
			p.Size = &x
		}
		if fs.System != nil {
			x := *fs.System
			p.System = &x
		}
		p.HostMountPointData = mountPointsToMountPointDatas(fs)
		if p.Id != nil && *p.Id != "" {
			m[*p.Id] = p
		}
	}
	return &m, nil
}

// filesystemUUIDToPartitionID converts a filesystem UUID to a partition ID.
// It looks up the device path from /dev/disk/by-uuid/ and then finds the
// corresponding ID from /dev/disk/by-id/.
// Returns the UUID prefixed with "by-uuid-" if the ID cannot be found.
func filesystemUUIDToPartitionID(uuid *string) (*string, error) {
	// First, resolve the UUID to the actual device path
	if strings.HasPrefix(*uuid, "by-id-") {
		trimmed := strings.TrimPrefix(*uuid, "by-id-")
		return &trimmed, nil
	}
	if strings.HasPrefix(*uuid, "by-uuid-") {
		trimmed := strings.TrimPrefix(*uuid, "by-uuid-")
		uuid = &trimmed
	}
	uuidPath := filepath.Join("/dev/disk/by-uuid/", *uuid)
	devicePath, err := filepath.EvalSymlinks(uuidPath)
	if err != nil {
		// UUID symlink not found or cannot be resolved, return prefixed UUID
		x := "by-uuid-" + *uuid
		return &x, err
	}

	// Now find the corresponding ID in /dev/disk/by-id/
	entries, err := os.ReadDir("/dev/disk/by-id/")
	if err != nil {
		// Cannot read by-id directory, return prefixed UUID
		x := "by-uuid-" + *uuid
		return &x, err
	}

	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			idPath := filepath.Join("/dev/disk/by-id/", entry.Name())
			resolvedID, err := filepath.EvalSymlinks(idPath)
			if err != nil {
				continue
			}
			// Check if this ID symlink points to the same device
			if resolvedID == devicePath {
				x := entry.Name()
				return &x, nil
			}
		}
	}

	// No matching ID found, return prefixed UUID as fallback
	x := "by-uuid-" + *uuid
	return &x, fmt.Errorf(" No matching ID found, return prefixed UUID as fallback uuid: %s", *uuid)
}

func partitionIDToFilesystemUUID(id *string) (*string, error) {
	// First, resolve the UUID to the actual device path
	if strings.HasPrefix(*id, "by-uuid-") {
		trimmed := strings.TrimPrefix(*id, "by-uuid-")
		return &trimmed, nil
	}
	if strings.HasPrefix(*id, "by-id-") {
		trimmed := strings.TrimPrefix(*id, "by-id-")
		id = &trimmed
	}
	uuidPath := filepath.Join("/dev/disk/by-id/", *id)
	devicePath, err := filepath.EvalSymlinks(uuidPath)
	if err != nil {
		x := "by-id-" + *id
		return &x, err
	}

	// Now find the corresponding ID in /dev/disk/by-uuid/
	entries, err := os.ReadDir("/dev/disk/by-uuid/")
	if err != nil {
		// Cannot read by-id directory, return prefixed UUID
		x := "by-id-" + *id
		return &x, err
	}

	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			idPath := filepath.Join("/dev/disk/by-uuid/", entry.Name())
			resolvedID, err := filepath.EvalSymlinks(idPath)
			if err != nil {
				continue
			}
			// Check if this ID symlink points to the same device
			if resolvedID == devicePath {
				x := entry.Name()
				return &x, nil
			}
		}
	}

	// No matching ID found, return prefixed UUID as fallback
	x := "by-uuid-" + *id
	return &x, fmt.Errorf(" No matching ID found, return prefixed UUID as fallback uuid: %s", *id)
}
