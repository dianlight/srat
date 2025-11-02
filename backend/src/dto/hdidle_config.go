package dto

// HDIdleDeviceDTO represents per-device configuration for API
type HDIdleDeviceDTO struct {
	DevicePath     string        `json:"device_path"`                            // e.g., "sda" or "/dev/disk/by-id/..."
	IdleTime       int           `json:"idle_time"`                              // 0 = use default
	CommandType    HdidleCommand `json:"command_type,omitempty" enum:"scsi,ata"` // empty = use default
	PowerCondition uint8         `json:"power_condition"`
	// Enabled tri-state flag: "default", "yes", or "no".
	Enabled HdidleEnabled `json:"enabled,omitempty" enum:"default,yes,no"`
}
