// Package smartmontools provides Go bindings for interfacing with smartmontools
// to monitor and manage storage device health using S.M.A.R.T. data.
//
// The library wraps the smartctl command-line utility and provides a clean,
// idiomatic Go API for accessing SMART information from storage devices.
package smartmontools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SMART attribute IDs for SSD detection
const (
	SmartAttrSSDLifeLeft       = 231 // SSD Life Left attribute
	SmartAttrSandForceInternal = 233 // SandForce Internal (SSD-specific)
	SmartAttrTotalLBAsWritten  = 234 // Total LBAs Written (SSD-specific)
)

// Commander interface for executing commands
type Commander interface {
	Command(name string, arg ...string) Cmd
}

// Cmd interface for command execution
type Cmd interface {
	Output() ([]byte, error)
	Run() error
}

// execCommander implements Commander using os/exec
type execCommander struct{}

func (e execCommander) Command(name string, arg ...string) Cmd {
	slog.Debug("Executing command", "name", name, "args", arg)
	return exec.Command(name, arg...)
}

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

// SMARTInfo represents SMART information for a device
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
	SmartStatus                SmartStatus                 `json:"smart_status,omitempty"`
	SmartSupport               *SmartSupport               `json:"smart_support,omitempty"`
	AtaSmartData               *AtaSmartData               `json:"ata_smart_data,omitempty"`
	NvmeSmartHealth            *NvmeSmartHealth            `json:"nvme_smart_health_information_log,omitempty"`
	NvmeControllerCapabilities *NvmeControllerCapabilities `json:"nvme_controller_capabilities,omitempty"`
	Temperature                *Temperature                `json:"temperature,omitempty"`
	PowerOnTime                *PowerOnTime                `json:"power_on_time,omitempty"`
	PowerCycleCount            int                         `json:"power_cycle_count,omitempty"`
	Smartctl                   *SmartctlInfo               `json:"smartctl,omitempty"`
}

// SmartStatus represents the overall SMART health status
type SmartStatus struct {
	Passed bool `json:"passed"`
}

// SmartSupport represents SMART availability and enablement status
type SmartSupport struct {
	Available bool `json:"available"`
	Enabled   bool `json:"enabled"`
}

// SMARTSupportInfo represents SMART support and enablement information
type SMARTSupportInfo struct {
	Supported bool
	Enabled   bool
}

// AtaSmartData represents ATA SMART attributes
type AtaSmartData struct {
	OfflineDataCollection *OfflineDataCollection `json:"offline_data_collection,omitempty"`
	SelfTest              *SelfTest              `json:"self_test,omitempty"`
	Capabilities          *Capabilities          `json:"capabilities,omitempty"`
	Table                 []SmartAttribute       `json:"table,omitempty"`
}

// OfflineDataCollection represents offline data collection status
type StatusField struct {
	Value  int    `json:"value"`
	String string `json:"string"`
	Passed *bool  `json:"passed,omitempty"`
}

// UnmarshalJSON allows StatusField to be parsed from either a JSON string
// (e.g., "completed") or a structured object with fields {value, string, passed}.
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
	return nil
}

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

// SmartClient interface defines the methods for interacting with smartmontools
type SmartClient interface {
	ScanDevices() ([]Device, error)
	GetSMARTInfo(devicePath string) (*SMARTInfo, error)
	CheckHealth(devicePath string) (bool, error)
	GetDeviceInfo(devicePath string) (map[string]interface{}, error)
	RunSelfTest(devicePath string, testType string) error
	RunSelfTestWithProgress(ctx context.Context, devicePath string, testType string, callback ProgressCallback) error
	GetAvailableSelfTests(devicePath string) (*SelfTestInfo, error)
	IsSMARTSupported(devicePath string) (*SMARTSupportInfo, error)
	EnableSMART(devicePath string) error
	DisableSMART(devicePath string) error
	AbortSelfTest(devicePath string) error
}

