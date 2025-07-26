package converter

import (
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/lsblk"
)

// goverter:converter
// goverter:output:file ./lsblk_to_dto_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:default:update
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:skipCopySameType
// goverter:enum:unknown @error
type LsblkToDtoConverter interface {
	// goverter:update target
	// goverter:useUnderlyingTypeMethods
	// goverter:ignore IsInvalid InvalidError Warnings Flags CustomFlags IsToMountAtStartup Shares
	// goverter:map Name Device
	// goverter:map Mountpoint Path
	// goverter:map Fstype FSType
	// goverter:map Mountpoint IsMounted | isMounted
	// goverter:map Mountpoint Type | pathToType
	// goverter:map Mountpoint PathHash | github.com/shomali11/util/xhashes:SHA1
	// goverter:useZeroValueOnPointerInconsistency
	LsblkInfoToMountPointData(source *lsblk.LSBKInfo, target *dto.MountPointData) error
}

func isMounted(mountpoint string) bool {
	return mountpoint != ""
}
