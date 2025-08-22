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
	haService        HomeAssistantServiceInterface
}

func NewBroadcasterService(ctx context.Context, haService HomeAssistantServiceInterface) (broker BroadcasterServiceInterface) {
	// Instantiate a broker
	return &BroadcasterService{
		ctx:       ctx,
		relay:     broadcast.NewRelay[any](),
		haService: haService,
	}
}

func (broker *BroadcasterService) BroadcastMessage(msg any) (any, error) {
	if _, ok := msg.(dto.HealthPing); !ok {
		tlog.Trace("Queued Message", "type", fmt.Sprintf("%T", msg), "msg", msg)
	}
	defer broker.SentCounter.Add(1)
	broker.relay.Broadcast(msg)

	// Send to Home Assistant if in secure mode
	broker.sendToHomeAssistant(msg)

	return msg, nil
}

func (broker *BroadcasterService) sendToHomeAssistant(msg any) {
	if broker.haService == nil {
		return
	}

	// Handle different message types
	switch v := msg.(type) {
	case *[]dto.Disk:
		if err := broker.haService.SendDiskEntities(v); err != nil {
			slog.Warn("Failed to send disk entities to Home Assistant", "error", err)
		}
		if err := broker.haService.SendVolumeStatusEntity(v); err != nil {
			slog.Warn("Failed to send volume status entity to Home Assistant", "error", err)
		}
	case []dto.Disk:
		if err := broker.haService.SendDiskEntities(&v); err != nil {
			slog.Warn("Failed to send disk entities to Home Assistant", "error", err)
		}
		if err := broker.haService.SendVolumeStatusEntity(&v); err != nil {
			slog.Warn("Failed to send volume status entity to Home Assistant", "error", err)
		}
	case *dto.DiskHealth:
		if err := broker.haService.SendDiskHealthEntities(v); err != nil {
			slog.Warn("Failed to send disk health entities to Home Assistant", "error", err)
		}
	case dto.DiskHealth:
		if err := broker.haService.SendDiskHealthEntities(&v); err != nil {
			slog.Warn("Failed to send disk health entities to Home Assistant", "error", err)
		}
	case *dto.SambaStatus:
		if err := broker.haService.SendSambaStatusEntity(v); err != nil {
			slog.Warn("Failed to send samba status entity to Home Assistant", "error", err)
		}
	case dto.SambaStatus:
		if err := broker.haService.SendSambaStatusEntity(&v); err != nil {
			slog.Warn("Failed to send samba status entity to Home Assistant", "error", err)
		}
	case *dto.SambaProcessStatus:
		if err := broker.haService.SendSambaProcessStatusEntity(v); err != nil {
			slog.Warn("Failed to send samba process status entity to Home Assistant", "error", err)
		}
	case dto.SambaProcessStatus:
		if err := broker.haService.SendSambaProcessStatusEntity(&v); err != nil {
			slog.Warn("Failed to send samba process status entity to Home Assistant", "error", err)
		}
	default:
		slog.Debug("Skipping Home Assistant entity update for unsupported message type", "type", fmt.Sprintf("%T", msg), "msg", msg)
	}
}

// ProcessHttpChannel processes an HTTP channel for Server-Sent Events (SSE).
// It filters out Home Assistant-specific events that should not be sent to web clients
// and only sends events that are registered with the SSE system.
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
			// Filter out Home Assistant-specific events that shouldn't go to SSE clients
			if broker.shouldSkipSSEEvent(event) {
				continue
			}

			err := send.Data(event)
			if err != nil {
				if !strings.Contains(err.Error(), "broken pipe") &&
					!strings.Contains(err.Error(), "context canceled") &&
					!strings.Contains(err.Error(), "connection reset by peer") &&
					!strings.Contains(err.Error(), "i/o timeout") {
					slog.Warn("Error sending event to client", "event", event, "err", err)
				}
				return
			}
		}
	}
}

// shouldSkipSSEEvent determines if an event should be skipped for SSE clients
// These events are meant for Home Assistant integration only
func (broker *BroadcasterService) shouldSkipSSEEvent(event any) bool {
	switch event.(type) {
	case dto.SambaStatus, *dto.SambaStatus:
		return true
	case dto.SambaProcessStatus, *dto.SambaProcessStatus:
		return true
	case dto.DiskHealth, *dto.DiskHealth:
		return true
	default:
		return false
	}
}
