package converter

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dianlight/srat/dbom"
	"github.com/u-root/u-root/pkg/mount"
)

// goverter:converter
// goverter:output:file ./mount_to_dbom_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:default:update
type MountToDbom interface {
	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:ignore CreatedAt UpdatedAt DeletedAt IsToMountAtStartup Shares Flags
	// g.overter:map Flags Flags | uintptrToMounFlags
	// goverter:map Data Data | stringToMounFlags
	// goverter:map Device Type | pathToType
	// goverter:map Device DeviceId | deviceToDeviceId
	MountToMountPointPath(source *mount.MountPoint, target *dbom.MountPointPath) error
}

func stringToMounFlags(source string) (*dbom.MounDataFlags, error) {
	var ret dbom.MounDataFlags
	slog.Debug("Converting mount data string to MounDataFlags", "data", source)
	err := ret.Scan(source)
	return &ret, err
}

func deviceToDeviceId(source string) (string, error) {
	deviceID := ""
	entries, err := os.ReadDir("/dev/disk/by-id/")
	if err == nil {
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				linkPath := filepath.Join("/dev/disk/by-id/", entry.Name())
				resolved, err := filepath.EvalSymlinks(linkPath)
				if err != nil {
					continue
				}
				//slog.Debug("Resolved symlink", "link", linkPath, "resolved", resolved, "source", source)
				if resolved == source || linkPath == source {
					deviceID = "by-id-" + entry.Name()
					break
				}
			}
		}
	}
	return deviceID, nil
}
