package converter

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/tlog"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/xorcare/pointer"
)

// goverter:converter
// goverter:output:file ./config_to_dto_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:extend StringToDtoUser
// goverter:extend DtoUserToString
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:default:update
type ConfigToDtoConverter interface {
	// g.overter:update target
	// g.overter:ignore Invalid MountPointData IsHAMounted _ HaStatus
	// g.overter:context users
	//ShareToSharedResourceNoMountPointData(source config.Share, target *dto.SharedResource, users []dto.User) error

	// g.overter:update target
	// g.overter:ignore  Flags CustomFlags IsInvalid InvalidError Warnings Shares IsToMountAtStartup
	// g.overter:map Path IsMounted | github.com/dianlight/srat/internal/osutil:IsMounted
	// g.overter:map Path DeviceId | PathToSource
	// g.overter:map Path Type | pathToType
	// g.overter:map Path PathHash | github.com/shomali11/util/xhashes:SHA1
	// g.overter:map FS FSType
	// g.overter:map Path IsWriteSupported | FSTypeIsWriteSupported
	// g.overter:map FS TimeMachineSupport | TimeMachineSupportFromFS
	// g.overter:map Path DiskLabel | DiskLabelFromPath
	// g.overter:map Path DiskSerial | DiskSerialFromPath
	// g.overter:map Path DiskSize | DiskSizeFromPath
	//ShareToMountPointData(source config.Share, target *dto.MountPointData) error

	// goverter:update target
	// goverter:map MountPointData.Path Path
	// goverter:map MountPointData.FSType FS
	// goverter:context users
	SharedResourceToShare(source dto.SharedResource, target *config.Share) error

	// g:overter:update target
	// g:overter:ignore IsAdmin _  RwShares RoShares
	//OtherUserToUser(source config.User, target *dto.User) error

	// goverter:update target
	UserToOtherUser(source dto.User, target *config.User) error

	// g.overter:update target
	// g.overter:update:ignoreZeroValueField no
	// g.overter:map UpdateChannel UpdateChannel | github.com/dianlight/srat/dto:ParseUpdateChannel
	// g.overter:map TelemetryMode TelemetryMode | github.com/dianlight/srat/dto:ParseTelemetryMode
	// g.overter:ignore ExportStatsToHA
	//ConfigToSettings(source config.Config, target *dto.Settings) error

	// g.overter:update target
	// g.overter:ignore CurrentFile
	// g.overter:ignore ConfigSpecVersion
	// g.overter:ignore Shares
	// g.overter:ignore DockerInterface DockerNet
	// g.overter:ignoreMissing
	// g.overter:context conv
	//SettingsToConfig(source dto.Settings, target *config.Config, conv ConfigToDtoConverter) error

	// g.overter:update target
	// g.overter:ignore IsAdmin _  RwShares RoShares
	//ConfigToUser(source config.Config, target *dto.User) error
}

// goverter:context users
func StringToDtoUser(username string, users []dto.User) (dto.User, error) {
	for _, u := range users {
		if u.Username == username {
			return u, nil
		}
	}
	return dto.User{Username: username}, fmt.Errorf("User not found: %s", username)
}

func DtoUserToString(user dto.User) string {
	return user.Username
}

func PathToSource(path string) string {
	info, err := osutil.LoadMountInfo()
	if err != nil {
		slog.Warn("Error loading mount info", "err", err)
		return ""
	}
	for _, m := range info {
		after, _ := strings.CutPrefix(m.MountSource, "/dev/")
		if m.MountDir == path {
			return after
		} else {
			same, _ := mount.SameFilesystem(path, m.MountDir)
			if same {
				return after
			}
		}

	}
	return ""
}

func pathToType(_ string) string {
	return "ADDON"
}

func DiskLabelFromPath(path string) *string {
	label, err := disk.Label(PathToSource(path))
	if err != nil {
		return nil
	}
	return &label
}

func DiskSerialFromPath(path string) *string {
	serial, err := disk.SerialNumber(PathToSource(path))
	if err != nil {
		return nil
	}
	return &serial
}

func DiskSizeFromPath(path string) *uint64 {
	usage, err := disk.Usage(path)
	if err != nil {
		return nil
	}
	return &usage.Total
}

// TimeMachineSupportFromFS returns the Time Machine support status for a given filesystem type.
func TimeMachineSupportFromFS(fsType string) *dto.TimeMachineSupport {
	switch fsType {
	case "ext2", "ext3", "ext4", "jfs", "squashfs", "xfs", "btrfs", "zfs", "ubifs", "yaffs2", "reiserfs", "reiserfs4", "orangefs", "lustre", "ocfs2":
		return &dto.TimeMachineSupports.SUPPORTED
	case "ntfs3", "ntfs":
		return &dto.TimeMachineSupports.EXPERIMENTAL
	case "vfat", "msdos", "iso9660", "erofs", "exfat":
		return &dto.TimeMachineSupports.UNSUPPORTED
	default:
		return &dto.TimeMachineSupports.UNKNOWN
	}
}

func FSTypeIsWriteSupported(path string) *bool {
	tlog.Trace("Checking if path is writable", "path", path, "isWritable", osutil.IsWritable(path))
	return pointer.Bool(osutil.IsWritable(path))

}
