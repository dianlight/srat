package api_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
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
	state           *dto.ContextState
	mockBroadcaster service.BroadcasterServiceInterface
	repairService   service.RepairServiceInterface
	mockProblemSvc  service.ProblemServiceInterface
	mockHAService   service.HomeAssistantServiceInterface
}

func TestWsHandlerSuite(t *testing.T) { suite.Run(t, new(WsHandlerSuite)) }

func (suite *WsHandlerSuite) SetupTest() {
	suite.wg = &sync.WaitGroup{}

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg)
				return context.WithCancel(ctx)
			},
			func() *dto.ContextState { return &dto.ContextState{} },
			service.NewUpgradeService,
			service.NewRepairService,
			service.NewBroadcasterService,
			events.NewEventBus,
			mock.Mock[service.ProblemServiceInterface],
			mock.Mock[service.HomeAssistantServiceInterface],
			mock.Mock[service.HaRootServiceInterface],
			func() *dto.DiskMap { return &dto.DiskMap{} },
			mock.Mock[service.ShareServiceInterface],
			///mock.Mock[service.BroadcasterServiceInterface],
		),
		fx.Populate(&suite.ctx, &suite.cancel, &suite.state),
		fx.Populate(&suite.mockBroadcaster),
		fx.Populate(&suite.repairService),
		fx.Populate(&suite.mockProblemSvc),
		fx.Populate(&suite.mockHAService),
	)

	suite.app.RequireStart()
}

func (suite *WsHandlerSuite) TestWebSocketUpgrade() {

	// Create handler
	h := api.NewWebSocketBroker(api.WebSocketHandlerParams{Ctx: suite.ctx, Broadcaster: suite.mockBroadcaster, RepairService: suite.repairService, State: suite.state})

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

	h := api.NewWebSocketBroker(api.WebSocketHandlerParams{Ctx: suite.ctx, Broadcaster: suite.mockBroadcaster, RepairService: suite.repairService, State: suite.state})
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
	suite.Require().NoError(conn.SetReadDeadline(time.Now().Add(1 * time.Second)))
	_, msg1, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(msg1), "event: hello")
	// Now broadcast an UpdateProgress message and ensure the client receives it
	suite.mockBroadcaster.BroadcastMessage(dto.UpdateProgress{Progress: 42, ReleaseAsset: &dto.ReleaseAsset{
		LastRelease: "v1.2.3",
		ArchAsset: dto.BinaryAsset{
			Name:               "srat-linux-amd64.zip",
			Size:               123456,
			ID:                 987654321,
			BrowserDownloadURL: "https://example.com/srat-linux-amd64.zip",
			Digest:             "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		},
	}, ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSCHECKING})

	_, msg2, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(msg2), "event: updating")
	suite.Contains(string(msg2), "v1.2.3")
}

func (suite *WsHandlerSuite) TestWebSocketAcceptsValidatedInboundHelo() {
	h := api.NewWebSocketBroker(api.WebSocketHandlerParams{Ctx: suite.ctx, Broadcaster: suite.mockBroadcaster, RepairService: suite.repairService, State: suite.state})
	r := mux.NewRouter()
	h.RegisterWs(r)

	srv := httptest.NewServer(r)
	defer srv.Close()

	url := "ws" + srv.URL[len("http"):] + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		suite.Failf("Dial failed", "err=%v resp=%v", err, resp)
		return
	}
	defer conn.Close()

	suite.Require().NoError(conn.SetReadDeadline(time.Now().Add(1 * time.Second)))
	_, msg1, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(msg1), "event: hello")

	err = conn.WriteJSON(dto.HeloMessage{
		Type:      dto.ClientEventTypes.CLIENTEVENTTYPEHELO.String(),
		Component: dto.HomeAssistantComponentSRAT,
		Version:   "2026.03.1",
	})
	suite.Require().NoError(err)
	suite.Eventually(func() bool {
		return suite.state.HAWsComponent != nil && suite.state.HAWsComponent.Version == "2026.03.1"
	}, time.Second, 10*time.Millisecond)
	suite.Require().NotNil(suite.state.HAWsComponent)
	suite.Equal(dto.HomeAssistantComponentSRAT, suite.state.HAWsComponent.Component)

	suite.mockBroadcaster.BroadcastMessage(dto.UpdateProgress{Progress: 7})
	_, msg2, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(msg2), "event: updating")
	suite.Contains(string(msg2), "\"progress\":7")

	err = conn.Close()
	suite.Require().NoError(err)
	suite.Eventually(func() bool {
		return suite.state.HAWsComponent == nil
	}, time.Second, 10*time.Millisecond)
}

