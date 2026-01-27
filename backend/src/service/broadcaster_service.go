package service

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/server/ws"
	"github.com/dianlight/tlog"
	"github.com/teivah/broadcast"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"

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
	eventBus         events.EventBusInterface
	volumeService    VolumeServiceInterface
	shareService     ShareServiceInterface
}

type broadcastEvent struct {
	ID      uint64
	Message any
}

func NewBroadcasterService(
	lc fx.Lifecycle,
	ctx context.Context,
	haService HomeAssistantServiceInterface,
	haRootService HaRootServiceInterface,
	state *dto.ContextState,
	eventBus events.EventBusInterface,
	volumeService VolumeServiceInterface,
	shareService ShareServiceInterface,
) (broker BroadcasterServiceInterface) {
	// Instantiate a broker
	b := &BroadcasterService{
		ctx:           ctx,
		relay:         broadcast.NewRelay[broadcastEvent](),
		haService:     haService,
		state:         state,
		SentCounter:   atomic.Uint64{},
		haRootService: haRootService,
		eventBus:      eventBus,
		volumeService: volumeService,
		shareService:  shareService,
	}

	unsubscribe := b.setupEventListeners()

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			tlog.TraceContext(ctx, "Starting BroadcasterService")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			tlog.TraceContext(ctx, "Stopping BroadcasterService")
			for _, unsub := range unsubscribe {
				unsub()
			}
			b.relay.Close()
			return nil
		},
	})

	return b
}

func (broker *BroadcasterService) setupEventListeners() []func() {
	ret := make([]func(), 6)
	// Listen for disk events
	ret[0] = broker.eventBus.OnDisk(func(ctx context.Context, event events.DiskEvent) errors.E {
		diskID := "unknown"
		if event.Disk.Id != nil {
			diskID = *event.Disk.Id
		}
		slog.DebugContext(ctx, "BroadcasterService received Disk event", "disk", diskID)
		broker.BroadcastMessage(broker.volumeService.GetVolumesData())
		return nil
	})
	// Listen for share events
	ret[1] = broker.eventBus.OnShare(func(ctx context.Context, event events.ShareEvent) errors.E {
		slog.DebugContext(ctx, "BroadcasterService received Share event", "share", event.Share.Name)
		shares, err := broker.shareService.ListShares()
		if err != nil {
			slog.ErrorContext(ctx, "Failed to list shares for broadcasting", "error", err)
			return nil
		}
		broker.BroadcastMessage(shares)
		return nil
	})

	// Listen for mount point events
	ret[2] = broker.eventBus.OnMountPoint(func(ctx context.Context, event events.MountPointEvent) errors.E {
		slog.DebugContext(ctx, "BroadcasterService received MountPointMounted event", "mount_point", event.MountPoint.Path)
		broker.BroadcastMessage(broker.volumeService.GetVolumesData())
		return nil
	})
	ret[3] = broker.eventBus.OnDirtyData(func(ctx context.Context, dde events.DirtyDataEvent) errors.E {
		slog.DebugContext(ctx, "BroadcasterService received DirtyData event", "tracker", dde.DataDirtyTracker)
		broker.BroadcastMessage(dde.DataDirtyTracker) // TODO: implement push of dirty data status only
		return nil
	})
	ret[4] = broker.eventBus.OnSmart(func(ctx context.Context, event events.SmartEvent) errors.E {
		slog.DebugContext(ctx, "BroadcasterService received SmartTestStatus event", "status", event.SmartTestStatus)
		if event.SmartTestStatus.DiskId != "" {
			broker.BroadcastMessage(event.SmartTestStatus)
		}
		return nil
	})
	ret[5] = broker.eventBus.OnHomeAssistant(func(ctx context.Context, event events.HomeAssistantEvent) errors.E {
		if event.Type != events.EventTypes.ERROR {
			return nil
		}
		slog.DebugContext(ctx, "BroadcasterService received Error event", "error", event.Error)
		broker.BroadcastMessage(event.Error)
		return nil
	})

	return ret
}

func (broker *BroadcasterService) BroadcastMessage(msg any) any {

	if reflect.ValueOf(msg).Kind() == reflect.Ptr {
		if reflect.ValueOf(msg).IsNil() {
			tlog.WarnContext(broker.ctx, "Attempted to broadcast nil pointer message", "type", fmt.Sprintf("%T", msg))
			return msg
		}
	}

	if _, ok := msg.(dto.HealthPing); !ok {
		tlog.TraceContext(broker.ctx, "Queued Message", "type", fmt.Sprintf("%T", msg), "msg", msg)
	}
	defer broker.SentCounter.Add(1)
	broker.relay.Broadcast(broadcastEvent{ID: broker.SentCounter.Load(), Message: msg})

	// Send to Home Assistant if in secure mode
	go broker.sendToHomeAssistant(msg)

	return msg
}

