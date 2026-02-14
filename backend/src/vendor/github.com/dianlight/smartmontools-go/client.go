package smartmontools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/tlog"
)

// SMART attribute IDs for SSD detection
const (
	SmartAttrSSDLifeLeft       = 231 // SSD Life Left attribute
	SmartAttrSandForceInternal = 233 // SandForce Internal (SSD-specific)
	SmartAttrTotalLBAsWritten  = 234 // Total LBAs Written (SSD-specific)
)

// ClientOption is a function that configures a Client
type ClientOption func(*Client)

// WithSmartctlPath sets a custom path to the smartctl binary
func WithSmartctlPath(path string) ClientOption {
	return func(c *Client) {
		c.smartctlPath = path
	}
}

// WithLogHandler sets a custom slog.Logger for the client.
func WithLogHandler(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logHandler = logger
	}
}

// WithTLogHandler sets a custom tlog.Logger for the client.
func WithTLogHandler(logger *tlog.Logger) ClientOption {
	return func(c *Client) {
		c.logHandler = logger
	}
}

// WithCommander sets a custom commander for testing purposes
func WithCommander(commander Commander) ClientOption {
	return func(c *Client) {
		c.commander = commander
	}
}

// WithContext sets a default context to use when methods are called with nil context
func WithContext(ctx context.Context) ClientOption {
	return func(c *Client) {
		c.defaultCtx = ctx
	}
}