func (suite *WsHandlerSuite) TestWebSocketIgnoresMalformedInboundPayload() {
	h := api.NewWebSocketBroker(api.WebSocketHandlerParams{Ctx: suite.ctx, Broadcaster: suite.mockBroadcaster, RepairService: suite.repairService, State: suite.state})
	r := mux.NewRouter()
	h.RegisterWs(r)

	srv := httptest.NewServer(r)
	defer srv.Close()

	url := "ws" + srv.URL[len("http"):] + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		suite.Failf("Dial failed", "err=%v resp=%v", err, resp)
		return
	}
	defer conn.Close()

	suite.Require().NoError(conn.SetReadDeadline(time.Now().Add(1 * time.Second)))
	_, msg1, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(msg1), "event: hello")

	err = conn.WriteMessage(websocket.TextMessage, []byte("{not-json"))
	suite.Require().NoError(err)
	time.Sleep(50 * time.Millisecond)
	suite.Nil(suite.state.HAWsComponent)

	suite.mockBroadcaster.BroadcastMessage(dto.UpdateProgress{Progress: 11})
	_, msg2, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(msg2), "event: updating")
	suite.Contains(string(msg2), fmt.Sprintf("\"progress\":%d", 11))
}

func (suite *WsHandlerSuite) TestWebSocketAcceptsValidatedInboundRepairLifecycle() {
	_, err := suite.repairService.Create(dto.RepairCommandMessage{
		CommandID:      "cmd-1",
		RepairID:       "disk_space_low",
		Action:         dto.RepairCommandActionUpsert,
		TranslationKey: "disk_space_low",
		Severity:       dto.RepairIssueSeverityWarning,
	})
	suite.Require().NoError(err)

	h := api.NewWebSocketBroker(api.WebSocketHandlerParams{Ctx: suite.ctx, Broadcaster: suite.mockBroadcaster, RepairService: suite.repairService, State: suite.state})
	r := mux.NewRouter()
	h.RegisterWs(r)

	srv := httptest.NewServer(r)
	defer srv.Close()

	url := "ws" + srv.URL[len("http"):] + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		suite.Failf("Dial failed", "err=%v resp=%v", err, resp)
		return
	}
	defer conn.Close()

	suite.Require().NoError(conn.SetReadDeadline(time.Now().Add(1 * time.Second)))
	_, welcomeMsg, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(welcomeMsg), "event: hello")

	err = conn.WriteJSON(dto.RepairLifecycleMessage{
		Type:      dto.ClientEventTypes.CLIENTEVENTTYPEREPAIRLIFECYCLE.String(),
		CommandID: "cmd-1",
		RepairID:  "disk_space_low",
		Status:    dto.RepairLifecycleStatusCreated,
	})
	suite.Require().NoError(err)

	suite.Eventually(func() bool {
		repair, ok := suite.repairService.Get("disk_space_low")
		return ok && repair != nil && repair.Lifecycle != nil &&
			repair.Lifecycle.RepairID == "disk_space_low" &&
			repair.Lifecycle.Status == dto.RepairLifecycleStatusCreated
	}, time.Second, 10*time.Millisecond)
}

func (suite *WsHandlerSuite) TestWebSocketRepairLifecycleSynchronizesRepairServiceState() {
	var noProblemError *string
	mock.When(
		suite.mockProblemSvc.ApplyLifecycle(
			mock.Exact("disk_space_low"),
			mock.Exact(dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSIGNORED),
			mock.Exact(noProblemError),
		),
	).ThenReturn(&dto.Problem{ProblemKey: "disk_space_low"}, nil)

	_, err := suite.repairService.Create(dto.RepairCommandMessage{
		CommandID:      "cmd-100",
		RepairID:       "disk_space_low",
		Action:         dto.RepairCommandActionUpsert,
		TranslationKey: "disk_space_low",
		Severity:       dto.RepairIssueSeverityWarning,
	})
	suite.Require().NoError(err)

	h := api.NewWebSocketBroker(api.WebSocketHandlerParams{Ctx: suite.ctx, Broadcaster: suite.mockBroadcaster, RepairService: suite.repairService, State: suite.state})
	r := mux.NewRouter()
	h.RegisterWs(r)

	srv := httptest.NewServer(r)
	defer srv.Close()

	url := "ws" + srv.URL[len("http"):] + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		suite.Failf("Dial failed", "err=%v resp=%v", err, resp)
		return
	}
	defer conn.Close()

	suite.Require().NoError(conn.SetReadDeadline(time.Now().Add(1 * time.Second)))
	_, welcomeMsg, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(welcomeMsg), "event: hello")

	err = conn.WriteJSON(dto.RepairLifecycleMessage{
		Type:      dto.ClientEventTypes.CLIENTEVENTTYPEREPAIRLIFECYCLE.String(),
		CommandID: "cmd-100",
		RepairID:  "disk_space_low",
		Status:    dto.RepairLifecycleStatusIgnored,
	})
	suite.Require().NoError(err)

	suite.Eventually(func() bool {
		repair, ok := suite.repairService.Get("disk_space_low")
		return ok && repair != nil && repair.Status == dto.RepairLifecycleStatusIgnored
	}, time.Second, 10*time.Millisecond)

	_, _ = mock.Verify(suite.mockProblemSvc, matchers.Times(1)).ApplyLifecycle(
		mock.Exact("disk_space_low"),
		mock.Exact(dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSIGNORED),
		mock.Exact(noProblemError),
	)
}

