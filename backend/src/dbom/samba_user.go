package dbom

import (
	"time"

	"gorm.io/gorm"
)

type SambaUsers []SambaUser

type SambaUser struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Username  string         `gorm:"primaryKey"`
	Password  string
	IsAdmin   bool
	RwShares  []ExportedShare `gorm:"many2many:user_rw_share;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	RoShares  []ExportedShare `gorm:"many2many:user_ro_share;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

/*
func (u *SambaUser) BeforeCreate(tx *gorm.DB) error {
	//if os.Getenv("SRAT_MOCK") == "true" {
	//		return nil
	//	}
	err := unixsamba.CreateSambaUser(tx.Statement.Context, u.Username, u.Password, unixsamba.UserOptions{
		CreateHome:    false,
		SystemAccount: false,
		Shell:         "/sbin/nologin",
	})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (u *SambaUser) BeforeUpdate(tx *gorm.DB) error {
	//if os.Getenv("SRAT_MOCK") == "true" {
	//	return nil
	//}
	/*
		if tx.Statement.Changed("Username") {
			newUsername := tx.Statement.Dest.(map[string]interface{})["username"].(string)
			err := unixsamba.RenameUsername(tx.Statement.Context, u.Username, newUsername, u.Password)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	* /
	newVar := tx.Statement.Changed()
	newVar1 := tx.Statement.Changed("Password")
	tlog.Debug("Before update:", "u", slog.Any("u", u), "tx.Statement.Changed()", newVar, "tx.Statement.Changed(\"Password\")", newVar1)
	if tx.Statement.Changed("Password") || tx.Statement.Changed("password") {
		newPassword := tx.Statement.Dest.(map[string]interface{})["password"].(string)
		err := unixsamba.ChangePassword(tx.Statement.Context, u.Username, newPassword)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	//if tx.Statement.Changed("DeletedAt") && !u.DeletedAt.Valid {
	//	return u.BeforeCreate(tx)
	//}
	return nil
}

func (u *SambaUser) AfterSave(tx *gorm.DB) error {
	//if os.Getenv("SRAT_MOCK") == "true" {
	//	return nil
	//}
	//tlog.Debug("After save:", "u", spew.Sdump(u), "tx.Statement.Changed()", tx.Statement.Changed(), "tx.Statement.Changed(\"Password\")", tx.Statement.Changed("Password"))
	if u.Username != "" && u.Password != "" {
		err := unixsamba.CheckSambaUser(tx.Statement.Context, u.Username, u.Password)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	/*
		if tx.Statement.Changed("Password") { // FIXME: Work only BeforeUpdate, not AfterUpdate, because in AfterUpdate the password is already updated and tx.Statement.Changed("Password") returns false
			err := unixsamba.ChangePassword(tx.Statement.Context, u.Username, u.Password)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	* /
	return nil
}

func (u *SambaUser) AfterDelete(tx *gorm.DB) (err error) {
	// Cancellazione Utende da Samba
	if u.Username == "" {
		return nil
	}
	//if os.Getenv("SRAT_MOCK") == "true" {
	//		return nil
	//	}

	err = unixsamba.DeleteSambaUser(tx.Statement.Context, u.Username)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
*/
