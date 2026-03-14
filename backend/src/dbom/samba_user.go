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
