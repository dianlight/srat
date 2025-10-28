package service

import (
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/smartmontools-go"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/tlog"
	gocache "github.com/patrickmn/go-cache"
	"gitlab.com/tozd/go/errors"
)

const (
	smartCacheKeyPrefix = "smart_"
	smartCacheExpiry    = 5 * time.Minute
	smartCacheCleanup   = 10 * time.Minute
)

type SmartServiceInterface interface {
	GetSmartInfo(deviceName string) (*dto.SmartInfo, errors.E)
	GetHealthStatus(devicePath string) (*dto.SmartHealthStatus, errors.E)
	StartSelfTest(devicePath string, testType dto.SmartTestType) errors.E
	AbortSelfTest(devicePath string) errors.E
	GetTestStatus(devicePath string) (*dto.SmartTestStatus, errors.E)
	EnableSMART(devicePath string) errors.E
	DisableSMART(devicePath string) errors.E
}

type smartService struct {
	cache  *gocache.Cache
	mutex  sync.Mutex
	client smartmontools.SmartClient
}

func NewSmartService() SmartServiceInterface {
	client, err := smartmontools.NewClient()
	if err != nil {
		// Fall back to a nil client if smartctl is not available
		// This allows the service to start but operations will fail gracefully
		slog.Warn("Failed to initialize smartmontools client", "error", err)
	}
	return &smartService{
		cache:  gocache.New(smartCacheExpiry, smartCacheCleanup),
		client: client,
	}
}

// checkDeviceExists verifies that the device path exists and is readable
func checkDeviceExists(devicePath string) errors.E {
	file, err := os.OpenFile(devicePath, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "does not exist")
		}
		if os.IsPermission(err) {
			return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "not readable")
		}
		return errors.Wrapf(err, "error checking device path '%s'", devicePath)
	}
	file.Close()
	return nil
}

func (s *smartService) GetSmartInfo(devicePath string) (*dto.SmartInfo, errors.E) {
	cacheKey := smartCacheKeyPrefix + devicePath
	// Try to get from cache first
	if cachedInfo, found := s.cache.Get(cacheKey); found {
		if info, ok := cachedInfo.(*dto.SmartInfo); ok {
			return info, nil
		}
	}

	// If not in cache, acquire lock to fetch
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Re-check cache after acquiring lock (another goroutine might have populated it)
	if cachedInfo, found := s.cache.Get(cacheKey); found {
		if info, ok := cachedInfo.(*dto.SmartInfo); ok {
			return info, nil
		}
	}

	// Check if the device exists before attempting to query it
	if err := checkDeviceExists(devicePath); err != nil {
		return nil, err
	}

	// Check if client is available
	if s.client == nil {
		return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "smartctl not available")
	}

	// Get SMART information using smartmontools-go
	smartInfo, err := s.client.GetSMARTInfo(devicePath)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "No such device") {
			return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "device not found")
		}
		return nil, errors.Wrapf(err, "failed to get SMART info for device %s", devicePath)
	}

	// Check if SMART is supported and enabled
	smartEnabled := false
	if smartInfo.SmartSupport != nil {
		smartEnabled = smartInfo.SmartSupport.Enabled
	}

	// Initialize the return structure
	ret := &dto.SmartInfo{
		Enabled: smartEnabled,
	}

	// Extract temperature
	if smartInfo.Temperature != nil {
		ret.Temperature.Value = smartInfo.Temperature.Current
	}

	// Extract power on hours
	if smartInfo.PowerOnTime != nil {
		ret.PowerOnHours.Value = smartInfo.PowerOnTime.Hours
	}

	// Extract power cycle count
	ret.PowerCycleCount.Value = smartInfo.PowerCycleCount

	// Process based on device type
	if smartInfo.AtaSmartData != nil {
		// ATA/SATA device
		ret.DiskType = "SATA"

		// Process SMART attributes
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
					ret.PowerCycleCount.Value = attr.Value
					ret.PowerCycleCount.Worst = attr.Worst
					ret.PowerCycleCount.Thresholds = attr.Thresh
					if attr.Raw.Value > 0 {
						ret.PowerCycleCount.Value = int(attr.Raw.Value)
					}
				case dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS.Code:
					// Power on hours
					ret.PowerOnHours.Value = attr.Value
					ret.PowerOnHours.Worst = attr.Worst
					ret.PowerOnHours.Thresholds = attr.Thresh
					if attr.Raw.Value > 0 {
						ret.PowerOnHours.Value = int(attr.Raw.Value)
					}
				default:
					// Other attributes
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
		ret.DiskType = "NVMe"

		// Extract NVMe-specific data
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

		// Add NVMe-specific attributes
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
	} else {
		// SCSI or unknown device type
		ret.DiskType = "SCSI"
	}

	// Cache the result
	s.cache.Set(cacheKey, ret, gocache.DefaultExpiration)

	return ret, nil
}

// GetHealthStatus returns the health status of a device by evaluating SMART attributes
func (s *smartService) GetHealthStatus(devicePath string) (*dto.SmartHealthStatus, errors.E) {
	// Check if client is available
	if s.client == nil {
		return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "smartctl not available")
	}

	// Get SMART info first (may return cached data)
	smartInfo, err := s.GetSmartInfo(devicePath)
	if err != nil {
		return nil, err
	}

	// Check if SMART is enabled
	if !smartInfo.Enabled {
		tlog.Warn("SMART is not enabled on device", "device", devicePath)
		return &dto.SmartHealthStatus{
			Passed:            false,
			OverallStatus:     "warning",
			FailingAttributes: []string{"SMART_not_enabled"},
		}, nil
	}

	// Use smartmontools-go to check health
	healthy, stdErr := s.client.CheckHealth(devicePath)
	if stdErr != nil {
		tlog.Warn("failed to check health status", "device", devicePath, "error", stdErr)
	}

	health := checkSMARTHealth(smartInfo, nil, nil)

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

