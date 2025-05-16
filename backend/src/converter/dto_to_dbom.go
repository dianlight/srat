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
	// goverter:map Flags Flags | stringsToMountDataFlags
	// goverter:useUnderlyingTypeMethods
	MountPointDataToMountPointPath(source dto.MountPointData, target *dbom.MountPointPath) error

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:useUnderlyingTypeMethods
	// goverter:map Flags Flags | mountDataFlagsToStrings
	MountPointPathToMountPointData(source dbom.MountPointPath, target *dto.MountPointData) error

	// goverter:update target
	// goverter:ignore _
	// goverter:update:ignoreZeroValueField:basic no
	SambaUserToUser(source dbom.SambaUser, target *dto.User) error

	// goverter:update target
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:ignoreMissing
	UserToSambaUser(source dto.User, target *dbom.SambaUser) error
}

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

func mountDataFlagsToStrings(source dbom.MounDataFlags) (dest []string) {
	for _, flag := range source {
		val, err := flag.Value()
		if err != nil {
			continue
		}
		for _, mflag := range dto.MountFlagValues() {
			if int(mflag) == val {
				dest = append(dest, mflag.String())
				break
			}
		}
		//		slog.Debug("Transf", "flag", flag, "val", val, "dest", dest)
	}
	return dest

}

func exportedShareToString(source dbom.ExportedShare) string {
	return source.Name
}

func stringToExportedShare(source string) dbom.ExportedShare {
	return dbom.ExportedShare{
		Name: source,
	}
}
