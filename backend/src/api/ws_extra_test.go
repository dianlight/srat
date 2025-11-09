package api_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/suite"
)

type WsExtraSuite struct {
	suite.Suite
}

func TestWsExtraSuite(t *testing.T) { suite.Run(t, new(WsExtraSuite)) }

// Test that the WebSocket handler invokes the broadcaster's ProcessWebSocketChannel
// and the client receives the welcome message and a subsequent event.
func (suite *WsExtraSuite) TestWebSocketReceivesMessagesFromBroadcaster() {
	// Use a real BroadcasterService so we can broadcast messages into the WebSocket handler.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	state := &dto.ContextState{}
	broker := service.NewBroadcasterService(ctx, nil, nil, state, nil, nil)

	h := api.NewWebSocketBroker(ctx, broker)
	r := mux.NewRouter()
	h.RegisterWs(r)

	srv := httptest.NewServer(r)
	defer srv.Close()

	url := "ws" + srv.URL[len("http"):] + "/ws"
	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.Dial(url, nil)
	if err != nil {
		suite.Failf("Dial failed", "err=%v resp=%v", err, resp)
		return
	}
	defer conn.Close()

	// Read two text messages and assert they contain expected event names
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msg1, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(msg1), "event: hello")
	// Now broadcast an UpdateProgress message and ensure the client receives it
	broker.BroadcastMessage(dto.UpdateProgress{Progress: 42, LastRelease: "v1.2.3"})

	_, msg2, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(msg2), "event: updating")
	suite.Contains(string(msg2), "v1.2.3")
}
