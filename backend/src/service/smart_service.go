package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/smartmontools-sdk/bindings/go/v8"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"

	"github.com/dianlight/tlog"
	gocache "github.com/patrickmn/go-cache"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

const (
	// smartInfoCacheTTL bounds how long a per-device GetSmartInfo result
	// (including "not supported" errors) is reused. SMART probing issues raw
	// SCSI commands that can trigger kernel partition rescans and udev event
	// storms; the hardware snapshot is already served up-to-30-min stale by
	// design, so a short TTL here stays within the existing freshness envelope.
	smartInfoCacheTTL = 2 * time.Minute
	// smartInfoCacheCleanup is the janitor interval for the SMART info cache.
	smartInfoCacheCleanup = 5 * time.Minute
)

// smartInfoCacheEntry stores either a successful SmartInfo or the error that
// should be replayed for the TTL window. Negative caching matters: devices
// without SMART support are exactly the ones re-probed most often.
type smartInfoCacheEntry struct {
	info *dto.SmartInfo
	err  errors.E
}

// selfTestTrackerEntry records the last known state of a SMART self-test
// started through StartSelfTest. It exists because some backends (notably the
// lib/direct backend) omit ata_smart_data.self_test.status from GetSMARTInfo,
// leaving the drive query blind: without the tracker, GetTestStatus would
// report unknown/unknown even while a test started by us is running (#1196).
type selfTestTrackerEntry struct {
	testType        string
	running         bool
	progress        int
	status          string
	updatedAt       time.Time
	expectedMinutes int
}

// selfTestExpectedMinutes mirrors the SDK's default test durations used for
// progress polling when capability durations are unavailable.
func selfTestExpectedMinutes(testType string) int {
	switch testType {
	case "long", "extended":
		return 120
	case "conveyance":
		return 5
	case "offline":
		return 10
	default:
		return 2 // short
	}
}

// selfTestPollIntervalSecs mirrors the SDK's adaptive polling interval: at
// most 24 samples over the expected duration, clamped between 5 s and 60 s.
func selfTestPollIntervalSecs(expectedMinutes int) int {
	v := expectedMinutes * 60 / 24
	if v < 5 {
		return 5
	}
	if v > 60 {
		return 60
	}
	return v
}

// fresh reports whether a running entry is recent enough to trust: the SDK
// invokes the progress callback every poll interval, so an entry older than
// two intervals plus margin means polling stopped and the drive query (however
// blind) is all that remains.
func (e *selfTestTrackerEntry) fresh(now time.Time) bool {
	if e == nil || !e.running {
		return false
	}
	interval := time.Duration(selfTestPollIntervalSecs(e.expectedMinutes)) * time.Second
	return now.Sub(e.updatedAt) <= 2*interval+30*time.Second
}

// reportable reports whether a terminal (completed/aborted) entry is still
// worth reporting instead of falling back to the drive query.
func (e *selfTestTrackerEntry) reportable(now time.Time) bool {
	if e == nil {
		return false
	}
	if e.running {
		return e.fresh(now)
	}
	return now.Sub(e.updatedAt) <= 10*time.Minute
}

type SmartServiceInterface interface {
	GetSmartInfo(ctx context.Context, deviceId string) (*dto.SmartInfo, errors.E)
	GetSmartStatus(ctx context.Context, deviceId string) (*dto.SmartStatus, errors.E)
	GetHealthStatus(ctx context.Context, deviceId string) (*dto.SmartHealthStatus, errors.E)
	StartSelfTest(ctx context.Context, deviceId string, testType dto.SmartTestType) errors.E
	AbortSelfTest(ctx context.Context, deviceId string) errors.E
	GetTestStatus(ctx context.Context, deviceId string) (*dto.SmartTestStatus, errors.E)
	EnableSMART(ctx context.Context, deviceId string) errors.E
	DisableSMART(ctx context.Context, deviceId string) errors.E
	MockDeviceToDevice(func(string) (string, error))
	MockVerifyClient(smartmontools.SmartClient)
}

type smartService struct {
	mutex  sync.Mutex
	client smartmontools.SmartClient
	// verifyClient is an exec-backend client used to double-check suspicious
	// lib-backend results (see newVerifyClient). Nil when unavailable or when
	// the primary client already uses the exec backend.
	verifyClient     smartmontools.SmartClient
	conv             converter.SmartMonToolsToDtoImpl
	eventBus         events.EventBusInterface
	deviceIdToDevice func(string) (string, error)
	// infoCache caches GetSmartInfo results per deviceId (successes and
	// errors) to avoid re-issuing side-effecting SCSI probes on every
	// hardware re-enumeration.
	infoCache *gocache.Cache
	// testTracker records self-test progress from StartSelfTest callbacks so
	// GetTestStatus can report running tests even when the backend omits
	// self-test status from GetSMARTInfo. Guarded by testMu (never hold
	// s.mutex while holding testMu).
	testMu      sync.Mutex
	testTracker map[string]*selfTestTrackerEntry
}

