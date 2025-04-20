package converter

import (
	"strings"

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
	// goverter:ignore CreatedAt UpdatedAt DeletedAt DeviceId IsInvalid InvalidError Warnings IsMounted
	// goverter:map Device Device | removeDevPrefix
	// goverter:map Flags Flags | uintptrToMounDataFlags
	// goverter:map Device Type | pathToType
	MountToMountPointPath(source *mount.MountPoint, target *dbom.MountPointPath) error
}

func uintptrToMounDataFlags(source uintptr) (dbom.MounDataFlags, error) {
	var ret dbom.MounDataFlags
	err := ret.Scan(source)
	return ret, err
}

func removeDevPrefix(source string) (string, error) {
	ret, _ := strings.CutPrefix(source, "/dev/")
	return ret, nil
}
