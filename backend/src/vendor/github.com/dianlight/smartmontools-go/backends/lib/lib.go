//go:build linux || darwin

// Package lib provides a Backend implementation that loads the smartmon wrapper
// library via purego (no CGO required). Build the wrapper library once with:
//
//	scripts/setup-lib-backend.sh
//
// The script downloads the pre-built libsmartmon.a static library from
// https://github.com/dianlight/smartmontools-sdk releases and compiles the
// thin C++ wrapper (backends/lib/csrc/smartmon_c_api.cpp) into
// backends/lib/sdk/libsmartmon_go.{so,dylib}.
//
// Then point the backend at it via SMARTMON_LIB_PATH or WithLibraryPath.
//
//go:generate ../../scripts/setup-lib-backend.sh
package lib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"github.com/dianlight/tlog"
	"github.com/ebitengine/purego"

	smtypes "github.com/dianlight/smartmontools-go/internal/types"
)

// defaultLibNames are tried first using the dynamic linker (respects
// LD_LIBRARY_PATH / DYLD_LIBRARY_PATH / rpath).
var defaultLibNames = []string{
	"libsmartmon_go.so",
	"libsmartmon_go.dylib",
}

// defaultLibPaths are absolute paths tried as a fallback.
var defaultLibPaths = []string{
	"/usr/local/lib/libsmartmon_go.so",
	"/usr/lib/libsmartmon_go.so",
	"/usr/lib/x86_64-linux-gnu/libsmartmon_go.so",
	"/usr/lib/aarch64-linux-gnu/libsmartmon_go.so",
	"/opt/homebrew/lib/libsmartmon_go.dylib",
	"/usr/local/lib/libsmartmon_go.dylib",
}

// libFuncs holds C function pointers loaded from the wrapper library via purego.
// All functions match the signatures declared in smartmon_c_api.h.
type libFuncs struct {
	init          func() int32
	cleanup       func()
	scanDevices   func(outJSON *unsafe.Pointer) int32
	getSmartData  func(device, devType string, outJSON *unsafe.Pointer) int32
	checkHealth   func(device, devType string, outHealthy *int32) int32
	enableSmart   func(device, devType string) int32
	disableSmart  func(device, devType string) int32
	runSelftest   func(device, devType, testType string) int32
	abortSelftest func(device, devType string) int32
	freeString    func(s unsafe.Pointer)
	lastError     func() unsafe.Pointer
}

// Option configures a LibBackend.
type Option func(*LibBackend)

// LibBackend implements Backend by loading the smartmon wrapper library via
// purego at runtime. No CGO is required. Build the wrapper once with:
//
//	scripts/setup-lib-backend.sh
//
// The script downloads libsmartmon.a from github.com/dianlight/smartmontools-sdk
// releases and compiles the thin C++ wrapper into
// backends/lib/sdk/libsmartmon_go.{so,dylib}.
type LibBackend struct {
	libHandle  uintptr
	libPath    string
	funcs      *libFuncs
	logHandler LogAdapter
	closeOnce  sync.Once
	mu         sync.RWMutex
	closed     bool
}

// Ensure LibBackend satisfies the Backend interface at compile time.
var _ Backend = (*LibBackend)(nil)

// WithLibraryPath sets a custom path to the smartmon wrapper shared library.
// When not set, New checks SMARTMON_LIB_PATH, then defaultLibNames and defaultLibPaths.
func WithLibraryPath(path string) Option {
	return func(b *LibBackend) {
		b.libPath = path
	}
}

// WithSlogHandler sets a slog.Logger for the backend.
func WithSlogHandler(logger *slog.Logger) Option {
	return withLogHandler(logger)
}

// WithTLogHandler sets a tlog.Logger for the backend.
func WithTLogHandler(logger *tlog.Logger) Option {
	return withLogHandler(logger)
}

// WithLogHandler sets a custom logger adapter for the backend.
func WithLogHandler(logger LogAdapter) Option {
	return withLogHandler(logger)
}

