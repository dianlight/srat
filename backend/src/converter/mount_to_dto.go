package converter

import (
	"log/slog"

	"github.com/dianlight/srat/dto"
	"github.com/u-root/u-root/pkg/mount"
)

// goverter:converter
// goverter:output:file ./mount_to_dto_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:default:update
type MountToDto interface {
	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:ignore IsToMountAtStartup Shares InvalidError Warnings RefreshVersion
	// goverter:map Data CustomFlags | stringToMountFlags
	// goverter:map Device Type | pathToType
	// goverter:map Device DeviceId | deviceToDeviceId
	// goverter:map Path DiskLabel | DiskLabelFromPath
	// goverter:map Path DiskSerial | DiskSerialFromPath
	// goverter:map Path DiskSize | DiskSizeFromPath
	// goverter:map Path IsInvalid | isPathDirNotExists
	// goverter:map Path IsMounted | github.com/dianlight/srat/internal/osutil:IsMounted
	// goverter:map Path PathHash | github.com/shomali11/util/xhashes:SHA1
	// goverter:map Path IsWriteSupported | FSTypeIsWriteSupported
	// goverter:map Device Partition | partitionFromDevice
	// goverter:map FSType TimeMachineSupport | TimeMachineSupportFromFS
	// goverter:map Flags Flags | uintptrToMountFlags
	// goverter:context disks
	MountToMountPointData(source *mount.MountPoint, target *dto.MountPointData, disks []dto.Disk) error
}

func stringToMountFlags(source string) (*dto.MountFlags, error) {
	var ret dto.MountFlags
	slog.Debug("Converting mount data string to MounFlags", "data", source)
	err := ret.Scan(source)
	return &ret, err
}

// goverter:context disks
func partitionFromDevice(device string, disks []dto.Disk) *dto.Partition {
	for _, d := range disks {
		for _, p := range *d.Partitions {
			if p.DevicePath != nil && *p.DevicePath == device {
				return &p
			}
		}
	}
	return nil
}

func uintptrToMountFlags(source uintptr) (*dto.MountFlags, error) {
	var ret dto.MountFlags
	slog.Debug("Converting mount uintptr to MounFlags", "data", source)
	err := ret.Scan(source)
	return &ret, err
}