// Client represents a smartmontools client
type Client struct {
	smartctlPath       string
	commander          Commander
	deviceTypeCache    map[string]string // Maps device path to device type (e.g., "sat")
	deviceTypeCacheMux sync.RWMutex      // Protects deviceTypeCache
}

// NewClient creates a new smartmontools client
func NewClient() (SmartClient, error) {
	// Try to find smartctl in PATH
	path, err := exec.LookPath("smartctl")
	if err != nil {
		return nil, fmt.Errorf("smartctl not found in PATH: %w", err)
	}

	// Ensure smartctl is a compatible version (JSON output requires >= 7.0)
	if err := ensureCompatibleSmartctl(path); err != nil {
		return nil, err
	}

	return &Client{
		smartctlPath:    path,
		commander:       execCommander{},
		deviceTypeCache: loadDrivedbAddendum(),
	}, nil
}

// NewClientWithPath creates a new smartmontools client with a specific smartctl path
func NewClientWithPath(smartctlPath string) SmartClient {
	return &Client{
		smartctlPath:    smartctlPath,
		commander:       execCommander{},
		deviceTypeCache: loadDrivedbAddendum(),
	}
}

// NewClientWithCommander creates a new client with a custom commander (for testing)
func NewClientWithCommander(smartctlPath string, commander Commander) SmartClient {
	return &Client{
		smartctlPath:    smartctlPath,
		commander:       commander,
		deviceTypeCache: make(map[string]string),
	}
}

// ScanDevices scans for available storage devices
func (c *Client) ScanDevices() ([]Device, error) {
	cmd := c.commander.Command(c.smartctlPath, "--scan-open", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to scan devices: %w", err)
	}

	var result struct {
		Devices []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"devices"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse scan output: %w", err)
	}

	devices := make([]Device, len(result.Devices))
	for i, d := range result.Devices {
		devices[i] = Device{
			Name: d.Name,
			Type: d.Type,
		}
	}

	return devices, nil
}

// getCachedDeviceType retrieves a cached device type for the given device path
func (c *Client) getCachedDeviceType(devicePath string) (string, bool) {
	c.deviceTypeCacheMux.RLock()
	defer c.deviceTypeCacheMux.RUnlock()
	deviceType, ok := c.deviceTypeCache[devicePath]
	return deviceType, ok
}

// setCachedDeviceType stores a device type in the cache for the given device path
func (c *Client) setCachedDeviceType(devicePath, deviceType string) {
	c.deviceTypeCacheMux.Lock()
	defer c.deviceTypeCacheMux.Unlock()
	c.deviceTypeCache[devicePath] = deviceType
	slog.Debug("Cached device type", "devicePath", devicePath, "deviceType", deviceType)
}

