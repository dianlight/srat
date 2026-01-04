package dto

import (
	"time"
)

// HDIdleDevice represents per-device configuration for API
type HDIdleDevice struct {
	HDIdleDeviceSupport
	DiskId         string        `json:"disk_id,omitempty"`                      // Unique and persistent id for drive.
	IdleTime       time.Duration `json:"idle_time"`                              // 0 = use default
	CommandType    HdidleCommand `json:"command_type,omitempty" enum:"scsi,ata"` // empty = use default
	PowerCondition uint8         `json:"power_condition"`
	Enabled        HdidleEnabled `json:"enabled,omitempty" enum:"yes,custom,no"`
}

// HDIdleDeviceStatus represents the HD idle status for a single disk.
type HDIdleDeviceStatus struct {
	Name       string    `json:"name,omitempty"` // Resolved device name, e.g., "sda"
	SpunDown   bool      `json:"spun_down"`
	LastIOAt   time.Time `json:"last_io_at,omitempty"`   // ISO8601 timestamp
	SpinDownAt time.Time `json:"spin_down_at,omitempty"` // ISO8601 timestamp
	SpinUpAt   time.Time `json:"spin_up_at,omitempty"`   // ISO8601 timestamp
}

// HDIdleDeviceSupport represents the HD idle support status for a device
type HDIdleDeviceSupport struct {
	Supported          bool           `json:"supported,omitempty" readonly:"true"`           // Supported indicates if the device supports HD idle spindown commands
	SupportsSCSI       bool           `json:"supports_scsi,omitempty" readonly:"true"`       // SupportsSCSI indicates if the device supports SCSI spindown commands
	SupportsATA        bool           `json:"supports_ata,omitempty" readonly:"true"`        // SupportsATA indicates if the device supports ATA spindown commands
	RecommendedCommand *HdidleCommand `json:"recommended_command,omitempty" readonly:"true"` // RecommendedCommand is the recommended command type for this device
	DevicePath         string         `json:"device_path,omitempty" readonly:"true"`         // DevicePath is the resolved real path of the device
	ErrorMessage       string         `json:"error_message,omitempty" readonly:"true"`       // ErrorMessage contains any error message if the device is not supported
}
