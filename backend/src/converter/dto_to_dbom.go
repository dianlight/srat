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
	// goverter:ignore Users RoUsers MountPointData MountPointDataID
	// -goverter:map MountPointData.DeviceId DeviceId
	// goverter:useUnderlyingTypeMethods
	SharedResourceToExportedShareNoUsersNoMountPointPath(source dto.SharedResource, target *dbom.ExportedShare) error

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:ignore DeviceId
	// -goverter:map  DeviceId BlockDeviceId
	// goverter:useUnderlyingTypeMethods
	MountPointDataToMountPointPath(source dto.MountPointData, target *dbom.MountPointPath) error

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// -goverter:ignore CreatedAt UpdatedAt DeletedAt
	// -goverter:map   BlockDeviceId DeviceId
	// goverter:useUnderlyingTypeMethods
	MountPointPathToMountPointData(source dbom.MountPointPath, target *dto.MountPointData) error

	// goverter:update target
	// goverter:update:ignoreZeroValueField:basic no
	SambaUserToUser(source dbom.SambaUser, target *dto.User) error

	// goverter:update target
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:ignoreMissing
	UserToSambaUser(source dto.User, target *dbom.SambaUser) error

	// goverter:update target
	// goverter:ignore CreatedAt UpdatedAt DeletedAt ID PrimaryPath
	// goverter:ignore IsInvalid InvalidError Warnings
	// goverter:map Name Source
	// goverter:map MountPoint Path
	// goverter:map Type FSType
	// goverter:map PartitionFlags Flags
	// goverter:map MountPoint IsMounted | isMountPointValid
	BlockPartitionToMountPointPath(source dto.BlockPartition, target *dbom.MountPointPath) error
}

func isMountPointValid(source string) bool {
	return source != ""
}
