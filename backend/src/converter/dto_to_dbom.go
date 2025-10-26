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

	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	HDIdleDeviceDTOToHDIdleDevice(source dto.HDIdleDeviceDTO) (dbom.HDIdleDevice, error)

	HDIdleDeviceToHDIdleDeviceDTO(source dbom.HDIdleDevice) (dto.HDIdleDeviceDTO, error)

	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:map MountPointData.Path MountPointDataPath
	sharedResourceToExportedShare(source dto.SharedResource) (dbom.ExportedShare, error)

	// goverter:ignore Invalid IsHAMounted HaStatus
	exportedShareToSharedResource(source dbom.ExportedShare) (dto.SharedResource, error)

	// goverter:map Path PathHash | github.com/shomali11/util/xhashes:SHA1
	// goverter:map Path IsMounted | github.com/dianlight/srat/internal/osutil:IsMounted
	// goverter:map Path IsInvalid | isPathDirNotExists
	// goverter:ignore Partition
	// goverter:map Data CustomFlags
	// goverter:ignore InvalidError Warnings
	// goverter:map Path IsWriteSupported | FSTypeIsWriteSupported
	// goverter:map FSType TimeMachineSupport | TimeMachineSupportFromFS
	// goverter:map Path DiskLabel | DiskLabelFromPath
	// goverter:map Path DiskSerial | DiskSerialFromPath
	// goverter:map Path DiskSize | DiskSizeFromPath
	mountPointPathToMountPointData(source dbom.MountPointPath) (dto.MountPointData, error)

	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:ignore DeviceId
	// goverter:map  CustomFlags Data
	mountPointDataToMountPointPath(source dto.MountPointData) (dbom.MountPointPath, error)

	// goverter:ignore Description ValueDescription ValueValidationRegex
	mountDataFlagToMountFlag(source dbom.MounDataFlag) (dest dto.MountFlag, err error)

	MountFlagsToMountDataFlags(source []dto.MountFlag) (dest dbom.MounDataFlags)

	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	userToSambaUser(source dto.User) (dbom.SambaUser, error)

	// goverter:update target
	// goverter:ignore Invalid IsHAMounted HaStatus
	// goverter:useUnderlyingTypeMethods
	// goverter:ignore MountPointData
	// goverter:useZeroValueOnPointerInconsistency
	ExportedShareToSharedResourceNoMountPointData(source dbom.ExportedShare, target *dto.SharedResource) error

	// goverter:useUnderlyingTypeMethods
	// goverter:useZeroValueOnPointerInconsistency
	ExportedSharesToSharedResources(source *[]dbom.ExportedShare) (target *[]dto.SharedResource, err error)

	// goverter:useUnderlyingTypeMethods
	// goverter:useZeroValueOnPointerInconsistency
	SharedResourcesToExportedShares(source *[]dto.SharedResource) (target *[]dbom.ExportedShare, err error)

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
	// goverter:map Flags Flags
	// goverter:map CustomFlags Data
	// goverter:useUnderlyingTypeMethods
	MountPointDataToMountPointPath(source dto.MountPointData, target *dbom.MountPointPath) error

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:useUnderlyingTypeMethods
	// goverter:ignore InvalidError Warnings
	// goverter:map Flags Flags
	// goverter:map Data CustomFlags
	// goverter:map Path IsInvalid | isPathDirNotExists
	// goverter:map Path IsMounted | github.com/dianlight/srat/internal/osutil:IsMounted
	// goverter:map Path PathHash | github.com/shomali11/util/xhashes:SHA1
	// goverter:map Path IsWriteSupported | FSTypeIsWriteSupported
	// goverter:map FSType TimeMachineSupport | TimeMachineSupportFromFS
	// goverter:map Path DiskLabel | DiskLabelFromPath
	// goverter:map Path DiskSerial | DiskSerialFromPath
	// goverter:map Path DiskSize | DiskSizeFromPath
	// goverter:map DeviceId Partition | partitionFromDeviceId
	// goverter:context disks
	MountPointPathToMountPointData(source dbom.MountPointPath, target *dto.MountPointData, disks []dto.Disk) error

	// goverter:update target
	// goverter:ignore _
	// goverter:update:ignoreZeroValueField:basic no
	SambaUserToUser(source dbom.SambaUser, target *dto.User) error

	// goverter:update target
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:ignoreMissing
	UserToSambaUser(source dto.User, target *dbom.SambaUser) error
}

func exportedShareToString(source dbom.ExportedShare) string {
	return source.Name
}

func stringToExportedShare(source string) dbom.ExportedShare {
	return dbom.ExportedShare{
		Name: source,
	}
}

// goverter:context disks
func partitionFromDeviceId(source string, disks []dto.Disk) *dto.Partition {
	for _, d := range disks {
		for _, p := range *d.Partitions {
			if p.Id != nil && *p.Id == source {
				return &p
			}
		}
	}
	return nil
}