func (broker *BroadcasterService) sendToHomeAssistant(msg any) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(broker.ctx, "Panic in sendToHomeAssistant", "panic", r)
		}
	}()

	if broker.haService == nil || !broker.state.HACoreReady {
		return
	}

	// Handle different message types
	switch v := msg.(type) {
	case *[]*dto.Disk:
		if err := broker.haService.SendDiskEntities(v); err != nil {
			slog.WarnContext(broker.ctx, "Failed to send disk entities to Home Assistant", "error", err)
		}
		if err := broker.haService.SendVolumeStatusEntity(v); err != nil {
			slog.WarnContext(broker.ctx, "Failed to send volume status entity to Home Assistant", "error", err)
		}
	case []*dto.Disk:
		if err := broker.haService.SendDiskEntities(&v); err != nil {
			slog.WarnContext(broker.ctx, "Failed to send disk entities to Home Assistant", "error", err)
		}
		if err := broker.haService.SendVolumeStatusEntity(&v); err != nil {
			slog.WarnContext(broker.ctx, "Failed to send volume status entity to Home Assistant", "error", err)
		}
	case *dto.DiskHealth:
		if err := broker.haService.SendDiskHealthEntities(v); err != nil {
			slog.WarnContext(broker.ctx, "Failed to send disk health entities to Home Assistant", "error", err)
		}
	case dto.DiskHealth:
		if err := broker.haService.SendDiskHealthEntities(&v); err != nil {
			slog.WarnContext(broker.ctx, "Failed to send disk health entities to Home Assistant", "error", err)
		}
	case *dto.SambaStatus:
		if err := broker.haService.SendSambaStatusEntity(v); err != nil {
			slog.WarnContext(broker.ctx, "Failed to send samba status entity to Home Assistant", "error", err)
		}
	case dto.SambaStatus:
		if err := broker.haService.SendSambaStatusEntity(&v); err != nil {
			slog.WarnContext(broker.ctx, "Failed to send samba status entity to Home Assistant", "error", err)
		}
	case *dto.ServerProcessStatus:
		if err := broker.haService.SendSambaProcessStatusEntity(v); err != nil {
			slog.WarnContext(broker.ctx, "Failed to send samba process status entity to Home Assistant", "error", err)
		}
	case dto.ServerProcessStatus:
		if err := broker.haService.SendSambaProcessStatusEntity(&v); err != nil {
			slog.WarnContext(broker.ctx, "Failed to send samba process status entity to Home Assistant", "error", err)
		}
	default:
		tlog.TraceContext(broker.ctx, "Skipping Home Assistant entity update for unsupported message type", "type", fmt.Sprintf("%T", msg), "msg", msg)
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

	slog.DebugContext(broker.ctx, "SSE Connected client", "actual clients", broker.ConnectedClients.Load())

	err := send(sse.Message{
		ID:    0,
		Retry: 1000,
		Data:  broker.createWelcomeMessage(),
	})
	if err != nil {
		slog.WarnContext(broker.ctx, "Error sending welcome message to SSE client", "err", err)
	}

	for {
		select {
		case <-broker.ctx.Done():
			slog.InfoContext(broker.ctx, "SSE Process Closed", "err", broker.ctx.Err(), "active clients", broker.ConnectedClients.Load())
			return
		case event := <-listener.Ch():
			// Filter out Home Assistant-specific events that shouldn't go to SSE clients
			if dto.WebEventMap.IsValidEvent(event.Message) {
				continue
			}

			err := send(sse.Message{
				ID:    int(event.ID),
				Retry: 1000,
				Data:  event.Message,
			})
			if err != nil {
				slog.WarnContext(broker.ctx, "Error sending event to client", "event", event, "err", err, "active clients", broker.ConnectedClients.Load())
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

	slog.DebugContext(broker.ctx, "WebSocket Connected client", "actual clients", broker.ConnectedClients.Load())

	// Send welcome message
	err := send(ws.Message{
		ID:   0,
		Data: broker.createWelcomeMessage(),
	})
	if err != nil {
		slog.WarnContext(broker.ctx, "Error sending welcome message to SSE client", "err", err)
	}

	for {
		select {
		case <-broker.ctx.Done():
			slog.InfoContext(broker.ctx, "WebSocket Process Closed", "err", broker.ctx.Err(), "active clients", broker.ConnectedClients.Load())
			return
		case event := <-listener.Ch():
			// Filter out Home Assistant-specific events that shouldn't go to WebSocket clients
			if dto.WebEventMap.IsValidEvent(event.Message) {
				err := send(ws.Message{
					ID:   int(event.ID),
					Data: event.Message,
				})
				if err != nil {
					if !strings.Contains(err.Error(), ": broken pipe") {
						tlog.DebugContext(broker.ctx, "Error sending event to client", "event", event, "err", err, "active clients", broker.ConnectedClients.Load())
					}
					return
				}
			}
		}
	}
}

func (broker *BroadcasterService) createWelcomeMessage() dto.Welcome {
	welcomeMsg := dto.Welcome{
		Message:         "Welcome to SRAT WebSocket",
		ActiveClients:   broker.ConnectedClients.Load(),
		SupportedEvents: dto.WebEventTypes.All(),
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
			slog.DebugContext(broker.ctx, "Error getting system info for machine_id", "err", err)
			welcomeMsg.MachineId = nil
		} else if sysInfo != nil && sysInfo.MachineId != nil {
			welcomeMsg.MachineId = sysInfo.MachineId
		}
	}
	return welcomeMsg
}
