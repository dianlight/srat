// Package compare provides a virtual Backend that runs multiple backends in
// parallel, uses the first (master) backend's results as the response, and
// logs discrepancies between backends for testing and validation purposes.
package compare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	smtypes "github.com/dianlight/smartmontools-go/internal/types"
	"github.com/dianlight/tlog"
)

var (
	_ Backend          = (*CompareBackend)(nil)
	_ DiscoveryBackend = (*CompareBackend)(nil)
)

// Option configures a CompareBackend.
type Option func(*CompareBackend)

// CompareBackend is a virtual Backend that runs multiple backends in parallel,
// returns the first (master) backend's results, and logs discrepancies.
//
// All backends execute concurrently so overall latency is determined by the
// slowest backend. Callers should set an appropriate timeout on the context
// to bound latency when using slow or unreliable secondaries.
type CompareBackend struct {
	backends []Backend
	log      LogAdapter
}

// WithLogHandler sets a custom LogAdapter for the compare backend.
func WithLogHandler(logger LogAdapter) Option {
	return func(b *CompareBackend) {
		b.log = logger
	}
}

// WithSlogHandler sets a custom slog.Logger for the compare backend.
func WithSlogHandler(logger *slog.Logger) Option {
	return WithLogHandler(logger)
}

// WithTLogHandler sets a custom tlog.Logger for the compare backend.
func WithTLogHandler(logger *tlog.Logger) Option {
	return WithLogHandler(logger)
}

// NewCompareBackend creates a CompareBackend with at least two backends.
// The first backend is the master; its results are always returned to the caller.
// Secondary backends run in parallel with the master; result mismatches are
// logged as warnings and secondary errors are logged as errors.
func NewCompareBackend(backends []Backend, opts ...Option) (*CompareBackend, error) {
	if len(backends) < 2 {
		return nil, fmt.Errorf("compare backend requires at least 2 backends, got %d", len(backends))
	}
	copied := make([]Backend, len(backends))
	for i, b := range backends {
		if b == nil {
			return nil, fmt.Errorf("compare backend: nil backend at index %d", i)
		}
		copied[i] = b
	}
	cb := &CompareBackend{
		backends: copied,
		log:      tlog.NewLoggerWithLevel(tlog.LevelDebug),
	}
	for _, opt := range opts {
		opt(cb)
	}
	return cb, nil
}

// Name returns "compare".
func (c *CompareBackend) Name() string {
	return "compare"
}

// Close closes all backends, combining any errors into a single return value.
func (c *CompareBackend) Close() error {
	errs := make([]error, 0, len(c.backends))
	for _, b := range c.backends {
		if err := b.Close(); err != nil {
			errs = append(errs, fmt.Errorf("backend %q: %w", b.Name(), err))
		}
	}
	return errors.Join(errs...)
}

// ScanDevices scans for storage devices from all backends and returns the master's result.
func (c *CompareBackend) ScanDevices(ctx context.Context) ([]Device, error) {
	type res struct {
		devices []Device
		err     error
	}
	results := make([]res, len(c.backends))
	var wg sync.WaitGroup
	for i, b := range c.backends {
		wg.Go(func() {
			devices, err := b.ScanDevices(ctx)
			results[i] = res{devices, err}
		})
	}
	wg.Wait()

	master := results[0]
	for i, r := range results[1:] {
		name := c.backends[i+1].Name()
		if r.err != nil {
			c.logSecondaryError(ctx, "ScanDevices", name, r.err)
			continue
		}
		if !jsonEqual(sortedDevicesCopy(master.devices), sortedDevicesCopy(r.devices)) {
			c.logMismatch(ctx, "ScanDevices", name, master.devices, r.devices)
		}
	}
	return master.devices, master.err
}

// GetSMARTInfo retrieves SMART information from all backends and returns the master's result.
func (c *CompareBackend) GetSMARTInfo(ctx context.Context, devicePath string) (*SMARTInfo, error) {
	type res struct {
		info *SMARTInfo
		err  error
	}
	results := make([]res, len(c.backends))
	var wg sync.WaitGroup
	for i, b := range c.backends {
		wg.Go(func() {
			info, err := b.GetSMARTInfo(ctx, devicePath)
			results[i] = res{info, err}
		})
	}
	wg.Wait()

	master := results[0]
	for i, r := range results[1:] {
		name := c.backends[i+1].Name()
		if r.err != nil {
			c.logSecondaryError(ctx, "GetSMARTInfo", name, r.err)
			continue
		}
		if !jsonEqual(master.info, r.info) {
			c.logMismatch(ctx, "GetSMARTInfo", name, master.info, r.info)
		}
	}
	return master.info, master.err
}

