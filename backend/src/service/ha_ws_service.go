package service

import (
	"context"
	"encoding/json"
	"log/slog"

	"go.uber.org/fx"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/websocket"
	"github.com/dianlight/tlog"
)

// HaWsServiceInterface defines callbacks for Home Assistant lifecycle events
type HaWsServiceInterface interface {
}

// HaWsService is a lightweight service that subscribes to HA start/stop events
type HaWsService struct {
	ctx    context.Context
	client websocket.ClientInterface
	state  *dto.ContextState
	// optional unsubscribe functions
	unsubStarted func() error
	unsubStopped func() error
	unsubConn    func()
	//	haReadySubscribers []func(ready bool)
	EventBus events.EventBusInterface
}

type HaWsServiceParams struct {
	fx.In
	Ctx      context.Context
	State    *dto.ContextState
	WsClient websocket.ClientInterface `optional:"true"`
	EventBus events.EventBusInterface
}

// NewHaWsService wires the service into FX lifecycle and subscribes to events when started
func NewHaWsService(lc fx.Lifecycle, params HaWsServiceParams) (HaWsServiceInterface, error) {
	s := &HaWsService{
		ctx:      params.Ctx,
		client:   params.WsClient,
		state:    params.State,
		EventBus: params.EventBus,
	}

	// if websocket client is not available, nothing to subscribe
	if s.client == nil {
		slog.Debug("ha_ws_service: no websocket client provided; skipping subscriptions")
		return s, nil
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// subscribe to connection lifecycle events
			if s.client != nil {
				unsubC, err := s.client.SubscribeConnectionEvents(func(ev websocket.ConnectionEvent) {
					switch ev.Type {
					case websocket.ConnEventConnected:
						tlog.Trace("ha_ws_service: websocket connected")
						s.onHaConnected()
					case websocket.ConnEventDisconnected:
						tlog.Trace("ha_ws_service: websocket disconnected")
						s.onHaDisconnected()
					}
				})
				if err != nil {
					slog.Warn("ha_ws_service: subscribe connection events failed", "error", err)
				} else {
					s.unsubConn = unsubC
				}
				err = s.client.Connect(s.ctx)
				if err != nil {
					slog.Warn("ha_ws_service: websocket connection failed", "error", err)
					return err
				}
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.unsubscribeFromHaEvents()
			return nil
		},
	})

	return s, nil
}

// notifyHaReady calls all registered handlers with the new ready state.
func (s *HaWsService) onHaStarted() {
	slog.Debug("ha_ws_service: OnHaStarted called")
	s.state.HACoreReady = true
	s.subscribeToHaEvents()
	s.EventBus.EmitHomeAssistant(events.HomeAssistantEvent{
		Event: events.Event{
			Type: events.EventTypes.START,
		},
	})
}

// onHaStopped default implementation logs the event.
func (s *HaWsService) onHaStopped() {
	slog.Debug("ha_ws_service: OnHaStopped called")
	s.state.HACoreReady = false
	s.unsubscribeFromHaEvents()
	s.EventBus.EmitHomeAssistant(events.HomeAssistantEvent{
		Event: events.Event{
			Type: events.EventTypes.STOP,
		},
	})
}

// onHaConnected default implementation logs the websocket connection event.
func (s *HaWsService) onHaConnected() {
	slog.Debug("ha_ws_service: OnHaConnected called")
	s.state.HACoreReady = true
	s.subscribeToHaEvents()
	s.EventBus.EmitHomeAssistant(events.HomeAssistantEvent{
		Event: events.Event{
			Type: events.EventTypes.START,
		},
	})
}

// onHaDisconnected default implementation logs the websocket disconnection event.
func (s *HaWsService) onHaDisconnected() {
	slog.Debug("ha_ws_service: OnHaDisconnected called")
	s.state.HACoreReady = false
	s.unsubscribeFromHaEvents()
	s.EventBus.EmitHomeAssistant(events.HomeAssistantEvent{
		Event: events.Event{
			Type: events.EventTypes.STOP,
		},
	})
}

func (s *HaWsService) subscribeToHaEvents() {
	// subscribe to homeassistant_started
	unsubStart, err := s.client.SubscribeEvents(s.ctx, "homeassistant_started", func(ev json.RawMessage) {
		tlog.Trace("ha_ws_service: received homeassistant_started event")
		s.onHaStarted()
	})
	if err != nil {
		slog.Warn("ha_ws_service: subscribe homeassistant_started failed", "error", err)
	} else {
		s.unsubStarted = unsubStart
	}

	// subscribe to homeassistant_stop (the docs mention homeassistant_stop/homeassistant_started)
	unsubStop, err2 := s.client.SubscribeEvents(s.ctx, "homeassistant_stop", func(ev json.RawMessage) {
		tlog.Trace("ha_ws_service: received homeassistant_stop event")
		s.onHaStopped()
	})
	if err2 != nil {
		slog.Warn("ha_ws_service: subscribe homeassistant_stop failed", "error", err2)
	} else {
		s.unsubStopped = unsubStop
	}

}

func (s *HaWsService) unsubscribeFromHaEvents() {
	if s.unsubStarted != nil {
		_ = s.unsubStarted()
		s.unsubStarted = nil
	}
	if s.unsubStopped != nil {
		_ = s.unsubStopped()
		s.unsubStopped = nil
	}
	if s.unsubConn != nil {
		// unsubscribe connection events
		s.unsubConn()
		s.unsubConn = nil
	}
}
