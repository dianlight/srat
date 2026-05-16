package smartmontools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/tlog"
)

// SMART attribute IDs for SSD detection and wear-level computation
const (
	SmartAttrSSDLifeUsed       = 173 // SSD Life Used — raw value is percent used (0 = new)
	SmartAttrWearLevelingCount = 177 // Wear Leveling Count — normalized value is remaining life (100 = new)
	SmartAttrSSDLifeLeft       = 231 // SSD Life Left — normalized value is remaining life (100 = new)
	SmartAttrSandForceInternal = 233 // SandForce Internal (SSD-specific, used for drive-type detection)
	SmartAttrTotalLBAsWritten  = 234 // Total LBAs Written (SSD-specific, used for drive-type detection)
)

// Valid self-test types for SMART testing
var validSelfTestTypes = []string{"short", "long", "conveyance", "offline"}

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
		c.defaultCommander = false
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
	GetAvailableSelfTestsFromInfo(smartInfo *SMARTInfo) *SelfTestInfo
	IsSMARTSupported(ctx context.Context, devicePath string) (*SmartSupport, error)
	GetSMARTSupportFromInfo(smartInfo *SMARTInfo) *SmartSupport
	EnableSMART(ctx context.Context, devicePath string) error
	DisableSMART(ctx context.Context, devicePath string) error
	AbortSelfTest(ctx context.Context, devicePath string) error
	DiscoverDevices(ctx context.Context) ([]DiscoveryResult, error)
}

// Client represents a smartmontools client
type Client struct {
	smartctlPath       string
	commander          Commander
	defaultCommander   bool              // true when using the built-in exec commander
	deviceTypeCache    map[string]string // Maps device path to device type (e.g., "sat")
	deviceTypeCacheMux sync.RWMutex      // Protects deviceTypeCache
	healthBitsCache    map[string]int    // Maps device path to last-seen health bits (bits 3–7)
	healthBitsCacheMux sync.RWMutex      // Protects healthBitsCache
	logHandler         logAdapter        // Logger for the client
	defaultCtx         context.Context   // Default context to use when nil is passed
}

