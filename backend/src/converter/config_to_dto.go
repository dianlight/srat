package converter

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/u-root/u-root/pkg/mount"
)

// goverter:converter
// goverter:output:file ./config_to_dto_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:extend StringToDtoUser
// goverter:extend DtoUserToString
// goverter:extend secretToString
// goverter:extend stringToSecret
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:default:update
type ConfigToDtoConverter interface {
	// g.overter:update target
	// g.overter:ignore Invalid MountPointData IsHAMounted _ HaStatus
	// g.overter:context users
	//ShareToSharedResourceNoMountPointData(source config.Share, target *dto.SharedResource, users []dto.User) error

	// g.overter:update target
	// g.overter:ignore  Flags CustomFlags IsInvalid InvalidError Warnings Share IsToMountAtStartup
	// goverter:ignore Flags CustomFlags InvalidError Warnings Share RefreshVersion
	// goverter:ignore Partition
	// goverter:map Path IsMounted | github.com/dianlight/srat/internal/osutil:IsMounted
	// goverter:map IsInvalid | falseConst
	// goverter:map IsToMountAtStartup | truePConst
	// goverter:map Path DeviceId | mountPathToDeviceId
	// goverter:map Path Type | pathToType
	// goverter:map FS FSType
	// goverter:map Path IsWriteSupported | isWriteSupported
	// goverter:map FS TimeMachineSupport | TimeMachineSupportFromFS
	// goverter:map Path DiskLabel | DiskLabelFromPath
	// goverter:map Path DiskSerial | DiskSerialFromPath
	// goverter:map Path DiskSize | DiskSizeFromPath
	// goverter:map Path Root
	ShareToMountPointData(source config.Share) (*dto.MountPointData, error)

	// goverter:ignore _ Status
	// goverter:map Users Users | StringsToDtoUsers
	// goverter:map . MountPointData
	// goverter:context users
	ShareToSharedResource(source config.Share, users []dto.User) (dto.SharedResource, error)

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

	// goverter:update target
	// goverter:update:ignoreZeroValueField no
	// goverter:map TelemetryMode TelemetryMode | github.com/dianlight/srat/dto:ParseTelemetryMode
	// goverter:map HDIdleDefaultCommandType HDIdleDefaultCommandType | github.com/dianlight/srat/dto:ParseHdidleCommand
	// goverter:ignore HASmbPassword
	ConfigToSettings(source config.Config, target *dto.Settings) error

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

// goverter:context users
func StringsToDtoUsers(usernames []string, users []dto.User) ([]dto.User, error) {
	var result []dto.User
	for _, username := range usernames {
		user, err := StringToDtoUser(username, users)
		if err != nil {
			return nil, err
		}
		result = append(result, user)
	}
	return result, nil
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

func pathToType(path string) string {
	if path == "/mnt" || strings.HasPrefix(path, "/mnt/") {
		return "ADDON"
	}
	return "HOST"
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
