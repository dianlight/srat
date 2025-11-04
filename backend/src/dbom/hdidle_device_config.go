package dbom

import (
	"time"

	"github.com/dianlight/srat/dto"
	"gorm.io/gorm"
)

// HDIdleDevice represents per-device configuration
type HDIdleDevice struct {
	DevicePath     string             `gorm:"primaryKey"`              // e.g., "sda" or "/dev/disk/by-id/..."
	IdleTime       int                `gorm:"default:0"`               // 0 = use default
	CommandType    *dto.HdidleCommand `gorm:"default:"`                // empty = use default
	PowerCondition uint8              `gorm:"default:0"`               // Power condition to send with the issued SCSI START STOP UNIT command. Possible values are 0-15 (inclusive). The default value of 0 works fine for disks accessible via the SCSI layer (USB, IEEE1394, ...), but it will NOT work as intended with real SCSI / SAS disks. A stopped SAS disk will not start up automatically on access, but requires a startup command for reactivation. Useful values for SAS disks are 2 for idle and 3 for standby.
	Enabled        dto.HdidleEnabled  `gorm:"type:text;default:'yes'"` // tri-state flag: "yes", "custom", or "no"
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}
