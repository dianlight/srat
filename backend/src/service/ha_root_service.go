package service

import (
	"context"
	"sync"
	"time"

	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"

	"github.com/dianlight/srat/homeassistant/root"
	gocache "github.com/patrickmn/go-cache"
)

const (
	haRootSystemInfoCacheKey     = "ha_root_system_info"
	haRootSystemInfoCacheExpiry  = 24 * time.Hour
	haRootSystemInfoCacheCleanup = 1 * time.Hour
)

var haRootSystemInfoCache = gocache.New(haRootSystemInfoCacheExpiry, haRootSystemInfoCacheCleanup)

var _ha_root_api_mutex sync.Mutex

type HaRootServiceInterface interface {
	GetSystemInfo() (*root.SystemInfo, error)
	GetAvailableUpdates() ([]root.UpdateItem, error)
	RefreshUpdates() error
	ReloadUpdates() error
}

type HaRootService struct {
	apiContext       context.Context
	apiContextCancel context.CancelFunc
	client           root.ClientWithResponsesInterface
	// No cache here; use package-level cache helpers
}

type HaRootServiceParams struct {
	fx.In
	ApiContext       context.Context
	ApiContextCancel context.CancelFunc
	Client           root.ClientWithResponsesInterface `optional:"true"`
}

func NewHaRootService(in HaRootServiceParams) HaRootServiceInterface {
	return &HaRootService{
		apiContext:       in.ApiContext,
		apiContextCancel: in.ApiContextCancel,
		client:           in.Client,
	}
}

func (s *HaRootService) GetSystemInfo() (*root.SystemInfo, error) {
	_ha_root_api_mutex.Lock()
	defer _ha_root_api_mutex.Unlock()
	// Try cache first
	if cached, found := getCachedSystemInfo(); found {
		return cached, nil
	}
	if s.client == nil {
		return nil, errors.New("HA Root client is not initialized")
	}
	resp, err := s.client.GetSystemInfoWithResponse(s.apiContext)
	if err != nil {
		return nil, errors.Errorf("Error getting system info from ha_root: %w", err)
	}
	if resp.StatusCode() != 200 || resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil, errors.Errorf("Error getting system info from ha_root: %d", resp.StatusCode())
	}
	setCachedSystemInfo(resp.JSON200.Data)
	return resp.JSON200.Data, nil
}

func (s *HaRootService) GetAvailableUpdates() ([]root.UpdateItem, error) {
	_ha_root_api_mutex.Lock()
	defer _ha_root_api_mutex.Unlock()
	if s.client == nil {
		return nil, errors.New("HA Root client is not initialized")
	}
	resp, err := s.client.GetAvailableUpdatesWithResponse(s.apiContext)
	if err != nil {
		return nil, errors.Errorf("Error getting available updates from ha_root: %w", err)
	}
	if resp.StatusCode() != 200 || resp.JSON200 == nil || resp.JSON200.Data == nil || resp.JSON200.Data.AvailableUpdates == nil {
		return nil, errors.Errorf("Error getting available updates from ha_root: %d", resp.StatusCode())
	}
	return *resp.JSON200.Data.AvailableUpdates, nil
}

func (s *HaRootService) RefreshUpdates() error {
	_ha_root_api_mutex.Lock()
	defer _ha_root_api_mutex.Unlock()
	if s.client == nil {
		return errors.New("HA Root client is not initialized")
	}
	resp, err := s.client.RefreshUpdatesWithResponse(s.apiContext)
	if err != nil {
		return errors.Errorf("Error refreshing updates from ha_root: %w", err)
	}
	if resp.StatusCode() != 200 {
		return errors.Errorf("Error refreshing updates from ha_root: %d", resp.StatusCode())
	}
	return nil
}

func (s *HaRootService) ReloadUpdates() error {
	_ha_root_api_mutex.Lock()
	defer _ha_root_api_mutex.Unlock()
	if s.client == nil {
		return errors.New("HA Root client is not initialized")
	}
	resp, err := s.client.ReloadUpdatesWithResponse(s.apiContext)
	if err != nil {
		return errors.Errorf("Error reloading updates from ha_root: %w", err)
	}
	if resp.StatusCode() != 200 {
		return errors.Errorf("Error reloading updates from ha_root: %d", resp.StatusCode())
	}
	return nil
}

func getCachedSystemInfo() (*root.SystemInfo, bool) {
	if cached, found := haRootSystemInfoCache.Get(haRootSystemInfoCacheKey); found {
		if info, ok := cached.(*root.SystemInfo); ok {
			return info, true
		}
	}
	return nil, false
}

func setCachedSystemInfo(info *root.SystemInfo) {
	haRootSystemInfoCache.Set(haRootSystemInfoCacheKey, info, haRootSystemInfoCacheExpiry)
}