// GetSMARTInfo retrieves SMART information for a device
func (c *Client) GetSMARTInfo(devicePath string) (*SMARTInfo, error) {
	// Check if we have a cached device type for this device
	var args []string
	var isATA bool
	if cachedType, ok := c.getCachedDeviceType(devicePath); ok {
		isATA = isATADevice(cachedType)
		if isATA {
			args = []string{"-d", cachedType, "--nocheck=standby", "-a", "-j", devicePath}
		} else {
			args = []string{"-d", cachedType, "-a", "-j", devicePath}
		}
	} else {
		// Assume ATA by default for --nocheck=standby
		args = []string{"--nocheck=standby", "-a", "-j", devicePath}
		isATA = true
	}

	cmd := c.commander.Command(c.smartctlPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// smartctl returns non-zero exit codes for various conditions
		// Bit 1 (exit code 2) indicates device is in standby mode
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			// Device is in standby mode
			var smartInfo SMARTInfo
			if len(output) > 0 && json.Unmarshal(output, &smartInfo) == nil {
				smartInfo.InStandby = true
				smartInfo.DiskType = determineDiskType(&smartInfo)
				return &smartInfo, nil
			}
			// If no JSON output, return minimal info
			return &SMARTInfo{
				Device:    Device{Name: devicePath},
				InStandby: true,
			}, nil
		}

		// We still want to parse the output if available and it's valid JSON
		if len(output) > 0 {
			var smartInfo SMARTInfo
			if json.Unmarshal(output, &smartInfo) == nil {
				// Valid JSON, treat error as warning
				//slog.Debug("smartctl returned error but provided valid JSON output", "error", err)
				// Check for error messages in the output
				if smartInfo.Smartctl != nil && len(smartInfo.Smartctl.Messages) > 0 {
					for _, msg := range smartInfo.Smartctl.Messages {
						slog.Warn("smartctl message", "severity", msg.Severity, "message", msg.String)
					}
				}

				// Check if this is an unknown USB bridge error and we haven't cached a type yet
				if isUnknownUSBBridge(&smartInfo) {
					_, hasCached := c.getCachedDeviceType(devicePath)
					if !hasCached {
						// First, check if this USB bridge is in our standard drivedb
						usbBridgeID := extractUSBBridgeID(&smartInfo)
						var deviceType string
						if usbBridgeID != "" {
							if knownType, ok := c.getCachedDeviceType(usbBridgeID); ok {
								deviceType = knownType
								slog.Info("Found USB bridge in drivedb", "usbBridgeID", usbBridgeID, "deviceType", deviceType)
							}
						}

						// If not in drivedb, default to sat
						if deviceType == "" {
							deviceType = "sat"
							slog.Info("Unknown USB bridge detected, retrying with -d sat", "devicePath", devicePath)
						}

						// Retry with the determined device type and --nocheck=standby
						retryArgs := []string{"-d", deviceType, "--nocheck=standby", "-a", "-j", devicePath}
						retryCmd := c.commander.Command(c.smartctlPath, retryArgs...)
						retryOutput, retryErr := retryCmd.Output()

						// Check for standby mode on retry
						if retryErr != nil {
							if exitErr, ok := retryErr.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
								var retrySmartInfo SMARTInfo
								if len(retryOutput) > 0 && json.Unmarshal(retryOutput, &retrySmartInfo) == nil {
									c.setCachedDeviceType(devicePath, deviceType)
									retrySmartInfo.InStandby = true
									retrySmartInfo.DiskType = determineDiskType(&retrySmartInfo)
									return &retrySmartInfo, nil
								}
								return &SMARTInfo{
									Device:    Device{Name: devicePath, Type: deviceType},
									InStandby: true,
								}, nil
							}
						}

						if retryErr == nil || len(retryOutput) > 0 {
							var retrySmartInfo SMARTInfo
							if json.Unmarshal(retryOutput, &retrySmartInfo) == nil {
								// Check if SMART is supported with the device type
								if retrySmartInfo.Device.Name != "" {
									// Success! Cache the device type for this device path
									c.setCachedDeviceType(devicePath, deviceType)
									slog.Info("Successfully accessed device", "devicePath", devicePath, "deviceType", deviceType)
									retrySmartInfo.DiskType = determineDiskType(&retrySmartInfo)
									return &retrySmartInfo, nil
								}
							}
						}
						// If retry didn't work, log the failure
						slog.Debug("Retry with device type failed", "devicePath", devicePath, "deviceType", deviceType, "error", retryErr)
					}
				}

				smartInfo.DiskType = determineDiskType(&smartInfo)
				// If we have valid device information, return it without error
				// If device name is empty, SMART is likely not supported
				if smartInfo.Device.Name != "" {
					return &smartInfo, nil
				}
				return &smartInfo, fmt.Errorf("SMART Not Supported")
			}
		}
		return nil, fmt.Errorf("failed to get SMART info: %w", err)
	}

	var smartInfo SMARTInfo
	if err := json.Unmarshal(output, &smartInfo); err != nil {
		return nil, fmt.Errorf("failed to parse SMART info: %w", err)
	}

	// Check for messages in the output even when command succeeded
	if smartInfo.Smartctl != nil && len(smartInfo.Smartctl.Messages) > 0 {
		for _, msg := range smartInfo.Smartctl.Messages {
			slog.Warn("smartctl message", "severity", msg.Severity, "message", msg.String)
		}
	}

	// Determine disk type based on rotation rate and device type
	smartInfo.DiskType = determineDiskType(&smartInfo)

	return &smartInfo, nil
}