// NewClient creates a new smartmontools client with optional configuration.
// If no smartctl path is provided, it will search for smartctl in PATH.
// If no log handler is provided, it will use a tlog debug-level logger for diagnostic output.
// If no context is provided, context.Background() will be used as the default.
// The device type cache is pre-populated with drivedb entries on initialization.
func NewClient(opts ...ClientOption) (SmartClient, error) {
	// Create client with defaults
	// Pre-loaded drivedb cache is populated at package init time
	client := &Client{
		commander:        execCommander{},
		defaultCommander: true,
		deviceTypeCache:  cloneDeviceTypeCache(),
		healthBitsCache:  make(map[string]int),
		// Use a debug-level logger by default so library emits diagnostic output.
		// Use NewLoggerWithLevel to obtain a *tlog.Logger (tlog.WithLevel returns *slog.Logger).
		logHandler: tlog.NewLoggerWithLevel(tlog.LevelDebug),
		defaultCtx: context.Background(),
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	// If no smartctl path was provided, search PATH then platform-specific locations.
	if client.smartctlPath == "" {
		path, err := resolveSmartctlPath()
		if err != nil {
			return nil, err
		}
		client.smartctlPath = path
	}

	// Only ensure smartctl is compatible if using the built-in exec commander
	// (skip validation for mock/test commanders)
	if client.defaultCommander {
		if err := ensureCompatibleSmartctl(client.smartctlPath); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// ScanDevices scans for available storage devices.
// It first attempts --scan-open (which performs an open on each drive to verify
// accessibility) and falls back to --scan on failure. --scan-open may fail in
// container sandboxes, on older kernels, or when the caller lacks the required
// permissions; --scan still returns the device list without the open step.
func (c *Client) ScanDevices(ctx context.Context) ([]Device, error) {
	ctx = c.resolveCtx(ctx)
	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, "--scan-open", "--json")
	output, err := cmd.Output()
	if err != nil {
		// Fall back to --scan when --scan-open is unsupported or fails.
		c.logHandler.WarnContext(ctx, "--scan-open failed, retrying with --scan", "err", err)
		fallbackCmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, "--scan", "--json")
		output, err = fallbackCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to scan devices: %w", err)
		}
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

	// Pre-allocate slice with exact capacity needed and fill using index loop
	devices := make([]Device, len(result.Devices))
	for i, d := range result.Devices {
		devices[i] = Device{
			Name: d.Name,
			Type: d.Type,
		}
		// Cache device type discovered by --scan-open so all subsequent methods
		// can use --nocheck=standby and the correct -d <type> argument without
		// needing an extra disk query.
		if d.Name != "" && d.Type != "" {
			if _, cached := c.getCachedDeviceType(d.Name); !cached {
				c.setCachedDeviceType(d.Name, d.Type)
			}
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

// resolveCtx returns ctx if non-nil, otherwise returns the client's default context.
func (c *Client) resolveCtx(ctx context.Context) context.Context {
	if ctx == nil {
		return c.defaultCtx
	}
	return ctx
}

// buildArgs assembles smartctl arguments for devicePath, prepending flags and
// inserting --nocheck=standby (ATA only) plus -d <type> when the device type
// is already known from the cache. Falls back to the ATA-safe default when the
// cache is cold.
func (c *Client) buildArgs(devicePath string, flags ...string) []string {
	if cachedType, ok := c.getCachedDeviceType(devicePath); ok {
		args := append([]string(nil), flags...)
		if isATADevice(cachedType) {
			args = append(args, "--nocheck=standby")
		}
		return append(args, "-d", cachedType, devicePath)
	}
	// Unknown device type — assume ATA and add --nocheck=standby.
	return append(append([]string(nil), flags...), "--nocheck=standby", devicePath)
}

// logSmartctlMessages logs messages from a smartctl response, deduplicating via
// the global TTL cache so the same message is not repeated on every poll cycle.
func (c *Client) logSmartctlMessages(ctx context.Context, info *SMARTInfo) {
	if info.Smartctl == nil {
		return
	}
	for _, msg := range info.Smartctl.Messages {
		severity := msg.Severity
		if severity == "" {
			severity = "default"
		}
		if !globalMessageCache.shouldLog(msg.String, severity) {
			continue
		}
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

// retryWithDeviceType retries the SMART query for devicePath using an explicit
// -d <deviceType> flag and --nocheck=standby. It is the common implementation
// behind both the execution-failure SAT-probe path and the USB bridge
// protocol-selection path.
//
// On success the device type is written to the device type cache so subsequent
// calls use buildArgs directly without re-probing.
//
// Returns (info, true) when the attempt produces a usable result (including
// standby). Returns (nil, false) when the device cannot be opened with this
// type, the output cannot be parsed, or the response has an empty device name
// indicating the protocol did not produce valid SMART data.
func (c *Client) retryWithDeviceType(ctx context.Context, devicePath, deviceType string) (*SMARTInfo, bool) {
	args := []string{"-a", "-j", "--nocheck=standby", "-d", deviceType, devicePath}
	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, args...)
	output, err := cmd.Output()

	if err != nil {
		exitErr, isExit := err.(*exec.ExitError)
		if !isExit {
			return nil, false
		}
		code := exitErr.ExitCode()
		// Execution failure bits 0 or 2: device still cannot be read with this type.
		if code&0x05 != 0 {
			return nil, false
		}
		// Bit 1: device is in standby but responds to this protocol — cache the
		// type so future buildArgs invocations use the correct -d flag.
		if code&0x02 != 0 {
			c.setCachedDeviceType(devicePath, deviceType)
			if len(output) > 0 {
				var info SMARTInfo
				if json.Unmarshal(output, &info) == nil {
					info.InStandby = true
					info.DiskType = determineDiskType(&info)
					info.SmartStatus = checkSmartStatus(&info)
					return &info, true
				}
			}
			return &SMARTInfo{
				Device:       Device{Name: devicePath, Type: deviceType},
				InStandby:    true,
				SmartSupport: &SmartSupport{Available: true, Enabled: true},
			}, true
		}
	}

	if len(output) == 0 {
		return nil, false
	}
	var info SMARTInfo
	if jsonErr := json.Unmarshal(output, &info); jsonErr != nil {
		return nil, false
	}
	// An empty device name indicates the protocol couldn't read SMART data.
	if info.Device.Name == "" {
		return nil, false
	}
	c.setCachedDeviceType(devicePath, deviceType)
	c.logHandler.InfoContext(ctx, "Device type retry succeeded", "devicePath", devicePath, "deviceType", deviceType)
	info.DiskType = determineDiskType(&info)
	info.SmartStatus = checkSmartStatus(&info)
	c.logHealthBits(ctx, devicePath, &info)
	c.logSmartctlMessages(ctx, &info)
	return &info, true
}

// retrySATFallback is called when the initial smartctl query failed with
// execution-failure bits (bits 0–2 of the smartctl exit code), indicating a
// protocol mismatch — common on Synology /dev/sata* paths, USB-to-SATA
// bridges, and RAID passthrough devices.
//
// On success the "sat" protocol is written to the device type cache so that
// all subsequent calls use it directly without re-probing.
//
// Returns (info, true) when the SAT attempt produces a usable result
// (including standby). Returns (nil, false) when the SAT attempt also fails
// with execution failure bits or produces unparseable output.
func (c *Client) retrySATFallback(ctx context.Context, devicePath string) (*SMARTInfo, bool) {
	c.logHandler.InfoContext(ctx, "execution failure with default protocol, retrying with -d sat", "devicePath", devicePath)
	return c.retryWithDeviceType(ctx, devicePath, "sat")
}

// getSMARTInfoInternal is the implementation behind GetSMARTInfo. The second
// return value is true when the internal SAT fallback (retrySATFallback) was
// invoked and succeeded, allowing DiscoverDevices to surface SATFallbackRequired
// without changing the public GetSMARTInfo signature.
func (c *Client) getSMARTInfoInternal(ctx context.Context, devicePath string) (*SMARTInfo, bool, error) {
	ctx = c.resolveCtx(ctx)
	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, c.buildArgs(devicePath, "-a", "-j")...)
	output, err := cmd.Output()
	if err != nil {
		// smartctl returns non-zero exit codes for various conditions
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()

			// Bits 0, 2 (mask 0x05): execution failures — retry with -d sat on
			// first contact. Handles Synology /dev/sata* paths, USB bridges, and
			// RAID passthrough devices that fail with the auto-detected protocol.
			// Bit 1 (standby) is excluded: --nocheck=standby is always passed, so
			// bit 1 means the device is in standby mode, not a protocol mismatch.
			// The standby check below handles it without triggering a SAT probe.
			if exitCode&0x05 != 0 {
				if _, hasCached := c.getCachedDeviceType(devicePath); !hasCached {
					if info, satOK := c.retrySATFallback(ctx, devicePath); satOK {
						return info, true, nil
					}
				}
			}

			// Bit 1 (value 2): Device is in standby/sleep mode
			if exitCode&2 != 0 {
				// Parse partial output if available
				if len(output) > 0 {
					var smartInfo SMARTInfo
					if jsonErr := json.Unmarshal(output, &smartInfo); jsonErr == nil {
						smartInfo.InStandby = true
						// Cache the device type returned by the standby response.
						// The previous !isATA guard was wrong: isATA defaults to true
						// when no type is cached yet (the common first-contact case),
						// which silently prevented ATA/SAT devices from being cached.
						if smartInfo.Device.Type != "" {
							if _, cached := c.getCachedDeviceType(devicePath); !cached {
								c.setCachedDeviceType(devicePath, smartInfo.Device.Type)
							}
						}
						smartInfo.DiskType = determineDiskType(&smartInfo)
						smartInfo.SmartStatus = checkSmartStatus(&smartInfo)
						return &smartInfo, false, nil
					}
				}
				// If parsing fails, return a minimal SMARTInfo indicating standby
				return &SMARTInfo{InStandby: true}, false, nil
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

				c.logSmartctlMessages(ctx, &smartInfo)

				// Check if this is an unknown USB bridge error and we haven't cached a type yet
				if isUnknownUSBBridge(&smartInfo) {
					if _, hasCached := c.getCachedDeviceType(devicePath); !hasCached {
						// Prefer a type from drivedb for known bridges; fall back to sat.
						deviceType := "sat"
						if usbBridgeID := extractUSBBridgeID(&smartInfo); usbBridgeID != "" {
							if knownType, ok := c.getCachedDeviceType(usbBridgeID); ok {
								deviceType = knownType
								c.logHandler.InfoContext(ctx, "Found USB bridge in drivedb", "usbBridgeID", usbBridgeID, "deviceType", deviceType)
							}
						}
						if deviceType == "sat" {
							c.logHandler.InfoContext(ctx, "Unknown USB bridge detected, retrying with -d sat", "devicePath", devicePath)
						}
						if info, ok := c.retryWithDeviceType(ctx, devicePath, deviceType); ok {
							return info, false, nil
						}
						c.logHandler.ErrorContext(ctx, "Retry with device type failed", "devicePath", devicePath, "deviceType", deviceType)
					}
				}

				smartInfo.DiskType = determineDiskType(&smartInfo)
				smartInfo.SmartStatus = checkSmartStatus(&smartInfo)
				// If device name is empty after USB bridge fallback, SMART is likely not supported
				if smartInfo.Device.Name == "" {
					return &smartInfo, false, fmt.Errorf("SMART Not Supported")
				}
				return &smartInfo, false, nil
			}
		}
		return nil, false, fmt.Errorf("failed to get SMART info: %w", err)
	}

	var smartInfo SMARTInfo
	if err := json.Unmarshal(output, &smartInfo); err != nil {
		return nil, false, fmt.Errorf("failed to parse SMART info: %w", err)
	}

	c.logSmartctlMessages(ctx, &smartInfo)

	// Determine disk type based on rotation rate and device type
	smartInfo.DiskType = determineDiskType(&smartInfo)
	// Populate SmartStatus.Running field based on test status
	smartInfo.SmartStatus = checkSmartStatus(&smartInfo)
	c.logHealthBits(ctx, devicePath, &smartInfo)

	// Cache the device type from the successful response so all subsequent
	// methods can use --nocheck=standby and the correct -d <type> argument
	// without issuing another disk query.
	if smartInfo.Device.Type != "" {
		if _, cached := c.getCachedDeviceType(devicePath); !cached {
			c.setCachedDeviceType(devicePath, smartInfo.Device.Type)
		}
	}

	return &smartInfo, false, nil
}

// GetSMARTInfo retrieves SMART information for a device
func (c *Client) GetSMARTInfo(ctx context.Context, devicePath string) (*SMARTInfo, error) {
	info, _, err := c.getSMARTInfoInternal(ctx, devicePath)
	return info, err
}

func checkSmartStatus(sMARTInfo *SMARTInfo) *SmartStatus {
	if sMARTInfo.SmartStatus == nil {
		sMARTInfo.SmartStatus = &SmartStatus{}
	}

	var damaged, critical bool
	if sMARTInfo.Smartctl != nil {
		exitStatus := sMARTInfo.Smartctl.ExitStatus
		damaged = exitStatus&0x08 != 0
		critical = exitStatus&0x10 != 0

		// Populate ExitCodeInfo so consumers can inspect exit status bits
		// programmatically without parsing error strings.
		if exitStatus != 0 {
			sMARTInfo.ExitCodeInfo = &ExitCodeInfo{
				ExecBits:   exitStatus & 0x07,
				HealthBits: exitStatus & 0xF8,
			}
		}
	}

	s := &SmartStatus{Passed: sMARTInfo.SmartStatus.Passed, Damaged: damaged, Critical: critical}
	switch {
	case sMARTInfo.AtaSmartData != nil && sMARTInfo.AtaSmartData.SelfTest != nil && sMARTInfo.AtaSmartData.SelfTest.Status != nil:
		v := sMARTInfo.AtaSmartData.SelfTest.Status.Value
		s.Running = v >= 240 && v <= 253
	case sMARTInfo.NvmeSmartTestLog != nil:
		s.Running = sMARTInfo.NvmeSmartTestLog.CurrentOpeation != nil && *sMARTInfo.NvmeSmartTestLog.CurrentOpeation != 0
	}
	return s
}

// logHealthBits emits a single WARNING per device per unique health-bit pattern.
// When a drive enters a stable-but-degraded state (e.g., pre-failure attributes
// below threshold), subsequent polls produce the same bits and are suppressed to
// avoid flooding the caller's log.
func (c *Client) logHealthBits(ctx context.Context, devicePath string, info *SMARTInfo) {
	if info == nil || info.ExitCodeInfo == nil || info.ExitCodeInfo.HealthBits == 0 {
		return
	}
	bits := info.ExitCodeInfo.HealthBits

	c.healthBitsCacheMux.Lock()
	prev, seen := c.healthBitsCache[devicePath]
	if seen && prev == bits {
		c.healthBitsCacheMux.Unlock()
		return
	}
	c.healthBitsCache[devicePath] = bits
	c.healthBitsCacheMux.Unlock()

	c.logHandler.WarnContext(ctx, "SMART health flags detected",
		"devicePath", devicePath,
		"healthBits", bits,
		"diskFailing", bits&0x08 != 0,
		"prefailAttr", bits&0x10 != 0,
		"pastPrefail", bits&0x20 != 0,
		"errorLog", bits&0x40 != 0,
		"selfTestLog", bits&0x80 != 0,
	)
}

// CheckHealth checks if a device is healthy according to SMART
func (c *Client) CheckHealth(ctx context.Context, devicePath string) (bool, error) {
	ctx = c.resolveCtx(ctx)
	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, c.buildArgs(devicePath, "-H")...)
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
	ctx = c.resolveCtx(ctx)
	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, c.buildArgs(devicePath, "-i", "-j")...)
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
	ctx = c.resolveCtx(ctx)
	// Valid test types: short, long, conveyance, offline
	if !slices.Contains(validSelfTestTypes, testType) {
		return fmt.Errorf("invalid test type: %s (must be one of: short, long, conveyance, offline)", testType)
	}

	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, "-t", testType, devicePath)
	if err := cmd.Run(); err != nil {
		output, _ := cmd.CombinedOutput()
		return fmt.Errorf("failed to run self-test: %w (devicePath: %s, testType: %s, output: %s)", err, devicePath, testType, string(output))
	}

	return nil
}

