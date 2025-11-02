package dbom

import (
	"time"

	"github.com/dianlight/srat/dto"
	"gorm.io/gorm"
)

// HDIdleDevice represents per-device configuration
type HDIdleDevice struct {
	DevicePath     string             `gorm:"primaryKey"` // e.g., "sda" or "/dev/disk/by-id/..."
	IdleTime       int                `gorm:"default:0"`  // 0 = use default
	CommandType    *dto.HdidleCommand `gorm:"default:"`   // empty = use default
	PowerCondition uint8              `gorm:"default:0"`
	Enabled        dto.HdidleEnabled  `gorm:"type:text;default:'default'"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}
