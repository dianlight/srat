package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server/ws"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/tlog"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"gitlab.com/tozd/go/errors"
)

type WebSocketHandler struct {
	ctx         context.Context
	broadcaster service.BroadcasterServiceInterface
	upgrader    websocket.Upgrader
	eventMap    map[string]any
	reverseMap  map[string]string
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
		dto.WebEventTypes.EVENTHELLO.String():     dto.Welcome{},
		dto.WebEventTypes.EVENTUPDATING.String():  dto.UpdateProgress{},
		dto.WebEventTypes.EVENTVOLUMES.String():   []dto.Disk{},
		dto.WebEventTypes.EVENTHEARTBEAT.String(): dto.HealthPing{},
		dto.WebEventTypes.EVENTSHARE.String():     []dto.SharedResource{},
	}

	reverseMap := reverseMap(eventMap)

	return &WebSocketHandler{
		ctx:         ctx,
		broadcaster: broadcaster,
		upgrader:    upgrader,
		eventMap:    eventMap,
		reverseMap:  reverseMap,
	}
}

func reverseMap(m map[string]any) map[string]string {
	n := make(map[string]string, len(m))
	for k, v := range m {
		n[reflect.TypeOf(v).Name()] = k
	}
	return n
}

// HandleWebSocket handles the WebSocket upgrade and connection
// This method should be called from an HTTP handler that matches the /ws path
func (self *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := self.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection to WebSocket", "error", err)
		return
	}
	defer conn.Close()

	// Handle ping/pong for connection health
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	slog.Debug("WebSocket client connected")

	self.broadcaster.ProcessWebSocketChannel(func(msg ws.Message) errors.E {

		// Marshal the event data to JSON
		eventData, err := json.Marshal(msg.Data)
		if err != nil {
			return errors.WithDetails(err, "message", "Failed to marshal event data", "event", msg)
		}

		typeName, ok := self.reverseMap[reflect.TypeOf(msg.Data).Name()]
		if !ok {
			return errors.Errorf("unknown event type for WebSocket: %T", msg.Data)
		}

		// Send the event data
		err = conn.WriteMessage(websocket.TextMessage,
			[]byte(fmt.Sprintf("id: %d\nevent: %s\ndata: %s\n\n", msg.ID, typeName, eventData)),
		)
		if err != nil {
			return errors.WithDetails(err, "message", "Failed to write message to WebSocket", "event", msg)
		}
		return nil
	})

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
				tlog.Trace("Error sending ping to WebSocket client", "err", err)
				return
			}
		}
	}
}

func (self *WebSocketHandler) RegisterWs(router *mux.Router) {
	router.HandleFunc("/ws", self.HandleWebSocket)
}