func (suite *WsHandlerSuite) TestWebSocketRepairLifecycleErrorSynchronizesProblemServiceState() {
	problemErr := "ha_apply_failed"
	problemLifecycleCalled := make(chan *string, 1)
	mock.When(
		suite.mockProblemSvc.ApplyLifecycle(
			mock.Exact("disk_space_low"),
			mock.Exact(dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSERROR),
			mock.Any[*string](),
		),
	).ThenAnswer(func(args []any) []any {
		errValue, _ := args[2].(*string)
		problemLifecycleCalled <- errValue
		return []any{&dto.Problem{ProblemKey: "disk_space_low"}, nil}
	})

	_, err := suite.repairService.Create(dto.RepairCommandMessage{
		CommandID:      "cmd-101",
		RepairID:       "disk_space_low",
		Action:         dto.RepairCommandActionUpsert,
		TranslationKey: "disk_space_low",
		Severity:       dto.RepairIssueSeverityWarning,
	})
	suite.Require().NoError(err)

	h := api.NewWebSocketBroker(api.WebSocketHandlerParams{Ctx: suite.ctx, Broadcaster: suite.mockBroadcaster, RepairService: suite.repairService, State: suite.state})
	r := mux.NewRouter()
	h.RegisterWs(r)

	srv := httptest.NewServer(r)
	defer srv.Close()

	url := "ws" + srv.URL[len("http"):] + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		suite.Failf("Dial failed", "err=%v resp=%v", err, resp)
		return
	}
	defer conn.Close()

	suite.Require().NoError(conn.SetReadDeadline(time.Now().Add(1 * time.Second)))
	_, welcomeMsg, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(welcomeMsg), "event: hello")

	err = conn.WriteJSON(dto.RepairLifecycleMessage{
		Type:      dto.ClientEventTypes.CLIENTEVENTTYPEREPAIRLIFECYCLE.String(),
		CommandID: "cmd-101",
		RepairID:  "disk_space_low",
		Status:    dto.RepairLifecycleStatusError,
		Error:     &problemErr,
	})
	suite.Require().NoError(err)

	suite.Eventually(func() bool {
		repair, ok := suite.repairService.Get("disk_space_low")
		return ok && repair != nil &&
			repair.Status == dto.RepairLifecycleStatusError &&
			repair.LastError != nil &&
			*repair.LastError == problemErr
	}, time.Second, 10*time.Millisecond)

	select {
	case errValue := <-problemLifecycleCalled:
		suite.Require().NotNil(errValue)
		suite.Equal(problemErr, *errValue)
	case <-time.After(time.Second):
		suite.Fail("timed out waiting for mirrored problem lifecycle call")
	}
}

func (suite *WsHandlerSuite) TestWebSocketFlushesQueuedRepairCommandsAfterHelo() {
	err := suite.repairService.EnqueueCommand(dto.RepairCommandMessage{
		CommandID:      "cmd-queued-1",
		RepairID:       "disk_space_low",
		Action:         dto.RepairCommandActionUpsert,
		TranslationKey: "disk_space_low",
		Severity:       dto.RepairIssueSeverityWarning,
		IsPersistent:   true,
	})
	suite.Require().NoError(err)

	h := api.NewWebSocketBroker(api.WebSocketHandlerParams{Ctx: suite.ctx, Broadcaster: suite.mockBroadcaster, RepairService: suite.repairService, State: suite.state})
	r := mux.NewRouter()
	h.RegisterWs(r)

	srv := httptest.NewServer(r)
	defer srv.Close()

	url := "ws" + srv.URL[len("http"):] + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		suite.Failf("Dial failed", "err=%v resp=%v", err, resp)
		return
	}
	defer conn.Close()

	suite.Require().NoError(conn.SetReadDeadline(time.Now().Add(2 * time.Second)))
	_, welcomeMsg, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(welcomeMsg), "event: hello")

	err = conn.WriteJSON(dto.HeloMessage{
		Type:      dto.ClientEventTypes.CLIENTEVENTTYPEHELO.String(),
		Component: dto.HomeAssistantComponentSRAT,
		Version:   "2026.03.1",
	})
	suite.Require().NoError(err)

	_, flushedMsg, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(flushedMsg), "event: repair_command")
	suite.Contains(string(flushedMsg), "\"repair_id\":\"disk_space_low\"")
	suite.Equal(0, suite.repairService.QueueSize())
}

