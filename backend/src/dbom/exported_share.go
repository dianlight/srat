package dbom

import (
	"time"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

type ExportedShare struct {
	Name               string `gorm:"primarykey"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
	Disabled           bool
	Users              []SambaUser `gorm:"many2many:user_rw_share"`
	RoUsers            []SambaUser `gorm:"many2many:user_ro_share"`
	TimeMachine        bool
	Usage              dto.HAMountUsage
	MountPointDataPath string
	MountPointData     MountPointPath `gorm:"foreignKey:MountPointDataPath;references:Path;"`
}

func (u *ExportedShare) BeforeSave(tx *gorm.DB) error {
	if u.Name == "" {
		return errors.Errorf("Invalid name for exported share")
	}
	/*
		if errors.Is(tx.First(&ExportedShare{}, "name = ?", u.Name).Error, gorm.ErrRecordNotFound) {
			// Create without users
			saveuses := u.Users
			u.Users = nil
			saveuses_r := u.RoUsers
			u.RoUsers = nil
			if err := tx.Session(&gorm.Session{SkipHooks: true}).Create(u).Error; err != nil {
				return errors.WithStack(err)
			}
			u.Users = saveuses
			u.RoUsers = saveuses_r
		}
	*/
	return nil
}
