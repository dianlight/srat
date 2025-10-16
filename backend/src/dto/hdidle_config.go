package dto

// HDIdleConfigDTO represents the HDIdle configuration for API
type HDIdleConfigDTO struct {
	ID                      uint              `json:"id,omitempty"`
	Enabled                 bool              `json:"enabled"`
	DefaultIdleTime         int               `json:"default_idle_time"` // seconds
	DefaultCommandType      string            `json:"default_command_type"`
	DefaultPowerCondition   uint8             `json:"default_power_condition"`
	Debug                   bool              `json:"debug"`
	LogFile                 string            `json:"log_file,omitempty"`
	SymlinkPolicy           int               `json:"symlink_policy"` // 0=once, 1=retry
	IgnoreSpinDownDetection bool              `json:"ignore_spin_down_detection"`
	Devices                 []HDIdleDeviceDTO `json:"devices,omitempty"`
}

// HDIdleDeviceDTO represents per-device configuration for API
type HDIdleDeviceDTO struct {
	ID             uint   `json:"id,omitempty"`
	Name           string `json:"name"`                   // e.g., "sda" or "/dev/disk/by-id/..."
	IdleTime       int    `json:"idle_time"`              // 0 = use default
	CommandType    string `json:"command_type,omitempty"` // empty = use default
	PowerCondition uint8  `json:"power_condition"`
}
