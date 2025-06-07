package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/dianlight/srat/dto"

	"github.com/danielgtaylor/huma/v2/sse"
)

type BroadcasterServiceInterface interface {
	BroadcastMessage(msg any) (any, error)
	ProcessHttpChannel(send sse.Sender)
}

type BroadcasterService struct {
	ctx         context.Context
	notifier    chan any
	SentCounter int64
	DropCounter int64
}

func NewBroadcasterService(ctx context.Context) (broker BroadcasterServiceInterface) {
	// Instantiate a broker
	rbroker := &BroadcasterService{
		ctx:         ctx,
		notifier:    make(chan any, 10),
		SentCounter: 0,
		DropCounter: 0,
	}

	broker = rbroker
	return
}

func (broker *BroadcasterService) BroadcastMessage(msg any) (any, error) {
	select {
	case broker.notifier <- msg:
		broker.SentCounter++
		if _, ok := msg.(dto.HealthPing); !ok {
			slog.Debug("Queued Message", "msg", msg)
		}
	default:
		slog.Debug("Dropped Message", "msg", msg)
		broker.DropCounter++
	}
	return msg, nil
}

func (broker *BroadcasterService) ProcessHttpChannel(send sse.Sender) {
	for {
		select {
		case <-broker.ctx.Done():
			slog.Info("Run process closed", "err", broker.ctx.Err())
			return
		case event := <-broker.notifier:
			err := send.Data(event)
			if err != nil {
				if !strings.Contains(err.Error(), "write: broken pipe") {
					slog.Warn("Error sending event to client", "event", event, "err", err)
				}
				return
			}
		}
	}
}
