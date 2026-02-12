package service

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/dianlight/smartmontools-go"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"

	"github.com/dianlight/tlog"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

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
}

type smartService struct {
	mutex            sync.Mutex
	client           smartmontools.SmartClient
	conv             converter.SmartMonToolsToDtoImpl
	eventBus         events.EventBusInterface
	deviceIdToDevice func(string) (string, error)
}

type SmartServiceParams struct {
	fx.In
	Client   smartmontools.SmartClient `optional:"true"`
	EventBus events.EventBusInterface
}

func NewSmartService(in SmartServiceParams) SmartServiceInterface {
	return &smartService{
		client:           in.Client,
		eventBus:         in.EventBus,
		conv:             converter.SmartMonToolsToDtoImpl{},
		deviceIdToDevice: converter.DeviceIdToDevice,
	}
}

func (s *smartService) MockDeviceToDevice(mock func(string) (string, error)) {
	s.deviceIdToDevice = mock
}

func (s *smartService) GetSmartInfo(ctx context.Context, deviceId string) (*dto.SmartInfo, errors.E) {

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

	// Get SMART information using smartmontools-go
	smartInfo, err := s.client.GetSMARTInfo(ctx, devicePath)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "No such device") || strings.Contains(err.Error(), "SMART Not Supported") {
			return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", err.Error())
		}
		return nil, errors.Errorf("failed to get SMART info for device %s %w", devicePath, err)
	}

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

	// Get SMART information using smartmontools-go
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

	// Process based on device type
	if smartInfo.AtaSmartData != nil {
		// ATA/SATA device - process SMART attributes
		if smartInfo.AtaSmartData.Table != nil {
			others := make(map[string]dto.SmartRangeValue)

			for _, attr := range smartInfo.AtaSmartData.Table {
				switch attr.ID {
				case dto.SmartAttributeCodes.SMARTATTRTEMPERATURECELSIUS.Code:
					// Temperature attribute
					ret.Temperature.Value = attr.Value
					if attr.Raw.Value > 0 {
						ret.Temperature.Value = int(attr.Raw.Value)
					}
				case dto.SmartAttributeCodes.SMARTATTRPOWERCYCLECOUNT.Code:
					// Power cycle count
					ret.PowerCycleCount.Code = attr.ID
					ret.PowerCycleCount.Value = attr.Value
					ret.PowerCycleCount.Worst = attr.Worst
					ret.PowerCycleCount.Thresholds = attr.Thresh
					if attr.Raw.Value > 0 {
						ret.PowerCycleCount.Value = int(attr.Raw.Value)
					}
				case dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS.Code:
					// Power on hours
					ret.PowerOnHours.Code = attr.ID
					ret.PowerOnHours.Value = attr.Value
					ret.PowerOnHours.Worst = attr.Worst
					ret.PowerOnHours.Thresholds = attr.Thresh
					if attr.Raw.Value > 0 {
						ret.PowerOnHours.Value = int(attr.Raw.Value)
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

	// Use smartmontools-go to check health
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

	// Check if device is about to fail and trigger callback
	if !health.Passed {
		tlog.Warn("SMART pre-failure detected", "device", devicePath,
			"failing_attributes", health.FailingAttributes)
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

	// Start the self-test using smartmontools-go
	if err := s.client.RunSelfTestWithProgress(ctx, devicePath, testType.String(), func(progress int, status string) {
		s.eventBus.EmitSmart(events.SmartEvent{
			Event: events.Event{
				Type: events.EventTypes.UPDATE,
			},
			SmartTestStatus: dto.SmartTestStatus{
				TestType:        testType.String(),
				Running:         true,
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

	// Abort the self-test using smartmontools-go
	if err := s.client.AbortSelfTest(ctx, devicePath); err != nil {
		if strings.Contains(err.Error(), "not supported") {
			return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath,
				"reason", "self-test abort not supported")
		}
		return errors.Wrapf(err, "failed to abort SMART self-test")
	}

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
		st := smartInfo.AtaSmartData.SelfTest.Status

		status := &dto.SmartTestStatus{
			Running:  false,
			DiskId:   deviceId,
			Status:   "unknown",
			TestType: "unknown",
		}

		if st != nil {
			status.Status = st.String
			ls := strings.ToLower(st.String)
			// Determine test type if available from status string
			if strings.Contains(ls, "short") {
				status.TestType = "short"
			} else if strings.Contains(ls, "long") || strings.Contains(ls, "extended") {
				status.TestType = "long"
			} else if strings.Contains(ls, "conveyance") {
				status.TestType = "conveyance"
			}
		}

		return status, nil
	}

	// Return a default status if no self-test info is available
	return &dto.SmartTestStatus{
		Status:   "idle",
		TestType: "none",
	}, nil
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

	// Enable SMART using smartmontools-go
	if err := s.client.EnableSMART(ctx, devicePath); err != nil {
		return errors.Wrapf(err, "failed to enable SMART")
	}

	// Verify SMART is now enabled
	supportInfo, err := s.client.IsSMARTSupported(ctx, devicePath)
	if err != nil {
		tlog.Warn("SMART enabled but verification failed", "device", devicePath, "error", err)
		return errors.Wrap(err, "SMART enable succeeded but verification failed")
	}

	if !supportInfo.Enabled {
		return errors.WithDetails(dto.ErrorSMARTOperationFailed, "device", devicePath,
			"reason", "SMART enable command executed but device reports disabled")
	}

	slog.DebugContext(ctx, "SMART enabled and verified", "device", devicePath)

	smartInfo, err := s.GetSmartInfo(ctx, deviceId)
	if err != nil {
		return errors.Wrapf(err, "failed to get SMART info after enabling SMART")
	}

	s.eventBus.EmitSmart(events.SmartEvent{
		Event: events.Event{
			Type: events.EventTypes.UPDATE,
		},
		SmartInfo: *smartInfo,
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

	// Disable SMART using smartmontools-go
	if err := s.client.DisableSMART(ctx, devicePath); err != nil {
		return errors.Wrapf(err, "failed to disable SMART")
	}

	// Verify SMART is now disabled (optional, for informational purposes)
	supportInfo, err := s.client.IsSMARTSupported(ctx, devicePath)
	if err != nil {
		tlog.WarnContext(ctx, "SMART disabled but verification failed", "device", devicePath, "error", err)
	} else if supportInfo.Enabled {
		tlog.WarnContext(ctx, "SMART disable command executed but device still reports enabled", "device", devicePath)
	}

	slog.DebugContext(ctx, "SMART disabled", "device", devicePath)

	smartInfo, err := s.GetSmartInfo(ctx, deviceId)
	if err != nil {
		return errors.Errorf("failed to get SMART info after disabling SMART %w", err)
	}

	s.eventBus.EmitSmart(events.SmartEvent{
		Event: events.Event{
			Type: events.EventTypes.UPDATE,
		},
		SmartInfo: *smartInfo,
	})

	return nil
}