// TestWebSocketRepairLifecycleFixedCallsHARestart verifies that when a "fixed" repair
// lifecycle message arrives for the custom_component_restart_required repair, the
// backend calls RestartHomeAssistant on the HA service.
func (suite *WsHandlerSuite) TestWebSocketRepairLifecycleFixedCallsHARestart() {
	ctrl := mock.NewMockController(suite.T())

	restartCalled := make(chan struct{}, 1)
	dismissCalled := make(chan struct{}, 1)

	mockHASrv := mock.Mock[service.HomeAssistantServiceInterface](ctrl)
	mock.When(mockHASrv.RestartHomeAssistant(mock.AnyContext())).
		ThenAnswer(func(_ []any) []any {
			restartCalled <- struct{}{}
			return []any{nil}
		})

	mockHAComponentSvc := mock.Mock[service.HomeAssistantComponentServiceInterface](ctrl)
	mock.When(mockHAComponentSvc.DismissRestartRequiredRepair(mock.AnyContext())).
		ThenAnswer(func(_ []any) []any {
			dismissCalled <- struct{}{}
			return []any{nil}
		})

	h := api.NewWebSocketBroker(api.WebSocketHandlerParams{
		Ctx:            suite.ctx,
		Broadcaster:    suite.mockBroadcaster,
		RepairService:  suite.repairService,
		HAService:      mockHASrv,
		HAComponentSvc: mockHAComponentSvc,
		State:          suite.state,
	})
	r := mux.NewRouter()
	h.RegisterWs(r)

	srv := httptest.NewServer(r)
	defer srv.Close()

	url := "ws" + srv.URL[len("http"):] + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		suite.Failf("Dial failed", "err=%v resp=%v", err, resp)
		return
	}
	defer conn.Close()

	suite.Require().NoError(conn.SetReadDeadline(time.Now().Add(1 * time.Second)))
	_, welcomeMsg, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(welcomeMsg), "event: hello")

	err = conn.WriteJSON(dto.RepairLifecycleMessage{
		Type:     dto.ClientEventTypes.CLIENTEVENTTYPEREPAIRLIFECYCLE.String(),
		RepairID: "custom_component_restart_required",
		Status:   dto.RepairLifecycleStatusFixed,
	})
	suite.Require().NoError(err)

	select {
	case <-dismissCalled:
	case <-time.After(time.Second):
		suite.Fail("timed out waiting for DismissRestartRequiredRepair to be called")
	}

	select {
	case <-restartCalled:
	case <-time.After(time.Second):
		suite.Fail("timed out waiting for RestartHomeAssistant to be called")
	}
}

func (suite *WsHandlerSuite) TestWebSocketAcceptsValidatedInboundProblemLifecycle() {
	updated := &dto.Problem{
		ProblemKey: "custom_component_restart_required",
		Title:      "Restart Home Assistant",
		Severity:   dto.ProblemSeverities.PROBLEMSEVERITYERROR,
		Status:     dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED,
	}
	var noProblemError *string
	mock.When(
		suite.mockProblemSvc.ApplyLifecycle(
			mock.Exact("custom_component_restart_required"),
			mock.Exact(dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED),
			mock.Exact(noProblemError),
		),
	).ThenReturn(updated, nil)

	h := api.NewWebSocketBroker(api.WebSocketHandlerParams{
		Ctx:            suite.ctx,
		Broadcaster:    suite.mockBroadcaster,
		RepairService:  suite.repairService,
		ProblemService: suite.mockProblemSvc,
		State:          suite.state,
	})
	r := mux.NewRouter()
	h.RegisterWs(r)

	srv := httptest.NewServer(r)
	defer srv.Close()

	url := "ws" + srv.URL[len("http"):] + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		suite.Failf("Dial failed", "err=%v resp=%v", err, resp)
		return
	}
	defer conn.Close()

	suite.Require().NoError(conn.SetReadDeadline(time.Now().Add(1 * time.Second)))
	_, welcomeMsg, err := conn.ReadMessage()
	suite.Require().NoError(err)
	suite.Contains(string(welcomeMsg), "event: hello")

	err = conn.WriteJSON(dto.ProblemLifecycleMessage{
		Type:       dto.ClientEventTypes.CLIENTEVENTTYPEPROBLEMLIFECYCLE.String(),
		ProblemKey: "custom_component_restart_required",
		Status:     dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED,
	})
	suite.Require().NoError(err)

	time.Sleep(100 * time.Millisecond)
	_, _ = mock.Verify(suite.mockProblemSvc, matchers.Times(1)).ApplyLifecycle(
		mock.Exact("custom_component_restart_required"),
		mock.Exact(dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED),
		mock.Exact(noProblemError),
	)
}
