package converter

import (
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
)

// goverter:converter
// goverter:output:file ./config_to_dbom_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:extend StringToSambaUser
// goverter:extend SambaUserToString
// goverter:update:ignoreZeroValueField
// goverter:useZeroValueOnPointerInconsistency
// -goverter:useUnderlyingTypeMethods
// goverter:default:update
// goverter:wrapErrorsUsing github.com/dianlight/srat/converter/patherr
type ConfigToDbomConverter interface {
	// goverter:update target
	// goverter:ignore ID DeviceId MountPointData
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:context users
	ShareToExportedShareNoMountPointData(source config.Share, target *dbom.ExportedShare, users *dbom.SambaUsers) error

	// goverter:update target
	// goverter:ignore DefaultPath Flags Data BlockDeviceId
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:map FS FSType
	ShareToMountPointData(source config.Share, target *dbom.MountPointData) error

	// goverter:update target
	// goverter:map MountPointData.Path Path
	// goverter:map MountPointData.FSType FS
	ExportedShareToShare(source dbom.ExportedShare, target *config.Share) error

	// goverter:update target
	// goverter:ignore IsAdmin
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	UserToUSambaUser(source config.User, target *dbom.SambaUser) error

	// goverter:update target
	SambaUserToUser(source dbom.SambaUser, target *config.User) error

	// goverter:update target
	// -goverter:map Options.Username Username
	// -goverter:map Options.Password Password
	// goverter:ignore IsAdmin CreatedAt UpdatedAt DeletedAt
	ConfigToSambaUser(source config.Config, target *dbom.SambaUser) error
}

// goverter:context users
func StringToSambaUser(username string, users *dbom.SambaUsers) (dbom.SambaUser, error) {
	for _, u := range *users {
		if u.Username == username {
			return u, nil
		}
	}
	u := dbom.SambaUser{Username: username}
	*users = append(*users, u)
	return u, nil
}

func SambaUserToString(user dbom.SambaUser) string {
	return user.Username
}
