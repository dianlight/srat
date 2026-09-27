package api

import (
	"context"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/apps"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/tlog"
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
	sambaService        service.ServerServiceInterface
	dirtyService        service.DirtyDataServiceInterface
	addonsService       service.AddonsServiceInterface
	diskStatsService    service.DiskStatsService
	networkStatsService service.NetworkStatsService
	haRootService       service.HaRootServiceInterface
	callTimeouts        healthCallTimeouts
}

// healthCallTimeouts bounds the blocking service calls made from the run loop
// so that a single wedged service can never stop heartbeat broadcasts
// (issue #1252).
type healthCallTimeouts struct {
	// heavy bounds addon stats (HTTP to the supervisor), samba process
	// enumeration and smbstatus subprocess calls. Default 10s.
	heavy time.Duration
	// light bounds disk/network stats and the dirty tracker, which mostly
	// block on in-process mutexes. Default 5s.
	light time.Duration
}

func defaultHealthCallTimeouts() healthCallTimeouts {
	return healthCallTimeouts{
		heavy: 10 * time.Second,
		light: 5 * time.Second,
	}
}

// withDefaults fills zero/negative values with the defaults so handlers
// constructed without explicit timeouts (e.g. tests) still get bounded calls.
func (t healthCallTimeouts) withDefaults() healthCallTimeouts {
	d := defaultHealthCallTimeouts()
	if t.heavy <= 0 {
		t.heavy = d.heavy
	}
	if t.light <= 0 {
		t.light = d.light
	}
	return t
}

type HealthHandlerParams struct {
	fx.In
	Ctx                context.Context
	State              *dto.ContextState
	Broadcaster        service.BroadcasterServiceInterface
	SambaService       service.ServerServiceInterface
	DirtyService       service.DirtyDataServiceInterface
	AddonsService      service.AddonsServiceInterface
	NetworkStatService service.NetworkStatsService
	DiskStatsService   service.DiskStatsService
	HaRootService      service.HaRootServiceInterface `optional:"true"`
}

func isExpectedStartupHealthError(component string, err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	switch component {
	case "disk stats":
		return strings.Contains(msg, "disk stats not initialized")
	case "samba status":
		return strings.Contains(msg, "smbstatus returned non-json output") ||
			strings.Contains(msg, "smbstatus returned empty output")
	default:
		return false
	}
}

func logHealthFetchError(ctx context.Context, component string, err error) {
	if isExpectedStartupHealthError(component, err) {
		tlog.TraceContext(ctx, "Health subsystem still warming up", "component", component, "err", err)
		return
	}

	tlog.DebugContext(ctx, "Warning getting "+component+" for health ping", "err", err)
}

// callWithTimeout runs fn in a goroutine bounded by timeout so that a service
// call blocking indefinitely cannot wedge the health run loop (issue #1252).
// It returns true when fn finished within the timeout, false otherwise.
//
// On timeout the abandoned goroutine may still complete later, so fn must only
// write to call-local state; callers apply results to the handler after a true
// return and keep the previous values on false.
func (self *HealthHanler) callWithTimeout(timeout time.Duration, component string, fn func()) bool {
	ctx, cancel := context.WithTimeout(self.ctx, timeout)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		if self.ctx.Err() == nil {
			tlog.WarnContext(self.ctx, "Health check call timed out; skipping update",
				"component", component, "timeout", timeout)
		}
		return false
	}
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
	p.callTimeouts = defaultHealthCallTimeouts()

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if wg, ok := p.ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok && wg != nil {
				wg.Go(func() {
					if err := p.run(); err != nil && !errors.Is(err, context.Canceled) {
						slog.WarnContext(p.ctx, "Health handler loop stopped with error", "error", err)
					}
				})
			}
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
	return &struct{ Body bool }{Body: self.Alive}, nil
}

