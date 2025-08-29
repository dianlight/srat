package api

import (
	"log/slog"
	"net/http"

	"github.com/dianlight/srat/service"
	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	broadcaster service.BroadcasterServiceInterface
	upgrader    websocket.Upgrader
}

func NewWebSocketBroker(broadcaster service.BroadcasterServiceInterface) *WebSocketHandler {
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

	return &WebSocketHandler{
		broadcaster: broadcaster,
		upgrader:    upgrader,
	}
}

/*
func (self *WebSocketHandler) RegisterSystemHandler(api huma.API) {
	huma.Get(api, "/ws", self.HandleWebSocket, huma.OperationTags("system"))
}
*/
// HandleWebSocket handles the WebSocket upgrade and connection
// This method should be called from an HTTP handler that matches the /ws path
func (self *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := self.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection to WebSocket", "error", err)
		return
	}
	defer conn.Close()

	slog.Debug("WebSocket client connected")
	self.broadcaster.ProcessWebSocketChannel(conn)
}
