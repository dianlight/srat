package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/dianlight/tlog"
)

var (
	_ Backend          = (*ExecBackend)(nil)
	_ DiscoveryBackend = (*ExecBackend)(nil)
)

// smartctlSearchPaths contains platform-specific locations tried in order when
// smartctl is not found in PATH. Ordered from most-common to most-specific to
// minimise stat calls on standard Linux installs.
var smartctlSearchPaths = []string{
	// Standard Linux (Debian, Ubuntu, RHEL, Fedora, Arch, Alpine, Proxmox, OMV)
	"/usr/sbin/smartctl",
	// FreeBSD / TrueNAS CORE / OpenBSD
	"/usr/local/sbin/smartctl",
	// macOS Homebrew (Intel)
	"/usr/local/bin/smartctl",
	// macOS Homebrew (Apple Silicon)
	"/opt/homebrew/bin/smartctl",
	// MacPorts
	"/opt/local/sbin/smartctl",
	// Synology DSM native package
	"/usr/syno/bin/smartctl",
	// Synology SynoCommunity QPKG
	"/volume1/@appstore/smartmontools/usr/sbin/smartctl",
	// QNAP Entware
	"/opt/sbin/smartctl",
	// QNAP QPKG
	"/share/CACHEDEV1_DATA/.qpkg/smartmontools/bin/smartctl",
	// NixOS system profile
	"/run/current-system/sw/sbin/smartctl",
}

// Option configures an [ExecBackend].
type Option func(*ExecBackend)

// ExecBackend is a [Backend] implementation that shells out to the smartctl binary.
type ExecBackend struct {
	smartctlPath       string
	commander          Commander
	defaultCommander   bool
	deviceTypeCache    map[string]string
	deviceTypeCacheMux sync.RWMutex
	healthBitsCache    map[string]int
	healthBitsCacheMux sync.RWMutex
	logHandler         LogAdapter
}

// WithSmartctlPath sets a custom path to the smartctl binary.
func WithSmartctlPath(path string) Option {
	return func(b *ExecBackend) {
		b.smartctlPath = path
	}
}

// WithCommander sets a custom commander, typically for testing.
func WithCommander(commander Commander) Option {
	return func(b *ExecBackend) {
		b.commander = commander
		b.defaultCommander = false
	}
}

// WithSlogHandler sets a custom slog.Logger for the backend.
func WithSlogHandler(logger *slog.Logger) Option {
	return withLogHandler(logger)
}

// WithTLogHandler sets a custom tlog.Logger for the backend.
func WithTLogHandler(logger *tlog.Logger) Option {
	return withLogHandler(logger)
}

// WithLogHandler sets a custom logger adapter for the backend.
func WithLogHandler(logger LogAdapter) Option {
	return withLogHandler(logger)
}

func withLogHandler(logger LogAdapter) Option {
	return func(b *ExecBackend) {
		b.logHandler = logger
	}
}

// New creates a new exec-backed SMART backend.
func New(opts ...Option) (*ExecBackend, error) {
	b := &ExecBackend{
		commander:        execCommander{},
		defaultCommander: true,
		deviceTypeCache:  cloneDeviceTypeCache(),
		healthBitsCache:  make(map[string]int),
		logHandler:       tlog.NewLoggerWithLevel(tlog.LevelDebug),
	}
	for _, opt := range opts {
		opt(b)
	}
	if b.smartctlPath == "" {
		path, err := resolveSmartctlPath()
		if err != nil {
			return nil, err
		}
		b.smartctlPath = path
	}
	if b.defaultCommander {
		if err := ensureCompatibleSmartctl(b.smartctlPath); err != nil {
			return nil, err
		}
	}
	return b, nil
}

// resolveSmartctlPath searches PATH and then platform-specific fallback
// locations for a usable smartctl binary. The WithSmartctlPath option always
// takes precedence and bypasses this function entirely.
func resolveSmartctlPath() (string, error) {
	// 1. Prefer PATH so that user-installed or version-managed binaries win.
	if path, err := exec.LookPath("smartctl"); err == nil {
		return path, nil
	}

	// 2. Search known platform-specific paths.
	for _, candidate := range smartctlSearchPaths {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		if info.Mode()&0o111 == 0 {
			continue // not executable
		}
		return candidate, nil
	}

	return "", fmt.Errorf(
		"smartctl not found in PATH or known locations.\n" +
			"Install smartmontools for your platform:\n" +
			"  Linux (Debian/Ubuntu): sudo apt install smartmontools\n" +
			"  Linux (RHEL/Fedora):   sudo dnf install smartmontools\n" +
			"  macOS (Homebrew):      brew install smartmontools\n" +
			"  Synology:              Install SynoCli Disk Tools from SynoCommunity\n" +
			"  QNAP:                  Install smartmontools via Entware (opkg install smartmontools)\n" +
			"  FreeBSD/TrueNAS:       pkg install smartmontools\n" +
			"More info: https://www.smartmontools.org/wiki/Download",
	)
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
// "smartctl 7.3 2022-02-28 r5338 ..." or "smartctl 7.5 ...".
func parseSmartctlVersion(output string) (int, int, error) {
	// Find first occurrence of "smartctl X.Y"
	re := regexp.MustCompile(`(?m)\bsmartctl\s+(\d+)\.(\d+)\b`)
	m := re.FindStringSubmatch(output)
	if len(m) != 3 {
		return 0, 0, fmt.Errorf("version pattern not found in output")
	}
	// Convert captures to ints using strconv for better performance
	major, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse major version: %w", err)
	}
	minor, err := strconv.Atoi(m[2])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse minor version: %w", err)
	}
	return major, minor, nil
}

