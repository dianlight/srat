package service

import (
	"context"
	"sync"
	"time"

	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"

	gocache "github.com/patrickmn/go-cache"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/host"
	"github.com/dianlight/srat/repository"
)

const (
	hostnameCacheKey            = "hostname"
	defaultCacheExpiration      = 1 * time.Hour
	defaultCacheCleanupInterval = 10 * time.Minute
)

type HostServiceInterface interface {
	GetHostName() (string, error)
}

type HostService struct {
	apiContext    context.Context
	host_client   host.ClientWithResponsesInterface
	state         *dto.ContextState
	hostnameCache *gocache.Cache
	hostnameMutex sync.Mutex
}

type HostServiceParams struct {
	fx.In
	ApiContext       context.Context
	ApiContextCancel context.CancelFunc
	HostClient       host.ClientWithResponsesInterface `optional:"true"`
	PropertyRepo     repository.PropertyRepositoryInterface
	State            *dto.ContextState
}

func NewHostService(in HostServiceParams) HostServiceInterface {
	p := &HostService{}
	p.apiContext = in.ApiContext
	p.host_client = in.HostClient
	p.state = in.State
	p.hostnameCache = gocache.New(defaultCacheExpiration, defaultCacheCleanupInterval)
	return p
}

func (self *HostService) GetHostName() (string, error) {
	self.hostnameMutex.Lock()
	defer self.hostnameMutex.Unlock()

	if self.state.HACoreReady == false {
		return "homeassistant", errors.New("Home Assistant core is not ready")
	}

	if name, found := self.hostnameCache.Get(hostnameCacheKey); found {
		if strName, ok := name.(string); ok {
			return strName, nil
		}
	}

	// If still not in cache, proceed to fetch
	if self.state.SupervisorURL != "demo" {
		if self.host_client == nil {
			return "homeassistant", errors.New("Host client is not initialized")
		}
		resp, err := self.host_client.GetHostInfoWithResponse(self.apiContext)
		if err != nil {
			return "homeassistant", errors.Errorf("Error getting info from ha_Host: %w", err)
		}
		if resp.StatusCode() != 200 {
			return "homeassistant", errors.Errorf("Error getting info from ha_Host: %d %s", resp.StatusCode(), string(resp.Body))
		}
		if resp.JSON200 == nil || resp.JSON200.Data.Hostname == nil {
			return "homeassistant", errors.New("Error getting info from ha_Host: response data is nil")
		}
		hostname := *resp.JSON200.Data.Hostname
		self.hostnameCache.Set(hostnameCacheKey, hostname, gocache.DefaultExpiration)
		return hostname, nil
	} else {
		return "demo", nil
	}
}