// isATADevice checks if a device type is ATA-based (ata, sat, sata, etc.)
func isATADevice(deviceType string) bool {
	if deviceType == "" {
		return false
	}
	dt := strings.ToLower(deviceType)
	return strings.Contains(dt, "ata") || strings.Contains(dt, "sat") || dt == "scsi"
}

// determineDiskType determines the type of disk based on available information
func determineDiskType(info *SMARTInfo) string {
	// Check for NVMe devices first
	if info.Device.Type == "nvme" || info.NvmeSmartHealth != nil || info.NvmeControllerCapabilities != nil {
		return "NVMe"
	}

	// Check rotation rate for ATA/SATA devices
	if info.RotationRate != nil {
		if *info.RotationRate == 0 {
			return "SSD"
		}
		return "HDD"
	}

	// Check device type from smartctl
	deviceType := strings.ToLower(info.Device.Type)
	if strings.Contains(deviceType, "nvme") {
		return "NVMe"
	}
	if strings.Contains(deviceType, "sata") || strings.Contains(deviceType, "ata") || strings.Contains(deviceType, "sat") {
		// If we have ATA SMART data but no rotation rate, try to infer
		if info.AtaSmartData != nil {
			// Look for SSD-specific attributes
			if info.AtaSmartData.Table != nil {
				for _, attr := range info.AtaSmartData.Table {
					if attr.ID == SmartAttrSSDLifeLeft || attr.ID == SmartAttrSandForceInternal || attr.ID == SmartAttrTotalLBAsWritten {
						return "SSD"
					}
				}
			}
		}
	}

	// If we can't determine, return Unknown
	return "Unknown"
}

// CheckHealth checks if a device is healthy according to SMART
func (c *Client) CheckHealth(devicePath string) (bool, error) {
	// Check if we have a cached device type and add --nocheck=standby for ATA devices
	var args []string
	if cachedType, ok := c.getCachedDeviceType(devicePath); ok {
		if isATADevice(cachedType) {
			args = []string{"-d", cachedType, "--nocheck=standby", "-H", devicePath}
		} else {
			args = []string{"-d", cachedType, "-H", devicePath}
		}
	} else {
		// Assume ATA by default for --nocheck=standby
		args = []string{"--nocheck=standby", "-H", devicePath}
	}

	cmd := c.commander.Command(c.smartctlPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// Exit code 2: device in standby
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 2 {
				// Device is in standby mode, cannot check health without waking it
				return false, fmt.Errorf("device is in standby mode")
			}
			// Parse output to determine health
			outputStr := string(exitErr.Stderr)
			if len(outputStr) == 0 {
				outputStr = string(output)
			}
			return strings.Contains(outputStr, "PASSED"), nil
		}
		return false, fmt.Errorf("failed to check health: %w", err)
	}

	outputStr := string(output)
	return strings.Contains(outputStr, "PASSED"), nil
}

// GetDeviceInfo retrieves basic device information
func (c *Client) GetDeviceInfo(devicePath string) (map[string]interface{}, error) {
	// Check if we have a cached device type and add --nocheck=standby for ATA devices
	var args []string
	if cachedType, ok := c.getCachedDeviceType(devicePath); ok {
		if isATADevice(cachedType) {
			args = []string{"-d", cachedType, "--nocheck=standby", "-i", "-j", devicePath}
		} else {
			args = []string{"-d", cachedType, "-i", "-j", devicePath}
		}
	} else {
		// Assume ATA by default for --nocheck=standby
		args = []string{"--nocheck=standby", "-i", "-j", devicePath}
	}

	cmd := c.commander.Command(c.smartctlPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// Exit code 2: device in standby
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			return nil, fmt.Errorf("device is in standby mode")
		}
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	var info map[string]interface{}
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse device info: %w", err)
	}

	return info, nil
}

