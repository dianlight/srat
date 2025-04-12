package converter

import (
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/lsblk"
)

// goverter:converter
// goverter:output:file ./lsblk_to_dbom_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:default:update
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:skipCopySameType
// goverter:enum:unknown @error
type LsblkToDbomConverter interface {
	// goverter:update target
	// goverter:useUnderlyingTypeMethods
	// goverter:ignore DeviceId IsInvalid InvalidError Warnings Flags CreatedAt UpdatedAt DeletedAt
	// goverter:map Name Device
	// goverter:map Mountpoint Path
	// goverter:map Fstype FSType
	// goverter:map Mountpoint IsMounted | isMounted
	// goverter:useZeroValueOnPointerInconsistency
	LsblkInfoToMountPointPath(source *lsblk.LSBKInfo, target *dbom.MountPointPath) error
}

func isMounted(mountpoint string) bool {
	return mountpoint != ""
}
