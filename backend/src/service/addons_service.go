package service

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/addons"
	"github.com/dianlight/srat/tlog"
	gocache "github.com/patrickmn/go-cache"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// AddonsServiceInterface defines the contract for addon-related operations.
type AddonsServiceInterface interface {
	// CheckProtectedMode checks if a given Home Assistant addon is marked as protected.
	// It returns true if the addon is protected, false otherwise.
	// An error is returned if the check cannot be performed (e.g., API error, addon not found).
	CheckProtectedMode() (bool, errors.E)
	// GetStats retrieves the resource usage statistics for the current addon.
	// It returns an AddonStatsData object on success.
	GetStats() (*addons.AddonStatsData, errors.E)
}

// AddonsService provides methods to interact with Home Assistant addons.
type AddonsService struct {
	ctx                context.Context
	apictx             *dto.ContextState // Context state for the API, can be used for logging or passing additional information.
	addonsClient       addons.ClientWithResponsesInterface
	haWsService        HaWsServiceInterface
	protectedModeCache *gocache.Cache
	protectedModeMutex sync.Mutex
	statsCache         *gocache.Cache
	statsMutex         sync.Mutex
}

// AddonsServiceParams holds the dependencies for AddonsService.
type AddonsServiceParams struct {
	fx.In
	Ctx          context.Context
	Apictx       *dto.ContextState
	AddonsClient addons.ClientWithResponsesInterface `optional:"true"`
	HaWsService  HaWsServiceInterface
}

const (
	protectedModeCacheKey     = "protectedMode"
	protectedModeCacheExpiry  = 24 * time.Hour
	protectedModeCacheCleanup = 1 * time.Hour
	statsCacheKey             = "addonStats"
	statsCacheExpiry          = 30 * time.Second
	statsCacheCleanup         = 1 * time.Minute
)

// NewAddonsService creates a new instance of AddonsService.
func NewAddonsService(lc fx.Lifecycle, params AddonsServiceParams) AddonsServiceInterface {
	if params.AddonsClient == nil {
		tlog.Debug("AddonsClient is not available for AddonsService. Operations requiring it will fail.")
	}
	p := &AddonsService{
		ctx:                params.Ctx,
		apictx:             params.Apictx,
		addonsClient:       params.AddonsClient,
		protectedModeCache: gocache.New(protectedModeCacheExpiry, protectedModeCacheCleanup),
		statsCache:         gocache.New(statsCacheExpiry, statsCacheCleanup),
		haWsService:        params.HaWsService,
	}

	p.haWsService.SubscribeToHaEvents(func(ready bool) {
		if ready {
			p.apictx.ProtectedMode, _ = p.CheckProtectedMode()
		}
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			p.apictx.ProtectedMode, _ = p.CheckProtectedMode()
			return nil
		},
	})
	return p
}

// CheckProtectedMode implements the AddonsServiceInterface.
func (s *AddonsService) CheckProtectedMode() (bool, errors.E) {
	if !s.apictx.HACoreReady {
		return true, nil // If HA Core is not ready, we cannot determine protected mode but assume to true
	}

	// Try to get from cache first
	if protected, found := s.protectedModeCache.Get(protectedModeCacheKey); found {
		if p, ok := protected.(bool); ok {
			return p, nil
		}
	}

	// If not in cache, acquire lock to fetch
	s.protectedModeMutex.Lock()
	defer s.protectedModeMutex.Unlock()

	// Re-check cache after acquiring lock
	if protected, found := s.protectedModeCache.Get(protectedModeCacheKey); found {
		if p, ok := protected.(bool); ok {
			return p, nil
		}
	}

	if s.addonsClient == nil {
		return false, errors.New("addons client is not initialized")
	}

	resp, err := s.addonsClient.GetSelfAddonInfoWithResponse(s.ctx)
	if err != nil || resp == nil {
		return false, errors.Wrapf(err, "failed to get addon info for '%s'", "self")
	}

	if resp.StatusCode() != http.StatusOK {
		return false, errors.Errorf("failed to get addon info for '%s': status %d, body: %s", "self", resp.StatusCode(), string(resp.Body))
	}

	if resp.JSON200 == nil || resp.JSON200.Data.Protected == nil {
		// If protected status is not explicitly provided, assume not protected or data is incomplete.
		return false, errors.Errorf("protected status not available or data incomplete for addon '%s'", "self")
	}

	isProtected := *resp.JSON200.Data.Protected
	s.protectedModeCache.Set(protectedModeCacheKey, isProtected, gocache.DefaultExpiration)

	return isProtected, nil
}

// GetStats implements the AddonsServiceInterface.
func (s *AddonsService) GetStats() (*addons.AddonStatsData, errors.E) {
	// Try to get from cache first
	if cachedStats, found := s.statsCache.Get(statsCacheKey); found {
		if stats, ok := cachedStats.(*addons.AddonStatsData); ok {
			return stats, nil
		}
	}

	// If not in cache, acquire lock to fetch
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	// Re-check cache after acquiring lock
	if cachedStats, found := s.statsCache.Get(statsCacheKey); found {
		if stats, ok := cachedStats.(*addons.AddonStatsData); ok {
			return stats, nil
		}
	}

	if s.addonsClient == nil {
		return nil, errors.New("addons client is not initialized")
	}

	resp, err := s.addonsClient.GetSelfAddonStatsWithResponse(s.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get addon stats")
	}

	if resp.StatusCode() != http.StatusOK {
		if resp != nil && resp.Body != nil && strings.Contains(string(resp.Body), "System is not ready with state: shutdown") {
			return nil, errors.WithDetails(dto.ErrorInvalidStateForOperation, string(resp.Body))
		}
		return nil, errors.Errorf("failed to get addon stats: status %d, body: %s", resp.StatusCode(), string(resp.Body))
	}

	if resp.JSON200 == nil {
		return nil, errors.New("addon stats not available or data incomplete")
	}

	stats := &resp.JSON200.Data
	s.statsCache.Set(statsCacheKey, stats, gocache.DefaultExpiration)

	return stats, nil
}
