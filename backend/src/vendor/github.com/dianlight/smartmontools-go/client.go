package smartmontools

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	smtypes "github.com/dianlight/smartmontools-go/internal/types"
	"github.com/dianlight/tlog"
)

// SMART attribute IDs for SSD detection and wear-level computation.
const (
	SmartAttrSSDLifeUsed       = smtypes.SmartAttrSSDLifeUsed
	SmartAttrWearLevelingCount = smtypes.SmartAttrWearLevelingCount
	SmartAttrSSDLifeLeft       = smtypes.SmartAttrSSDLifeLeft
	SmartAttrSandForceInternal = smtypes.SmartAttrSandForceInternal
	SmartAttrTotalLBAsWritten  = smtypes.SmartAttrTotalLBAsWritten
)

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithSmartctlPath sets a custom path to the smartctl binary.
// This option is only effective when using the default ExecBackend.
// It is silently ignored when WithBackend is also provided.
func WithSmartctlPath(path string) ClientOption {
	return func(c *Client) {
		c.pendingExecOpts = append(c.pendingExecOpts, WithExecSmartctlPath(path))
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

// WithCommander sets a custom commander for testing purposes.
// This option is only effective when using the default ExecBackend.
// It is silently ignored when WithBackend is also provided.
func WithCommander(commander Commander) ClientOption {
	return func(c *Client) {
		c.pendingExecOpts = append(c.pendingExecOpts, WithExecCommander(commander))
	}
}

// WithContext sets a default context to use when methods are called with nil context.
func WithContext(ctx context.Context) ClientOption {
	return func(c *Client) {
		c.defaultCtx = ctx
	}
}

// WithBackend sets an explicit Backend implementation, bypassing the default
// ExecBackend. When WithBackend is provided, options such as WithSmartctlPath
// and WithCommander have no effect.
func WithBackend(backend Backend) ClientOption {
	return func(c *Client) {
		c.backend = backend
	}
}

// LogAdapter captures the logging methods used by this package.
type LogAdapter = smtypes.LogAdapter

var (
	_ LogAdapter = (*tlog.Logger)(nil)
	_ LogAdapter = (*slog.Logger)(nil)
)

// SmartClient interface defines the methods for interacting with smartmontools.
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
	Close() error
}

// Client represents a smartmontools client that delegates SMART operations
// to a pluggable [Backend]. The default backend is [ExecBackend].
type Client struct {
	backend         Backend
	logHandler      LogAdapter // staging: propagated to ExecBackend during NewClient
	defaultCtx      context.Context
	pendingExecOpts []ExecBackendOption // staging: collected during option application, consumed by NewClient
}

// NewClient creates a new smartmontools client with optional configuration.
// If no Backend is provided via WithBackend, an ExecBackend is created using
// any pending exec options (e.g., from WithSmartctlPath or WithCommander).
// If no log handler is provided, a tlog debug-level logger is used.
// If no context is provided, context.Background() is used as the default.
func NewClient(opts ...ClientOption) (SmartClient, error) {
	client := &Client{
		logHandler: tlog.NewLoggerWithLevel(tlog.LevelDebug),
		defaultCtx: context.Background(),
	}
	for _, opt := range opts {
		opt(client)
	}
	if client.backend == nil {
		execOpts := append([]ExecBackendOption{WithExecLogHandler(client.logHandler)}, client.pendingExecOpts...)
		backend, err := NewExecBackend(execOpts...)
		if err != nil {
			return nil, err
		}
		client.backend = backend
	}
	client.pendingExecOpts = nil
	return client, nil
}

// resolveCtx returns ctx if non-nil, otherwise returns the client's default context.
func (c *Client) resolveCtx(ctx context.Context) context.Context {
	if ctx == nil {
		return c.defaultCtx
	}
	return ctx
}

// Close releases any resources held by the active backend.
func (c *Client) Close() error {
	return c.backend.Close()
}

// ScanDevices scans for available storage devices.
func (c *Client) ScanDevices(ctx context.Context) ([]Device, error) {
	return c.backend.ScanDevices(c.resolveCtx(ctx))
}

// GetSMARTInfo retrieves SMART information for a device.
func (c *Client) GetSMARTInfo(ctx context.Context, devicePath string) (*SMARTInfo, error) {
	return c.backend.GetSMARTInfo(c.resolveCtx(ctx), devicePath)
}

// CheckHealth checks if a device is healthy according to SMART.
func (c *Client) CheckHealth(ctx context.Context, devicePath string) (bool, error) {
	return c.backend.CheckHealth(c.resolveCtx(ctx), devicePath)
}

// GetDeviceInfo retrieves basic device information.
func (c *Client) GetDeviceInfo(ctx context.Context, devicePath string) (map[string]interface{}, error) {
	return c.backend.GetDeviceInfo(c.resolveCtx(ctx), devicePath)
}

// RunSelfTest initiates a SMART self-test.
func (c *Client) RunSelfTest(ctx context.Context, devicePath string, testType string) error {
	return c.backend.RunSelfTest(c.resolveCtx(ctx), devicePath, testType)
}

// RunSelfTestWithProgress starts a SMART self-test and reports progress.
func (c *Client) RunSelfTestWithProgress(ctx context.Context, devicePath string, testType string, callback ProgressCallback) error {
	ctx = c.resolveCtx(ctx)
	// Valid test types: short, long, conveyance, offline
	if !slices.Contains(smtypes.ValidSelfTestTypes, testType) {
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

// GetAvailableSelfTests returns the list of available self-test types and their durations for a device.
func (c *Client) GetAvailableSelfTests(ctx context.Context, devicePath string) (*SelfTestInfo, error) {
	return c.backend.GetAvailableSelfTests(c.resolveCtx(ctx), devicePath)
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
// // Get and cache SMART info once
// info, err := client.GetSMARTInfo(ctx, devicePath)
//
//	if err != nil {
//	   return err
//	}
//
// // Extract self-test types without another disk query
// selfTests := client.GetAvailableSelfTestsFromInfo(info)
//
//	for _, testType := range selfTests.Available {
//	   fmt.Printf("Test: %s (%d min)\n", testType, selfTests.Durations[testType])
//	}
func (c *Client) GetAvailableSelfTestsFromInfo(smartInfo *SMARTInfo) *SelfTestInfo {
	info := &SelfTestInfo{
		Available: []string{},
		Durations: make(map[string]int),
	}
	if smartInfo == nil {
		return info
	}
	smtypes.PopulateSelfTestInfo(info, smartInfo.AtaSmartData, smartInfo.NvmeControllerCapabilities, nil)
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
// // Get and cache SMART info once
// info, err := client.GetSMARTInfo(ctx, devicePath)
//
//	if err != nil {
//	   return err
//	}
//
// // Check SMART status from cached info (no disk access)
// support := client.GetSMARTSupportFromInfo(info)
//
//	if support.Available && support.Enabled {
//	   // SMART is available and enabled
//	}
//
// // After EnableSMART/DisableSMART, refresh the cache:
//
//	if err := client.EnableSMART(ctx, devicePath); err != nil {
//	   return err
//	}
//
// // Refresh cache after state change
// info, err = client.GetSMARTInfo(ctx, devicePath)
//
//	if err != nil {
//	   return err
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
// // Initial query (performed once or when SMART status changes)
// info, err := client.GetSMARTInfo(ctx, devicePath)
//
//	if err != nil {
//	   return err
//	}
//
// // Cache the info and check SMART status without disk I/O
// support := client.GetSMARTSupportFromInfo(info)
//
//	if !support.Enabled {
//	   // Skip SMART monitoring when disabled
//	   return
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

// EnableSMART enables SMART monitoring on a device.
func (c *Client) EnableSMART(ctx context.Context, devicePath string) error {
	return c.backend.EnableSMART(c.resolveCtx(ctx), devicePath)
}

// DisableSMART disables SMART monitoring on a device.
func (c *Client) DisableSMART(ctx context.Context, devicePath string) error {
	return c.backend.DisableSMART(c.resolveCtx(ctx), devicePath)
}

// AbortSelfTest aborts a running self-test on a device.
func (c *Client) AbortSelfTest(ctx context.Context, devicePath string) error {
	return c.backend.AbortSelfTest(c.resolveCtx(ctx), devicePath)
}

// DiscoverDevices scans all available storage devices and probes each one to
// determine SMART readability and protocol compatibility.
func (c *Client) DiscoverDevices(ctx context.Context) ([]DiscoveryResult, error) {
	ctx = c.resolveCtx(ctx)
	if db, ok := c.backend.(DiscoveryBackend); ok {
		return db.DiscoverDevices(ctx)
	}
	// Generic fallback for backends that don't implement DiscoveryBackend.
	// No SAT-fallback details are available in this path.
	devices, err := c.backend.ScanDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan devices for discovery: %w", err)
	}
	results := make([]DiscoveryResult, 0, len(devices))
	for _, dev := range devices {
		result := DiscoveryResult{DevicePath: dev.Name, DetectedProtocol: dev.Type}
		info, infoErr := c.backend.GetSMARTInfo(ctx, dev.Name)
		if infoErr == nil && info != nil {
			result.SMARTReadable = true
			result.DetectedProtocol = info.Device.Type
			result.Model = info.ModelName
			if result.Model == "" {
				result.Model = info.ModelFamily
			}
			result.Serial = info.SerialNumber
		}
		results = append(results, result)
	}
	return results, nil
}
