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
// goverter:default:update
// g.overter:wrapErrorsUsing gitlab.com/tozd/go/errors
type ConfigToDbomConverter interface {
	// goverter:update target
	// goverter:ignore MountPointData
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:map Path MountPointDataPath
	// goverter:map Path MountPointDataRoot
	// goverter:context users
	ShareToExportedShareNoMountPointPath(source config.Share, target *dbom.ExportedShare, users *dbom.SambaUsers) error

	// goverter:update target
	// goverter:map MountPointData.Path Path
	// goverter:map MountPointData.FSType FS
	ExportedShareToShare(source dbom.ExportedShare, target *config.Share) error

	// goverter:update target
	SambaUserToUser(source dbom.SambaUser, target *config.User) error
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
