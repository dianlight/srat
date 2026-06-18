package types

import (
	"encoding/json"
)

// Device represents a storage device
type Device struct {
	Name string
	Type string
}

// NvmeControllerCapabilities represents NVMe controller capabilities
type NvmeControllerCapabilities struct {
	SelfTest bool `json:"self_test,omitempty"`
}

// NvmeSmartHealth represents NVMe SMART health information
type NvmeSmartHealth struct {
	CriticalWarning      int   `json:"critical_warning,omitempty"`
	Temperature          int   `json:"temperature,omitempty"`
	AvailableSpare       int   `json:"available_spare,omitempty"`
	AvailableSpareThresh int   `json:"available_spare_threshold,omitempty"`
	PercentageUsed       int   `json:"percentage_used,omitempty"`
	DataUnitsRead        int64 `json:"data_units_read,omitempty"`
	DataUnitsWritten     int64 `json:"data_units_written,omitempty"`
	HostReadCommands     int64 `json:"host_read_commands,omitempty"`
	HostWriteCommands    int64 `json:"host_write_commands,omitempty"`
	ControllerBusyTime   int64 `json:"controller_busy_time,omitempty"`
	PowerCycles          int64 `json:"power_cycles,omitempty"`
	PowerOnHours         int64 `json:"power_on_hours,omitempty"`
	UnsafeShutdowns      int64 `json:"unsafe_shutdowns,omitempty"`
	MediaErrors          int64 `json:"media_errors,omitempty"`
	NumErrLogEntries     int64 `json:"num_err_log_entries,omitempty"`
	WarningTempTime      int   `json:"warning_temp_time,omitempty"`
	CriticalCompTime     int   `json:"critical_comp_time,omitempty"`
	TemperatureSensors   []int `json:"temperature_sensors,omitempty"`
}

type NvmeSmartTestLog struct {
	CurrentOpeation   *int `json:"current_operation,omitempty"`
	CurrentCompletion *int `json:"current_completion,omitempty"`
}

// UserCapacity represents storage device capacity information
type UserCapacity struct {
	Blocks int64 `json:"blocks"`
	Bytes  int64 `json:"bytes"`
}

// SMARTInfo represents comprehensive SMART information for a storage device
type SMARTInfo struct {
	Device                     Device                      `json:"device"`
	ModelFamily                string                      `json:"model_family,omitempty"`
	ModelName                  string                      `json:"model_name,omitempty"`
	SerialNumber               string                      `json:"serial_number,omitempty"`
	Firmware                   string                      `json:"firmware_version,omitempty"`
	UserCapacity               *UserCapacity               `json:"user_capacity,omitempty"`
	RotationRate               *int                        `json:"rotation_rate,omitempty"` // Rotation rate in RPM (0 for SSDs, >0 for HDDs, nil if not available or not applicable)
	DiskType                   string                      `json:"-"`                       // Computed disk type: "SSD", "HDD", "NVMe", or "Unknown"
	InStandby                  bool                        `json:"in_standby,omitempty"`    // True if device is in standby/sleep mode (ATA only)
	ExitCodeInfo               *ExitCodeInfo               `json:"-"`                       // Computed from Smartctl.ExitStatus; nil when exit status is zero
	SmartStatus                *SmartStatus                `json:"smart_status,omitempty"`
	SmartSupport               *SmartSupport               `json:"smart_support,omitempty"`
	AtaSmartData               *AtaSmartData               `json:"ata_smart_data,omitempty"`
	NvmeSmartHealth            *NvmeSmartHealth            `json:"nvme_smart_health_information_log,omitempty"`
	NvmeSmartTestLog           *NvmeSmartTestLog           `json:"nvme_smart_test_log,omitempty"`
	NvmeControllerCapabilities *NvmeControllerCapabilities `json:"nvme_controller_capabilities,omitempty"`
	Temperature                *Temperature                `json:"temperature,omitempty"`
	PowerOnTime                *PowerOnTime                `json:"power_on_time,omitempty"`
	PowerCycleCount            int                         `json:"power_cycle_count,omitempty"`
	Smartctl                   *SmartctlInfo               `json:"smartctl,omitempty"` // Exec-backend metadata (smartctl version, exit_status, messages)
}

// SmartStatus represents the overall SMART health status
type SmartStatus struct {
	Running  bool `json:"running"`
	Passed   bool `json:"passed"`
	Damaged  bool `json:"damaged,omitempty"`
	Critical bool `json:"critical,omitempty"`
}

// SmartSupport represents SMART availability and enablement status.
type SmartSupport struct {
	Available bool `json:"available"`
	Enabled   bool `json:"enabled"`
}