// CheckHealth checks device health from all backends and returns the master's result.
func (c *CompareBackend) CheckHealth(ctx context.Context, devicePath string) (bool, error) {
	type res struct {
		healthy bool
		err     error
	}
	results := make([]res, len(c.backends))
	var wg sync.WaitGroup
	for i, b := range c.backends {
		wg.Go(func() {
			healthy, err := b.CheckHealth(ctx, devicePath)
			results[i] = res{healthy, err}
		})
	}
	wg.Wait()

	master := results[0]
	for i, r := range results[1:] {
		name := c.backends[i+1].Name()
		if r.err != nil {
			c.logSecondaryError(ctx, "CheckHealth", name, r.err)
			continue
		}
		if master.healthy != r.healthy {
			c.logMismatch(ctx, "CheckHealth", name, master.healthy, r.healthy)
		}
	}
	return master.healthy, master.err
}

// GetDeviceInfo retrieves device info from all backends and returns the master's result.
func (c *CompareBackend) GetDeviceInfo(ctx context.Context, devicePath string) (map[string]any, error) {
	type res struct {
		info map[string]any
		err  error
	}
	results := make([]res, len(c.backends))
	var wg sync.WaitGroup
	for i, b := range c.backends {
		wg.Go(func() {
			info, err := b.GetDeviceInfo(ctx, devicePath)
			results[i] = res{info, err}
		})
	}
	wg.Wait()

	master := results[0]
	for i, r := range results[1:] {
		name := c.backends[i+1].Name()
		if r.err != nil {
			c.logSecondaryError(ctx, "GetDeviceInfo", name, r.err)
			continue
		}
		if !jsonEqual(master.info, r.info) {
			c.logMismatch(ctx, "GetDeviceInfo", name, master.info, r.info)
		}
	}
	return master.info, master.err
}

// RunSelfTest runs a self-test on all backends and returns the master's error.
func (c *CompareBackend) RunSelfTest(ctx context.Context, devicePath string, testType string) error {
	errs := make([]error, len(c.backends))
	var wg sync.WaitGroup
	for i, b := range c.backends {
		wg.Go(func() {
			errs[i] = b.RunSelfTest(ctx, devicePath, testType)
		})
	}
	wg.Wait()

	for i, err := range errs[1:] {
		name := c.backends[i+1].Name()
		switch {
		case err != nil && errs[0] == nil:
			c.logSecondaryError(ctx, "RunSelfTest", name, err)
		case err == nil && errs[0] != nil:
			c.logMismatch(ctx, "RunSelfTest", name, errs[0], err)
		}
	}
	return errs[0]
}

// GetAvailableSelfTests retrieves self-test info from all backends and returns the master's result.
func (c *CompareBackend) GetAvailableSelfTests(ctx context.Context, devicePath string) (*SelfTestInfo, error) {
	type res struct {
		info *SelfTestInfo
		err  error
	}
	results := make([]res, len(c.backends))
	var wg sync.WaitGroup
	for i, b := range c.backends {
		wg.Go(func() {
			info, err := b.GetAvailableSelfTests(ctx, devicePath)
			results[i] = res{info, err}
		})
	}
	wg.Wait()

	master := results[0]
	for i, r := range results[1:] {
		name := c.backends[i+1].Name()
		if r.err != nil {
			c.logSecondaryError(ctx, "GetAvailableSelfTests", name, r.err)
			continue
		}
		if !jsonEqual(normalizeSelfTestInfo(master.info), normalizeSelfTestInfo(r.info)) {
			c.logMismatch(ctx, "GetAvailableSelfTests", name, master.info, r.info)
		}
	}
	return master.info, master.err
}

