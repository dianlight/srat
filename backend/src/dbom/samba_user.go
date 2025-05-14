package dbom

import (
	"time"

	"github.com/dianlight/srat/unixsamba"
	"gitlab.com/tozd/go/errors"
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
}

func (u *SambaUser) BeforeSave(tx *gorm.DB) error {
	if u.Username == "" {
		return errors.Errorf("username cannot be empty")
	}
	return nil
}

func (u *SambaUser) BeforeCreate(tx *gorm.DB) error {
	err := unixsamba.CreateSambaUser(u.Username, u.Password, unixsamba.UserOptions{
		CreateHome:    false,
		SystemAccount: false,
		Shell:         "/sbin/nologin",
	})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (u *SambaUser) AfterDelete(tx *gorm.DB) (err error) {
	// Cancellazione Utende da Samba
	err = unixsamba.DeleteSambaUser(u.Username, true, true)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