type SmartServiceParams struct {
	fx.In
	// Client is optional: when provided (e.g. in tests via mock injection) it is
	// used as-is. When nil (production), NewSmartService initialises the client
	// internally by probing the lib backend first, then falling back to exec.
	Client   smartmontools.SmartClient `optional:"true"`
	ApiCtx   *dto.ContextState         `optional:"true"`
	EventBus events.EventBusInterface
}

// recordLibSmartBackendOutcome records the lib SMART backend availability and,
// when unavailable, the reason, on the runtime context. It is shared by the
// smartlib and !smartlib build variants so both report a consistent capability.
func recordLibSmartBackendOutcome(apiCtx *dto.ContextState, available bool, reason string) {
	if apiCtx == nil {
		return
	}
	apiCtx.LibSmartAvailable = available
	apiCtx.LibSmartUnavailableReason = ""
	if !available {
		apiCtx.LibSmartUnavailableReason = reason
	}
}

func NewSmartService(in SmartServiceParams) SmartServiceInterface {
	client := in.Client
	prodInit := false
	if client == nil {
		client = initSmartClient(in.ApiCtx)
		prodInit = true
	}
	return &smartService{
		client:           client,
		verifyClient:     newVerifyClient(in.ApiCtx, prodInit),
		eventBus:         in.EventBus,
		conv:             converter.SmartMonToolsToDtoImpl{},
		deviceIdToDevice: converter.DeviceIdToDevice,
		infoCache:        gocache.New(smartInfoCacheTTL, smartInfoCacheCleanup),
		testTracker:      make(map[string]*selfTestTrackerEntry),
	}
}

// newExecSmartClient builds the exec-backend verify client. It is a variable
// to allow fault injection in tests.
var newExecSmartClient = func() (smartmontools.SmartClient, error) {
	return smartmontools.NewClient(
		smartmontools.WithTLogHandler(tlog.NewLoggerWithLevel(tlog.LevelInfo)),
	)
}

// newVerifyClient builds an exec-backend client used to double-check
// suspicious lib-backend results. The bundled libsmartmon JSON schema omits
// self-test execution status and overall-health, and its CheckHealth can
// report failure on drives that `smartctl -H` passes (observed on Kingston
// SV300, #1196). The fallback is only created for production lib-backend
// deployments; it stays nil when the primary client already uses exec, when
// the backend is unknown (tests injecting a mock client), or when
// construction fails.
func newVerifyClient(apiCtx *dto.ContextState, prodInit bool) smartmontools.SmartClient {
	if !prodInit || apiCtx == nil || !apiCtx.LibSmartAvailable {
		return nil
	}
	client, err := newExecSmartClient()
	if err != nil {
		slog.Warn("SMART exec verify client unavailable; lib-backend results will not be double-checked", "error", err)
		return nil
	}
	return client
}

func (s *smartService) MockDeviceToDevice(mock func(string) (string, error)) {
	s.deviceIdToDevice = mock
}

// MockVerifyClient installs an exec-backend equivalent used to double-check
// suspicious lib-backend results. Test seam mirroring MockDeviceToDevice.
func (s *smartService) MockVerifyClient(client smartmontools.SmartClient) {
	s.verifyClient = client
}

func (s *smartService) smartInfoFromSMARTInfo(devicePath string, smartInfo *smartmontools.SMARTInfo) (*dto.SmartInfo, errors.E) {
	ret, err := s.conv.SmartMonToolsSmartInfoToSmartInfo(smartInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert SMART info for device %s", devicePath)
	}
	ret.DiskId = devicePath

	if ret.DiskType == "" {
		if smartInfo.AtaSmartData != nil {
			// ATA/SATA device
			ret.DiskType = "SATA"
		} else if smartInfo.NvmeSmartHealth != nil || smartInfo.NvmeControllerCapabilities != nil {
			// NVMe device
			ret.DiskType = "NVMe"
		}
	}

	return ret, nil
}

func (s *smartService) GetSmartInfo(ctx context.Context, deviceId string) (*dto.SmartInfo, errors.E) {
	// Cache lookup: replay a previous result (success or error) within the
	// TTL window instead of re-issuing SCSI probes.
	if s.infoCache != nil {
		if cached, ok := s.infoCache.Get(deviceId); ok {
			if entry, castOk := cached.(smartInfoCacheEntry); castOk {
				tlog.DebugContext(ctx, "Returning SMART info from cache", "device", deviceId)
				return entry.info, entry.err
			}
			s.infoCache.Delete(deviceId)
		}
	}

	info, err := s.getSmartInfoUncached(ctx, deviceId)

	if s.infoCache != nil {
		s.infoCache.SetDefault(deviceId, smartInfoCacheEntry{info: info, err: err})
	}
	return info, err
}

