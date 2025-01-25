package converter

import (
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/u-root/u-root/pkg/mount"
)

// goverter:converter
// goverter:output:file ./mount_to_dbom_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// -goverter:extend funcToInt64
// -goverter:extend sliceToInt
// -goverter:extend int64ToTime
// goverter:default:update
type MountToDbom interface {
	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:ignore CreatedAt UpdatedAt DeletedAt ID DeviceId PrimaryPath Flags IsInvalid InvalidError Warnings IsMounted
	// goverter:map Device Source
	// goverter:map Flags Flags | uintptrToMounDataFlags
	// -goverter:map Connections Connections | sliceToLen
	MountToMountPointPath(source *mount.MountPoint, target *dbom.MountPointPath) error
}

func uintptrToMounDataFlags(source uintptr) (dto.MounDataFlags, error) {
	var ret dto.MounDataFlags
	err := ret.Scan(source)
	return ret, err
}

/* func int64ToTime(source int64) (time.Time, error) {
	return time.Unix(source/1000, 0), nil
}

func sliceToLen(source any) (int, error) {
	return len(source.([]any)), nil
}
*/