// StartSelfTest initiates a SMART self-test on the device
func (s *smartService) StartSelfTest(devicePath string, testType dto.SmartTestType) errors.E {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Map test type to smartctl test string - validate first
	var testTypeStr string
	switch testType {
	case dto.SmartTestTypeShort:
		testTypeStr = "short"
	case dto.SmartTestTypeLong:
		testTypeStr = "long"
	case dto.SmartTestTypeConveyance:
		testTypeStr = "conveyance"
	default:
		return errors.WithDetails(dto.ErrorInvalidParameter, "test_type", testType)
	}

	// Check if client is available
	if s.client == nil {
		return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "smartctl not available")
	}

	// Check if device exists
	if err := checkDeviceExists(devicePath); err != nil {
		return err
	}

	// Start the self-test using smartmontools-go
	if err := s.client.RunSelfTest(devicePath, testTypeStr); err != nil {
		if strings.Contains(err.Error(), "not supported") {
			return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath,
				"reason", "self-test not supported")
		}
		return errors.Wrapf(err, "failed to start SMART self-test")
	}

	slog.Info("SMART self-test started", "device", devicePath, "type", testType)
	return nil
}

// AbortSelfTest aborts the currently running SMART self-test on the device
func (s *smartService) AbortSelfTest(devicePath string) errors.E {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if client is available
	if s.client == nil {
		return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "smartctl not available")
	}

	// Check if device exists
	if err := checkDeviceExists(devicePath); err != nil {
		return err
	}

	// Abort the self-test using smartmontools-go
	if err := s.client.AbortSelfTest(devicePath); err != nil {
		if strings.Contains(err.Error(), "not supported") {
			return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath,
				"reason", "self-test abort not supported")
		}
		return errors.Wrapf(err, "failed to abort SMART self-test")
	}

	slog.Info("SMART self-test aborted", "device", devicePath)
	return nil
}

// GetTestStatus returns the status of the currently running or last SMART self-test
func (s *smartService) GetTestStatus(devicePath string) (*dto.SmartTestStatus, errors.E) {
	// Check if client is available
	if s.client == nil {
		return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "smartctl not available")
	}

	// Check if device exists
	if err := checkDeviceExists(devicePath); err != nil {
		return nil, err
	}

	// Get SMART info which includes self-test status
	smartInfo, err := s.client.GetSMARTInfo(devicePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get SMART info")
	}

	// Parse self-test status from ATA SMART data
	if smartInfo.AtaSmartData != nil && smartInfo.AtaSmartData.SelfTest != nil {
		status := &dto.SmartTestStatus{
			Status:   smartInfo.AtaSmartData.SelfTest.Status,
			TestType: "unknown",
		}

		// Determine test type if available
		if strings.Contains(smartInfo.AtaSmartData.SelfTest.Status, "short") {
			status.TestType = "short"
		} else if strings.Contains(smartInfo.AtaSmartData.SelfTest.Status, "long") || strings.Contains(smartInfo.AtaSmartData.SelfTest.Status, "extended") {
			status.TestType = "long"
		} else if strings.Contains(smartInfo.AtaSmartData.SelfTest.Status, "conveyance") {
			status.TestType = "conveyance"
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
func (s *smartService) EnableSMART(devicePath string) errors.E {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if client is available
	if s.client == nil {
		return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "smartctl not available")
	}

	// Check if device exists
	if err := checkDeviceExists(devicePath); err != nil {
		return err
	}

	// Enable SMART using smartmontools-go
	if err := s.client.EnableSMART(devicePath); err != nil {
		return errors.Wrapf(err, "failed to enable SMART")
	}

	// Verify SMART is now enabled
	supportInfo, err := s.client.IsSMARTSupported(devicePath)
	if err != nil {
		tlog.Warn("SMART enabled but verification failed", "device", devicePath, "error", err)
		return errors.Wrap(err, "SMART enable succeeded but verification failed")
	}

	if !supportInfo.Enabled {
		return errors.WithDetails(dto.ErrorSMARTOperationFailed, "device", devicePath,
			"reason", "SMART enable command executed but device reports disabled")
	}

	slog.Info("SMART enabled and verified", "device", devicePath)
	return nil
}

// DisableSMART disables SMART functionality on the device
func (s *smartService) DisableSMART(devicePath string) errors.E {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if client is available
	if s.client == nil {
		return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "smartctl not available")
	}

	// Check if device exists
	if err := checkDeviceExists(devicePath); err != nil {
		return err
	}

	// Disable SMART using smartmontools-go
	if err := s.client.DisableSMART(devicePath); err != nil {
		return errors.Wrapf(err, "failed to disable SMART")
	}

	// Verify SMART is now disabled (optional, for informational purposes)
	supportInfo, err := s.client.IsSMARTSupported(devicePath)
	if err != nil {
		tlog.Warn("SMART disabled but verification failed", "device", devicePath, "error", err)
		tlog.Info("SMART disable command executed (verification failed)", "device", devicePath)
	} else if supportInfo.Enabled {
		tlog.Warn("SMART disable command executed but device still reports enabled", "device", devicePath)
	}

	slog.Info("SMART disabled", "device", devicePath)
	return nil
}
