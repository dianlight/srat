package converter

import (
	"fmt"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
)

// goverter:converter
// goverter:output:file ./config_to_dto_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:extend StringToDtoUser
// goverter:extend DtoUserToString
// goverter:useZeroValueOnPointerInconsistency
// goverter:update:ignoreZeroValueField
// goverter:default:update
type ConfigToDtoConverter interface {
	// goverter:update target
	// goverter:ignore ID DeviceId Invalid
	// goverter:context users
	ShareToSharedResource(source config.Share, target *dto.SharedResource, users []dto.User) error

	// goverter:update target
	// goverter:context users
	SharedResourceToShare(source dto.SharedResource, target *config.Share) error

	// goverter:update target
	// goverter:ignore IsAdmin
	OtherUserToUser(source config.User, target *dto.User) error

	// goverter:update target
	UserToOtherUser(source dto.User, target *config.User) error

	// goverter:update target
	// -goverter:map Options.Workgroup Workgroup
	// -goverter:map Options.Mountoptions Mountoptions
	// -goverter:map Options.AllowHost AllowHost
	// -goverter:map Options.VetoFiles VetoFiles
	// -goverter:map Options.CompatibilityMode CompatibilityMode
	// -goverter:map Options.EnableRecycleBin EnableRecycleBin
	// -goverter:map Options.Interfaces Interfaces
	// -goverter:map Options.BindAllInterfaces BindAllInterfaces
	// -goverter:map Options.LogLevel LogLevel
	// -goverter:map Options.MultiChannel MultiChannel
	ConfigToSettings(source config.Config, target *dto.Settings) error

	// goverter:update target
	// -goverter:map . Options | SettingsToOptions
	// goverter:ignore CurrentFile
	// goverter:ignore ConfigSpecVersion
	// goverter:ignore Shares
	// goverter:ignore DockerInterface DockerNet
	// goverter:ignoreMissing
	// goverter:context conv
	SettingsToConfig(source dto.Settings, target *config.Config, conv ConfigToDtoConverter) error

	// -goverter:update target
	// -goverter:ignore Username Password
	// -goverter:ignore Automount
	// -goverter:ignore Moredisks AvailableDiskLog Medialibrary WSDD WSDD2 HDDIdle Smart MQTTNextGen MQTTEnable
	// -goverter:ignore MQTTHost MQTTUsername MQTTPassword MQTTPort MQTTTopic
	// -goverter:ignore Autodiscovery MOF
	// -goverter:ignore OtherUsers ACL
	//_SettingsToOptions(source dto.Settings, target *config.Options) error

	// goverter:update target
	// -goverter:map Options.Username Username
	// -goverter:map Options.Password Password
	// goverter:ignore IsAdmin
	ConfigToUser(source config.Config, target *dto.User) error
}

// goverter:context users
func StringToDtoUser(username string, users []dto.User) (dto.User, error) {
	for _, u := range users {
		if *u.Username == username {
			return u, nil
		}
	}
	return dto.User{Username: &username}, fmt.Errorf("User not found: %s", username)
}

func DtoUserToString(user dto.User) string {
	return *user.Username
}

/*
// goverter:context conv
func SettingsToOptions(source dto.Settings, conv ConfigToDtoConverter) (config.Options, error) {
	var target config.Options
	err := conv._SettingsToOptions(source, &target)
	return target, err
}
*/