func (s *smartService) getSmartInfoUncached(ctx context.Context, deviceId string) (*dto.SmartInfo, errors.E) {
	devicePath, err := s.deviceIdToDevice(deviceId)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.WithDetails(dto.ErrorNotFound, "device", deviceId)
		}
		return nil, errors.Wrapf(err, "failed to resolve device path for device ID %s", deviceId)
	}

	// Check if client is available
	if s.client == nil {
		return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "smartctl not available")
	}

	// Get SMART information using the smartmontools bindings
	smartInfo, err := s.client.GetSMARTInfo(ctx, devicePath)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "No such device") || strings.Contains(err.Error(), "SMART Not Supported") {
			return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", err.Error())
		}
		return nil, errors.Errorf("failed to get SMART info for device %s %w", devicePath, err)
	}

	smartInfoDto, errE := s.smartInfoFromSMARTInfo(devicePath, smartInfo)
	if errE != nil {
		return nil, errE
	}
	// Override DiskId: smartInfoFromSMARTInfo sets it to the raw device path,
	// but callers expect it to be the canonical deviceId.
	smartInfoDto.DiskId = deviceId
	return smartInfoDto, nil
}

// invalidateSmartInfoCache drops the cached GetSmartInfo entry for a device,
// used after SMART enable/disable changes device state.
func (s *smartService) invalidateSmartInfoCache(deviceId string) {
	if s.infoCache != nil {
		s.infoCache.Delete(deviceId)
	}
}

// getTestTracker returns a copy of the tracked self-test state for a device,
// or false when nothing was ever tracked.
func (s *smartService) getTestTracker(deviceId string) (selfTestTrackerEntry, bool) {
	s.testMu.Lock()
	defer s.testMu.Unlock()
	if s.testTracker == nil {
		return selfTestTrackerEntry{}, false
	}
	entry, ok := s.testTracker[deviceId]
	if !ok || entry == nil {
		return selfTestTrackerEntry{}, false
	}
	return *entry, true
}

// setTestTracker replaces the tracked self-test state for a device.
func (s *smartService) setTestTracker(deviceId string, entry selfTestTrackerEntry) {
	s.testMu.Lock()
	defer s.testMu.Unlock()
	if s.testTracker == nil {
		s.testTracker = make(map[string]*selfTestTrackerEntry)
	}
	cp := entry
	s.testTracker[deviceId] = &cp
}

