package converter

import (
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
)

// goverter:converter
// goverter:output:file ./dto_to_dbom_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:default:update
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:skipCopySameType
// goverter:extend exportedShareToString
// goverter:extend stringToExportedShare
// goverter:ignoreUnexported
// goverter:enum:unknown @error
type DtoToDbomConverter interface {
	// goverter:update target
	// goverter:ignore Invalid
	// goverter:useUnderlyingTypeMethods
	// goverter:ignore MountPointData
	// goverter:useZeroValueOnPointerInconsistency
	ExportedShareToSharedResourceNoMountPointData(source dbom.ExportedShare, target *dto.SharedResource) error

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:ignore Users RoUsers MountPointData
	// goverter:map MountPointData.Path MountPointDataPath
	// goverter:useUnderlyingTypeMethods
	SharedResourceToExportedShareNoUsersNoMountPointPath(source dto.SharedResource, target *dbom.ExportedShare) error

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:ignore DeviceId
	// goverter:map Flags Flags
	// goverter:map CustomFlags Data
	// goverter:useUnderlyingTypeMethods
	MountPointDataToMountPointPath(source dto.MountPointData, target *dbom.MountPointPath) error

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:useUnderlyingTypeMethods
	// goverter:map Flags Flags
	// goverter:map Data CustomFlags
	// goverter:map Path PathHash | github.com/shomali11/util/xhashes:MD5
	MountPointPathToMountPointData(source dbom.MountPointPath, target *dto.MountPointData) error

	// goverter:update target
	// goverter:ignore _
	// goverter:update:ignoreZeroValueField:basic no
	SambaUserToUser(source dbom.SambaUser, target *dto.User) error

	// goverter:update target
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:ignoreMissing
	UserToSambaUser(source dto.User, target *dbom.SambaUser) error

	// goverter:ignore Description
	mountDataFlagToMountFlag(source dbom.MounDataFlag) (dest dto.MountFlag, err error)
}

/*
	func stringsToMountDataFlags(source []string) (dest dbom.MounDataFlags) {
		tmp := dto.MountFlags{}
		tmp.Scan(source)
		for _, flag := range tmp {
			val, err := flag.Value()
			if err != nil {
				continue
			}
			var tmp1 dbom.MounDataFlag
			tmp1.Scan(val)
			dest.Add(tmp1)
		}
		return dest
	}

	func mountFlagsToStrings(source dto.MountFlags) (dest []string) {
		for _, flag := range source {
			val, err := flag.Value()
			if err != nil {
				continue
			}
			dest = append(dest, val.(string))
		}
		return dest
	}

	func stringToMountFlags(source []string) (dest []dto.MountFlag) {
		for _, flag := range source {
			var tmp dto.MountFlag
			err := tmp.Scan(flag)
			if err != nil {
				continue
			}
			dest = append(dest, tmp)
		}
		return dest
	}
*/
func exportedShareToString(source dbom.ExportedShare) string {
	return source.Name
}

func stringToExportedShare(source string) dbom.ExportedShare {
	return dbom.ExportedShare{
		Name: source,
	}
}
