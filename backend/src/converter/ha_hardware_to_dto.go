package converter

import (
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
	// goverter:ignore SmartInfo DevicePath LegacyDevicePath HDIdleStatus RefreshVersion
	DriveToDisk(source hardware.Drive, target *dto.Disk) error

	// goverter:useZeroValueOnPointerInconsistency
	// goverter:useUnderlyingTypeMethods
	// goverter:ignore MountPointData  DevicePath FsType RefreshVersion DiskId
	// goverter:map Device LegacyDevicePath
	// goverter:map Device LegacyDeviceName | trimDevPrefix
	// goverter:map . HostMountPointData | mountPointsToMountPointDatas
	filesystemToPartition(source hardware.Filesystem) dto.Partition

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:useUnderlyingTypeMethods
	// goverter:ignore MountPointData  DevicePath FsType RefreshVersion DiskId
	// goverter:map Device LegacyDevicePath
	// goverter:map Device LegacyDeviceName | trimDevPrefix
	// goverter:map . HostMountPointData | mountPointsToMountPointDatas
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
func filesystemsToPartitionsMap(source *[]hardware.Filesystem) *map[string]dto.Partition {
	m := make(map[string]dto.Partition)
	if source == nil || len(*source) == 0 {
		return &m
	}
	for _, fs := range *source {
		var p dto.Partition
		if fs.Device != nil {
			x := *fs.Device
			p.LegacyDevicePath = &x
		}
		p.LegacyDeviceName = trimDevPrefix(fs.Device)
		if fs.Id != nil {
			x := *fs.Id
			p.Id = &x
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
	return &m
}
