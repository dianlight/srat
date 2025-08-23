package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"go.uber.org/fx"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/websocket"
	"github.com/dianlight/srat/tlog"
)

// HaWsServiceInterface defines callbacks for Home Assistant lifecycle events
type HaWsServiceInterface interface {
	// OnHaStarted is invoked when Home Assistant signals it finished starting
	//OnHaStarted()
	// OnHaStopped is invoked when Home Assistant signals it will stop
	//OnHaStopped()
	// OnHaConnected is invoked when the websocket connection to Home Assistant is established
	//OnHaConnected()
	// OnHaDisconnected is invoked when the websocket connection to Home Assistant is lost
	//OnHaDisconnected()
	SubscribeToHaEvents(handler func(ready bool)) func()
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
	// subscribers for HACoreReady changes
	haReadyMu          sync.Mutex
	haReadySubscribers []func(ready bool)
}

type HaWsServiceParams struct {
	fx.In
	Ctx      context.Context
	State    *dto.ContextState
	WsClient websocket.ClientInterface `optional:"true"`
}

// NewHaWsService wires the service into FX lifecycle and subscribes to events when started
func NewHaWsService(lc fx.Lifecycle, params HaWsServiceParams) (HaWsServiceInterface, error) {
	s := &HaWsService{
		ctx:    params.Ctx,
		client: params.WsClient,
		state:  params.State,
	}

	// if websocket client is not available, nothing to subscribe
	if s.client == nil {
		slog.Debug("ha_ws_service: no websocket client provided; skipping subscriptions")
		return s, nil
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// subscribe to homeassistant_started
			unsubStart, err := s.client.SubscribeEvents(ctx, "homeassistant_started", func(ev json.RawMessage) {
				tlog.Trace("ha_ws_service: received homeassistant_started event")
				s.OnHaStarted()
			})
			if err != nil {
				slog.Warn("ha_ws_service: subscribe homeassistant_started failed", "error", err)
			} else {
				s.unsubStarted = unsubStart
			}

			// subscribe to homeassistant_stop (the docs mention homeassistant_stop/homeassistant_started)
			unsubStop, err2 := s.client.SubscribeEvents(ctx, "homeassistant_stop", func(ev json.RawMessage) {
				tlog.Trace("ha_ws_service: received homeassistant_stop event")
				s.OnHaStopped()
			})
			if err2 != nil {
				slog.Warn("ha_ws_service: subscribe homeassistant_stop failed", "error", err2)
			} else {
				s.unsubStopped = unsubStop
			}

			// subscribe to connection lifecycle events
			if s.client != nil {
				unsubC, err := s.client.SubscribeConnectionEvents(func(ev websocket.ConnectionEvent) {
					switch ev.Type {
					case websocket.ConnEventConnected:
						tlog.Trace("ha_ws_service: websocket connected")
						s.OnHaConnected()
					case websocket.ConnEventDisconnected:
						tlog.Trace("ha_ws_service: websocket disconnected")
						s.OnHaDisconnected()
					}
				})
				if err != nil {
					slog.Warn("ha_ws_service: subscribe connection events failed", "error", err)
				} else {
					s.unsubConn = unsubC
				}
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
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
			return nil
		},
	})

	return s, nil
}

// OnHaStarted default implementation logs the event. Consumers can type-assert
// the returned implementation to override behavior or embed this service.
func (s *HaWsService) OnHaStarted() {
	slog.Debug("ha_ws_service: OnHaStarted called")
	s.state.HACoreReady = true
	s.notifyHaReady(true)
}

// OnHaStopped default implementation logs the event.
func (s *HaWsService) OnHaStopped() {
	slog.Debug("ha_ws_service: OnHaStopped called")
	s.state.HACoreReady = false
	s.notifyHaReady(false)
}

// OnHaConnected default implementation logs the websocket connection event.
func (s *HaWsService) OnHaConnected() {
	slog.Debug("ha_ws_service: OnHaConnected called")
	s.state.HACoreReady = true
	s.notifyHaReady(true)
}

// OnHaDisconnected default implementation logs the websocket disconnection event.
func (s *HaWsService) OnHaDisconnected() {
	slog.Debug("ha_ws_service: OnHaDisconnected called")
	s.state.HACoreReady = false
	s.notifyHaReady(false)
}

// SubscribeToHaEvents registers a handler that will be called whenever the
// Home Assistant core readiness changes. The handler receives the new ready state.
// It returns an unsubscribe function that the caller should invoke to remove
// the handler when no longer needed.
func (s *HaWsService) SubscribeToHaEvents(handler func(ready bool)) func() {
	s.haReadyMu.Lock()
	defer s.haReadyMu.Unlock()
	s.haReadySubscribers = append(s.haReadySubscribers, handler)
	idx := len(s.haReadySubscribers) - 1

	// return unsubscribe func
	return func() {
		s.haReadyMu.Lock()
		defer s.haReadyMu.Unlock()
		// simple removal by index: preserve order by shifting
		if idx < 0 || idx >= len(s.haReadySubscribers) {
			return
		}
		copy(s.haReadySubscribers[idx:], s.haReadySubscribers[idx+1:])
		s.haReadySubscribers[len(s.haReadySubscribers)-1] = nil
		s.haReadySubscribers = s.haReadySubscribers[:len(s.haReadySubscribers)-1]
	}
}

// notifyHaReady calls all registered handlers with the new ready state.
func (s *HaWsService) notifyHaReady(ready bool) {
	s.haReadyMu.Lock()
	subs := make([]func(bool), len(s.haReadySubscribers))
	copy(subs, s.haReadySubscribers)
	s.haReadyMu.Unlock()

	for _, h := range subs {
		if h == nil {
			continue
		}
		// call handlers in goroutines to avoid blocking the service
		go func(fn func(bool)) {
			defer func() {
				// recover from panics in handlers
				_ = recover()
			}()
			fn(ready)
		}(h)
	}
}
