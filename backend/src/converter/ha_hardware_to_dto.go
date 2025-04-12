package converter

import (
	"log/slog"

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
	// goverter:map Filesystems Partitions
	// goverter:useUnderlyingTypeMethods
	// goverter:skipCopySameType
	DriveToDisk(source hardware.Drive, target *dto.Disk) error

	// goverter:useZeroValueOnPointerInconsistency
	// goverter:useUnderlyingTypeMethods
	// goverter:map . MountPointData | mountPointsToMountPointDatas
	filesystemToPartition(source hardware.Filesystem) dto.Partition

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:useUnderlyingTypeMethods
	// goverter:map . MountPointData | mountPointsToMountPointDatas
	FilesystemToPartition(source hardware.Filesystem, target *dto.Partition) error
}

func mountPointsToMountPointDatas(source hardware.Filesystem) *[]dto.MountPointData {
	var mountPointDatas []dto.MountPointData

	fstype, flags, err := mount.FSFromBlock(*source.Device)
	if err != nil {
		slog.Warn("Failed to get filesystem type and flags", "device", source.Device, "error", err)
		return nil
	}

	M_flags := dto.MountFlags{}
	M_flags.Scan(flags)

	for _, s := range *source.MountPoints {
		mountPointDatas = append(mountPointDatas, dto.MountPointData{
			Path:      s,
			Device:    *source.Device,
			FSType:    fstype,
			Flags:     M_flags.Strings(),
			IsMounted: true,
		})
	}
	return &mountPointDatas
}