// checkSamba checks the status of the Samba process using the sambaService,
// bounded by timeout so a hung process scan cannot wedge the run loop
// (issue #1252). It updates the HealthPing.SambaProcessStatus field only when
// a fresh, non-nil result arrives within the timeout; on error, nil result or
// timeout the previous value is kept.
func (self *HealthHanler) checkSamba(timeout time.Duration) {
	var sambaProcess *dto.ServerProcessStatus
	var fetchErr errors.E
	if !self.callWithTimeout(timeout, "samba process status", func() {
		sambaProcess, fetchErr = self.sambaService.GetServerProcesses()
		if fetchErr != nil {
			tlog.ErrorContext(self.ctx, "Error reading processes", "err", fetchErr)
		}
	}) {
		return
	}
	if sambaProcess == nil {
		return
	}
	self.SambaProcessStatus = *sambaProcess
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
//
// Every service call is bounded by callTimeouts (issue #1252): a call that
// exceeds its budget is abandoned and the loop continues with the previous
// value for that field, so heartbeat broadcasts never stop.
func (self *HealthHanler) run() error {
	var ticks int
	timeouts := self.callTimeouts.withDefaults()
	for {
		select {
		case <-self.ctx.Done():
			slog.InfoContext(self.ctx, "Run process closed", "err", self.ctx.Err())
			self.OutputEventsInterleave = time.Duration(math.MaxInt64) // FIX legacy issue #32
			return errors.WithStack(self.ctx.Err())
		case <-time.After(self.OutputEventsInterleave): // Use a timer to control loop frequency
			// Get Addon Stats
			self.Uptime = time.Since(self.state.StartTime).Milliseconds()

			self.UpdateAvailable = self.state.UpdateAvailable

			// Only run expensive process and samba status checks every 12 ticks (~60s)
			// to avoid blocking the high-frequency health loop and to reduce disk access.
			isHeavyTick := (ticks == 0)
			ticks = (ticks + 1) % 12

			var stats *apps.AppStatsData
			if self.callWithTimeout(timeouts.heavy, "addon stats", func() {
				var err errors.E
				stats, err = self.addonsService.GetStats()
				if err != nil {
					slog.WarnContext(self.ctx, "Warning getting addon stats for health ping", "err", err)
					stats = nil
				}
			}) {
				self.AddonStats = stats // Clear stats on error, keep previous on timeout
			}
			if isHeavyTick {
				self.checkSamba(timeouts.heavy)
			}
			var diskStats *dto.DiskHealth
			if self.callWithTimeout(timeouts.light, "disk stats", func() {
				var err errors.E
				diskStats, err = self.diskStatsService.GetDiskStats()
				if err != nil {
					logHealthFetchError(self.ctx, "disk stats", err)
					diskStats = nil
				}
			}) {
				self.DiskHealth = diskStats
				if diskStats != nil {
					// Also broadcast the disk health separately for Home Assistant integration
					self.broadcaster.BroadcastMessage(*diskStats)
				}
			}
			var netStats *dto.NetworkStats
			if self.callWithTimeout(timeouts.light, "network stats", func() {
				var err errors.E
				netStats, err = self.networkStatsService.GetNetworkStats()
				if err != nil {
					slog.WarnContext(self.ctx, "Warning getting network stats for health ping", "err", err)
					netStats = nil
				}
			}) {
				self.NetworkHealth = netStats
			}
			if isHeavyTick {
				var sambaStatus *dto.SambaStatus
				if self.callWithTimeout(timeouts.heavy, "samba status", func() {
					var err errors.E
					sambaStatus, err = self.sambaService.GetSambaStatus()
					if err != nil {
						logHealthFetchError(self.ctx, "samba status", err)
						sambaStatus = nil
					}
				}) {
					self.SambaStatus = sambaStatus
					if sambaStatus != nil {
						// Also broadcast the samba status separately for Home Assistant integration
						self.broadcaster.BroadcastMessage(*sambaStatus)
					}
				}

				// Also broadcast the samba process status separately for Home Assistant integration
				self.broadcaster.BroadcastMessage(self.SambaProcessStatus)
			}

			var dirty dto.DataDirtyTracker
			if self.callWithTimeout(timeouts.light, "dirty data tracker", func() {
				dirty = self.dirtyService.GetDirtyDataTracker()
			}) {
				self.Dirty = dirty
			}
			self.AliveTime = time.Now().UnixMilli()
			self.broadcaster.BroadcastMessage(self.HealthPing)

		}
	}
}
