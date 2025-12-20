package dto

import (
	"time"
)

// HDIdleDevice represents per-device configuration for API
type HDIdleDevice struct {
	DiskId         string        `json:"disk_id,omitempty"`                      // Unique and persistent id for drive.
	Supported      bool          `json:"supported"`                              // Whether HD idle is supported for this device
	DevicePath     string        `json:"device_path"`                            // e.g., "sda" or "/dev/disk/by-id/..."
	IdleTime       time.Duration `json:"idle_time"`                              // 0 = use default
	CommandType    HdidleCommand `json:"command_type,omitempty" enum:"scsi,ata"` // empty = use default
	PowerCondition uint8         `json:"power_condition"`
	// Enabled tri-state flag: "yes", "custom", or "no".
	Enabled HdidleEnabled `json:"enabled,omitempty" enum:"yes,custom,no"`
}

// HDIdleDeviceStatus represents the HD idle status for a single disk.
type HDIdleDeviceStatus struct {
	Name string `json:"name,omitempty"` // Resolved device name, e.g., "sda"
	//GivenName      string    `json:"given_name,omitempty"` // Given device name, e.g., "/dev/disk/by-id/..."
	SpunDown       bool      `json:"spun_down"`
	LastIOAt       time.Time `json:"last_io_at,omitempty"`       // ISO8601 timestamp
	SpinDownAt     time.Time `json:"spin_down_at,omitempty"`     // ISO8601 timestamp
	SpinUpAt       time.Time `json:"spin_up_at,omitempty"`       // ISO8601 timestamp
	IdleTimeMillis int64     `json:"idle_time_millis,omitempty"` // Configured idle time in milliseconds
	CommandType    string    `json:"command_type,omitempty"`     // "scsi" or "ata"
	Supported      bool      `json:"supported"`                  // Whether HD idle is supported for this device
	Enabled        bool      `json:"enabled"`                    // Whether HD idle monitoring is enabled for this device
}

// HDIdleDeviceSupport represents the HD idle support status for a device
type HDIdleDeviceSupport struct {
	// Supported indicates if the device supports HD idle spindown commands
	Supported bool
	// SupportsSCSI indicates if the device supports SCSI spindown commands
	SupportsSCSI bool
	// SupportsATA indicates if the device supports ATA spindown commands
	SupportsATA bool
	// RecommendedCommand is the recommended command type for this device
	RecommendedCommand *HdidleCommand
	// DevicePath is the resolved real path of the device
	DevicePath string
	// ErrorMessage contains any error message if the device is not supported
	ErrorMessage string
}