func withLogHandler(logger LogAdapter) Option {
	return func(b *LibBackend) {
		if logger != nil {
			b.logHandler = logger
		}
	}
}

// New creates a LibBackend by loading the smartmon wrapper library.
//
// Library resolution order:
//  1. [WithLibraryPath] option
//  2. SMARTMON_LIB_PATH environment variable (if the file exists at that path).
//     If SMARTMON_LIB_PATH is set but the file is absent a warning is logged
//     and the search falls through to step 3.
//     If SMARTMON_LIB_PATH resolves to a different directory than a library
//     found in the system paths a warning is logged.
//  3. Standard system library paths (dynamic linker names, then absolute paths).
func New(opts ...Option) (*LibBackend, error) {
	b := &LibBackend{
		logHandler: tlog.NewLoggerWithLevel(tlog.LevelDebug),
	}
	for _, opt := range opts {
		opt(b)
	}

	if b.libPath == "" {
		if envPath := os.Getenv("SMARTMON_LIB_PATH"); envPath != "" {
			if _, err := os.Stat(envPath); err == nil {
				b.libPath = envPath
				// Warn when a library also exists in a different standard location,
				// which may indicate a stale or unintended installation.
				if sysPath, ok := findSystemLibPath(); ok && !sameDir(envPath, sysPath) {
					b.logHandler.WarnContext(context.Background(),
						"SMARTMON_LIB_PATH is set but library also found in a different system path",
						"env_path", envPath,
						"system_path", sysPath)
				}
			} else {
				// env path is set but the file is missing — warn and fall back to the
				// system-wide search so the backend can still start if the library is
				// installed globally.
				b.logHandler.WarnContext(context.Background(),
					"SMARTMON_LIB_PATH is set but the file was not found; falling back to system library search",
					"path", envPath)
			}
		}
	}

	if b.libPath == "" {
		path, err := resolveLibPath()
		if err != nil {
			return nil, err
		}
		b.libPath = path
	}

	handle, err := purego.Dlopen(b.libPath, purego.RTLD_LAZY|purego.RTLD_LOCAL)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", b.libPath, err)
	}

	funcs, err := registerFuncs(handle)
	if err != nil {
		_ = purego.Dlclose(handle)
		return nil, fmt.Errorf("failed to register library functions from %s: %w", b.libPath, err)
	}
	b.funcs = funcs
	b.libHandle = handle

	if rc := funcs.init(); rc != 0 {
		_ = purego.Dlclose(handle)
		return nil, fmt.Errorf("smartmon_init failed (code %d): %s", rc, goString(funcs.lastError()))
	}
	return b, nil
}

// Name returns the backend identifier.
func (b *LibBackend) Name() string { return "lib" }

// Close calls smartmon_cleanup and unloads the shared library.
// It is safe to call Close more than once; subsequent calls are no-ops.
func (b *LibBackend) Close() error {
	var closeErr error
	b.closeOnce.Do(func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		b.closed = true

		if b.funcs != nil {
			b.funcs.cleanup()
			b.funcs = nil
		}
		if b.libHandle != 0 {
			if err := purego.Dlclose(b.libHandle); err != nil {
				closeErr = err
			}
			b.libHandle = 0
		}
	})
	return closeErr
}

// ScanDevices returns the list of storage devices visible to the wrapper library.
func (b *LibBackend) ScanDevices(ctx context.Context) ([]Device, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, errors.New("backend is closed")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	jsonStr, err := b.callWithStringOut(func(out *unsafe.Pointer) int32 {
		return b.funcs.scanDevices(out)
	})
	if err != nil {
		return nil, fmt.Errorf("scan devices: %w", err)
	}

	var result struct {
		Devices []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"devices"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse scan output: %w", err)
	}

	devices := make([]Device, len(result.Devices))
	for i, d := range result.Devices {
		devices[i] = Device{Name: d.Name, Type: d.Type}
	}
	return devices, nil
}