// logAdapter captures the logging methods used by this package.
type logAdapter interface {
	Debug(msg string, args ...any)
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

var (
	_ logAdapter = (*tlog.Logger)(nil)
	_ logAdapter = (*slog.Logger)(nil)
)

// SmartClient interface defines the methods for interacting with smartmontools
type SmartClient interface {
	ScanDevices(ctx context.Context) ([]Device, error)
	GetSMARTInfo(ctx context.Context, devicePath string) (*SMARTInfo, error)
	CheckHealth(ctx context.Context, devicePath string) (bool, error)
	GetDeviceInfo(ctx context.Context, devicePath string) (map[string]interface{}, error)
	RunSelfTest(ctx context.Context, devicePath string, testType string) error
	RunSelfTestWithProgress(ctx context.Context, devicePath string, testType string, callback ProgressCallback) error
	GetAvailableSelfTests(ctx context.Context, devicePath string) (*SelfTestInfo, error)
	IsSMARTSupported(ctx context.Context, devicePath string) (*SmartSupport, error)
	GetSMARTSupportFromInfo(smartInfo *SMARTInfo) *SmartSupport
	EnableSMART(ctx context.Context, devicePath string) error
	DisableSMART(ctx context.Context, devicePath string) error
	AbortSelfTest(ctx context.Context, devicePath string) error
}

// Client represents a smartmontools client
type Client struct {
	smartctlPath       string
	commander          Commander
	deviceTypeCache    map[string]string // Maps device path to device type (e.g., "sat")
	deviceTypeCacheMux sync.RWMutex      // Protects deviceTypeCache
	logHandler         logAdapter        // Logger for the client
	defaultCtx         context.Context   // Default context to use when nil is passed
}

// NewClient creates a new smartmontools client with optional configuration.
// If no smartctl path is provided, it will search for smartctl in PATH.
// If no log handler is provided, it will use a tlog debug-level logger for diagnostic output.
// If no context is provided, context.Background() will be used as the default.
func NewClient(opts ...ClientOption) (SmartClient, error) {
	// Create client with defaults
	client := &Client{
		commander:       execCommander{},
		deviceTypeCache: loadDrivedbAddendum(),
		// Use a debug-level logger by default so library emits diagnostic output.
		// Use NewLoggerWithLevel to obtain a *tlog.Logger (tlog.WithLevel returns *slog.Logger).
		logHandler: tlog.NewLoggerWithLevel(tlog.LevelDebug),
		defaultCtx: context.Background(),
	}

	// Track if commander was set via options (for testing)
	defaultCommander := true

	// Apply options
	for _, opt := range opts {
		// Check if commander is being set
		beforeCommander := client.commander
		opt(client)
		if client.commander != beforeCommander {
			defaultCommander = false
		}
	}

	// If no smartctl path was provided, try to find it in PATH
	if client.smartctlPath == "" {
		path, err := exec.LookPath("smartctl")
		if err != nil {
			return nil, fmt.Errorf("smartctl not found in PATH: %w", err)
		}
		client.smartctlPath = path
	}

	// Only ensure smartctl is compatible if using the default commander
	// (skip validation for mock/test commanders)
	if defaultCommander {
		if err := ensureCompatibleSmartctl(client.smartctlPath); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// ScanDevices scans for available storage devices
func (c *Client) ScanDevices(ctx context.Context) ([]Device, error) {
	if ctx == nil {
		ctx = c.defaultCtx
	}
	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, "--scan-open", "--json")
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
	c.logHandler.Debug("Cached device type", "devicePath", devicePath, "deviceType", deviceType)
}

// GetSMARTInfo retrieves SMART information for a device
func (c *Client) GetSMARTInfo(ctx context.Context, devicePath string) (*SMARTInfo, error) {
	if ctx == nil {
		ctx = c.defaultCtx
	}
	// Check if we have a cached device type for this device
	var args []string
	var isATA bool
	if cachedType, ok := c.getCachedDeviceType(devicePath); ok {
		isATA = isATADevice(cachedType)
		args = []string{"-a", "-j"}
		if isATA {
			args = append(args, "--nocheck=standby")
		}
		args = append(args, "-d", cachedType, devicePath)
	} else {
		// Assume ATA by default for --nocheck=standby
		args = []string{"-a", "-j", "--nocheck=standby", devicePath}
		isATA = true
	}

	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// smartctl returns non-zero exit codes for various conditions
		// Bit 1 (exit code 2) indicates device is in standby mode
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()

			// Bit 1 (value 2): Device is in standby/sleep mode
			if exitCode&2 != 0 {
				// Parse partial output if available
				if len(output) > 0 {
					var smartInfo SMARTInfo
					if jsonErr := json.Unmarshal(output, &smartInfo); jsonErr == nil {
						smartInfo.InStandby = true
						// Cache device type if not cached yet
						if smartInfo.Device.Type != "" && !isATA {
							c.setCachedDeviceType(devicePath, smartInfo.Device.Type)
						}
						smartInfo.DiskType = determineDiskType(&smartInfo)
						smartInfo.SmartStatus = checkSmartStatus(&smartInfo)
						return &smartInfo, nil
					}
				}
				// If parsing fails, return a minimal SMARTInfo indicating standby
				return &SMARTInfo{InStandby: true}, nil
			}
		}

		// We still want to parse the output if available and it's valid JSON
		if len(output) > 0 {
			var smartInfo SMARTInfo
			if jsonErr := json.Unmarshal(output, &smartInfo); jsonErr == nil {
				// Cache device type if not cached yet
				if smartInfo.Device.Type != "" {
					// Detect device type from output
					if _, cached := c.getCachedDeviceType(devicePath); !cached {
						c.setCachedDeviceType(devicePath, smartInfo.Device.Type)
					}
				}

				// Cache messages from the output to avoid duplicate logging
				// Messages are cached with TTL based on severity:
				// - information: 1h, warning: 30min, error: 5min, default: 2h
				if smartInfo.Smartctl != nil && len(smartInfo.Smartctl.Messages) > 0 {
					for _, msg := range smartInfo.Smartctl.Messages {
						severity := msg.Severity
						if severity == "" {
							severity = "default"
						}
						// Cache the message; skip if already cached and not expired
						if globalMessageCache.shouldLog(msg.String, severity) {
							switch severity {
							case "error":
								c.logHandler.ErrorContext(ctx, msg.String)
							case "warning":
								c.logHandler.WarnContext(ctx, msg.String)
							default:
								c.logHandler.InfoContext(ctx, msg.String)
							}
						}
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
								c.logHandler.InfoContext(ctx, "Found USB bridge in drivedb", "usbBridgeID", usbBridgeID, "deviceType", deviceType)
							}
						}
						// If not in drivedb, default to sat
						if deviceType == "" {
							deviceType = "sat"
							c.logHandler.InfoContext(ctx, "Unknown USB bridge detected, retrying with -d sat", "devicePath", devicePath)
						}

						// Retry with the determined device type and --nocheck=standby
						retryArgs := []string{"-a", "-j", "--nocheck=standby", "-d", deviceType, devicePath}
						retryCmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, retryArgs...)
						retryOutput, retryErr := retryCmd.Output()

						// Check for standby mode on retry
						if retryErr != nil {
							if exitErr, ok := retryErr.(*exec.ExitError); ok && exitErr.ExitCode()&2 != 0 {
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
									SmartSupport: &SmartSupport{
										Available: true,
										Enabled:   true,
									},
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
									c.logHandler.InfoContext(ctx, "Successfully accessed device", "devicePath", devicePath, "deviceType", deviceType)
									retrySmartInfo.DiskType = determineDiskType(&retrySmartInfo)
									return &retrySmartInfo, nil
								}
							}
						}
						// If retry didn't work, log the failure
						c.logHandler.DebugContext(ctx, "Retry with device type failed", "devicePath", devicePath, "deviceType", deviceType, "error", retryErr)
					}
				}

				smartInfo.DiskType = determineDiskType(&smartInfo)
				smartInfo.SmartStatus = checkSmartStatus(&smartInfo)
				// If device name is empty after USB bridge fallback, SMART is likely not supported
				if smartInfo.Device.Name == "" {
					return &smartInfo, fmt.Errorf("SMART Not Supported")
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

	// Cache messages from the output to avoid duplicate logging
	// Messages are cached with TTL based on severity:
	// - information: 1h, warning: 30min, error: 5min, default: 2h
	if smartInfo.Smartctl != nil && len(smartInfo.Smartctl.Messages) > 0 {
		for _, msg := range smartInfo.Smartctl.Messages {
			severity := msg.Severity
			if severity == "" {
				severity = "default"
			}
			// Cache the message; skip if already cached and not expired
			if globalMessageCache.shouldLog(msg.String, severity) {
				switch severity {
				case "error":
					c.logHandler.ErrorContext(ctx, msg.String)
				case "warning":
					c.logHandler.WarnContext(ctx, msg.String)
				default:
					c.logHandler.InfoContext(ctx, msg.String)
				}
			}
		}
	}

	// Determine disk type based on rotation rate and device type
	smartInfo.DiskType = determineDiskType(&smartInfo)
	// Populate SmartStatus.Running field based on test status
	smartInfo.SmartStatus = checkSmartStatus(&smartInfo)

	return &smartInfo, nil
}