// GetSmartStatus returns dynamic SMART status data for a device
func (s *smartService) GetSmartStatus(ctx context.Context, deviceId string) (*dto.SmartStatus, errors.E) {

	devicePath, err := s.deviceIdToDevice(deviceId)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve device path for device ID %s", deviceId)
	}

	// Check if client is available
	if s.client == nil {
		return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", deviceId, "reason", "smartctl not available")
	}

	// Get SMART information using the smartmontools bindings
	smartInfo, err := s.client.GetSMARTInfo(ctx, devicePath)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "No such device") || strings.Contains(err.Error(), "SMART Not Supported") {
			return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", err.Error())
		}
		return nil, errors.Wrapf(err, "failed to get SMART status for device %s", devicePath)
	}

	if smartInfo.SmartSupport != nil && !smartInfo.SmartSupport.Available {
		return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath)
	}

	ret, err := s.conv.SmartMonToolsSmartInfoToSmartStatus(smartInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert SMART status for device %s", devicePath)
	}

	// The lib backend emits power_on_time.hours as a 64-bit packed value where
	// the low 32 bits hold the hours and the high 32 bits carry a sub-hour
	// counter (observed: 0x9b8a0000a587 → 42375h). Normalize it so the packed
	// value never reaches the UI; plausible plain values pass through untouched.
	if ret.PowerOnHours.Value > maxPlausibleSmartValue {
		if v := ret.PowerOnHours.Value & 0xFFFFFFFF; v > 0 && v < maxPlausibleSmartValue {
			ret.PowerOnHours.Value = v
		} else {
			ret.PowerOnHours.Value = 0
		}
	}

	// Process based on device type
	if smartInfo.AtaSmartData != nil {
		// ATA/SATA device - process SMART attributes
		if smartInfo.AtaSmartData.Table != nil {
			others := make(map[string]dto.SmartRangeValue)

			for _, attr := range smartInfo.AtaSmartData.Table {
				switch attr.ID {
				case dto.SmartAttributeCodes.SMARTATTRTEMPERATURECELSIUS.Code:
					// Temperature attribute. The converter already set
					// ret.Temperature.Value from the top-level smartctl
					// `temperature` object (protocol-independent °C). The ATA
					// attr `value` is a normalized 0-253 health score (often
					// 100/121), NOT the temperature, so it must not override
					// the Celsius value. The raw string ("51 (Min/Max -22/57)")
					// is only used when the top-level temperature is missing.
					if ret.Temperature.Value == 0 {
						ret.Temperature.Value = rawPlausibleInt(attr.Raw.String)
					}
				case dto.SmartAttributeCodes.SMARTATTRPOWERCYCLECOUNT.Code:
					// Power cycle count. Prefer the value already set by the
					// converter (smartctl `power_cycle_count`); only override it
					// with the raw string when it is a plausible plain count.
					ret.PowerCycleCount.Code = attr.ID
					ret.PowerCycleCount.Worst = attr.Worst
					ret.PowerCycleCount.Thresholds = attr.Thresh
					if count := rawPlausibleInt(attr.Raw.String); count > 0 {
						ret.PowerCycleCount.Value = count
					}
				case dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS.Code:
					// Power on hours. The raw 48-bit integer packs hours + msec
					// (e.g. "42374h+52m+33.990s"); parse the leading integer from
					// the raw string only when plausible, otherwise keep the value
					// set by the converter (smartctl `power_on_time.hours`).
					ret.PowerOnHours.Code = attr.ID
					ret.PowerOnHours.Worst = attr.Worst
					ret.PowerOnHours.Thresholds = attr.Thresh
					if hours := rawPlausibleInt(attr.Raw.String); hours > 0 {
						ret.PowerOnHours.Value = hours
					}
				default:
					// Other dynamic attributes
					if attr.Name != "" {
						others[attr.Name] = dto.SmartRangeValue{
							Code:       attr.ID,
							Value:      attr.Value,
							Worst:      attr.Worst,
							Thresholds: attr.Thresh,
						}
					}
				}
			}

			if len(others) > 0 {
				ret.Additional = others
			}

		}
	} else if smartInfo.NvmeSmartHealth != nil {
		// NVMe device
		// Extract NVMe-specific dynamic data
		if smartInfo.NvmeSmartHealth.Temperature > 0 {
			ret.Temperature.Value = smartInfo.NvmeSmartHealth.Temperature
		}
		if smartInfo.NvmeSmartHealth.WarningTempTime > 0 {
			ret.Temperature.OvertempCounter = smartInfo.NvmeSmartHealth.WarningTempTime
		}
		if smartInfo.NvmeSmartHealth.PowerOnHours > 0 {
			ret.PowerOnHours.Value = int(smartInfo.NvmeSmartHealth.PowerOnHours)
		}
		if smartInfo.NvmeSmartHealth.PowerCycles > 0 {
			ret.PowerCycleCount.Value = int(smartInfo.NvmeSmartHealth.PowerCycles)
		}

		// Add NVMe-specific dynamic attributes
		others := make(map[string]dto.SmartRangeValue)
		others["AvailableSpare"] = dto.SmartRangeValue{
			Value:      smartInfo.NvmeSmartHealth.AvailableSpare,
			Thresholds: smartInfo.NvmeSmartHealth.AvailableSpareThresh,
		}
		others["PercentageUsed"] = dto.SmartRangeValue{
			Value: smartInfo.NvmeSmartHealth.PercentageUsed,
		}
		others["CriticalWarning"] = dto.SmartRangeValue{
			Value: smartInfo.NvmeSmartHealth.CriticalWarning,
		}
		ret.Additional = others

	}

	return ret, nil
}

