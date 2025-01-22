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
	SharedResourceToExportedShareNoUsersNoMountPointData(source dto.SharedResource, target *dbom.ExportedShare) error

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:ignore DeviceId
	// -goverter:map  DeviceId BlockDeviceId
	// goverter:useUnderlyingTypeMethods
	DtoMountPointDataToMountPointData(source dto.MountPointData, target *dbom.MountPointData) error

	// goverter:update target
	// goverter:useZeroValueOnPointerInconsistency
	// -goverter:ignore CreatedAt UpdatedAt DeletedAt
	// -goverter:map   BlockDeviceId DeviceId
	// goverter:useUnderlyingTypeMethods
	MountPointDataToDtoMountPointData(source dbom.MountPointData, target *dto.MountPointData) error

	// goverter:update target
	// goverter:update:ignoreZeroValueField:basic no
	SambaUserToUser(source dbom.SambaUser, target *dto.User) error

	// goverter:update target
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:ignoreMissing
	UserToSambaUser(source dto.User, target *dbom.SambaUser) error

	// goverter:update target
	// goverter:map Options.Workgroup Workgroup
	// goverter:map Options.Mountoptions Mountoptions
	// goverter:map Options.AllowHost AllowHost
	// goverter:map Options.VetoFiles VetoFiles
	// goverter:map Options.CompatibilityMode CompatibilityMode
	// goverter:map Options.EnableRecycleBin EnableRecycleBin
	// goverter:map Options.Interfaces Interfaces
	// goverter:map Options.BindAllInterfaces BindAllInterfaces
	// goverter:map Options.LogLevel LogLevel
	// goverter:map Options.MultiChannel MultiChannel
	//PropertiesToSettings(source dbom.Propertie, target *dto.Settings) error

	// goverter:update target
	// goverter:map . Options | SettingsToOptions
	// goverter:ignore CurrentFile
	// goverter:ignore ConfigSpecVersion
	// goverter:ignore Shares
	// goverter:ignore DockerInterface DockerNet
	// goverter:ignoreMissing
	// goverter:context conv
	//SettingsToConfig(source dto.Settings, target *config.Config, conv ConfigToDtoConverter) error

	// goverter:update target
	// goverter:ignore Username Password
	// goverter:ignore Automount
	// goverter:ignore Moredisks AvailableDiskLog Medialibrary WSDD WSDD2 HDDIdle Smart MQTTNextGen MQTTEnable
	// goverter:ignore MQTTHost MQTTUsername MQTTPassword MQTTPort MQTTTopic
	// goverter:ignore Autodiscovery MOF
	// goverter:ignore OtherUsers ACL
	//_SettingsToOptions(source dto.Settings, target *config.Options) error

	// goverter:update target
	// goverter:map Options.Username Username
	// goverter:map Options.Password Password
	// goverter:ignore IsAdmin
	//ConfigToUser(source config.Config, target *dto.User) error
}
