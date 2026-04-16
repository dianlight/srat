package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server/ws"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/tlog"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

type WebSocketHandler struct {
	ctx            context.Context
	broadcaster    service.BroadcasterServiceInterface
	repairService  service.RepairServiceInterface
	problemService service.ProblemServiceInterface
	haService      service.HomeAssistantServiceInterface
	haComponentSvc service.HomeAssistantComponentServiceInterface
	state          *dto.ContextState
	upgrader       websocket.Upgrader
	eventMap       map[string]any
	ObjectMap      map[string]string
}

type WebSocketHandlerParams struct {
	fx.In
	Ctx            context.Context
	Broadcaster    service.BroadcasterServiceInterface
	RepairService  service.RepairServiceInterface
	ProblemService service.ProblemServiceInterface                `optional:"true"`
	HAService      service.HomeAssistantServiceInterface          `optional:"true"`
	HAComponentSvc service.HomeAssistantComponentServiceInterface `optional:"true"`
	State          *dto.ContextState
}

func NewWebSocketBroker(p WebSocketHandlerParams) *WebSocketHandler {
	// Instantiate a WebSocket broker
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Allow connections from any origin in development
			// In production, you should check the origin
			return true
		},
	}

	reverseMap := reverseMap(dto.WebEventMap)

	return &WebSocketHandler{
		ctx:            p.Ctx,
		broadcaster:    p.Broadcaster,
		repairService:  p.RepairService,
		problemService: p.ProblemService,
		haService:      p.HAService,
		haComponentSvc: p.HAComponentSvc,
		state:          p.State,
		upgrader:       upgrader,
		eventMap:       dto.WebEventMap,
		ObjectMap:      reverseMap,
	}
}

func reverseMap(m map[string]any) map[string]string {
	n := make(map[string]string, len(m))
	for k, v := range m {
		n[reflect.TypeOf(v).String()] = k
	}
	return n
}

type WsMessageSender struct {
	Connection *websocket.Conn
	ObjectMap  map[string]string
	Mutex      sync.Mutex
}

func (self *WsMessageSender) writeMessage(messageType int, data []byte) errors.E {
	self.Mutex.Lock()
	defer self.Mutex.Unlock()

	if self.Connection == nil {
		return errors.New("WebSocket connection is nil")
	}

	err := self.Connection.WriteMessage(messageType, data)
	if err != nil {
		return errors.WithDetails(err, "message", "Failed to write message to WebSocket")
	}

	return nil
}

func (self *WsMessageSender) SendFunc(msg ws.Message) (retErr errors.E) {
	defer func() {
		if recovered := recover(); recovered != nil {
			retErr = errors.WithDetails(
				errors.Errorf("panic while sending WebSocket event: %v", recovered),
				"message", "Failed to marshal event data",
				"event_type", fmt.Sprintf("%T", msg.Data),
			)
		}
	}()

	safeData := dto.SanitizeWebEventData(msg.Data)
	eventData, err := json.Marshal(safeData)
	if err != nil {
		return errors.WithDetails(err, "message", "Failed to marshal event data", "event_type", fmt.Sprintf("%T", msg.Data))
	}

	typeName, ok := self.ObjectMap[reflect.TypeOf(msg.Data).String()]
	if !ok {
		return errors.Errorf("unknown event type for WebSocket: %T", msg.Data)
	}

	err = self.writeMessage(websocket.TextMessage, fmt.Appendf(nil, "id: %d\nevent: %s\ndata: %s\n\n", msg.ID, typeName, eventData))
	if err != nil {
		return errors.WithDetails(err, "message", "Failed to write message to WebSocket", "event", msg)
	}
	return nil
}

func (self *WsMessageSender) SendPing() errors.E {
	return self.writeMessage(websocket.PingMessage, nil)
}

func (self *WebSocketHandler) clearHomeAssistantComponentConnection() {
	if self.state == nil {
		return
	}

	self.state.HAWsComponent = nil
}

func (self *WebSocketHandler) setHomeAssistantComponentConnection(message dto.HeloMessage) {
	if self.state == nil {
		return
	}

	self.state.HAWsComponent = &dto.HomeAssistantComponentConnection{
		Component:   message.Component,
		Version:     message.Version,
		HAVersion:   message.HAVersion,
		EntryID:     message.EntryID,
		ConnectedAt: time.Now(),
	}

	if self.repairService != nil {
		for _, queued := range self.repairService.FlushQueuedCommands() {
			self.broadcaster.BroadcastMessage(queued)
		}
	}
}