// GetHealthStatus returns the health status of a device by evaluating SMART attributes
func (s *smartService) GetHealthStatus(ctx context.Context, deviceId string) (*dto.SmartHealthStatus, errors.E) {
	// Check if client is available
	if s.client == nil {
		return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", deviceId, "reason", "smartctl not available")
	}

	devicePath, errS := s.deviceIdToDevice(deviceId)
	if errS != nil {
		return nil, errors.Wrapf(errS, "failed to resolve device path for device ID %s", deviceId)
	}

	// Get SMART status first (may return cached data)
	smartStatus, err := s.GetSmartStatus(ctx, deviceId)
	if err != nil {
		if errors.Is(err, dto.ErrorSMARTNotSupported) {
			return &dto.SmartHealthStatus{
				Passed:        false,
				OverallStatus: "unknown",
			}, nil
		}
		return nil, err
	}

	// Check if SMART is enabled
	if !smartStatus.Enabled {
		tlog.WarnContext(ctx, "SMART is not enabled on device", "device", deviceId, "status", smartStatus)
		return &dto.SmartHealthStatus{
			Passed:            false,
			OverallStatus:     "warning",
			FailingAttributes: []string{"SMART_not_enabled"},
		}, nil
	}

	// Use the smartmontools bindings to check health
	healthy, stdErr := s.client.CheckHealth(ctx, devicePath)
	if stdErr != nil {
		tlog.Warn("failed to check health status", "device", devicePath, "error", stdErr)
	}

	health := checkSMARTHealth(smartStatus, nil, nil)

	// Override with smartctl health check result
	if stdErr == nil {
		health.Passed = healthy
		if healthy {
			health.OverallStatus = "healthy"
		} else if health.OverallStatus == "healthy" {
			health.OverallStatus = "failing"
		}
	}

	// Reconcile with the drive-reported overall-health (smart_status.passed,
	// the same source as `smartctl -H`): the lib backend's CheckHealth can
	// report failure while the drive reports PASSED. With zero failing
	// attributes there is no evidence of degradation, so the drive's own
	// assessment wins instead of flipping a healthy drive to failing (#1196).
	if !health.Passed && len(health.FailingAttributes) == 0 && smartStatus.IsTestPassed {
		tlog.WarnContext(ctx, "SMART CheckHealth disagrees with device overall-health; trusting device PASSED",
			"device", devicePath)
		health.Passed = true
		health.OverallStatus = "healthy"
	}

	// Double-check via the exec backend (`smartctl -H` semantics) when the
	// lib backend reports failure without evidence: its CheckHealth can false
	// alarm on drives smartctl passes, and its JSON omits smart_status, so
	// the reconciliation above cannot see the drive's own assessment (#1196).
	if !health.Passed && len(health.FailingAttributes) == 0 && s.verifyClient != nil {
		if verified, verr := s.verifyClient.CheckHealth(ctx, devicePath); verr != nil {
			tlog.WarnContext(ctx, "SMART verify CheckHealth failed; keeping primary result",
				"device", devicePath, "error", verr)
		} else if verified {
			tlog.WarnContext(ctx, "SMART lib CheckHealth disagreed with smartctl; trusting smartctl PASSED",
				"device", devicePath)
			health.Passed = true
			health.OverallStatus = "healthy"
		}
	}

	// Check if device is about to fail and trigger callback
	if !health.Passed {
		if len(health.FailingAttributes) > 0 {
			tlog.Warn("SMART pre-failure detected", "device", devicePath,
				"failing_attributes", health.FailingAttributes)
		} else {
			tlog.Warn("SMART health check failed without failing attributes", "device", devicePath,
				"overall_status", health.OverallStatus)
		}
	}

	return health, nil
}

// checkSMARTHealth evaluates SMART attributes to determine disk health
func checkSMARTHealth(smartStatus *dto.SmartStatus, _ any, _ any) *dto.SmartHealthStatus {
	health := &dto.SmartHealthStatus{
		Passed:        true,
		OverallStatus: "healthy",
	}

	failingAttrs := []string{}

	// Check if any critical attributes are below threshold
	for code, attr := range smartStatus.Additional {
		if attr.Thresholds > 0 && attr.Value < attr.Thresholds {
			failingAttrs = append(failingAttrs, code)
			health.Passed = false
		}
	}

	// Check power cycle count threshold
	if smartStatus.PowerCycleCount.Thresholds > 0 &&
		smartStatus.PowerCycleCount.Value < smartStatus.PowerCycleCount.Thresholds {
		failingAttrs = append(failingAttrs, "PowerCycleCount")
		health.Passed = false
	}

	// Check power on hours threshold
	if smartStatus.PowerOnHours.Thresholds > 0 &&
		smartStatus.PowerOnHours.Value < smartStatus.PowerOnHours.Thresholds {
		failingAttrs = append(failingAttrs, "PowerOnHours")
		health.Passed = false
	}

	if len(failingAttrs) > 0 {
		health.FailingAttributes = failingAttrs
		health.OverallStatus = "failing"
	}

	return health
}

