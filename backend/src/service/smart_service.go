package service

import (
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/anatol/smart.go"
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
	cache *gocache.Cache
	mutex sync.Mutex
}

func NewSmartService() SmartServiceInterface {
	return &smartService{
		cache: gocache.New(smartCacheExpiry, smartCacheCleanup),
	}
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
	// Check if the device path exists and is readable before attempting to open it with smart.go
	file, err := os.OpenFile(devicePath, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			tlog.Trace("SMART: device path does not exist on filesystem, skipping.", "device", devicePath)
			return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "does not exist")
		}
		if os.IsPermission(err) {
			tlog.Trace("SMART: device path is not readable, skipping.", "device", devicePath, "error", err)
			return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "not readable")
		}
		return nil, errors.Wrapf(err, "error checking device path '%s'", devicePath)
	}
	file.Close() // We just wanted to check if we can open it.

	dev, err := smart.Open(devicePath)
	if err != nil {
		if strings.Contains(err.Error(), "unknown drive type") || strings.Contains(err.Error(), "not a valid device") {
			return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath, "reason", "unsupported device")
		}
		return nil, errors.Wrapf(err, "failed to open device %s", devicePath)
	}
	defer dev.Close()

	attrs, err := dev.ReadGenericAttributes()
	if err != nil {
		return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath)
	}

	ret := &dto.SmartInfo{
		Temperature: dto.SmartTempValue{
			Value: int(attrs.Temperature),
		},
		PowerOnHours: dto.SmartRangeValue{
			Value: int(attrs.PowerOnHours),
		},
		PowerCycleCount: dto.SmartRangeValue{
			Value: int(attrs.PowerCycles),
		},
	}

	switch sm := dev.(type) {
	case *smart.SataDevice:
		ret.DiskType = "SATA"
		data, err := sm.ReadSMARTData()
		if err != nil {
			slog.Warn("SMART: failed to read SMART data", "device", devicePath, "error", err)
			break
		}
		if attr, ok := data.Attrs[uint8(dto.SmartAttributeCodes.SMARTATTRTEMPERATURECELSIUS.Code)]; ok {
			if temp, min, max, overtempCounter, errs := attr.ParseAsTemperature(); errs == nil {
				ret.Temperature.Value = temp
				ret.Temperature.Min = min
				ret.Temperature.Max = max
				ret.Temperature.OvertempCounter = overtempCounter
			}
		}
		if attr, ok := data.Attrs[uint8(dto.SmartAttributeCodes.SMARTATTRPOWERCYCLECOUNT.Code)]; ok {
			ret.PowerCycleCount.Value = int(attr.Current)
			ret.PowerCycleCount.Worst = int(attr.Worst)
		}
		if attr, ok := data.Attrs[uint8(dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS.Code)]; ok {
			ret.PowerOnHours.Value = int(attr.Current)
			ret.PowerOnHours.Worst = int(attr.Worst)
		}
		others := make(map[string]dto.SmartRangeValue)
		for code, attr := range data.Attrs {
			if attr.Name == "" || code == uint8(dto.SmartAttributeCodes.SMARTATTRTEMPERATURECELSIUS.Code) ||
				code == uint8(dto.SmartAttributeCodes.SMARTATTRPOWERCYCLECOUNT.Code) ||
				code == uint8(dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS.Code) {
				continue
			}
			others[attr.Name] = dto.SmartRangeValue{
				Code:  int(code),
				Value: int(attr.Current),
				Worst: int(attr.Worst),
			}
		}
		if len(others) > 0 {
			ret.Additional = others
		}

		// Read thresholds if available
		thdata, err := sm.ReadSMARTThresholds()
		if err != nil {
			slog.Warn("SMART: failed to read SMART thresholds", "device", devicePath, "error", err)
			break
		}
		if attr, ok := thdata.Thresholds[uint8(dto.SmartAttributeCodes.SMARTATTRPOWERCYCLECOUNT.Code)]; ok {
			ret.PowerCycleCount.Thresholds = int(attr)
		}
		if attr, ok := thdata.Thresholds[uint8(dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS.Code)]; ok {
			ret.PowerOnHours.Thresholds = int(attr)
		}
		for code, attr := range ret.Additional {
			if thattr, ok := thdata.Thresholds[uint8(attr.Code)]; ok {
				val := ret.Additional[code]
				val.Thresholds = int(thattr)
				ret.Additional[code] = val
			}
		}
	case *smart.ScsiDevice:
		ret.DiskType = "SCSI"
	case *smart.NVMeDevice:
		ret.DiskType = "NVMe"
		nvmsmart, err := sm.ReadSMART()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read SMART data from NVMe device %s", devicePath)
		}
		ret.Temperature.OvertempCounter = int(nvmsmart.WarningTempTime)
		others := make(map[string]dto.SmartRangeValue)
		others["AvailableSpare"] = dto.SmartRangeValue{
			Value:      int(nvmsmart.AvailSpare),
			Thresholds: int(nvmsmart.SpareThresh),
		}
		others["PercentageUsed"] = dto.SmartRangeValue{
			Value: int(nvmsmart.PercentUsed),
		}
		others["CriticalWarning"] = dto.SmartRangeValue{
			Value: int(nvmsmart.CritWarning),
		}
		ret.Additional = others
	}

	s.cache.Set(cacheKey, ret, gocache.DefaultExpiration)

	return ret, nil
}