// GetSMARTInfo returns comprehensive SMART information for the given device.
func (b *LibBackend) GetSMARTInfo(ctx context.Context, devicePath string) (*SMARTInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, errors.New("backend is closed")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	jsonStr, err := b.getSmartJSON(devicePath)
	if err != nil {
		return nil, err
	}

	var info SMARTInfo
	if err := json.Unmarshal([]byte(jsonStr), &info); err != nil {
		return nil, fmt.Errorf("failed to parse SMART info for %s: %w", devicePath, err)
	}
	return &info, nil
}

// CheckHealth returns true when the device passes its SMART overall-health assessment.
func (b *LibBackend) CheckHealth(ctx context.Context, devicePath string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return false, errors.New("backend is closed")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}

	var healthy int32
	rc := b.funcs.checkHealth(devicePath, "", &healthy)
	if rc != 0 {
		return false, fmt.Errorf("check health for %s: %w", devicePath, b.lastError())
	}
	return healthy != 0, nil
}

// GetDeviceInfo returns raw key/value device information for the given device.
func (b *LibBackend) GetDeviceInfo(ctx context.Context, devicePath string) (map[string]any, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, errors.New("backend is closed")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	jsonStr, err := b.getSmartJSON(devicePath)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse device info for %s: %w", devicePath, err)
	}
	return result, nil
}

// RunSelfTest starts a SMART self-test of the given type on the device.
// testType must be "short", "long", or "conveyance".
func (b *LibBackend) RunSelfTest(ctx context.Context, devicePath string, testType string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return errors.New("backend is closed")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	rc := b.funcs.runSelftest(devicePath, "", testType)
	if rc != 0 {
		return fmt.Errorf("run self-test (%s) on %s: %w", testType, devicePath, b.lastError())
	}
	return nil
}

