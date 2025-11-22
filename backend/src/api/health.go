package api

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/tlog"
)

var _healthHanlerIntance *HealthHanler
var _healthHanlerIntanceMutex sync.Mutex

type HealthHanler struct {
	ctx                    context.Context
	state                  *dto.ContextState
	OutputEventsCount      uint64
	OutputEventsInterleave time.Duration
	dto.HealthPing
	broadcaster         service.BroadcasterServiceInterface
	sambaService        service.SambaServiceInterface
	dirtyService        service.DirtyDataServiceInterface
	addonsService       service.AddonsServiceInterface
	diskStatsService    service.DiskStatsService
	networkStatsService service.NetworkStatsService
	haRootService       service.HaRootServiceInterface
}

type HealthHandlerParams struct {
	fx.In
	Ctx                context.Context
	State              *dto.ContextState
	Broadcaster        service.BroadcasterServiceInterface
	SambaService       service.SambaServiceInterface
	DirtyService       service.DirtyDataServiceInterface
	AddonsService      service.AddonsServiceInterface
	NetworkStatService service.NetworkStatsService
	DiskStatsService   service.DiskStatsService
	HaRootService      service.HaRootServiceInterface `optional:"true"`
}

func NewHealthHandler(lc fx.Lifecycle, param HealthHandlerParams) *HealthHanler {
	_healthHanlerIntanceMutex.Lock()
	defer _healthHanlerIntanceMutex.Unlock()
	if _healthHanlerIntance != nil {
		return _healthHanlerIntance
	}

	p := new(HealthHanler)
	p.Alive = true
	p.AliveTime = time.Now().UnixMilli()
	p.LastError = ""
	p.ctx = param.Ctx
	p.state = param.State
	p.broadcaster = param.Broadcaster
	p.sambaService = param.SambaService
	p.OutputEventsCount = 0
	p.addonsService = param.AddonsService
	p.dirtyService = param.DirtyService
	p.diskStatsService = param.DiskStatsService
	p.networkStatsService = param.NetworkStatService
	p.haRootService = param.HaRootService
	if param.State.Heartbeat > 0 {
		p.OutputEventsInterleave = time.Duration(param.State.Heartbeat) * time.Second
	} else {
		p.OutputEventsInterleave = 5 * time.Second
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			p.ctx.Value("wg").(*sync.WaitGroup).Add(1)
			go func() {
				defer p.ctx.Value("wg").(*sync.WaitGroup).Done()
				p.run()
			}()
			return nil
		},
	})
	_healthHanlerIntance = p
	return p
}

// RegisterVolumeHandlers registers the health check handler for the API.
// It sets up an endpoint at "/health" that responds to GET requests with the
// HealthCheckHandler function. The endpoint is tagged with "system".
func (self *HealthHanler) RegisterVolumeHandlers(api huma.API) {
	huma.Get(api, "/health", self.HealthCheckHandler, huma.OperationTags("system"))
	huma.Get(api, "/status", self.HealthStatusHandler, huma.OperationTags("system"))
}

// HealthCheckHandler handles the health check request and returns the health status.
//
// Parameters:
// - ctx: The context for the request.
// - input: An empty struct as input.
//
// Returns:
// - A struct containing the health status in the Body field.
// - An error if the health check fails.
func (self *HealthHanler) HealthCheckHandler(ctx context.Context, input *struct{}) (*struct{ Body dto.HealthPing }, error) {
	return &struct{ Body dto.HealthPing }{Body: self.HealthPing}, nil
}

func (self *HealthHanler) HealthStatusHandler(ctx context.Context, input *struct{}) (*struct{ Body bool }, error) {
	return &struct{ Body bool }{Body: self.HealthPing.Alive}, nil
}

// checkSamba checks the status of the Samba process using the sambaService.
// If the Samba process is running, it converts the process information to a
// SambaProcessStatus DTO and updates the HealthPing.SambaProcessStatus field.
// If the Samba process is not running or an error occurs, it sets the
// HealthPing.SambaProcessStatus.Pid to -1.
func (self *HealthHanler) checkSamba() {
	sambaProcess, err := self.sambaService.GetSambaProcess()
	if err != nil {
		tlog.ErrorContext(self.ctx, "Error reading processes", "err", err)
	}
	self.HealthPing.SambaProcessStatus = *sambaProcess
}

// run is a method of HealthHandler that continuously monitors the health status
// of the system. It listens for a cancellation signal from the context and
// performs health checks at regular intervals. If the context is cancelled,
// it logs the closure and wraps the context error before returning it.
//
// The method performs the following actions in a loop:
// 1. Checks if the context is done and exits the loop if it is.
// 2. Calls checkSamba to perform a Samba health check.
// 3. Updates the HealthPing.Dirty status with the latest dirty data tracker.
// 4. Updates the AliveTime with the current time in milliseconds.
// 5. Emits a health message using the EventEmitter and logs any errors.
// 6. Sleeps for the duration specified by OutputEventsInterleave before repeating.
func (self *HealthHanler) run() error {
	for {
		select {
		case <-self.ctx.Done():
			slog.InfoContext(self.ctx, "Run process closed", "err", self.ctx.Err())
			self.OutputEventsInterleave = time.Duration(math.MaxInt64) // FIX rollbar#32
			return errors.WithStack(self.ctx.Err())
		case <-time.After(self.OutputEventsInterleave): // Use a timer to control loop frequency
			//			self.ProtectedMode = self.apictx.ProtectedMode
			// Get Addon Stats
			self.HealthPing.Uptime = time.Since(self.state.StartTime).Milliseconds()

			stats, err := self.addonsService.GetStats()
			if err != nil {
				slog.WarnContext(self.ctx, "Error getting addon stats for health ping", "err", err)
				self.HealthPing.AddonStats = nil // Clear stats on error
			} else {
				self.HealthPing.AddonStats = stats
			}
			self.checkSamba()
			diskStats, err := self.diskStatsService.GetDiskStats()
			if err != nil {
				slog.WarnContext(self.ctx, "Error getting disk stats for health ping", "err", err)
				self.HealthPing.DiskHealth = nil
			} else {
				self.HealthPing.DiskHealth = diskStats
				// Also broadcast the disk health separately for Home Assistant integration
				self.broadcaster.BroadcastMessage(*diskStats)
			}
			netStats, err := self.networkStatsService.GetNetworkStats()
			if err != nil {
				slog.WarnContext(self.ctx, "Error getting network stats for health ping", "err", err)
				self.HealthPing.NetworkHealth = nil
			} else {
				self.HealthPing.NetworkHealth = netStats
			}
			sambaStatus, err := self.sambaService.GetSambaStatus()
			if err != nil {
				slog.WarnContext(self.ctx, "Error getting samba status for health ping", "err", err)
				self.HealthPing.SambaStatus = nil
			} else {
				self.HealthPing.SambaStatus = sambaStatus
				// Also broadcast the samba status separately for Home Assistant integration
				self.broadcaster.BroadcastMessage(*sambaStatus)
			}

			// Also broadcast the samba process status separately for Home Assistant integration
			self.broadcaster.BroadcastMessage(self.HealthPing.SambaProcessStatus)

			self.HealthPing.Dirty = self.dirtyService.GetDirtyDataTracker()
			self.AliveTime = time.Now().UnixMilli()
			self.broadcaster.BroadcastMessage(self.HealthPing)

		}
	}
}