// AtaSmartData represents ATA SMART attributes
type AtaSmartData struct {
	OfflineDataCollection *OfflineDataCollection `json:"offline_data_collection,omitempty"`
	SelfTest              *SelfTest              `json:"self_test,omitempty"`
	Capabilities          *Capabilities          `json:"capabilities,omitempty"`
	Table                 []SmartAttribute       `json:"table,omitempty"`
}

// StatusField represents a status field that can be either a simple string or a complex object
type StatusField struct {
	Value            int    `json:"value"`
	String           string `json:"string"`
	Passed           *bool  `json:"passed,omitempty"`
	RemainingPercent *int   `json:"remaining_percent,omitempty"`
}

// UnmarshalJSON allows StatusField to be parsed from either a JSON string
// (e.g., "completed") or a structured object with fields {value, string, passed, remaining_percent}.
func (s *StatusField) UnmarshalJSON(data []byte) error {
	// If the JSON value starts with a quote, it's a simple string
	if len(data) > 0 && data[0] == '"' {
		// Trim quotes and assign to String
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		s.String = str
		// Leave Value and Passed as zero values
		return nil
	}
	// Otherwise, parse as the structured form
	type alias StatusField
	var tmp alias
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	s.Value = tmp.Value
	s.String = tmp.String
	s.Passed = tmp.Passed
	s.RemainingPercent = tmp.RemainingPercent
	return nil
}

// OfflineDataCollection represents offline data collection status
type OfflineDataCollection struct {
	Status            *StatusField `json:"status,omitempty"`
	CompletionSeconds int          `json:"completion_seconds,omitempty"`
}

// PollingMinutes represents polling minutes for different test types
type PollingMinutes struct {
	Short      int `json:"short,omitempty"`
	Extended   int `json:"extended,omitempty"`
	Conveyance int `json:"conveyance,omitempty"`
}

// SelfTest represents self-test information
type SelfTest struct {
	Status         *StatusField    `json:"status,omitempty"`
	PollingMinutes *PollingMinutes `json:"polling_minutes,omitempty"`
}

// Capabilities represents SMART capabilities
type Capabilities struct {
	Values                      []int `json:"values,omitempty"`
	ExecOfflineImmediate        bool  `json:"exec_offline_immediate_supported,omitempty"`
	SelfTestsSupported          bool  `json:"self_tests_supported,omitempty"`
	ConveyanceSelfTestSupported bool  `json:"conveyance_self_test_supported,omitempty"`
}

// SelfTestInfo represents available self-tests and their durations
type SelfTestInfo struct {
	Available []string       `json:"available"`
	Durations map[string]int `json:"durations"`
}

// NvmeOptionalAdminCommands represents NVMe optional admin commands
type NvmeOptionalAdminCommands struct {
	SelfTest bool `json:"self_test,omitempty"`
}

// CapabilitiesOutput represents the output of smartctl -c -j
type CapabilitiesOutput struct {
	AtaSmartData               *AtaSmartData               `json:"ata_smart_data,omitempty"`
	NvmeControllerCapabilities *NvmeControllerCapabilities `json:"nvme_controller_capabilities,omitempty"`
	NvmeOptionalAdminCommands  *NvmeOptionalAdminCommands  `json:"nvme_optional_admin_commands,omitempty"`
}

// SmartAttribute represents a single SMART attribute
type SmartAttribute struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Value      int    `json:"value"`
	Worst      int    `json:"worst"`
	Thresh     int    `json:"thresh"`
	WhenFailed string `json:"when_failed,omitempty"`
	Flags      Flags  `json:"flags"`
	Raw        Raw    `json:"raw"`
}

// Flags represents attribute flags
type Flags struct {
	Value         int    `json:"value"`
	String        string `json:"string"`
	PreFailure    bool   `json:"prefailure"`
	UpdatedOnline bool   `json:"updated_online"`
	Performance   bool   `json:"performance"`
	ErrorRate     bool   `json:"error_rate"`
	EventCount    bool   `json:"event_count"`
	AutoKeep      bool   `json:"auto_keep"`
}

// Raw represents raw attribute value
type Raw struct {
	Value  int64  `json:"value"`
	String string `json:"string"`
}

// Temperature represents device temperature
type Temperature struct {
	Current int `json:"current"`
}

// PowerOnTime represents power on time
type PowerOnTime struct {
	Hours int `json:"hours"`
}

// Message represents a message from smartctl
type Message struct {
	String   string `json:"string"`
	Severity string `json:"severity,omitempty"`
}

