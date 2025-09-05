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

type SmartService interface {
	GetSmartInfo(deviceName string) (*dto.SmartInfo, errors.E)
}

type smartService struct {
	cache *gocache.Cache
	mutex sync.Mutex
}

func NewSmartService() SmartService {
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
