package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server/ws"
	"github.com/dianlight/srat/tlog"
	"github.com/teivah/broadcast"

	"github.com/danielgtaylor/huma/v2/sse"
)

type BroadcasterServiceInterface interface {
	BroadcastMessage(msg any) any
	ProcessHttpChannel(send sse.Sender)
	ProcessWebSocketChannel(send ws.Sender)
}

type BroadcasterService struct {
	ctx              context.Context
	state            *dto.ContextState
	SentCounter      atomic.Uint64
	ConnectedClients atomic.Int32
	relay            *broadcast.Relay[broadcastEvent]
	haService        HomeAssistantServiceInterface
	haRootService    HaRootServiceInterface
}

type broadcastEvent struct {
	ID      uint64
	Message any
}

func NewBroadcasterService(
	ctx context.Context,
	haService HomeAssistantServiceInterface,
	haRootService HaRootServiceInterface,
	state *dto.ContextState,
) (broker BroadcasterServiceInterface) {
	// Instantiate a broker
	return &BroadcasterService{
		ctx:           ctx,
		relay:         broadcast.NewRelay[broadcastEvent](),
		haService:     haService,
		state:         state,
		SentCounter:   atomic.Uint64{},
		haRootService: haRootService,
	}
}

func (broker *BroadcasterService) BroadcastMessage(msg any) any {
	if _, ok := msg.(dto.HealthPing); !ok {
		tlog.Trace("Queued Message", "type", fmt.Sprintf("%T", msg), "msg", msg)
	}
	defer broker.SentCounter.Add(1)
	broker.relay.Broadcast(broadcastEvent{ID: broker.SentCounter.Load(), Message: msg})

	// Send to Home Assistant if in secure mode
	go broker.sendToHomeAssistant(msg)

	return msg
}

func (broker *BroadcasterService) sendToHomeAssistant(msg any) {
	if broker.haService == nil || !broker.state.HACoreReady {
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
		tlog.Trace("Skipping Home Assistant entity update for unsupported message type", "type", fmt.Sprintf("%T", msg), "msg", msg)
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

	err := send(sse.Message{
		ID:    0,
		Retry: 1000,
		Data:  broker.createWelcomeMessage(),
	})
	if err != nil {
		slog.Warn("Error sending welcome message to SSE client", "err", err)
	}

	for {
		select {
		case <-broker.ctx.Done():
			slog.Info("SSE Process Closed", "err", broker.ctx.Err(), "active clients", broker.ConnectedClients.Load())
			return
		case event := <-listener.Ch():
			// Filter out Home Assistant-specific events that shouldn't go to SSE clients
			if broker.shouldSkipSSEEvent(event.Message) {
				continue
			}

			err := send(sse.Message{
				ID:    int(event.ID),
				Retry: 1000,
				Data:  event.Message,
			})
			if err != nil {
				/* 				if !strings.Contains(err.Error(), "broken pipe") &&
				!strings.Contains(err.Error(), "context canceled") &&
				!strings.Contains(err.Error(), "connection reset by peer") &&
				!strings.Contains(err.Error(), "i/o timeout") {
				*/slog.Warn("Error sending event to client", "event", event, "err", err, "active clients", broker.ConnectedClients.Load())
				//				}
				return
			}
		}
	}
}

// ProcessWebSocketChannel processes a WebSocket connection for real-time events.
// It filters out Home Assistant-specific events that should not be sent to web clients
// and only sends events that are registered with the WebSocket system.
func (broker *BroadcasterService) ProcessWebSocketChannel(send ws.Sender) {
	broker.ConnectedClients.Add(1)
	defer broker.ConnectedClients.Add(-1)

	listener := broker.relay.Listener(5)
	defer listener.Close() // Close the listener when done

	slog.Debug("WebSocket Connected client", "actual clients", broker.ConnectedClients.Load())

	// Send welcome message
	err := send(ws.Message{
		ID:   0,
		Data: broker.createWelcomeMessage(),
	})
	if err != nil {
		slog.Warn("Error sending welcome message to SSE client", "err", err)
	}

	for {
		select {
		case <-broker.ctx.Done():
			slog.Info("WebSocket Process Closed", "err", broker.ctx.Err(), "active clients", broker.ConnectedClients.Load())
			return
		case event := <-listener.Ch():
			// Filter out Home Assistant-specific events that shouldn't go to WebSocket clients
			if broker.shouldSkipSSEEvent(event.Message) {
				continue
			}

			err := send(ws.Message{
				ID:   int(event.ID),
				Data: event.Message,
			})
			if err != nil {
				tlog.Trace("Error sending event to client", "event", event, "err", err, "active clients", broker.ConnectedClients.Load())
				return
			}
		}
	}
}

func (broker *BroadcasterService) createWelcomeMessage() dto.Welcome {
	welcomeMsg := dto.Welcome{
		Message:         "Welcome to SRAT WebSocket",
		ActiveClients:   broker.ConnectedClients.Load(),
		SupportedEvents: dto.EventTypes.All(),
		UpdateChannel:   broker.state.UpdateChannel.String(),
		ReadOnly:        broker.state.ReadOnlyMode,
		SecureMode:      broker.state.SecureMode,
		BuildVersion:    config.BuildVersion(),
		ProtectedMode:   broker.state.ProtectedMode,
		StartTime:       broker.state.StartTime.Unix(),
	}

	// Get machine_id from ha_root service if available
	if broker.haRootService != nil {
		sysInfo, err := broker.haRootService.GetSystemInfo()
		if err != nil {
			slog.Debug("Error getting system info for machine_id", "err", err)
			welcomeMsg.MachineId = nil
		} else if sysInfo != nil && sysInfo.MachineId != nil {
			welcomeMsg.MachineId = sysInfo.MachineId
		}
	}
	return welcomeMsg
}

// shouldSkipSSEEvent determines if an event should be skipped for web clients (SSE/WebSocket)
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
