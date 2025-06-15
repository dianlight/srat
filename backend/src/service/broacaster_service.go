package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/tlog"
	"github.com/teivah/broadcast"

	"github.com/danielgtaylor/huma/v2/sse"
)

type BroadcasterServiceInterface interface {
	BroadcastMessage(msg any) (any, error)
	ProcessHttpChannel(send sse.Sender)
}

type BroadcasterService struct {
	ctx              context.Context
	SentCounter      atomic.Uint64
	ConnectedClients atomic.Int32
	relay            *broadcast.Relay[any]
}

func NewBroadcasterService(ctx context.Context) (broker BroadcasterServiceInterface) {
	// Instantiate a broker
	return &BroadcasterService{
		ctx:   ctx,
		relay: broadcast.NewRelay[any](),
	}
}

func (broker *BroadcasterService) BroadcastMessage(msg any) (any, error) {
	if _, ok := msg.(dto.HealthPing); !ok {
		tlog.Trace("Queued Message", "type", fmt.Sprintf("%T", msg), "msg", msg)
	}
	defer broker.SentCounter.Add(1)
	broker.relay.Broadcast(msg)

	return msg, nil
}

func (broker *BroadcasterService) ProcessHttpChannel(send sse.Sender) {
	broker.ConnectedClients.Add(1)
	defer broker.ConnectedClients.Add(-1)

	listener := broker.relay.Listener(5)
	defer listener.Close() // Close the listener when done

	slog.Debug("SSE Connected client", "actual clients", broker.ConnectedClients.Load())

	for {
		select {
		case <-broker.ctx.Done():
			slog.Info("SSE Progess Closed", "err", broker.ctx.Err())
			return
		case event := <-listener.Ch():
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
