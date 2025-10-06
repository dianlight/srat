package api_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/service"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
)

type WsHandlerSuite struct {
	suite.Suite
}

func TestWsHandlerSuite(t *testing.T) { suite.Run(t, new(WsHandlerSuite)) }

func (suite *WsHandlerSuite) TestWebSocketUpgrade() {
	// Use a real broadcaster mock but we only need upgrade to succeed
	// Provide a MockController to create the mock broadcaster
	ctrl := mock.NewMockController(suite.T())
	mockBroadcaster := mock.Mock[service.BroadcasterServiceInterface](ctrl)

	// Create handler
	h := api.NewWebSocketBroker(context.Background(), mockBroadcaster)

	// Register on router
	r := mux.NewRouter()
	h.RegisterWs(r)

	srv := httptest.NewServer(r)
	defer srv.Close()

	// Dial websocket
	url := "ws" + srv.URL[len("http"):] + "/ws"
	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.Dial(url, nil)
	if err != nil {
		// Some environments may block websockets; assert response instead
		suite.Failf("Dial failed", "err=%v, resp=%v", err, resp)
		return
	}
	defer conn.Close()
	suite.NotNil(conn)
}