func checkSmartStatus(sMARTInfo *SMARTInfo) SmartStatus {
	damaged := false
	critical := false
	// Populate SmartStatus Damaged and Critical
	if sMARTInfo.Smartctl != nil {
		damaged = (sMARTInfo.Smartctl.ExitStatus & 0x00000100) != 0
		critical = (sMARTInfo.Smartctl.ExitStatus & 0x00001000) != 0
	}
	// Popolate SmartStatus Running
	if sMARTInfo.AtaSmartData != nil && sMARTInfo.AtaSmartData.SelfTest != nil {
		return SmartStatus{
			Running:  sMARTInfo.AtaSmartData.SelfTest.Status.Value >= 240 && sMARTInfo.AtaSmartData.SelfTest.Status.Value <= 253,
			Passed:   sMARTInfo.SmartStatus.Passed,
			Damaged:  damaged,
			Critical: critical,
		}
	} else if sMARTInfo.NvmeSmartTestLog != nil {
		return SmartStatus{
			Running:  sMARTInfo.NvmeSmartTestLog.CurrentOpeation != nil && *sMARTInfo.NvmeSmartTestLog.CurrentOpeation != 0,
			Passed:   sMARTInfo.SmartStatus.Passed,
			Damaged:  damaged,
			Critical: critical,
		}
	} else {
		return SmartStatus{
			Running:  false,
			Passed:   sMARTInfo.SmartStatus.Passed,
			Damaged:  damaged,
			Critical: critical,
		}
	}
}