// StartSelfTest initiates a SMART self-test on the device
func (s *smartService) StartSelfTest(ctx context.Context, deviceId string, testType dto.SmartTestType) errors.E {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !testType.IsValid() {
		return errors.WithDetails(dto.ErrorInvalidParameter, "test_type", testType)
	}

	// Check if client is available
	if s.client == nil {
		return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", deviceId, "reason", "smartctl not available")
	}

	devicePath, err := s.deviceIdToDevice(deviceId)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve device path for device ID %s", deviceId)
	}

	// Refuse a second start while the tracker reports a running test. The API
	// layer maps ErrorSMARTTestInProgress to 422.
	if tracked, ok := s.getTestTracker(deviceId); ok && tracked.fresh(time.Now()) {
		return errors.WithDetails(dto.ErrorSMARTTestInProgress, "device", devicePath,
			"test_type", tracked.testType)
	}

	expectedMinutes := selfTestExpectedMinutes(testType.String())
	// Start the self-test using the smartmontools bindings. The SDK reports
	// progress asynchronously: completion is emitted from the callback below,
	// never synchronously, so callers observe Running=true until the test
	// actually finishes (#1196).
	if err := s.client.RunSelfTestWithProgress(ctx, devicePath, testType.String(), func(progress int, status string) {
		running := progress < 100 && !strings.Contains(strings.ToLower(status), "cancel")
		if progress > 100 {
			progress = 100
		}
		s.setTestTracker(deviceId, selfTestTrackerEntry{
			testType:        testType.String(),
			running:         running,
			progress:        progress,
			status:          status,
			updatedAt:       time.Now(),
			expectedMinutes: expectedMinutes,
		})
		s.eventBus.EmitSmart(events.SmartEvent{
			Type: events.EventTypes.UPDATE,
			SmartTestStatus: dto.SmartTestStatus{
				TestType:        testType.String(),
				Running:         running,
				DiskId:          deviceId,
				Status:          status,
				PercentComplete: progress,
			},
		})
	}); err != nil {
		if strings.Contains(err.Error(), "not supported") {
			return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath,
				"reason", "self-test not supported")
		}
		return errors.Wrapf(err, "failed to start SMART self-test")
	}

	slog.DebugContext(ctx, "SMART self-test started", "device", devicePath, "type", testType)
	return nil
}

// AbortSelfTest aborts the currently running SMART self-test on the device
func (s *smartService) AbortSelfTest(ctx context.Context, deviceId string) errors.E {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if client is available
	if s.client == nil {
		return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", deviceId, "reason", "smartctl not available")
	}

	devicePath, err := s.deviceIdToDevice(deviceId)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve device path for device ID %s", deviceId)
	}

	// Abort the self-test using the smartmontools bindings
	if err := s.client.AbortSelfTest(ctx, devicePath); err != nil {
		if strings.Contains(err.Error(), "not supported") {
			return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath,
				"reason", "self-test abort not supported")
		}
		return errors.Wrapf(err, "failed to abort SMART self-test")
	}

	// Record the abort so GetTestStatus reports Running=false even when the
	// backend omits self-test status from GetSMARTInfo. The previous test
	// type is preserved when known.
	testType := ""
	if tracked, ok := s.getTestTracker(deviceId); ok {
		testType = tracked.testType
	}
	s.setTestTracker(deviceId, selfTestTrackerEntry{
		testType:        testType,
		running:         false,
		status:          "aborted by user",
		updatedAt:       time.Now(),
		expectedMinutes: selfTestExpectedMinutes(testType),
	})

	slog.DebugContext(ctx, "SMART self-test aborted", "device", devicePath)
	return nil
}

// GetTestStatus returns the status of the currently running or last SMART self-test
func (s *smartService) GetTestStatus(ctx context.Context, deviceId string) (*dto.SmartTestStatus, errors.E) {
	// Check if client is available
	if s.client == nil {
		return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", deviceId, "reason", "smartctl not available")
	}

	devicePath, err := s.deviceIdToDevice(deviceId)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve device path for device ID %s", deviceId)
	}

	// Get SMART info which includes self-test status
	smartInfo, err := s.client.GetSMARTInfo(ctx, devicePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get SMART info")
	}

	// Parse self-test status from ATA SMART data
	if smartInfo.AtaSmartData != nil && smartInfo.AtaSmartData.SelfTest != nil {
		if st := smartInfo.AtaSmartData.SelfTest.Status; st != nil {
			return parseAtaSelfTestStatus(deviceId, st.String), nil
		}
		// Some backends (notably lib/direct) report self-test capabilities
		// but omit the execution status. Ask the exec verify client for
		// ground truth before falling back to tracked progress (#1196).
		if s.verifyClient != nil {
			if vinfo, verr := s.verifyClient.GetSMARTInfo(ctx, devicePath); verr != nil {
				tlog.DebugContext(ctx, "SMART verify GetSMARTInfo failed; using tracked progress", "device", devicePath, "error", verr)
			} else if vinfo.AtaSmartData != nil && vinfo.AtaSmartData.SelfTest != nil &&
				vinfo.AtaSmartData.SelfTest.Status != nil {
				return parseAtaSelfTestStatus(deviceId, vinfo.AtaSmartData.SelfTest.Status.String), nil
			}
		}
		if tracked, ok := s.testStatusFromTracker(deviceId); ok {
			return tracked, nil
		}
		return &dto.SmartTestStatus{
			Running:  false,
			DiskId:   deviceId,
			Status:   "unknown",
			TestType: "unknown",
		}, nil
	}

	// Consult the tracker before reporting a default: a test started through
	// this service may be running even when the backend exposes no self-test
	// data at all (non-ATA devices on blind backends).
	if tracked, ok := s.testStatusFromTracker(deviceId); ok {
		return tracked, nil
	}

	// Return a default status if no self-test info is available
	return &dto.SmartTestStatus{
		Status:   "idle",
		TestType: "none",
	}, nil
}

