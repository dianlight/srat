package service

import (
	"os"
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
	GetSmartInfo(deviceName string) (*dto.SmartInfo, error)
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

func (s *smartService) GetSmartInfo(devicePath string) (*dto.SmartInfo, error) {
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
		return nil, errors.Wrapf(err, "failed to open device %s", devicePath)
	}
	defer dev.Close()

	attrs, err := dev.ReadGenericAttributes()
	if err != nil {
		return nil, errors.WithDetails(dto.ErrorSMARTNotSupported, "device", devicePath)
	}

	ret := &dto.SmartInfo{
		Temperature:     attrs.Temperature,
		PowerOnHours:    attrs.PowerOnHours,
		PowerCycleCount: attrs.PowerCycles,
	}

	s.cache.Set(cacheKey, ret, gocache.DefaultExpiration)

	return ret, nil
}
