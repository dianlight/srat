package service

import (
	"sync"
	"time"

	"github.com/anatol/smart.go"
	"github.com/dianlight/srat/dto"
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

	dev, err := smart.Open(devicePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open device %s", devicePath)
	}
	defer dev.Close()

	attrs, err := dev.ReadGenericAttributes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read generic attributes for device %s", devicePath)
	}

	ret := &dto.SmartInfo{
		Temperature:     attrs.Temperature,
		PowerOnHours:    attrs.PowerOnHours,
		PowerCycleCount: attrs.PowerCycles,
	}

	s.cache.Set(cacheKey, ret, gocache.DefaultExpiration)

	return ret, nil
}