// CheckHealth checks if a device is healthy according to SMART
func (c *Client) CheckHealth(ctx context.Context, devicePath string) (bool, error) {
	if ctx == nil {
		ctx = c.defaultCtx
	}
	// Check if we have a cached device type and add --nocheck=standby for ATA devices
	var args []string
	if cachedType, ok := c.getCachedDeviceType(devicePath); ok {
		args = []string{"-H"}
		if isATADevice(cachedType) {
			args = append(args, "--nocheck=standby")
		}
		args = append(args, "-d", cachedType, devicePath)
	} else {
		// Assume ATA by default for --nocheck=standby
		args = []string{"-H", "--nocheck=standby", devicePath}
	}

	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// Exit code 2: device in standby
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			// exitCode == -1 means ProcessState is not set (mock/testing scenario)
			// exitCode&2 != 0 means device is in standby mode
			if exitCode != -1 && exitCode&2 != 0 {
				// Device in standby - cannot determine health
				c.logHandler.DebugContext(ctx, "Device in standby mode, cannot check health", "devicePath", devicePath)
				return false, nil
			}
			// Check output even if command returned non-zero exit code
			// smartctl often returns non-zero even with valid output
			if len(output) > 0 {
				outputStr := string(output)
				return strings.Contains(outputStr, "PASSED"), nil
			}
		}
		return false, fmt.Errorf("failed to check health: %w", err)
	}

	outputStr := string(output)
	return strings.Contains(outputStr, "PASSED"), nil
}

// GetDeviceInfo retrieves basic device information
func (c *Client) GetDeviceInfo(ctx context.Context, devicePath string) (map[string]interface{}, error) {
	if ctx == nil {
		ctx = c.defaultCtx
	}
	// Check if we have a cached device type and add --nocheck=standby for ATA devices
	var args []string
	if cachedType, ok := c.getCachedDeviceType(devicePath); ok {
		args = []string{"-i", "-j"}
		if isATADevice(cachedType) {
			args = append(args, "--nocheck=standby")
		}
		args = append(args, "-d", cachedType, devicePath)
	} else {
		// Assume ATA by default for --nocheck=standby
		args = []string{"-i", "-j", "--nocheck=standby", devicePath}
	}

	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// Exit code 2: device in standby
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode()&2 != 0 {
			return nil, fmt.Errorf("device in standby mode")
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
func (c *Client) RunSelfTest(ctx context.Context, devicePath string, testType string) error {
	if ctx == nil {
		ctx = c.defaultCtx
	}
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

	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, "-t", testType, devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run self-test: %w", err)
	}

	return nil
}

// RunSelfTestWithProgress starts a SMART self-test and reports progress
func (c *Client) RunSelfTestWithProgress(ctx context.Context, devicePath string, testType string, callback ProgressCallback) error {
	if ctx == nil {
		ctx = c.defaultCtx
	}
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
	selfTestInfo, err := c.GetAvailableSelfTests(ctx, devicePath)
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
	if err := c.RunSelfTest(ctx, devicePath, testType); err != nil {
		return err
	}
	go func() {

		if callback != nil {
			callback(0, "Test started")
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
			case <-ticker.C:
				elapsed += 5
				progress := (elapsed * 100) / (expectedMinutes * 60)
				if progress > 100 {
					progress = 100
				}

				// Check if test is complete
				info, err := c.GetSMARTInfo(ctx, devicePath)
				if err != nil {
					if callback != nil {
						callback(progress, fmt.Sprintf("Error checking status: %v", err))
					}
					continue
				}

				// Using SMART infor remaining_percent if available
				if info.AtaSmartData.SelfTest != nil && info.AtaSmartData.SelfTest.Status.RemainingPercent != nil {
					remaining := *info.AtaSmartData.SelfTest.Status.RemainingPercent
					calculatedProgress := 100 - remaining
					if calculatedProgress > progress {
						progress = calculatedProgress
					}
				}

				// Check ATA self-test status
				if info.AtaSmartData != nil && info.AtaSmartData.SelfTest != nil {
					if info.AtaSmartData.SelfTest.Status != nil {

						if callback != nil {
							callback(progress, info.AtaSmartData.SelfTest.Status.String)
						}

						if info.AtaSmartData.SelfTest.Status.Value <= 240 || progress >= 100 {
							// Test complete
							if callback != nil {
								callback(100, info.AtaSmartData.SelfTest.Status.String)
							}
							return
						}

					}
				}

				// For NVMe, check self-test log
				if info.NvmeSmartTestLog != nil {
					if info.NvmeSmartTestLog.CurrentOpeation != nil && *info.NvmeSmartTestLog.CurrentOpeation == 0 {
						// No current operation means test is complete
						if callback != nil {
							callback(100, "Test completed")
						}
						return
					} else if info.NvmeSmartTestLog.CurrentCompletion != nil {
						progress = *info.NvmeSmartTestLog.CurrentCompletion
						if callback != nil {
							callback(progress, "Test in progress")
						}
					}
				}

				// For NVMe devices, check completion differently
				// NVMe devices may not report progress the same way
				if info.NvmeSmartHealth != nil {
					// If we've reached expected duration, assume complete
					if elapsed >= expectedMinutes*60 {
						if callback != nil {
							callback(100, "Test completed")
						}
						return
					}
					if callback != nil {
						callback(progress, "Test in progress")
					}
				}

			case <-ctx.Done():
				if callback != nil {
					callback(0, "Test cancelled")
				}
				return
			}
		}
	}()
	return nil
}

