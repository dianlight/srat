package api_test

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type WsHandlerSuite struct {
	suite.Suite
	wg              *sync.WaitGroup
	app             *fxtest.App
	ctx             context.Context
	cancel          context.CancelFunc
	mockBroadcaster service.BroadcasterServiceInterface
}

func TestWsHandlerSuite(t *testing.T) { suite.Run(t, new(WsHandlerSuite)) }

func (suite *WsHandlerSuite) SetupTest() {
	suite.wg = &sync.WaitGroup{}

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), "wg", suite.wg)
				return context.WithCancel(ctx)
			},
			func() *dto.ContextState { return &dto.ContextState{} },
			service.NewUpgradeService,
			service.NewBroadcasterService,
			events.NewEventBus,
			mock.Mock[service.HomeAssistantServiceInterface],
			mock.Mock[service.HaRootServiceInterface],
			mock.Mock[service.VolumeServiceInterface],
			///mock.Mock[service.BroadcasterServiceInterface],
		),
		fx.Populate(&suite.ctx, &suite.cancel),
		fx.Populate(&suite.mockBroadcaster),
	)

	suite.app.RequireStart()
}

func (suite *WsHandlerSuite) TestWebSocketUpgrade() {

	// Create handler
	h := api.NewWebSocketBroker(suite.ctx, suite.mockBroadcaster)

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

// Test that the WebSocket handler invokes the broadcaster's ProcessWebSocketChannel
// and the client receives the welcome message and a subsequent event.
func (suite *WsHandlerSuite) TestWebSocketReceivesMessagesFromBroadcaster() {
	// Use a real BroadcasterService so we can broadcast messages into the WebSocket handler.

	h := api.NewWebSocketBroker(suite.ctx, suite.mockBroadcaster)
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
	suite.mockBroadcaster.BroadcastMessage(dto.UpdateProgress{Progress: 42, LastRelease: "v1.2.3"})

	_, msg2, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(msg2), "event: updating")
	suite.Contains(string(msg2), "v1.2.3")
}