// RunSelfTest initiates a SMART self-test
func (c *Client) RunSelfTest(devicePath string, testType string) error {
	// Valid test types: short, long, conveyance, offline
	validTypes := map[string]bool{
		"short":      true,
		"long":       true,
		"conveyance": true,
		"offline":    true,
	}

	if !validTypes[testType] {
		return fmt.Errorf("invalid test type: %s (must be one of: short, long, conveyance, offline)", testType)
	}

	cmd := c.commander.Command(c.smartctlPath, "-t", testType, devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run self-test: %w", err)
	}

	return nil
}

// ProgressCallback is a function type for reporting progress
type ProgressCallback func(progress int, status string)

// RunSelfTestWithProgress starts a SMART self-test and reports progress
func (c *Client) RunSelfTestWithProgress(ctx context.Context, devicePath string, testType string, callback ProgressCallback) error {
	// Valid test types: short, long, conveyance, offline
	validTypes := map[string]bool{
		"short":      true,
		"long":       true,
		"conveyance": true,
		"offline":    true,
	}

	if !validTypes[testType] {
		return fmt.Errorf("invalid test type: %s (must be one of: short, long, conveyance, offline)", testType)
	}

	// First check if self-tests are supported and get durations
	selfTestInfo, err := c.GetAvailableSelfTests(devicePath)
	if err != nil {
		return fmt.Errorf("failed to get self-test info: %w", err)
	}

	if len(selfTestInfo.Available) == 0 {
		return fmt.Errorf("self-tests are not supported by this device")
	}

	// Check if the requested test is available
	available := false
	for _, t := range selfTestInfo.Available {
		if t == testType {
			available = true
			break
		}
	}
	if !available {
		return fmt.Errorf("test type %s is not available for this device", testType)
	}

	// Start the self-test
	if err := c.RunSelfTest(devicePath, testType); err != nil {
		return fmt.Errorf("failed to start %s self-test: %w", testType, err)
	}

	if callback != nil {
		callback(0, fmt.Sprintf("%s self-test started", strings.ToUpper(string(testType[0]))+testType[1:]))
	}

	// Get expected duration based on test type
	expectedMinutes := map[string]int{
		"short":      2,
		"long":       120,
		"conveyance": 5,
		"offline":    10,
	}[testType]

	// Use duration from capabilities if available
	if duration, ok := selfTestInfo.Durations[testType]; ok && duration > 0 {
		expectedMinutes = duration
	}

	// Poll for completion
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	elapsed := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			elapsed += 5

			// Check current status
			currentInfo, err := c.GetSMARTInfo(devicePath)
			if err != nil {
				slog.Warn("Failed to get SMART info during polling", "error", err)
				continue
			}

			if currentInfo.AtaSmartData != nil && currentInfo.AtaSmartData.SelfTest != nil {
				status := currentInfo.AtaSmartData.SelfTest.Status
				if status != nil {
					ls := strings.ToLower(status.String)
					if strings.Contains(ls, "completed") || strings.Contains(ls, "aborted") || strings.Contains(ls, "interrupted") {
						if callback != nil {
							// Normalize message to expected phrasing
							msg := "Self-test "
							switch {
							case strings.Contains(ls, "completed"):
								msg += "completed"
							case strings.Contains(ls, "aborted"):
								msg += "aborted"
							case strings.Contains(ls, "interrupted"):
								msg += "interrupted"
							default:
								msg += status.String
							}
							callback(100, msg)
						}
						return nil
					}
				}

				// Try to get progress from Self-test execution status attribute (ID 231)
				progress := -1
				if currentInfo.AtaSmartData.Table != nil {
					for _, attr := range currentInfo.AtaSmartData.Table {
						if attr.ID == 231 {
							progress = attr.Value
							if progress > 100 {
								progress = 100
							}
							break
						}
					}
				}
				if progress == -1 {
					// Calculate progress based on elapsed time vs expected duration
					progress = (elapsed * 100) / (expectedMinutes * 60)
					if progress > 95 {
						progress = 95 // Don't show 100% until actually completed
					}
				}

				if callback != nil {
					msg := "Self-test in progress"
					if status != nil {
						msg = fmt.Sprintf("Self-test in progress (%s)", status.String)
					}
					callback(progress, msg)
				}
			} else {
				// Fallback progress calculation
				progress := (elapsed * 100) / (expectedMinutes * 60)
				if progress > 95 {
					progress = 95
				}
				if callback != nil {
					callback(progress, "Self-test in progress")
				}
			}

			// Timeout after 2x expected duration
			if elapsed > expectedMinutes*120 {
				return fmt.Errorf("self-test timed out after %d seconds", elapsed)
			}
		}
	}
}

