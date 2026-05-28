package service

import (
	"context"
	"log/slog"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// MDNSServiceInterface is the public contract for the mDNS registration service.
// It is called by the WebSocket broker when the Home Assistant custom component
// connects or disconnects via the HELO handshake.
type MDNSServiceInterface interface {
	// OnComponentConnected is called when the HA custom component sends a valid
	// HELO message. It broadcasts the current mDNS registration state so the
	// component can register or unregister the Samba service in Zeroconf.
	OnComponentConnected(message dto.HeloMessage)
	// OnComponentDisconnected is called when the WebSocket connection to the HA
	// custom component is closed. Per the chosen timeout semantics (Option B),
	// no disable message is queued — the component will re-sync on the next
	// HELO when it reconnects.
	OnComponentDisconnected()
}

// mdnsServiceParams groups all dependencies required by MDNSService via fx.In.
type mdnsServiceParams struct {
	fx.In
	Ctx            context.Context
	Broadcaster    BroadcasterServiceInterface
	SettingService SettingServiceInterface
	EventBus       events.EventBusInterface
}

// MDNSService broadcasts MdnsRegisterNotification events over WebSocket so the
// Home Assistant custom component can register or unregister the Samba server
// via Zeroconf / mDNS.
type MDNSService struct {
	ctx            context.Context
	broadcaster    BroadcasterServiceInterface
	settingService SettingServiceInterface
	eventBus       events.EventBusInterface

	// connected tracks whether a HA component is currently connected.
	connected bool
}

// NewMDNSService constructs the MDNSService and wires lifecycle / event hooks.
func NewMDNSService(lc fx.Lifecycle, params mdnsServiceParams) MDNSServiceInterface {
	svc := &MDNSService{
		ctx:            params.Ctx,
		broadcaster:    params.Broadcaster,
		settingService: params.SettingService,
		eventBus:       params.EventBus,
	}

	unsubscribes := svc.setupEventListeners()

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			for _, unsub := range unsubscribes {
				unsub()
			}
			return nil
		},
	})

	return svc
}

// setupEventListeners subscribes to the CLEAN server-process event so that
// settings changes are re-broadcast to the connected component.
func (svc *MDNSService) setupEventListeners() []func() {
	ret := make([]func(), 1)
	ret[0] = svc.eventBus.OnServerProccess(func(ctx context.Context, event events.ServerProcessEvent) errors.E {
		if event.Type == events.EventTypes.CLEAN && svc.connected {
			svc.broadcast(ctx)
		}
		return nil
	})
	return ret
}

// OnComponentConnected is called by the WebSocket broker after a successful HELO
// handshake. It immediately broadcasts the current mDNS registration state.
func (svc *MDNSService) OnComponentConnected(message dto.HeloMessage) {
	svc.connected = true
	svc.broadcast(svc.ctx)
}

// OnComponentDisconnected is called when the WebSocket connection drops.
// Per Option B timeout semantics, no disable message is sent — the component
// will re-sync when it reconnects.
func (svc *MDNSService) OnComponentDisconnected() {
	svc.connected = false
}

// broadcast reads the current settings and emits a MdnsRegisterNotification.
func (svc *MDNSService) broadcast(ctx context.Context) {
	settings, err := svc.settingService.Load()
	if err != nil {
		slog.ErrorContext(ctx, "mdns_service: failed to load settings", "err", err)
		return
	}

	enabled := false
	if settings.MDNSRegistration != nil {
		enabled = *settings.MDNSRegistration
	}

	notification := dto.MdnsRegisterNotification{
		Hostname: settings.Hostname,
		Port:     445, // Standard SMB/CIFS port
		Enabled:  enabled,
	}

	svc.broadcaster.BroadcastMessage(notification)
}