// EnableSMART enables SMART on all backends and returns the master's error.
func (c *CompareBackend) EnableSMART(ctx context.Context, devicePath string) error {
	errs := make([]error, len(c.backends))
	var wg sync.WaitGroup
	for i, b := range c.backends {
		wg.Go(func() {
			errs[i] = b.EnableSMART(ctx, devicePath)
		})
	}
	wg.Wait()
	for i, err := range errs[1:] {
		if err != nil {
			c.logSecondaryError(ctx, "EnableSMART", c.backends[i+1].Name(), err)
		}
	}
	return errs[0]
}

// DisableSMART disables SMART on all backends and returns the master's error.
func (c *CompareBackend) DisableSMART(ctx context.Context, devicePath string) error {
	errs := make([]error, len(c.backends))
	var wg sync.WaitGroup
	for i, b := range c.backends {
		wg.Go(func() {
			errs[i] = b.DisableSMART(ctx, devicePath)
		})
	}
	wg.Wait()
	for i, err := range errs[1:] {
		if err != nil {
			c.logSecondaryError(ctx, "DisableSMART", c.backends[i+1].Name(), err)
		}
	}
	return errs[0]
}

// AbortSelfTest aborts a running self-test on all backends and returns the master's error.
func (c *CompareBackend) AbortSelfTest(ctx context.Context, devicePath string) error {
	errs := make([]error, len(c.backends))
	var wg sync.WaitGroup
	for i, b := range c.backends {
		wg.Go(func() {
			errs[i] = b.AbortSelfTest(ctx, devicePath)
		})
	}
	wg.Wait()
	for i, err := range errs[1:] {
		if err != nil {
			c.logSecondaryError(ctx, "AbortSelfTest", c.backends[i+1].Name(), err)
		}
	}
	return errs[0]
}

// DiscoverDevices discovers devices from all backends and returns the master's result.
// Backends that do not implement DiscoveryBackend fall back to ScanDevices + GetSMARTInfo.
func (c *CompareBackend) DiscoverDevices(ctx context.Context) ([]DiscoveryResult, error) {
	type res struct {
		results []DiscoveryResult
		err     error
	}
	results := make([]res, len(c.backends))
	var wg sync.WaitGroup
	for i, b := range c.backends {
		wg.Go(func() {
			if db, ok := b.(DiscoveryBackend); ok {
				r, err := db.DiscoverDevices(ctx)
				results[i] = res{r, err}
			} else {
				r, err := genericDiscoverDevices(ctx, b)
				results[i] = res{r, err}
			}
		})
	}
	wg.Wait()

	master := results[0]
	for i, r := range results[1:] {
		name := c.backends[i+1].Name()
		if r.err != nil {
			c.logSecondaryError(ctx, "DiscoverDevices", name, r.err)
			continue
		}
		if !jsonEqual(sortedDiscoveryResultsCopy(master.results), sortedDiscoveryResultsCopy(r.results)) {
			c.logMismatch(ctx, "DiscoverDevices", name, master.results, r.results)
		}
	}
	return master.results, master.err
}

