package api

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
)

var _healthHanlerIntance *HealthHanler
var _healthHanlerIntanceMutex sync.Mutex

type HealthHanler struct {
	ctx                    context.Context
	apictx                 *dto.ContextState
	OutputEventsCount      uint64
	OutputEventsInterleave time.Duration
	dto.HealthPing
	broadcaster  service.BroadcasterServiceInterface
	sambaService service.SambaServiceInterface
	dirtyService service.DirtyDataServiceInterface
}

type HealthHandlerParams struct {
	fx.In
	Ctx          context.Context
	Apictx       *dto.ContextState
	Broadcaster  service.BroadcasterServiceInterface
	SambaService service.SambaServiceInterface
	DirtyService service.DirtyDataServiceInterface
	HaMode       bool `name:"ha_mode"`
}

func NewHealthHandler(param HealthHandlerParams) *HealthHanler {
	_healthHanlerIntanceMutex.Lock()
	defer _healthHanlerIntanceMutex.Unlock()
	if _healthHanlerIntance != nil {
		return _healthHanlerIntance
	}

	p := new(HealthHanler)
	p.Alive = true
	p.AliveTime = time.Now().UnixMilli()
	p.ReadOnly = param.Apictx.ReadOnlyMode
	p.SambaProcessStatus.Pid = -1
	p.LastError = ""
	p.ctx = param.Ctx
	p.apictx = param.Apictx
	p.broadcaster = param.Broadcaster
	p.sambaService = param.SambaService
	p.OutputEventsCount = 0
	p.dirtyService = param.DirtyService
	p.SecureMode = param.HaMode
	p.BuildVersion = config.BuildVersion()
	if param.Apictx.Heartbeat > 0 {
		p.OutputEventsInterleave = time.Duration(param.Apictx.Heartbeat) * time.Second
	} else {
		p.OutputEventsInterleave = 5 * time.Second
	}
	param.Ctx.Value("wg").(*sync.WaitGroup).Add(1)
	go func() {
		defer param.Ctx.Value("wg").(*sync.WaitGroup).Done()
		p.run()
	}()
	_healthHanlerIntance = p
	return p
}

// RegisterVolumeHandlers registers the health check handler for the API.
// It sets up an endpoint at "/health" that responds to GET requests with the
// HealthCheckHandler function. The endpoint is tagged with "system".
func (self *HealthHanler) RegisterVolumeHandlers(api huma.API) {
	huma.Get(api, "/health", self.HealthCheckHandler, huma.OperationTags("system"))
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

// EventEmitter broadcasts a health ping message using the broadcaster.
// It increments the OutputEventsCount if the message is successfully broadcasted.
// If an error occurs during broadcasting, it logs the error and wraps it before returning.
//
// Parameters:
//   - data: dto.HealthPing containing the health ping message to be broadcasted.
//
// Returns:
//   - error: An error if the broadcasting fails, otherwise nil.
func (self *HealthHanler) EventEmitter(data dto.HealthPing) error {
	_, err := self.broadcaster.BroadcastMessage(data)
	if err != nil {
		slog.Error("Error broadcasting health message: %w", "err", err)
		return errors.WithStack(err)
	}
	self.OutputEventsCount++
	return nil
}

// checkSamba checks the status of the Samba process using the sambaService.
// If the Samba process is running, it converts the process information to a
// SambaProcessStatus DTO and updates the HealthPing.SambaProcessStatus field.
// If the Samba process is not running or an error occurs, it sets the
// HealthPing.SambaProcessStatus.Pid to -1.
func (self *HealthHanler) checkSamba() {
	sambaProcess, err := self.sambaService.GetSambaProcess()
	if err == nil && sambaProcess != nil {
		var conv converter.ProcessToDtoImpl
		conv.ProcessToSambaProcessStatus(sambaProcess, &self.HealthPing.SambaProcessStatus)
	} else {
		self.HealthPing.SambaProcessStatus.Pid = -1
	}
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
			slog.Info("Run process closed", "err", self.ctx.Err())
			return errors.WithStack(self.ctx.Err())
		default:
			//slog.Debug("Richiesto aggiornamento per Healthy")
			self.checkSamba()
			self.HealthPing.Dirty = self.dirtyService.GetDirtyDataTracker()
			self.AliveTime = time.Now().UnixMilli()
			err := self.EventEmitter(self.HealthPing)
			if err != nil {
				slog.Error("Error emitting health message: %w", "err", err)
			}
			time.Sleep(self.OutputEventsInterleave)
		}
	}
}