// GetAvailableSelfTests returns the list of available self-test types and their durations for a device
func (c *Client) GetAvailableSelfTests(ctx context.Context, devicePath string) (*SelfTestInfo, error) {
	if ctx == nil {
		ctx = c.defaultCtx
	}
	// Check if we have a cached device type and add --nocheck=standby for ATA devices
	var args []string
	if cachedType, ok := c.getCachedDeviceType(devicePath); ok {
		args = []string{"-c", "-j"}
		if isATADevice(cachedType) {
			args = append(args, "--nocheck=standby")
		}
		args = append(args, "-d", cachedType, devicePath)
	} else {
		// Assume ATA by default for --nocheck=standby
		args = []string{"-c", "-j", "--nocheck=standby", devicePath}
	}

	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// Exit code 2: device in standby
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode()&2 != 0 {
			return nil, fmt.Errorf("device in standby mode")
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
			if caps.AtaSmartData.Capabilities.SelfTestsSupported {
				info.Available = append(info.Available, "short", "long")
			}
			if caps.AtaSmartData.Capabilities.ConveyanceSelfTestSupported {
				info.Available = append(info.Available, "conveyance")
			}
			if caps.AtaSmartData.Capabilities.ExecOfflineImmediate {
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

// GetSMARTSupportFromInfo extracts SMART support status from a SMARTInfo struct.
// This is a helper method that allows checking SMART availability and enablement status
// without performing additional disk I/O. Applications should call GetSMARTInfo once,
// cache the result, and use this method to check the SMART status from the cache.
//
// This pattern avoids periodic disk access and prevents waking disks from standby mode.
//
// Example usage:
//
//	// Get and cache SMART info once
//	info, err := client.GetSMARTInfo(ctx, devicePath)
//	if err != nil {
//	    return err
//	}
//
//	// Check SMART status from cached info (no disk access)
//	support := client.GetSMARTSupportFromInfo(info)
//	if support.Available && support.Enabled {
//	    // SMART is available and enabled
//	}
//
//	// After EnableSMART/DisableSMART, refresh the cache:
//	if err := client.EnableSMART(ctx, devicePath); err != nil {
//	    return err
//	}
//	// Refresh cache after state change
//	info, err = client.GetSMARTInfo(ctx, devicePath)
//	if err != nil {
//	    return err
//	}
func (c *Client) GetSMARTSupportFromInfo(smartInfo *SMARTInfo) *SmartSupport {
	return c.isSMARTSupported(smartInfo)
}

// isSMARTSupported checks if SMART is supported on a device and if it's enabled.
// This is an internal helper that extracts SMART support status from SMARTInfo.
func (c *Client) isSMARTSupported(smartInfo *SMARTInfo) *SmartSupport {

	supportInfo := &SmartSupport{}

	// Check if smartctl provided smart_support field (both ATA and NVMe devices)
	if smartInfo.SmartSupport != nil {
		supportInfo.Available = smartInfo.SmartSupport.Available
		supportInfo.Enabled = smartInfo.SmartSupport.Enabled
		return supportInfo
	}

	// Fallback: Check ATA SMART data presence for support
	// This handles older smartctl versions that don't provide smart_support field
	if smartInfo.AtaSmartData != nil {
		supportInfo.Available = true
		// If ATA SMART data is present, SMART is likely enabled
		// (older smartctl versions don't report disabled state in JSON)
		supportInfo.Enabled = true
		return supportInfo
	}

	// Fallback: Check NVMe SMART health information
	if smartInfo.NvmeSmartHealth != nil {
		supportInfo.Available = true
		supportInfo.Enabled = true
		return supportInfo
	}

	// Not supported
	supportInfo.Available = false
	supportInfo.Enabled = false
	return supportInfo
}

// IsSMARTSupported checks if SMART is supported on a device and if it's enabled.
//
// WARNING: This method performs disk I/O by calling GetSMARTInfo internally.
// For applications that need to check SMART status frequently (e.g., monitoring daemons),
// it's recommended to call GetSMARTInfo once, cache the result, and use
// GetSMARTSupportFromInfo to extract SMART support status from the cached data.
// This avoids periodic disk access and prevents waking disks from standby mode.
//
// Preferred usage pattern for periodic monitoring:
//
//	// Initial query (performed once or when SMART status changes)
//	info, err := client.GetSMARTInfo(ctx, devicePath)
//	if err != nil {
//	    return err
//	}
//
//	// Cache the info and check SMART status without disk I/O
//	support := client.GetSMARTSupportFromInfo(info)
//	if !support.Enabled {
//	    // Skip SMART monitoring when disabled
//	    return
//	}
//
// Only use IsSMARTSupported for one-off checks where disk access is acceptable.
func (c *Client) IsSMARTSupported(ctx context.Context, devicePath string) (*SmartSupport, error) {
	if ctx == nil {
		ctx = c.defaultCtx
	}
	smartInfo, err := c.GetSMARTInfo(ctx, devicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get SMART info: %w", err)
	}

	return c.isSMARTSupported(smartInfo), nil
}

// EnableSMART enables SMART monitoring on a device
func (c *Client) EnableSMART(ctx context.Context, devicePath string) error {
	if ctx == nil {
		ctx = c.defaultCtx
	}
	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, "-s", "on", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable SMART: %w", err)
	}
	return nil
}

// DisableSMART disables SMART monitoring on a device
func (c *Client) DisableSMART(ctx context.Context, devicePath string) error {
	if ctx == nil {
		ctx = c.defaultCtx
	}
	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, "-s", "off", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable SMART: %w", err)
	}
	return nil
}

// AbortSelfTest aborts a running self-test on a device
func (c *Client) AbortSelfTest(ctx context.Context, devicePath string) error {
	if ctx == nil {
		ctx = c.defaultCtx
	}
	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, "-X", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to abort self-test: %w", err)
	}
	return nil
}