// parseAtaSelfTestStatus converts a raw ATA self-test execution status string
// into a SmartTestStatus, detecting in-progress runs and the test type.
func parseAtaSelfTestStatus(deviceId string, statusStr string) *dto.SmartTestStatus {
	status := &dto.SmartTestStatus{
		Running:  false,
		DiskId:   deviceId,
		Status:   statusStr,
		TestType: "unknown",
	}
	ls := strings.ToLower(statusStr)
	// Detect a test currently in progress and parse the remaining percentage.
	// smartctl reports: "Self test routine in progress; N% remaining." or
	// similar; the smartmontools bindings normalise this to a lowercase
	// "in progress, N% remaining" form.
	if strings.Contains(ls, "in progress") {
		status.Running = true
		// Parse "N% remaining" → PercentComplete = 100 - N
		if idx := strings.Index(ls, "%"); idx > 0 {
			start := idx - 1
			for start > 0 && ls[start-1] >= '0' && ls[start-1] <= '9' {
				start--
			}
			if remaining := ls[start:idx]; remaining != "" {
				var pct int
				if _, scanErr := fmt.Sscanf(remaining, "%d", &pct); scanErr == nil {
					status.PercentComplete = 100 - pct
				}
			}
		}
	}
	// Determine test type if available from status string
	if strings.Contains(ls, "short") {
		status.TestType = "short"
	} else if strings.Contains(ls, "long") || strings.Contains(ls, "extended") {
		status.TestType = "long"
	} else if strings.Contains(ls, "conveyance") {
		status.TestType = "conveyance"
	}
	return status
}

// testStatusFromTracker reports the tracked self-test state for a device when
// it is still worth reporting (fresh running entry or recent terminal entry).
func (s *smartService) testStatusFromTracker(deviceId string) (*dto.SmartTestStatus, bool) {
	tracked, ok := s.getTestTracker(deviceId)
	if !ok || !tracked.reportable(time.Now()) {
		return nil, false
	}
	return &dto.SmartTestStatus{
		DiskId:          deviceId,
		Running:         tracked.running,
		Status:          tracked.status,
		TestType:        tracked.testType,
		PercentComplete: tracked.progress,
	}, true
}

// EnableSMART enables SMART functionality on the device
func (s *smartService) EnableSMART(ctx context.Context, deviceId string) errors.E {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if client is available
	if s.client == nil {
		return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", deviceId, "reason", "smartctl not available")
	}

	devicePath, err := s.deviceIdToDevice(deviceId)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve device path for device ID %s", deviceId)
	}

	// Enable SMART using the smartmontools bindings
	if err := s.client.EnableSMART(ctx, devicePath); err != nil {
		return errors.Wrapf(err, "failed to enable SMART")
	}
	s.invalidateSmartInfoCache(deviceId)

	// Verify SMART is now enabled (one-off check)
	supportInfo, err := s.client.IsSMARTSupported(ctx, devicePath)
	if err != nil {
		tlog.WarnContext(ctx, "SMART enabled but verification failed", "device", devicePath, "error", err)
		return errors.Wrap(err, "SMART enable succeeded but verification failed")
	}

	if !supportInfo.Enabled {
		return errors.WithDetails(dto.ErrorSMARTOperationFailed, "device", devicePath,
			"reason", "SMART enable command executed but device reports disabled")
	}

	slog.DebugContext(ctx, "SMART enabled and verified", "device", devicePath)

	smartInfo, err := s.client.GetSMARTInfo(ctx, devicePath)
	if err != nil {
		return errors.Wrapf(err, "failed to get SMART info after enabling SMART")
	}

	smartInfoDto, errE := s.smartInfoFromSMARTInfo(devicePath, smartInfo)
	if errE != nil {
		return errE
	}
	// Override DiskId: smartInfoFromSMARTInfo sets it to the raw device path,
	// but the DiskMap is indexed by the canonical deviceId.
	smartInfoDto.DiskId = deviceId

	s.eventBus.EmitSmart(events.SmartEvent{
		Type:      events.EventTypes.UPDATE,
		SmartInfo: *smartInfoDto,
	})

	return nil
}

