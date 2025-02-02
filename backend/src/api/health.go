package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ztrue/tracerr"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"
)

type HealthHanler struct {
	ctx                    context.Context
	OutputEventsCount      uint64
	OutputEventsInterleave time.Duration
	dto.HealthPing
	//gh *github.Client
	//updateChannel dto.UpdateChannel
	broadcaster  service.BroadcasterServiceInterface
	sambaService service.SambaServiceInterface
}

func NewHealthHandler(ctx context.Context, apictx *ContextState, broadcaster service.BroadcasterServiceInterface, sambaService service.SambaServiceInterface) *HealthHanler {

	p := new(HealthHanler)
	p.Alive = true
	p.AliveTime = time.Now()
	p.ReadOnly = apictx.ReadOnlyMode
	p.SambaProcessStatus.Pid = -1
	p.LastError = ""
	p.ctx = ctx
	p.broadcaster = broadcaster
	p.sambaService = sambaService
	p.OutputEventsCount = 0
	if apictx.Heartbeat > 0 {
		p.OutputEventsInterleave = time.Duration(apictx.Heartbeat) * time.Second
	} else {
		p.OutputEventsInterleave = 5 * time.Second
	}
	go p.run()
	return p
}

func (broker *HealthHanler) Patterns() []server.RouteDetail {
	return []server.RouteDetail{
		{Pattern: "/healt", Method: "GET", Handler: broker.HealthCheckHandler},
	}
}

// HealthCheckHandler godoc
//
//	@Summary		HealthCheck
//	@Description	HealthCheck
//	@Tags			system
//	@Produce		json
//	@Success		200 {object}	dto.HealthPing
//	@Failure		405	{object}	ErrorResponse
//	@Router			/health [get]
func (self *HealthHanler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	HttpJSONReponse(w, self, nil)
}

func (self *HealthHanler) EventEmitter(ctx context.Context, data dto.HealthPing) error {
	msg := dto.EventMessageEnvelope{
		Event: dto.EventHeartbeat,
		Data:  data,
	}
	_, err := self.broadcaster.BroadcastMessage(&msg)
	if err != nil {
		slog.Error("Error broadcasting health message: %w", "err", err)
		return tracerr.Wrap(err)
	}
	self.OutputEventsCount++
	return nil
}

func (self *HealthHanler) checkSamba() {
	sambaProcess, err := self.sambaService.GetSambaProcess()
	if err == nil && sambaProcess != nil {
		var conv converter.ProcessToDtoImpl
		conv.ProcessToSambaProcessStatus(sambaProcess, &self.HealthPing.SambaProcessStatus)
	} else {
		self.HealthPing.SambaProcessStatus.Pid = -1
	}
}

func (self *HealthHanler) run() error {
	for {
		select {
		case <-self.ctx.Done():
			slog.Debug("Run process closed", "err", self.ctx.Err())
			return tracerr.Wrap(self.ctx.Err())
		default:
			slog.Debug("Richiesto aggiornamento per Healthy")
			self.checkSamba()
			err := self.EventEmitter(self.ctx, self.HealthPing)
			if err != nil {
				slog.Error("Error emitting health message: %w", "err", err)
			}
			time.Sleep(self.OutputEventsInterleave)
		}
	}
}
