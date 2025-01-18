package converter

import (
	"fmt"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
)

// goverter:converter
// goverter:output:file ./config_to_dbom_conv_gen.go
// goverter:output:package github.com/dianlight/srat/converter
// goverter:extend StringToSambaUser
// goverter:extend SambaUserToString
// goverter:default:update
type ConfigToDbomConverter interface {
	// goverter:update target
	// goverter:ignore ID DeviceId Invalid
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	// goverter:context users
	ShareToExportedShare(source config.Share, target *dbom.ExportedShare, users dbom.SambaUsers) error

	// goverter:update target
	// goverter:context users
	ExportedShareToShare(source dbom.ExportedShare, target *config.Share) error

	// goverter:update target
	// goverter:ignore IsAdmin
	// goverter:ignore CreatedAt UpdatedAt DeletedAt
	UserToUSambaUser(source config.User, target *dbom.SambaUser) error

	// goverter:update target
	SambaUserToUser(source dbom.SambaUser, target *config.User) error

	// goverter:update target
	// goverter:map Options.Username Username
	// goverter:map Options.Password Password
	// goverter:ignore IsAdmin CreatedAt UpdatedAt DeletedAt
	ConfigToSambaUser(source config.Config, target *dbom.SambaUser) error
}

// goverter:context users
func StringToSambaUser(username string, users dbom.SambaUsers) (dbom.SambaUser, error) {
	for _, u := range users {
		if u.Username == username {
			return u, nil
		}
	}
	return dbom.SambaUser{Username: username}, fmt.Errorf("User not found: %s", username)
}

func SambaUserToString(user dbom.SambaUser) string {
	return user.Username
}
