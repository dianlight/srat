package dbom

import (
	"time"

	"gorm.io/gorm"
)

// HDIdleConfig represents the HDIdle configuration stored in the database
type HDIdleConfig struct {
	ID                      uint   `gorm:"primaryKey"`
	Enabled                 bool   `gorm:"default:false"`
	DefaultIdleTime         int    `gorm:"default:600"` // seconds
	DefaultCommandType      string `gorm:"default:scsi"`
	DefaultPowerCondition   uint8  `gorm:"default:0"`
	Debug                   bool   `gorm:"default:false"`
	LogFile                 string `gorm:"default:"`
	SymlinkPolicy           int    `gorm:"default:0"` // 0=once, 1=retry
	IgnoreSpinDownDetection bool   `gorm:"default:false"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
	DeletedAt               gorm.DeletedAt `gorm:"index"`
	Devices                 []HDIdleDevice `gorm:"foreignKey:ConfigID;constraint:OnDelete:CASCADE"`
}

// HDIdleDevice represents per-device configuration
type HDIdleDevice struct {
	ID             uint   `gorm:"primaryKey"`
	ConfigID       uint   `gorm:"index"`
	Name           string `gorm:"not null"` // e.g., "sda" or "/dev/disk/by-id/..."
	IdleTime       int    `gorm:"default:0"` // 0 = use default
	CommandType    string `gorm:"default:"`  // empty = use default
	PowerCondition uint8  `gorm:"default:0"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}
