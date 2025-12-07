package converter

import (
	"log/slog"

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
	// goverter:ignore CreatedAt UpdatedAt DeletedAt IsToMountAtStartup ExportedShare Flags
	// g.overter:map Flags Flags | uintptrToMounFlags
	// goverter:map Data Data | stringToMounFlags
	// goverter:map Device Type | pathToType
	// goverter:map Path Root
	// goverter:map Device DeviceId | deviceToDeviceId
	MountToMountPointPath(source *mount.MountPoint, target *dbom.MountPointPath) error
}

func stringToMounFlags(source string) (*dbom.MounDataFlags, error) {
	var ret dbom.MounDataFlags
	slog.Debug("Converting mount data string to MounDataFlags", "data", source)
	err := ret.Scan(source)
	return &ret, err
}