// DisableSMART disables SMART functionality on the device
func (s *smartService) DisableSMART(ctx context.Context, deviceId string) errors.E {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if client is available
	if s.client == nil {
		return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", deviceId, "reason", "smartctl not available")
	}

	devicePath, err := s.deviceIdToDevice(deviceId)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve device path for device ID %s", deviceId)
	}

	// Disable SMART using the smartmontools bindings
	if err := s.client.DisableSMART(ctx, devicePath); err != nil {
		return errors.Wrapf(err, "failed to disable SMART")
	}
	// Capture the last known full info before invalidation so a post-disable
	// read failure can still emit a disabled event with model details intact.
	var prevInfo *dto.SmartInfo
	if s.infoCache != nil {
		if cached, ok := s.infoCache.Get(deviceId); ok {
			if entry, castOk := cached.(smartInfoCacheEntry); castOk && entry.info != nil && entry.err == nil {
				cp := *entry.info
				prevInfo = &cp
			}
		}
	}
	s.invalidateSmartInfoCache(deviceId)

	// Verify SMART is now disabled (optional, for informational purposes)
	supportInfo, err := s.client.IsSMARTSupported(ctx, devicePath)
	if err != nil {
		tlog.WarnContext(ctx, "SMART disabled but verification failed", "device", devicePath, "error", err)
	} else if supportInfo.Enabled {
		tlog.WarnContext(ctx, "SMART disable command executed but device still reports enabled", "device", devicePath)
	}

	slog.DebugContext(ctx, "SMART disabled", "device", devicePath)

	// Best-effort refresh: a disabled device is expected to fail GetSMARTInfo,
	// and that must not turn a successful disable into an error. Emit a
	// disabled-state event preserving the last known model details so disk
	// caches observe the new state without a probe and without losing info.
	smartInfo, err := s.client.GetSMARTInfo(ctx, devicePath)
	if err != nil {
		tlog.WarnContext(ctx, "SMART disabled but post-disable info read failed (expected when disabled)", "device", devicePath, "error", err)
		if prevInfo != nil {
			prevInfo.DiskId = deviceId
			prevInfo.Enabled = false
			if supportInfo != nil {
				prevInfo.Supported = supportInfo.Available
			}
			s.eventBus.EmitSmart(events.SmartEvent{
				Type:      events.EventTypes.UPDATE,
				SmartInfo: *prevInfo,
			})
		} else if supportInfo != nil {
			s.eventBus.EmitSmart(events.SmartEvent{
				Type: events.EventTypes.UPDATE,
				SmartInfo: dto.SmartInfo{
					DiskId:    deviceId,
					Supported: supportInfo.Available,
					Enabled:   false,
				},
			})
		}
		slog.DebugContext(ctx, "SMART disabled", "device", devicePath)
		return nil
	}

	smartInfoDto, errE := s.smartInfoFromSMARTInfo(devicePath, smartInfo)
	if errE != nil {
		return errE
	}
	// Override DiskId: smartInfoFromSMARTInfo sets it to the raw device path,
	// but the DiskMap is indexed by the canonical deviceId.
	smartInfoDto.DiskId = deviceId

	s.eventBus.EmitSmart(events.SmartEvent{
		Type:      events.EventTypes.UPDATE,
		SmartInfo: *smartInfoDto,
	})

	return nil
}

// rawStringLeadingInt extracts the leading unsigned integer from a smartctl raw
// attribute string, e.g. "42374h+52m+33.990s" → 42374, "51 (Min/Max -22/57)" → 51,
// "67" → 67. Returns 0 when no leading integer is present.
func rawStringLeadingInt(raw string) int {
	trimmed := strings.TrimSpace(raw)
	i := 0
	for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0
	}
	n, err := strconv.Atoi(trimmed[:i])
	if err != nil {
		return 0
	}
	return n
}

// maxPlausibleSmartValue bounds any SMART counter a real drive can expose
// (power-on-hours, power-cycle count, temperature in °C). Firmware-packed
// 48-bit raw values and 64-bit packed lib values are always far above this.
const maxPlausibleSmartValue = 10_000_000

// rawPlausibleInt is rawStringLeadingInt with a sanity bound: firmware-packed
// 48-bit raw values (e.g. power-on-hours + msec rendered as a bare decimal by
// the lib backend) are rejected so they never replace the properly parsed
// top-level smartctl values.
func rawPlausibleInt(raw string) int {
	n := rawStringLeadingInt(raw)
	if n > 0 && n < maxPlausibleSmartValue {
		return n
	}
	return 0
}