// RunSelfTestWithProgress starts a SMART self-test and reports progress
func (c *Client) RunSelfTestWithProgress(ctx context.Context, devicePath string, testType string, callback ProgressCallback) error {
	ctx = c.resolveCtx(ctx)
	// Valid test types: short, long, conveyance, offline
	if !slices.Contains(validSelfTestTypes, testType) {
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
	if !slices.Contains(selfTestInfo.Available, testType) {
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

		// Poll for completion using an adaptive interval: at most 24 samples over
		// the expected duration, clamped between 5 s and 60 s. This avoids waking
		// the disk 1 440 times for a 2-hour long test.
		pollIntervalSecs := max(5, min(60, expectedMinutes*60/24))
		ticker := time.NewTicker(time.Duration(pollIntervalSecs) * time.Second)
		defer ticker.Stop()

		elapsed := 0
		for {
			select {
			case <-ticker.C:
				elapsed += pollIntervalSecs
				progress := (elapsed * 100) / (expectedMinutes * 60)
				if progress > 100 {
					progress = 100
				}

				// Check if test is complete
				info, err := c.GetSMARTInfo(ctx, devicePath)
				if err != nil {
					if callback != nil {
						callback(progress, fmt.Sprintf("Error checking status: %v (devicePath: %s, testType: %s)", err, devicePath, testType))
					}
					continue
				}

				// Using SMART infor remaining_percent if available
				if info.AtaSmartData != nil && info.AtaSmartData.SelfTest != nil && info.AtaSmartData.SelfTest.Status != nil && info.AtaSmartData.SelfTest.Status.RemainingPercent != nil {
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
							callback(progress, fmt.Sprintf("%s (devicePath: %s, testType: %s)", info.AtaSmartData.SelfTest.Status.String, devicePath, testType))
						}

						if info.AtaSmartData.SelfTest.Status.Value <= 240 || progress >= 100 {
							// Test complete
							if callback != nil {
								callback(100, fmt.Sprintf("%s (devicePath: %s, testType: %s)", info.AtaSmartData.SelfTest.Status.String, devicePath, testType))
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
							callback(100, fmt.Sprintf("Test completed (devicePath: %s, testType: %s)", devicePath, testType))
						}
						return
					} else if info.NvmeSmartTestLog.CurrentCompletion != nil {
						progress = *info.NvmeSmartTestLog.CurrentCompletion
						if callback != nil {
							callback(progress, fmt.Sprintf("Test in progress (devicePath: %s, testType: %s)", devicePath, testType))
						}
					}
				}

				// For NVMe devices, check completion differently
				// NVMe devices may not report progress the same way
				if info.NvmeSmartHealth != nil {
					// If we've reached expected duration, assume complete
					if elapsed >= expectedMinutes*60 {
						if callback != nil {
							callback(100, fmt.Sprintf("Test completed (devicePath: %s, testType: %s)", devicePath, testType))
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

// populateSelfTestInfo fills dst with available test types and durations extracted
// from ATA and NVMe capability data. It is shared by GetAvailableSelfTests and
// GetAvailableSelfTestsFromInfo to avoid duplicating the extraction logic.
func populateSelfTestInfo(info *SelfTestInfo, ata *AtaSmartData, nvmeCaps *NvmeControllerCapabilities, nvmeOptional *NvmeOptionalAdminCommands) {
	if ata != nil && ata.Capabilities != nil {
		caps := ata.Capabilities
		if caps.SelfTestsSupported {
			info.Available = append(info.Available, "short", "long")
		}
		if caps.ConveyanceSelfTestSupported {
			info.Available = append(info.Available, "conveyance")
		}
		if caps.ExecOfflineImmediate {
			info.Available = append(info.Available, "offline")
		}
		if ata.SelfTest != nil && ata.SelfTest.PollingMinutes != nil {
			pm := ata.SelfTest.PollingMinutes
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
	// NVMe — combine both capability fields to avoid appending "short" twice.
	if (nvmeCaps != nil && nvmeCaps.SelfTest) || (nvmeOptional != nil && nvmeOptional.SelfTest) {
		info.Available = append(info.Available, "short")
	}
}

// GetAvailableSelfTests returns the list of available self-test types and their durations for a device
func (c *Client) GetAvailableSelfTests(ctx context.Context, devicePath string) (*SelfTestInfo, error) {
	ctx = c.resolveCtx(ctx)
	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, c.buildArgs(devicePath, "-c", "-j")...)
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
	populateSelfTestInfo(info, caps.AtaSmartData, caps.NvmeControllerCapabilities, caps.NvmeOptionalAdminCommands)
	return info, nil
}

// GetAvailableSelfTestsFromInfo extracts available self-test types and their durations
// from a SMARTInfo struct without performing additional disk I/O. Applications that
// already hold a cached SMARTInfo (from GetSMARTInfo) should use this method instead of
// GetAvailableSelfTests to avoid an extra smartctl -c disk query.
//
// Note: NVMe detection uses NvmeControllerCapabilities, which is present in -a output.
// The NvmeOptionalAdminCommands field (available only via -c) is not stored in SMARTInfo,
// so NVMe self-test capability detection may be incomplete for some controllers.
//
// Example usage:
//
//	// Get and cache SMART info once
//	info, err := client.GetSMARTInfo(ctx, devicePath)
//	if err != nil {
//	    return err
//	}
//
//	// Extract self-test types without another disk query
//	selfTests := client.GetAvailableSelfTestsFromInfo(info)
//	for _, testType := range selfTests.Available {
//	    fmt.Printf("Test: %s (%d min)\n", testType, selfTests.Durations[testType])
//	}
func (c *Client) GetAvailableSelfTestsFromInfo(smartInfo *SMARTInfo) *SelfTestInfo {
	info := &SelfTestInfo{
		Available: []string{},
		Durations: make(map[string]int),
	}
	if smartInfo == nil {
		return info
	}
	populateSelfTestInfo(info, smartInfo.AtaSmartData, smartInfo.NvmeControllerCapabilities, nil)
	return info
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
	supportInfo := &SmartSupport{}
	if smartInfo.SmartSupport != nil {
		supportInfo.Available = smartInfo.SmartSupport.Available
		supportInfo.Enabled = smartInfo.SmartSupport.Enabled
		return supportInfo
	}
	if smartInfo.AtaSmartData != nil {
		supportInfo.Available = true
		supportInfo.Enabled = true
		return supportInfo
	}
	if smartInfo.NvmeSmartHealth != nil {
		supportInfo.Available = true
		supportInfo.Enabled = true
		return supportInfo
	}
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
	ctx = c.resolveCtx(ctx)
	smartInfo, err := c.GetSMARTInfo(ctx, devicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get SMART info: %w", err)
	}
	return c.GetSMARTSupportFromInfo(smartInfo), nil
}

// EnableSMART enables SMART monitoring on a device
func (c *Client) EnableSMART(ctx context.Context, devicePath string) error {
	ctx = c.resolveCtx(ctx)
	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, "-s", "on", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable SMART: %w", err)
	}
	return nil
}

// DisableSMART disables SMART monitoring on a device
// Note: NVMe devices do not support disabling SMART, an error will be returned
func (c *Client) DisableSMART(ctx context.Context, devicePath string) error {
	ctx = c.resolveCtx(ctx)

	// Check the cached device type first to avoid an unnecessary full disk query.
	// GetSMARTInfo populates the cache on its first successful call, so this path
	// is taken on every call after the initial scan or info query.
	if cachedType, ok := c.getCachedDeviceType(devicePath); ok {
		if strings.ToLower(cachedType) == "nvme" {
			return fmt.Errorf("cannot disable SMART: NVMe devices do not support SMART disable operation")
		}
	} else {
		// Cache is cold: query device info to determine the disk family.
		info, err := c.GetSMARTInfo(ctx, devicePath)
		if err != nil {
			return fmt.Errorf("failed to check device type: %w", err)
		}
		if determineDiskType(info) == "NVMe" {
			return fmt.Errorf("cannot disable SMART: NVMe devices do not support SMART disable operation")
		}
	}

	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, "-s", "off", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable SMART: %w", err)
	}
	return nil
}

// AbortSelfTest aborts a running self-test on a device
func (c *Client) AbortSelfTest(ctx context.Context, devicePath string) error {
	ctx = c.resolveCtx(ctx)
	cmd := c.commander.Command(ctx, c.logHandler, c.smartctlPath, "-X", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to abort self-test: %w", err)
	}
	return nil
}

// DiscoverDevices scans all available storage devices and probes each one to
// determine SMART readability and protocol compatibility.
//
// For each device found by ScanDevices, DiscoverDevices attempts to read SMART
// data using the auto-detected protocol. If that fails, it retries with an
// explicit -d sat flag and records whether the fallback was required. This
// gives callers a guided diagnostic path for devices that need manual
// WithSmartctlPath or protocol configuration.
//
// Example usage:
//
//	results, err := client.DiscoverDevices(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, r := range results {
//	    if !r.SMARTReadable {
//	        log.Printf("device %s is not SMART readable", r.DevicePath)
//	    }
//	    if r.SATFallbackRequired {
//	        log.Printf("device %s needs -d sat override", r.DevicePath)
//	    }
//	}
func (c *Client) DiscoverDevices(ctx context.Context) ([]DiscoveryResult, error) {
	ctx = c.resolveCtx(ctx)

	devices, err := c.ScanDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan devices for discovery: %w", err)
	}

	results := make([]DiscoveryResult, 0, len(devices))
	for _, dev := range devices {
		result := DiscoveryResult{
			DevicePath:       dev.Name,
			DetectedProtocol: dev.Type,
		}

		info, usedSATFallback, infoErr := c.getSMARTInfoInternal(ctx, dev.Name)
		if infoErr == nil && info != nil {
			result.SMARTReadable = true
			result.SATFallbackRequired = usedSATFallback
			result.DetectedProtocol = info.Device.Type
			result.Model = info.ModelName
			if result.Model == "" {
				result.Model = info.ModelFamily
			}
			result.Serial = info.SerialNumber
		} else {
			// The auto-detected protocol failed; try SAT explicitly.
			if satInfo, ok := c.retryWithDeviceType(ctx, dev.Name, "sat"); ok && satInfo != nil {
				result.SMARTReadable = true
				result.SATFallbackRequired = true
				result.DetectedProtocol = "sat"
				result.Model = satInfo.ModelName
				if result.Model == "" {
					result.Model = satInfo.ModelFamily
				}
				result.Serial = satInfo.SerialNumber
			}
		}

		results = append(results, result)
	}
	return results, nil
}
