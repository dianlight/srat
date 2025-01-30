package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ztrue/tracerr"

	"github.com/dianlight/srat/dto"
)

type Health struct {
	ctx                    context.Context
	OutputEventsCount      uint64
	OutputEventsInterleave time.Duration
	dto.HealthPing
}

func NewHealth(ctx context.Context, ro_mode bool) *Health {
	p := new(Health)
	p.Alive = true
	p.AliveTime = time.Now()
	p.ReadOnly = ro_mode
	p.SambaProcessStatus.Pid = -1
	p.LastError = ""
	p.ctx = ctx
	p.OutputEventsCount = 0
	if sec := ctx.Value("health_interlive_seconds"); sec != nil {
		p.OutputEventsInterleave = time.Duration(sec.(int)) * time.Second
	} else {
		p.OutputEventsInterleave = 10 * time.Second
	}
	go p.run()
	return p
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
func (self *Health) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	HttpJSONReponse(w, self, nil)
}

func (self *Health) EventEmitter(ctx context.Context, data dto.HealthPing) error {
	share := StateFromContext(ctx)
	msg := dto.EventMessageEnvelope{
		Event: dto.EventHeartbeat,
		Data:  data,
	}
	_, err := share.SSEBroker.BroadcastMessage(&msg)
	if err != nil {
		slog.Error("Error broadcasting health message: %w", "err", err)
		return tracerr.Wrap(err)
	}
	self.OutputEventsCount++
	return nil
}

func (self *Health) run() error {
	for {
		select {
		case <-self.ctx.Done():
			slog.Debug("Run process closed %w", "err", self.ctx.Err())
			return tracerr.Wrap(self.ctx.Err())
		default:
			// FIXME: Implement background process to retrieve all health information
			slog.Debug("Richiesto aggiornamento per Healthy")
			HealthAndUpdateDataRefeshHandlers(self.ctx)
			self.HealthPing = *healthData
			err := self.EventEmitter(self.ctx, self.HealthPing)
			if err != nil {
				slog.Error("Error emitting health message: %w", "err", err)
			}
			time.Sleep(self.OutputEventsInterleave)
			slog.Debug("Done")
		}
	}
}