// GetAvailableSelfTests returns the list of available self-test types and their durations for a device
func (c *Client) GetAvailableSelfTests(devicePath string) (*SelfTestInfo, error) {
	// Check if we have a cached device type and add --nocheck=standby for ATA devices
	var args []string
	if cachedType, ok := c.getCachedDeviceType(devicePath); ok {
		if isATADevice(cachedType) {
			args = []string{"-d", cachedType, "--nocheck=standby", "-c", "-j", devicePath}
		} else {
			args = []string{"-d", cachedType, "-c", "-j", devicePath}
		}
	} else {
		// Assume ATA by default for --nocheck=standby
		args = []string{"--nocheck=standby", "-c", "-j", devicePath}
	}

	cmd := c.commander.Command(c.smartctlPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// Exit code 2: device in standby
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			return nil, fmt.Errorf("device is in standby mode")
		}
		return nil, fmt.Errorf("failed to get capabilities: %w", err)
	}

	var caps CapabilitiesOutput
	if err := json.Unmarshal(output, &caps); err != nil {
		return nil, fmt.Errorf("failed to parse capabilities: %w", err)
	}

	info := &SelfTestInfo{
		Available: []string{},
		Durations: make(map[string]int),
	}

	// ATA
	if caps.AtaSmartData != nil {
		if caps.AtaSmartData.Capabilities != nil {
			capabilities := caps.AtaSmartData.Capabilities
			if capabilities.SelfTestsSupported {
				info.Available = append(info.Available, "short", "long")
			}
			if capabilities.ConveyanceSelfTestSupported {
				info.Available = append(info.Available, "conveyance")
			}
			if capabilities.ExecOfflineImmediate {
				info.Available = append(info.Available, "offline")
			}
		}
		if caps.AtaSmartData.SelfTest != nil && caps.AtaSmartData.SelfTest.PollingMinutes != nil {
			pm := caps.AtaSmartData.SelfTest.PollingMinutes
			if pm.Short > 0 {
				info.Durations["short"] = pm.Short
			}
			if pm.Extended > 0 {
				info.Durations["long"] = pm.Extended
			}
			if pm.Conveyance > 0 {
				info.Durations["conveyance"] = pm.Conveyance
			}
		}
	}

	// NVMe
	if caps.NvmeControllerCapabilities != nil && caps.NvmeControllerCapabilities.SelfTest {
		info.Available = append(info.Available, "short")
		// Durations not specified for NVMe in -c output
	}
	if caps.NvmeOptionalAdminCommands != nil && caps.NvmeOptionalAdminCommands.SelfTest {
		info.Available = append(info.Available, "short")
		// Durations not specified for NVMe in -c output
	}

	return info, nil
}

