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
	"strings"
	"time"
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
type SMARTInfo struct {
	Device                     Device                      `json:"device"`
	ModelFamily                string                      `json:"model_family,omitempty"`
	ModelName                  string                      `json:"model_name,omitempty"`
	SerialNumber               string                      `json:"serial_number,omitempty"`
	Firmware                   string                      `json:"firmware_version,omitempty"`
	UserCapacity               int64                       `json:"user_capacity,omitempty"`
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
type OfflineDataCollection struct {
	Status            string `json:"status,omitempty"`
	CompletionSeconds int    `json:"completion_seconds,omitempty"`
}

// PollingMinutes represents polling minutes for different test types
type PollingMinutes struct {
	Short      int `json:"short,omitempty"`
	Extended   int `json:"extended,omitempty"`
	Conveyance int `json:"conveyance,omitempty"`
}

// SelfTest represents self-test information
type SelfTest struct {
	Status         string          `json:"status,omitempty"`
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
	smartctlPath string
	commander    Commander
}

// NewClient creates a new smartmontools client
func NewClient() (SmartClient, error) {
	// Try to find smartctl in PATH
	path, err := exec.LookPath("smartctl")
	if err != nil {
		return nil, fmt.Errorf("smartctl not found in PATH: %w", err)
	}

	return &Client{
		smartctlPath: path,
		commander:    execCommander{},
	}, nil
}

// NewClientWithPath creates a new smartmontools client with a specific smartctl path
func NewClientWithPath(smartctlPath string) SmartClient {
	return &Client{
		smartctlPath: smartctlPath,
		commander:    execCommander{},
	}
}

// NewClientWithCommander creates a new client with a custom commander (for testing)
func NewClientWithCommander(smartctlPath string, commander Commander) SmartClient {
	return &Client{
		smartctlPath: smartctlPath,
		commander:    commander,
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

// GetSMARTInfo retrieves SMART information for a device
func (c *Client) GetSMARTInfo(devicePath string) (*SMARTInfo, error) {
	cmd := c.commander.Command(c.smartctlPath, "-a", "-j", devicePath)
	output, err := cmd.Output()
	if err != nil {
		// smartctl returns non-zero exit codes for various conditions
		// We still want to parse the output if available and it's valid JSON
		if len(output) > 0 {
			var smartInfo SMARTInfo
			if json.Unmarshal(output, &smartInfo) == nil {
				// Valid JSON, treat error as warning
				slog.Warn("smartctl returned error but provided valid JSON output", "error", err)
				// Check for error messages in the output
				if smartInfo.Smartctl != nil && len(smartInfo.Smartctl.Messages) > 0 {
					for _, msg := range smartInfo.Smartctl.Messages {
						slog.Warn("smartctl message", "severity", msg.Severity, "message", msg.String)
					}
				}
				return &smartInfo, nil
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

	return &smartInfo, nil
}

// CheckHealth checks if a device is healthy according to SMART
func (c *Client) CheckHealth(devicePath string) (bool, error) {
	cmd := c.commander.Command(c.smartctlPath, "-H", devicePath)
	output, err := cmd.Output()
	if err != nil {
		// Exit code 0: healthy, non-zero may indicate issues
		if exitErr, ok := err.(*exec.ExitError); ok {
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
	cmd := c.commander.Command(c.smartctlPath, "-i", "-j", devicePath)
	output, err := cmd.Output()
	if err != nil {
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
				if status == "completed" || status == "aborted" || status == "interrupted" {
					if callback != nil {
						callback(100, fmt.Sprintf("Self-test %s", status))
					}
					return nil
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
					callback(progress, fmt.Sprintf("Self-test in progress (%s)", status))
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
	cmd := c.commander.Command(c.smartctlPath, "-c", "-j", devicePath)
	output, err := cmd.Output()
	if err != nil {
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
