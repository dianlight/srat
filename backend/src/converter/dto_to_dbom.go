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
	// goverter:update:ignoreZeroValueField:basic no
	SambaUserToUser(source dbom.SambaUser, target *dto.User) error

	// goverter:update target
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:ignoreMissing
	UserToSambaUser(source dto.User, target *dbom.SambaUser) error
	/*
		// goverter:update target
		// goverter:ignore CreatedAt UpdatedAt DeletedAt ID
		// goverter:ignore IsInvalid InvalidError Warnings
		// goverter:map Name Device
		// goverter:map Type FSType
		// goverter:map PartitionFlags Flags | stringsToMountDataFlags
		// goverter:map MountPoint IsMounted | isMountPointValid
		BlockPartitionToMountPointPath(source dto.BlockPartition, target *dbom.MountPointPath) error
	*/
	/*
		// goverter:update target
		// goverter:ignore CreatedAt UpdatedAt DeletedAt DeviceId IsMounted
		// goverter:ignore IsInvalid InvalidError Warnings
		// goverter:map Id MountPoint | idToMountPoint
		// goverter:map Device FSType | deviceToFSType
		// goverter:map Device Flags | deviceToMounDataFlags
		// goverter:map Id ID
		PartitionToMountPointPath(source dto.Partition, target *dbom.MountPointPath) error
	*/
}

//func isMountPointValid(source string) bool {
//	return source != ""
//}

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

/*
func idToMountPoint(id *string) (string, error) {
	if id == nil || *id == "" {
		return "", nil
	}
	return "/mnt/" + *id, nil
}
*/

/*
	func deviceToFSType(device *string) (string, error) {
	fs, _, err := mount.FSFromBlock(*device)
	if err != nil {
		return "", err
	}
	return fs, nil
}
*/

/*
func deviceToMounDataFlags(device *string) (dbom.MounDataFlags, error) {
	_, flags, err := mount.FSFromBlock(*device)
	if err != nil {
		return nil, err
	}
	tmp := dbom.MounDataFlags{}
	tmp.Scan(flags)
	return tmp, nil
}
*/