// GetAvailableSelfTests returns the self-test types available on the device and
// their estimated durations, parsed from the full SMART data.
func (b *LibBackend) GetAvailableSelfTests(ctx context.Context, devicePath string) (*SelfTestInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, errors.New("backend is closed")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	jsonStr, err := b.getSmartJSON(devicePath)
	if err != nil {
		return nil, err
	}

	var caps struct {
		AtaSmartData               *smtypes.AtaSmartData               `json:"ata_smart_data,omitempty"`
		NvmeControllerCapabilities *smtypes.NvmeControllerCapabilities `json:"nvme_controller_capabilities,omitempty"`
		NvmeOptionalAdminCommands  *smtypes.NvmeOptionalAdminCommands  `json:"nvme_optional_admin_commands,omitempty"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &caps); err != nil {
		return nil, fmt.Errorf("failed to parse capabilities for %s: %w", devicePath, err)
	}

	info := &SelfTestInfo{
		Available: []string{},
		Durations: make(map[string]int),
	}
	smtypes.PopulateSelfTestInfo(info, caps.AtaSmartData, caps.NvmeControllerCapabilities, caps.NvmeOptionalAdminCommands)
	return info, nil
}

// EnableSMART enables SMART on the given ATA device.
func (b *LibBackend) EnableSMART(ctx context.Context, devicePath string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return errors.New("backend is closed")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	rc := b.funcs.enableSmart(devicePath, "")
	if rc != 0 {
		return fmt.Errorf("enable SMART on %s: %w", devicePath, b.lastError())
	}
	return nil
}

// DisableSMART disables SMART on the given ATA device.
func (b *LibBackend) DisableSMART(ctx context.Context, devicePath string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return errors.New("backend is closed")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	rc := b.funcs.disableSmart(devicePath, "")
	if rc != 0 {
		return fmt.Errorf("disable SMART on %s: %w", devicePath, b.lastError())
	}
	return nil
}

// AbortSelfTest aborts a running SMART self-test on the given device.
func (b *LibBackend) AbortSelfTest(ctx context.Context, devicePath string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return errors.New("backend is closed")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	rc := b.funcs.abortSelftest(devicePath, "")
	if rc != 0 {
		return fmt.Errorf("abort self-test on %s: %w", devicePath, b.lastError())
	}
	return nil
}

// resolveLibPath searches defaultLibNames (dynamic linker) then defaultLibPaths.
func resolveLibPath() (string, error) {
	for _, name := range defaultLibNames {
		if h, err := purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_LOCAL); err == nil {
			_ = purego.Dlclose(h)
			return name, nil
		}
	}
	for _, path := range defaultLibPaths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", errors.New("smartmon wrapper library (libsmartmon_go.{so,dylib}) not found; build it with scripts/setup-lib-backend.sh which downloads libsmartmon.a from github.com/dianlight/smartmontools-sdk and compiles the wrapper, then set SMARTMON_LIB_PATH or copy to a standard library directory")
}

// findSystemLibPath returns the first library found among defaultLibPaths.
// Unlike resolveLibPath it does not probe the dynamic linker, so it always
// returns an absolute path suitable for directory comparison.
func findSystemLibPath() (string, bool) {
	for _, path := range defaultLibPaths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

// sameDir reports whether two file paths share the same directory.
// Symlinks are resolved before comparing so that equivalent paths that differ
// only in their symlink structure are not reported as diverging.
func sameDir(a, b string) bool {
	dirA := filepath.Clean(filepath.Dir(a))
	dirB := filepath.Clean(filepath.Dir(b))
	if ra, err := filepath.EvalSymlinks(dirA); err == nil {
		dirA = ra
	}
	if rb, err := filepath.EvalSymlinks(dirB); err == nil {
		dirB = rb
	}
	return dirA == dirB
}

// registerFuncs binds all C symbols from the loaded library via purego.
func registerFuncs(lib uintptr) (*libFuncs, error) {
	f := &libFuncs{}
	for _, b := range []struct {
		ptr  any
		name string
	}{
		{&f.init, "smartmon_init"},
		{&f.cleanup, "smartmon_cleanup"},
		{&f.scanDevices, "smartmon_scan_devices"},
		{&f.getSmartData, "smartmon_get_smart_data"},
		{&f.checkHealth, "smartmon_check_health"},
		{&f.enableSmart, "smartmon_enable_smart"},
		{&f.disableSmart, "smartmon_disable_smart"},
		{&f.runSelftest, "smartmon_run_selftest"},
		{&f.abortSelftest, "smartmon_abort_selftest"},
		{&f.freeString, "smartmon_free_string"},
		{&f.lastError, "smartmon_last_error"},
	} {
		if err := registerFunc(lib, b.name, b.ptr); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// registerFunc registers a single C symbol, converting any panic into an error.
func registerFunc(lib uintptr, name string, fptr any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("symbol %q: %v", name, r)
		}
	}()
	purego.RegisterLibFunc(fptr, lib, name)
	return nil
}

// getSmartJSON calls smartmon_get_smart_data and returns the raw JSON string.
func (b *LibBackend) getSmartJSON(devicePath string) (string, error) {
	return b.callWithStringOut(func(out *unsafe.Pointer) int32 {
		return b.funcs.getSmartData(devicePath, "", out)
	})
}

// callWithStringOut calls a C function that writes a heap-allocated JSON string
// to *out, converts it to a Go string, and frees the C allocation.
// The caller must hold at least a read lock on b.mu.
func (b *LibBackend) callWithStringOut(fn func(out *unsafe.Pointer) int32) (string, error) {
	var outPtr unsafe.Pointer
	if rc := fn(&outPtr); rc != 0 {
		return "", b.lastError()
	}
	if outPtr == nil {
		return "", errors.New("library returned nil output")
	}
	result := goString(outPtr)
	b.funcs.freeString(outPtr)
	return result, nil
}

// lastError reads the last error from the C library as a Go error.
// The caller must hold at least a read lock on b.mu.
func (b *LibBackend) lastError() error {
	if b.funcs == nil {
		return errors.New("backend not initialised")
	}
	ptr := b.funcs.lastError()
	if ptr == nil {
		return errors.New("unknown library error")
	}
	return errors.New(goString(ptr))
}

// goString converts a null-terminated C string pointer into a Go string.
// The C memory is not modified or freed.
func goString(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Add(p, n)) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(p), n))
}