// SmartctlInfo represents smartctl metadata and messages
type SmartctlInfo struct {
	Version    []int     `json:"version,omitempty"`
	Messages   []Message `json:"messages,omitempty"`
	ExitStatus int       `json:"exit_status,omitempty"`
}

// ProgressCallback is a function type for reporting progress
type ProgressCallback func(progress int, status string)

// ExitCodeInfo breaks down the smartctl exit status into semantic groups.
//
// Bit assignments (from the smartctl man page, and the JSON exit_status field):
//
//	ExecBits (mask 0x07, bits 0–2):
//	  0x01  command line did not parse
//	  0x02  device open failed
//	  0x04  SMART or ATA command to the disk failed
//
//	HealthBits (mask 0xF8, bits 3–7):
//	  0x08  SMART status check returned "DISK FAILING"
//	  0x10  pre-failure attributes found at or below threshold
//	  0x20  attributes were at or below threshold in the past
//	  0x40  device error log contains records of errors
//	  0x80  self-test log contains records of errors
type ExitCodeInfo struct {
	// ExecBits holds bits 0–2 (mask 0x07) of the smartctl exit status.
	// Non-zero values indicate execution failures such as a missing binary,
	// permission denied, or a device that could not be opened.
	ExecBits int `json:"exec_bits"`

	// HealthBits holds bits 3–7 (mask 0xF8) of the smartctl exit status.
	// Non-zero values indicate drive health events. Each bit maps directly to
	// the corresponding bit in the exit status (bit 3 → 0x08, bit 4 → 0x10,
	// etc.), preserving the original semantics described in the smartctl man page.
	HealthBits int `json:"health_bits"`
}

// DiscoveryResult holds the outcome of probing a single device during
// DiscoverDevices. It reports whether SMART data was readable with the
// auto-detected protocol and whether a SAT fallback was required.
type DiscoveryResult struct {
	// DevicePath is the path of the storage device (e.g., "/dev/sda").
	DevicePath string `json:"device_path"`

	// DetectedProtocol is the device type string used for the successful read
	// (e.g., "ata", "sat", "nvme"). Empty if the device was not readable.
	DetectedProtocol string `json:"detected_protocol,omitempty"`

	// SMARTReadable is true when at least one SMART read attempt (native
	// protocol or SAT fallback) produced a valid response.
	SMARTReadable bool `json:"smart_readable"`

	// SATFallbackRequired is true when the auto-detected protocol failed but
	// a retry with the explicit -d sat flag succeeded.
	SATFallbackRequired bool `json:"sat_fallback_required,omitempty"`

	// Model is the drive model name or model family string from the SMART data.
	Model string `json:"model,omitempty"`

	// Serial is the drive serial number from the SMART data.
	Serial string `json:"serial,omitempty"`
}

// WearLevelPercent returns the percentage of drive life used (0 = new, 100 = worn out),
// or nil when the value cannot be determined (HDDs, unknown types, or missing data).
//
// The source depends on the drive type (SMARTInfo.DiskType):
//
//   - NVMe: nvme_smart_health_information_log.percentage_used
//   - SSD:  ATA SMART attributes, tried in priority order:
//     1. Attribute 231 (SSD Life Left)       — used = 100 − normalized value
//     2. Attribute 177 (Wear Leveling Count) — used = 100 − normalized value
//     3. Attribute 173 (SSD Life Used)       — used = raw value
//   - HDD / Unknown: nil
//
// The returned value is always clamped to [0, 100].
func (s *SMARTInfo) WearLevelPercent() *int {
	clamp := func(v int) *int {
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		return &v
	}

	switch s.DiskType {
	case "NVMe":
		if s.NvmeSmartHealth == nil {
			return nil
		}
		return clamp(s.NvmeSmartHealth.PercentageUsed)

	case "SSD":
		if s.AtaSmartData == nil {
			return nil
		}
		// Single-pass scan: track the best match found so far by priority.
		var byAttr231, byAttr177, byAttr173 *int
		for _, attr := range s.AtaSmartData.Table {
			switch attr.ID {
			case SmartAttrSSDLifeLeft: // 231 — normalized value = remaining life %
				byAttr231 = clamp(100 - attr.Value)
			case SmartAttrWearLevelingCount: // 177 — normalized value = remaining life %
				byAttr177 = clamp(100 - attr.Value)
			case SmartAttrSSDLifeUsed: // 173 — raw value = used life %
				byAttr173 = clamp(int(attr.Raw.Value))
			}
		}
		if byAttr231 != nil {
			return byAttr231
		}
		if byAttr177 != nil {
			return byAttr177
		}
		return byAttr173

	default:
		return nil
	}
}
