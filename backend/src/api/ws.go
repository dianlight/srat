package api

import (
	"context"
	"encoding/json"
	"fmt"
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
)

type WebSocketHandler struct {
	ctx         context.Context
	broadcaster service.BroadcasterServiceInterface
	upgrader    websocket.Upgrader
	eventMap    map[string]any
	ObjectMap   map[string]string
}

func NewWebSocketBroker(ctx context.Context, broadcaster service.BroadcasterServiceInterface) *WebSocketHandler {
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
	eventMap := map[string]any{
		dto.WebEventTypes.EVENTHELLO.String():        dto.Welcome{},
		dto.WebEventTypes.EVENTUPDATING.String():     dto.UpdateProgress{},
		dto.WebEventTypes.EVENTVOLUMES.String():      []*dto.Disk{},
		dto.WebEventTypes.EVENTHEARTBEAT.String():    dto.HealthPing{},
		dto.WebEventTypes.EVENTSHARES.String():       []dto.SharedResource{},
		dto.WebEventTypes.EVENTDIRTYTRACKER.String(): dto.DataDirtyTracker{},
	}

	reverseMap := reverseMap(eventMap)

	return &WebSocketHandler{
		ctx:         ctx,
		broadcaster: broadcaster,
		upgrader:    upgrader,
		eventMap:    eventMap,
		ObjectMap:   reverseMap,
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

func (self *WsMessageSender) SendFunc(msg ws.Message) errors.E {
	self.Mutex.Lock()
	defer self.Mutex.Unlock()

	eventData, err := json.Marshal(msg.Data)
	if err != nil {
		return errors.WithDetails(err, "message", "Failed to marshal event data", "event", msg)
	}

	typeName, ok := self.ObjectMap[reflect.TypeOf(msg.Data).String()]
	if !ok {
		return errors.Errorf("unknown event type for WebSocket: %T", msg.Data)
	}

	if self.Connection == nil {
		return errors.New("WebSocket connection is nil")
	}
	err = self.Connection.WriteMessage(websocket.TextMessage,
		fmt.Appendf(nil, "id: %d\nevent: %s\ndata: %s\n\n", msg.ID, typeName, eventData),
	)
	if err != nil {
		return errors.WithDetails(err, "message", "Failed to write message to WebSocket", "event", msg)
	}
	return nil
}

// HandleWebSocket handles the WebSocket upgrade and connection
// This method should be called from an HTTP handler that matches the /ws path
func (self *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := self.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.ErrorContext(self.ctx, "Failed to upgrade connection to WebSocket", "error", err)
		return
	}
	defer conn.Close()

	// Handle ping/pong for connection health
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	slog.DebugContext(self.ctx, "WebSocket client connected")

	wsMessageSender := &WsMessageSender{
		Connection: conn,
		ObjectMap:  self.ObjectMap,
		Mutex:      sync.Mutex{},
	}

	self.broadcaster.ProcessWebSocketChannel(wsMessageSender.SendFunc)

	// Start ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-self.ctx.Done():
			return
		case <-pingTicker.C:
			// Send ping to keep connection alive
			err := conn.WriteMessage(websocket.PingMessage, nil)
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