// IsSMARTSupported checks if SMART is supported on a device and if it's enabled
func (c *Client) IsSMARTSupported(devicePath string) (*SMARTSupportInfo, error) {
	smartInfo, err := c.GetSMARTInfo(devicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get SMART info: %w", err)
	}

	supportInfo := &SMARTSupportInfo{}

	// Check NVMe SMART support first
	if smartInfo.SmartSupport != nil {
		supportInfo.Supported = smartInfo.SmartSupport.Available
		supportInfo.Enabled = smartInfo.SmartSupport.Enabled
		return supportInfo, nil
	}

	// Check ATA SMART data presence for support
	if smartInfo.AtaSmartData != nil {
		supportInfo.Supported = true
		// For ATA devices, if SMART data is present, assume it's enabled
		// (ATA devices typically don't have a separate enabled/disabled status in JSON)
		supportInfo.Enabled = true
		return supportInfo, nil
	}

	// Check NVMe SMART health information as fallback
	if smartInfo.NvmeSmartHealth != nil {
		supportInfo.Supported = true
		supportInfo.Enabled = true
		return supportInfo, nil
	}

	// Not supported
	supportInfo.Supported = false
	supportInfo.Enabled = false
	return supportInfo, nil
}

// EnableSMART enables SMART monitoring on a device
func (c *Client) EnableSMART(devicePath string) error {
	cmd := c.commander.Command(c.smartctlPath, "-s", "on", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable SMART: %w", err)
	}
	return nil
}

// DisableSMART disables SMART monitoring on a device
func (c *Client) DisableSMART(devicePath string) error {
	cmd := c.commander.Command(c.smartctlPath, "-s", "off", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable SMART: %w", err)
	}
	return nil
}

// AbortSelfTest aborts a running self-test on a device
func (c *Client) AbortSelfTest(devicePath string) error {
	cmd := c.commander.Command(c.smartctlPath, "-X", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to abort self-test: %w", err)
	}
	return nil
}

// ensureCompatibleSmartctl runs "smartctl -V" and checks the version is supported.
// The library depends on JSON output (-j), which requires smartctl >= 7.0.
func ensureCompatibleSmartctl(smartctlPath string) error {
	out, err := exec.Command(smartctlPath, "-V").Output()
	if err != nil {
		return fmt.Errorf("failed to check smartctl version: %w", err)
	}
	major, minor, err := parseSmartctlVersion(string(out))
	if err != nil {
		return fmt.Errorf("unable to parse smartctl version: %w", err)
	}
	const minMajor, minMinor = 7, 0
	if major < minMajor || (major == minMajor && minor < minMinor) {
		return fmt.Errorf("unsupported smartctl version %d.%d; require >= %d.%d", major, minor, minMajor, minMinor)
	}
	return nil
}

// parseSmartctlVersion extracts the major and minor version numbers from
// the output of "smartctl -V". Expected forms include lines like:
//
//	"smartctl 7.3 2022-02-28 r5338 ..." or "smartctl 7.5 ...".
func parseSmartctlVersion(output string) (int, int, error) {
	// Find first occurrence of "smartctl X.Y"
	re := regexp.MustCompile(`(?m)\bsmartctl\s+(\d+)\.(\d+)\b`)
	m := re.FindStringSubmatch(output)
	if len(m) != 3 {
		return 0, 0, fmt.Errorf("version pattern not found in output")
	}
	// Convert captures to ints
	var (
		major int
		minor int
	)
	// Atoi without extra import by using fmt.Sscanf
	if _, err := fmt.Sscanf(m[1], "%d", &major); err != nil {
		return 0, 0, fmt.Errorf("invalid major version: %w", err)
	}
	if _, err := fmt.Sscanf(m[2], "%d", &minor); err != nil {
		return 0, 0, fmt.Errorf("invalid minor version: %w", err)
	}
	return major, minor, nil
}