func (self *WebSocketHandler) handleInboundMessage(messageType int, payload []byte) {
	if messageType != websocket.TextMessage {
		return
	}

	var envelope dto.ClientMessageEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		slog.WarnContext(self.ctx, "Ignoring invalid inbound WebSocket payload", "error", err)
		return
	}

	switch envelope.Type {
	case dto.ClientEventTypes.CLIENTEVENTTYPEHELO.String():
		var message dto.HeloMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			slog.WarnContext(self.ctx, "Ignoring malformed helo message", "error", err)
			return
		}
		if err := message.Validate(); err != nil {
			slog.WarnContext(self.ctx, "Ignoring invalid helo message", "error", err)
			return
		}
		self.setHomeAssistantComponentConnection(message)
		slog.InfoContext(self.ctx, "Home Assistant WebSocket handshake accepted", "component", message.Component, "version", message.Version, "entry_id", message.EntryID)
	case dto.ClientEventTypes.CLIENTEVENTTYPEREPAIRLIFECYCLE.String():
		var message dto.RepairLifecycleMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			slog.WarnContext(self.ctx, "Ignoring malformed repair lifecycle message", "error", err)
			return
		}
		if err := message.Validate(); err != nil {
			slog.WarnContext(self.ctx, "Ignoring invalid repair lifecycle message", "error", err)
			return
		}
		if self.repairService != nil {
			if _, err := self.repairService.ApplyLifecycle(message); err != nil && !errors.Is(err, dto.ErrorNotFound) {
				slog.WarnContext(self.ctx, "Failed to apply repair lifecycle to repair service", "error", err, "repair_id", message.RepairID)
			}
		}
		slog.DebugContext(self.ctx, "Accepted Home Assistant repair lifecycle message", "repair_id", message.RepairID, "status", message.Status, "command_id", message.CommandID)

		// When the user confirms the "restart required" repair fix, restart Home Assistant.
		if message.Status == dto.RepairLifecycleStatusFixed && self.haComponentSvc != nil {
			if err := self.haComponentSvc.DismissRestartRequiredRepair(self.ctx); err != nil {
				slog.WarnContext(self.ctx, "Failed to dismiss restart-required repair after fix", "error", err)
			}
			if self.haService != nil {
				if err := self.haService.RestartHomeAssistant(self.ctx); err != nil {
					slog.WarnContext(self.ctx, "Failed to restart Home Assistant after repair fix", "error", err)
				}
			}
		}
	case dto.ClientEventTypes.CLIENTEVENTTYPEPROBLEMLIFECYCLE.String():
		var message dto.ProblemLifecycleMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			slog.WarnContext(self.ctx, "Ignoring malformed problem lifecycle message", "error", err)
			return
		}
		if err := message.Validate(); err != nil {
			slog.WarnContext(self.ctx, "Ignoring invalid problem lifecycle message", "error", err)
			return
		}
		if self.problemService != nil {
			if _, err := self.problemService.ApplyLifecycle(message.ProblemKey, message.Status, message.Error); err != nil && !errors.Is(err, dto.ErrorNotFound) {
				slog.WarnContext(self.ctx, "Failed to apply problem lifecycle to problem service", "error", err, "problem_key", message.ProblemKey)
			}
		}
		slog.DebugContext(self.ctx, "Accepted Home Assistant problem lifecycle message", "problem_key", message.ProblemKey, "status", message.Status)

		if message.Status == dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED && message.ProblemKey == "custom_component_restart_required" && self.haComponentSvc != nil {
			if err := self.haComponentSvc.DismissRestartRequiredRepair(self.ctx); err != nil {
				slog.WarnContext(self.ctx, "Failed to dismiss restart-required problem after fix", "error", err)
			}
			if self.haService != nil {
				if err := self.haService.RestartHomeAssistant(self.ctx); err != nil {
					slog.WarnContext(self.ctx, "Failed to restart Home Assistant after problem fix", "error", err)
				}
			}
		}
	default:
		slog.WarnContext(self.ctx, "Ignoring unsupported inbound WebSocket message type", "type", envelope.Type)
	}
}

func (self *WebSocketHandler) readInboundMessages(conn *websocket.Conn, readErr chan<- error) {
	defer close(readErr)

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			readErr <- err
			return
		}

		if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			readErr <- err
			return
		}
		self.handleInboundMessage(messageType, payload)
	}
}

// HandleWebSocket handles the WebSocket upgrade and connection
// This method should be called from an HTTP handler that matches the /ws path
func (self *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			slog.ErrorContext(self.ctx, "WebSocket handler panicked", "panic", recoverErr)
		}
	}()

	if self.ctx.Err() != nil {
		http.Error(w, "service shutting down", http.StatusServiceUnavailable)
		return
	}

	conn, err := self.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.ErrorContext(self.ctx, "Failed to upgrade connection to WebSocket", "error", err)
		return
	}
	defer self.clearHomeAssistantComponentConnection()
	defer conn.Close()

	// Handle ping/pong for connection health
	if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		slog.WarnContext(self.ctx, "Failed to set initial WebSocket read deadline", "error", err)
		return
	}
	conn.SetPingHandler(func(appData string) error {
		if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			return err
		}
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second))
	})
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	slog.DebugContext(self.ctx, "WebSocket client connected")

	wsMessageSender := &WsMessageSender{
		Connection: conn,
		ObjectMap:  self.ObjectMap,
		Mutex:      sync.Mutex{},
	}
	if self.ctx.Err() != nil {
		slog.DebugContext(self.ctx, "Skipping WebSocket channel processing during shutdown")
		return
	}

	go self.broadcaster.ProcessWebSocketChannel(wsMessageSender.SendFunc)
	readErr := make(chan error, 1)
	go self.readInboundMessages(conn, readErr)

	// Start ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-self.ctx.Done():
			return
		case err, ok := <-readErr:
			if !ok {
				return
			}
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) || errors.Is(err, io.EOF) {
				tlog.TraceContext(self.ctx, "WebSocket client disconnected", "err", err)
				return
			}
			tlog.TraceContext(self.ctx, "Inbound WebSocket reader stopped", "err", err)
			return
		case <-pingTicker.C:
			// Send ping to keep connection alive
			err := wsMessageSender.SendPing()
			if err != nil {
				tlog.TraceContext(self.ctx, "Error sending ping to WebSocket client", "err", err)
				return
			}
		}
	}
}

func (self *WebSocketHandler) RegisterWs(router *mux.Router) {
	router.HandleFunc("/ws", self.HandleWebSocket)
}
