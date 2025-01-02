package config

import (
	"time"

	"gorm.io/gorm"
)

type SambaUser struct {
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Username  string         `json:"username" gorm:"primaryKey"`
	Password  string         `json:"password"`
}