// GetHealthStatus returns the health status of a device by evaluating SMART attributes
func (s *smartService) GetHealthStatus(devicePath string) (*dto.SmartHealthStatus, errors.E) {
	// Get SMART info first
	smartInfo, err := s.GetSmartInfo(devicePath)
	if err != nil {
		return nil, err
	}

	// Open device to read thresholds
	dev, stdErr := smart.Open(devicePath)
	if stdErr != nil {
		return nil, errors.Wrapf(stdErr, "failed to open device %s", devicePath)
	}
	defer dev.Close()

	var thresholds map[uint8]uint8
	var attrs map[uint8]interface{}

	// Get thresholds for SATA devices
	if sm, ok := dev.(*smart.SataDevice); ok {
		thdata, stdErr := sm.ReadSMARTThresholds()
		if stdErr == nil {
			thresholds = thdata.Thresholds
		}
		data, stdErr := sm.ReadSMARTData()
		if stdErr == nil {
			attrs = make(map[uint8]interface{})
			for k, v := range data.Attrs {
				attrs[k] = v
			}
		}
	}

	health := checkSMARTHealth(smartInfo, thresholds, attrs)

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

	// Validate test type
	var testByte byte
	switch testType {
	case dto.SmartTestTypeShort:
		testByte = _SMART_SHORT_SELFTEST
	case dto.SmartTestTypeLong:
		testByte = _SMART_LONG_SELFTEST
	case dto.SmartTestTypeConveyance:
		testByte = _SMART_CONVEYANCE_SELFTEST
	default:
		return errors.WithDetails(dto.ErrorInvalidParameter, "test_type", testType)
	}

	// Open device
	dev, stdErr := smart.Open(devicePath)
	if stdErr != nil {
		return errors.Wrapf(stdErr, "failed to open device %s", devicePath)
	}
	defer dev.Close()

	// Only SATA devices support self-tests via this method
	sm, ok := dev.(*smart.SataDevice)
	if !ok {
		return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath,
			"reason", "self-test only supported for SATA devices")
	}

	// Execute the test
	if err := executeSMARTTest(sm, testByte); err != nil {
		return errors.Wrap(err, "failed to start SMART self-test")
	}

	slog.Info("SMART self-test started", "device", devicePath, "type", testType)
	return nil
}

// AbortSelfTest aborts the currently running SMART self-test on the device
func (s *smartService) AbortSelfTest(devicePath string) errors.E {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Open device
	dev, stdErr := smart.Open(devicePath)
	if stdErr != nil {
		return errors.Wrapf(stdErr, "failed to open device %s", devicePath)
	}
	defer dev.Close()

	// Only SATA devices support self-test abort via this method
	sm, ok := dev.(*smart.SataDevice)
	if !ok {
		return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath,
			"reason", "self-test abort only supported for SATA devices")
	}

	// Execute the abort command
	if err := executeSMARTTest(sm, _SMART_ABORT_SELFTEST); err != nil {
		return errors.Wrap(err, "failed to abort SMART self-test")
	}

	slog.Info("SMART self-test aborted", "device", devicePath)
	return nil
}

// GetTestStatus returns the status of the currently running or last SMART self-test
func (s *smartService) GetTestStatus(devicePath string) (*dto.SmartTestStatus, errors.E) {
	// Open device
	dev, stdErr := smart.Open(devicePath)
	if stdErr != nil {
		return nil, errors.Wrapf(stdErr, "failed to open device %s", devicePath)
	}
	defer dev.Close()

	// Only SATA devices support self-test log reading
	sm, ok := dev.(*smart.SataDevice)
	if !ok {
		return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath,
			"reason", "self-test status only supported for SATA devices")
	}

	// Read self-test log
	log, stdErr := sm.ReadSMARTSelfTestLog()
	if stdErr != nil {
		return nil, errors.Wrapf(stdErr, "failed to read SMART self-test log")
	}

	status, err := parseSelfTestLog(log)
	if err != nil {
		return nil, err
	}

	return status, nil
}

// EnableSMART enables SMART functionality on the device
func (s *smartService) EnableSMART(devicePath string) errors.E {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Open device
	dev, stdErr := smart.Open(devicePath)
	if stdErr != nil {
		return errors.Wrapf(stdErr, "failed to open device %s", devicePath)
	}
	defer dev.Close()

	// Only SATA devices support enable/disable
	sm, ok := dev.(*smart.SataDevice)
	if !ok {
		return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath,
			"reason", "SMART enable/disable only supported for SATA devices")
	}

	if err := enableSMART(sm); err != nil {
		return err
	}

	slog.Info("SMART enabled", "device", devicePath)
	return nil
}

// DisableSMART disables SMART functionality on the device
func (s *smartService) DisableSMART(devicePath string) errors.E {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Open device
	dev, stdErr := smart.Open(devicePath)
	if stdErr != nil {
		return errors.Wrapf(stdErr, "failed to open device %s", devicePath)
	}
	defer dev.Close()

	// Only SATA devices support enable/disable
	sm, ok := dev.(*smart.SataDevice)
	if !ok {
		return errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath,
			"reason", "SMART enable/disable only supported for SATA devices")
	}

	if err := disableSMART(sm); err != nil {
		return err
	}

	slog.Info("SMART disabled", "device", devicePath)
	return nil
}