// Name returns the backend identifier.
func (b *ExecBackend) Name() string {
	return "exec"
}

// Close releases resources held by the backend.
func (b *ExecBackend) Close() error {
	return nil
}

// SmartctlPath returns the resolved path to the smartctl binary.
func (b *ExecBackend) SmartctlPath() string {
	return b.smartctlPath
}

// SetDeviceTypeHint stores a device type hint in the backend cache.
func (b *ExecBackend) SetDeviceTypeHint(path, deviceType string) {
	b.setCachedDeviceType(path, deviceType)
}

// DeviceTypeHint returns a cached device type hint for the provided path.
func (b *ExecBackend) DeviceTypeHint(path string) (string, bool) {
	return b.getCachedDeviceType(path)
}

// NewExecBackend preserves the legacy constructor name.
func NewExecBackend(opts ...Option) (*ExecBackend, error) {
	return New(opts...)
}

// ScanDevices scans for available storage devices.
// It first attempts --scan-open (which performs an open on each drive to verify
// accessibility) and falls back to --scan on failure. --scan-open may fail in
// container sandboxes, on older kernels, or when the caller lacks the required
// permissions; --scan still returns the device list without the open step.
func (b *ExecBackend) ScanDevices(ctx context.Context) ([]Device, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := b.commander.Command(ctx, b.logHandler, b.smartctlPath, "--scan-open", "--json")
	output, err := cmd.Output()
	if err != nil {
		// Fall back to --scan when --scan-open is unsupported or fails.
		b.logHandler.WarnContext(ctx, "--scan-open failed, retrying with --scan", "err", err)
		fallbackCmd := b.commander.Command(ctx, b.logHandler, b.smartctlPath, "--scan", "--json")
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
			if _, cached := b.getCachedDeviceType(d.Name); !cached {
				b.setCachedDeviceType(d.Name, d.Type)
			}
		}
	}

	return devices, nil
}

// GetSMARTInfo retrieves SMART information for a device.
func (b *ExecBackend) GetSMARTInfo(ctx context.Context, devicePath string) (*SMARTInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	info, _, err := b.getSMARTInfoInternal(ctx, devicePath)
	return info, err
}