// genericDiscoverDevices provides a discovery fallback for backends that do not
// implement DiscoveryBackend, using ScanDevices and GetSMARTInfo.
func genericDiscoverDevices(ctx context.Context, b Backend) ([]DiscoveryResult, error) {
	devices, err := b.ScanDevices(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]smtypes.DiscoveryResult, 0, len(devices))
	for _, dev := range devices {
		result := smtypes.DiscoveryResult{DevicePath: dev.Name, DetectedProtocol: dev.Type}
		info, infoErr := b.GetSMARTInfo(ctx, dev.Name)
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

// comparisonExcludedJSONKeys lists top-level JSON keys that are stripped from
// both sides before comparison. These are exec-backend-specific metadata fields
// absent from other backend implementations (e.g. lib backend).
var comparisonExcludedJSONKeys = []string{"smartctl"}

// jsonNormalize marshals v to JSON, then strips top-level excluded keys so that
// backend-specific metadata fields do not cause false-positive mismatches.
// Fields tagged json:"-" are also excluded by json.Marshal.
func jsonNormalize(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m any
	if err = json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if mp, ok := m.(map[string]any); ok {
		for _, key := range comparisonExcludedJSONKeys {
			delete(mp, key)
		}
	}
	return json.Marshal(m)
}

// jsonEqual reports whether a and b produce identical JSON after normalization.
// Fields tagged json:"-" and keys in comparisonExcludedJSONKeys are excluded,
// preventing false mismatches from backend-specific fields such as DiskType,
// ExitCodeInfo, and the exec-only "smartctl" metadata block.
func jsonEqual(a, b any) bool {
	aj, errA := jsonNormalize(a)
	bj, errB := jsonNormalize(b)
	if errA != nil || errB != nil {
		return false
	}
	return bytes.Equal(aj, bj)
}

// logMismatch logs a warning when a secondary backend returns a different result.
// The "diff" attribute contains only the JSON-serializable fields that actually
// differ, making it straightforward to pinpoint which data disagrees.
// It is a no-op when ctx has already been cancelled.
func (c *CompareBackend) logMismatch(ctx context.Context, method, backendName string, master, secondary any) {
	if ctx.Err() != nil {
		return
	}
	c.log.WarnContext(ctx, "compare: result mismatch",
		"method", method,
		"backend", backendName,
		"diff", jsonDiff(master, secondary),
	)
}

// jsonDiff returns a flat map of dot-notation paths → {master, secondary} pairs
// for every JSON-serializable field where master and secondary differ.
// Fields tagged json:"-" and keys in comparisonExcludedJSONKeys are excluded.
// Returns nil when the values are JSON-equal or cannot be marshaled.
func jsonDiff(master, secondary any) map[string]any {
	aj, errA := jsonNormalize(master)
	bj, errB := jsonNormalize(secondary)
	if errA != nil || errB != nil || bytes.Equal(aj, bj) {
		return nil
	}
	var am, bm any
	_ = json.Unmarshal(aj, &am)
	_ = json.Unmarshal(bj, &bm)
	diff := make(map[string]any)
	collectDiffs("", am, bm, diff)
	if len(diff) == 0 {
		return nil
	}
	return diff
}

// collectDiffs recursively walks two JSON-decoded values and populates diff
// with every leaf path where they disagree.
func collectDiffs(prefix string, a, b any, diff map[string]any) {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	if bytes.Equal(aj, bj) {
		return
	}
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)
	if aIsMap && bIsMap {
		keys := make(map[string]struct{}, len(aMap)+len(bMap))
		for k := range aMap {
			keys[k] = struct{}{}
		}
		for k := range bMap {
			keys[k] = struct{}{}
		}
		for k := range keys {
			child := k
			if prefix != "" {
				child = prefix + "." + k
			}
			collectDiffs(child, aMap[k], bMap[k], diff)
		}
		return
	}
	key := prefix
	if key == "" {
		key = "(root)"
	}
	diff[key] = map[string]any{"master": a, "secondary": b}
}

// logSecondaryError logs an error when a secondary backend fails.
// It is a no-op when ctx has already been cancelled.
func (c *CompareBackend) logSecondaryError(ctx context.Context, method, backendName string, err error) {
	if ctx.Err() != nil {
		return
	}
	c.log.ErrorContext(ctx, "compare: secondary backend error",
		"method", method,
		"backend", backendName,
		"error", err,
	)
}

// sortedDevicesCopy returns a sorted copy of devices by Name for order-independent comparison.
func sortedDevicesCopy(devices []Device) []Device {
	sorted := make([]Device, len(devices))
	copy(sorted, devices)
	slices.SortFunc(sorted, func(a, b Device) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return sorted
}

// sortedDiscoveryResultsCopy returns a sorted copy of discovery results by DevicePath.
func sortedDiscoveryResultsCopy(results []DiscoveryResult) []DiscoveryResult {
	sorted := make([]DiscoveryResult, len(results))
	copy(sorted, results)
	slices.SortFunc(sorted, func(a, b DiscoveryResult) int {
		if a.DevicePath < b.DevicePath {
			return -1
		}
		if a.DevicePath > b.DevicePath {
			return 1
		}
		return 0
	})
	return sorted
}

// normalizeSelfTestInfo returns a copy of SelfTestInfo with Available sorted,
// for order-independent comparison.
func normalizeSelfTestInfo(info *SelfTestInfo) *SelfTestInfo {
	if info == nil {
		return nil
	}
	sorted := make([]string, len(info.Available))
	copy(sorted, info.Available)
	slices.Sort(sorted)
	return &SelfTestInfo{
		Available: sorted,
		Durations: info.Durations,
	}
}
