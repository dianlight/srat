package service

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/addons"
	"github.com/dianlight/tlog"
	gocache "github.com/patrickmn/go-cache"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// AddonsServiceInterface defines the contract for addon-related operations.
type AddonsServiceInterface interface {
	// GetStats retrieves the resource usage statistics for the current addon.
	// It returns an AddonStatsData object on success.
	GetStats() (*addons.AddonStatsData, errors.E)
	GetLatestLogs(ctx context.Context) (string, errors.E)
	GetInfo(ctx context.Context) (*addons.AddonInfoData, errors.E)
}

// AddonsService provides methods to interact with Home Assistant addons.
type AddonsService struct {
	ctx          context.Context
	apictx       *dto.ContextState // Context state for the API, can be used for logging or passing additional information.
	addonsClient addons.ClientWithResponsesInterface
	haWsService  HaWsServiceInterface
	statsCache   *gocache.Cache
	statsMutex   sync.Mutex
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
	statsCacheKey     = "addonStats"
	statsCacheExpiry  = 30 * time.Second
	statsCacheCleanup = 1 * time.Minute
)

// NewAddonsService creates a new instance of AddonsService.
func NewAddonsService(lc fx.Lifecycle, params AddonsServiceParams) AddonsServiceInterface {
	if params.AddonsClient == nil {
		tlog.DebugContext(params.Ctx, "AddonsClient is not available for AddonsService. Operations requiring it will fail.")
	}
	p := &AddonsService{
		ctx:          params.Ctx,
		apictx:       params.Apictx,
		addonsClient: params.AddonsClient,
		statsCache:   gocache.New(statsCacheExpiry, statsCacheCleanup),
		haWsService:  params.HaWsService,
	}
	return p
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

func (s *AddonsService) GetLatestLogs(ctx context.Context) (string, errors.E) {
	if s.addonsClient == nil {
		return "", errors.New("addons client is not initialized")
	}

	resp, err := s.addonsClient.GetSelfAddonLogsLatestWithResponse(ctx, &addons.GetSelfAddonLogsLatestParams{
		Lines:  pointer.Int(1000),
		Accept: addons.GetSelfAddonLogsLatestParamsAcceptTextxLog,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to get addon logs")
	}

	if resp.StatusCode() != http.StatusOK {
		return "", errors.Errorf("failed to get addon logs: status %d, body: %s", resp.StatusCode(), string(resp.Body))
	}

	if resp.Body == nil {
		return "", errors.New("addon logs not available or data incomplete")
	}

	return string(resp.Body), nil
}

func (s *AddonsService) GetInfo(ctx context.Context) (*addons.AddonInfoData, errors.E) {
	if s.addonsClient == nil {
		return nil, errors.New("addons client is not initialized")
	}

	resp, err := s.addonsClient.GetSelfAddonInfoWithResponse(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get addon info")
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("failed to get addon info: status %d, body: %s", resp.StatusCode(), string(resp.Body))
	}

	if resp.JSON200 == nil {
		return nil, errors.New("addon info not available or data incomplete")
	}

	return &resp.JSON200.Data, nil
}