// CheckHealth checks if a device is healthy according to SMART.
func (b *ExecBackend) CheckHealth(ctx context.Context, devicePath string) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := b.commander.Command(ctx, b.logHandler, b.smartctlPath, b.buildArgs(devicePath, "-H")...)
	output, err := cmd.Output()
	if err != nil {
		// Exit code 2: device in standby
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			// exitCode == -1 means ProcessState is not set (mock/testing scenario)
			// exitCode&2 != 0 means device is in standby mode
			if exitCode != -1 && exitCode&2 != 0 {
				// Device in standby - cannot determine health
				b.logHandler.DebugContext(ctx, "Device in standby mode, cannot check health", "devicePath", devicePath)
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

// GetDeviceInfo retrieves basic device information.
func (b *ExecBackend) GetDeviceInfo(ctx context.Context, devicePath string) (map[string]interface{}, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := b.commander.Command(ctx, b.logHandler, b.smartctlPath, b.buildArgs(devicePath, "-i", "-j")...)
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

// RunSelfTest initiates a SMART self-test.
func (b *ExecBackend) RunSelfTest(ctx context.Context, devicePath string, testType string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	// Valid test types: short, long, conveyance, offline
	if !slices.Contains(validSelfTestTypes, testType) {
		return fmt.Errorf("invalid test type: %s (must be one of: short, long, conveyance, offline)", testType)
	}

	cmd := b.commander.Command(ctx, b.logHandler, b.smartctlPath, "-t", testType, devicePath)
	if err := cmd.Run(); err != nil {
		output, _ := cmd.CombinedOutput()
		return fmt.Errorf("failed to run self-test: %w (devicePath: %s, testType: %s, output: %s)", err, devicePath, testType, string(output))
	}

	return nil
}

// GetAvailableSelfTests returns the list of available self-test types and their durations for a device.
func (b *ExecBackend) GetAvailableSelfTests(ctx context.Context, devicePath string) (*SelfTestInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := b.commander.Command(ctx, b.logHandler, b.smartctlPath, b.buildArgs(devicePath, "-c", "-j")...)
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

// EnableSMART enables SMART monitoring on a device.
func (b *ExecBackend) EnableSMART(ctx context.Context, devicePath string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := b.commander.Command(ctx, b.logHandler, b.smartctlPath, "-s", "on", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable SMART: %w", err)
	}
	return nil
}

// DisableSMART disables SMART monitoring on a device.
// Note: NVMe devices do not support disabling SMART, an error will be returned.
func (b *ExecBackend) DisableSMART(ctx context.Context, devicePath string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// Check the cached device type first to avoid an unnecessary full disk query.
	// GetSMARTInfo populates the cache on its first successful call, so this path
	// is taken on every call after the initial scan or info query.
	if cachedType, ok := b.getCachedDeviceType(devicePath); ok {
		if strings.ToLower(cachedType) == "nvme" {
			return fmt.Errorf("cannot disable SMART: NVMe devices do not support SMART disable operation")
		}
	} else {
		// Cache is cold: query device info to determine the disk family.
		info, err := b.GetSMARTInfo(ctx, devicePath)
		if err != nil {
			return fmt.Errorf("failed to check device type: %w", err)
		}
		if determineDiskType(info) == "NVMe" {
			return fmt.Errorf("cannot disable SMART: NVMe devices do not support SMART disable operation")
		}
	}

	cmd := b.commander.Command(ctx, b.logHandler, b.smartctlPath, "-s", "off", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable SMART: %w", err)
	}
	return nil
}

// AbortSelfTest aborts a running self-test on a device.
func (b *ExecBackend) AbortSelfTest(ctx context.Context, devicePath string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := b.commander.Command(ctx, b.logHandler, b.smartctlPath, "-X", devicePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to abort self-test: %w", err)
	}
	return nil
}

// DiscoverDevices scans all available storage devices and probes each one to
// determine SMART readability and protocol compatibility.
func (b *ExecBackend) DiscoverDevices(ctx context.Context) ([]DiscoveryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	devices, err := b.ScanDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan devices for discovery: %w", err)
	}

	results := make([]DiscoveryResult, 0, len(devices))
	for _, dev := range devices {
		result := DiscoveryResult{
			DevicePath:       dev.Name,
			DetectedProtocol: dev.Type,
		}

		info, usedSATFallback, infoErr := b.getSMARTInfoInternal(ctx, dev.Name)
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
			if satInfo, ok := b.retryWithDeviceType(ctx, dev.Name, "sat"); ok && satInfo != nil {
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

// getCachedDeviceType retrieves a cached device type for the given device path.
func (b *ExecBackend) getCachedDeviceType(devicePath string) (string, bool) {
	b.deviceTypeCacheMux.RLock()
	defer b.deviceTypeCacheMux.RUnlock()
	deviceType, ok := b.deviceTypeCache[devicePath]
	return deviceType, ok
}

// setCachedDeviceType stores a device type in the cache for the given device path.
func (b *ExecBackend) setCachedDeviceType(devicePath, deviceType string) {
	b.deviceTypeCacheMux.Lock()
	defer b.deviceTypeCacheMux.Unlock()
	b.deviceTypeCache[devicePath] = deviceType
	b.logHandler.Debug("Cached device type", "devicePath", devicePath, "deviceType", deviceType)
}

// buildArgs assembles smartctl arguments for devicePath, prepending flags and
// inserting --nocheck=standby (ATA only) plus -d <type> when the device type
// is already known from the cache. Falls back to the ATA-safe default when the
// cache is cold.
func (b *ExecBackend) buildArgs(devicePath string, flags ...string) []string {
	if cachedType, ok := b.getCachedDeviceType(devicePath); ok {
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
func (b *ExecBackend) logSmartctlMessages(ctx context.Context, info *SMARTInfo) {
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
			b.logHandler.ErrorContext(ctx, msg.String)
		case "warning":
			b.logHandler.WarnContext(ctx, msg.String)
		default:
			b.logHandler.InfoContext(ctx, msg.String)
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
func (b *ExecBackend) retryWithDeviceType(ctx context.Context, devicePath, deviceType string) (*SMARTInfo, bool) {
	args := []string{"-a", "-j", "--nocheck=standby", "-d", deviceType, devicePath}
	cmd := b.commander.Command(ctx, b.logHandler, b.smartctlPath, args...)
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
			b.setCachedDeviceType(devicePath, deviceType)
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
	b.setCachedDeviceType(devicePath, deviceType)
	b.logHandler.InfoContext(ctx, "Device type retry succeeded", "devicePath", devicePath, "deviceType", deviceType)
	info.DiskType = determineDiskType(&info)
	info.SmartStatus = checkSmartStatus(&info)
	b.logHealthBits(ctx, devicePath, &info)
	b.logSmartctlMessages(ctx, &info)
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
func (b *ExecBackend) retrySATFallback(ctx context.Context, devicePath string) (*SMARTInfo, bool) {
	b.logHandler.InfoContext(ctx, "execution failure with default protocol, retrying with -d sat", "devicePath", devicePath)
	return b.retryWithDeviceType(ctx, devicePath, "sat")
}

// getSMARTInfoInternal is the implementation behind GetSMARTInfo. The second
// return value is true when the internal SAT fallback (retrySATFallback) was
// invoked and succeeded, allowing DiscoverDevices to surface SATFallbackRequired
// without changing the public GetSMARTInfo signature.
func (b *ExecBackend) getSMARTInfoInternal(ctx context.Context, devicePath string) (*SMARTInfo, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := b.commander.Command(ctx, b.logHandler, b.smartctlPath, b.buildArgs(devicePath, "-a", "-j")...)
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
				if _, hasCached := b.getCachedDeviceType(devicePath); !hasCached {
					if info, satOK := b.retrySATFallback(ctx, devicePath); satOK {
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
							if _, cached := b.getCachedDeviceType(devicePath); !cached {
								b.setCachedDeviceType(devicePath, smartInfo.Device.Type)
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
					if _, cached := b.getCachedDeviceType(devicePath); !cached {
						b.setCachedDeviceType(devicePath, smartInfo.Device.Type)
					}
				}

				b.logSmartctlMessages(ctx, &smartInfo)

				// Check if this is an unknown USB bridge error and we haven't cached a type yet
				if isUnknownUSBBridge(&smartInfo) {
					if _, hasCached := b.getCachedDeviceType(devicePath); !hasCached {
						// Prefer a type from drivedb for known bridges; fall back to sat.
						deviceType := "sat"
						if usbBridgeID := extractUSBBridgeID(&smartInfo); usbBridgeID != "" {
							if knownType, ok := b.getCachedDeviceType(usbBridgeID); ok {
								deviceType = knownType
								b.logHandler.InfoContext(ctx, "Found USB bridge in drivedb", "usbBridgeID", usbBridgeID, "deviceType", deviceType)
							}
						}
						if deviceType == "sat" {
							b.logHandler.InfoContext(ctx, "Unknown USB bridge detected, retrying with -d sat", "devicePath", devicePath)
						}
						if info, ok := b.retryWithDeviceType(ctx, devicePath, deviceType); ok {
							return info, false, nil
						}
						b.logHandler.ErrorContext(ctx, "Retry with device type failed", "devicePath", devicePath, "deviceType", deviceType)
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

	b.logSmartctlMessages(ctx, &smartInfo)

	// Determine disk type based on rotation rate and device type
	smartInfo.DiskType = determineDiskType(&smartInfo)
	// Populate SmartStatus.Running field based on test status
	smartInfo.SmartStatus = checkSmartStatus(&smartInfo)
	b.logHealthBits(ctx, devicePath, &smartInfo)

	// Cache the device type from the successful response so all subsequent
	// methods can use --nocheck=standby and the correct -d <type> argument
	// without issuing another disk query.
	if smartInfo.Device.Type != "" {
		if _, cached := b.getCachedDeviceType(devicePath); !cached {
			b.setCachedDeviceType(devicePath, smartInfo.Device.Type)
		}
	}

	return &smartInfo, false, nil
}

// logHealthBits emits a single WARNING per device per unique health-bit pattern.
// When a drive enters a stable-but-degraded state (e.g., pre-failure attributes
// below threshold), subsequent polls produce the same bits and are suppressed to
// avoid flooding the caller's log.
func (b *ExecBackend) logHealthBits(ctx context.Context, devicePath string, info *SMARTInfo) {
	if info == nil || info.ExitCodeInfo == nil || info.ExitCodeInfo.HealthBits == 0 {
		return
	}
	bits := info.ExitCodeInfo.HealthBits

	b.healthBitsCacheMux.Lock()
	prev, seen := b.healthBitsCache[devicePath]
	if seen && prev == bits {
		b.healthBitsCacheMux.Unlock()
		return
	}
	b.healthBitsCache[devicePath] = bits
	b.healthBitsCacheMux.Unlock()

	b.logHandler.WarnContext(ctx, "SMART health flags detected",
		"devicePath", devicePath,
		"healthBits", bits,
		"diskFailing", bits&0x08 != 0,
		"prefailAttr", bits&0x10 != 0,
		"pastPrefail", bits&0x20 != 0,
		"errorLog", bits&0x40 != 0,
		"selfTestLog", bits&0x80 != 0,
	)
}
