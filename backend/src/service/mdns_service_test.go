// Package service_test contains black-box unit tests for MDNSService.
package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// MDNSServiceTestSuite groups all unit tests for MDNSService.
type MDNSServiceTestSuite struct {
	suite.Suite

	app             *fxtest.App
	mdnsService     service.MDNSServiceInterface
	mockBroadcaster service.BroadcasterServiceInterface
	mockSettings    service.SettingServiceInterface
	eventBus        events.EventBusInterface
	ctrl            *matchers.MockController
	ctx             context.Context
	cancel          context.CancelFunc
	wg              *sync.WaitGroup
}

func TestMDNSServiceTestSuite(t *testing.T) {
	suite.Run(t, new(MDNSServiceTestSuite))
}

func (suite *MDNSServiceTestSuite) SetupTest() {
	suite.wg = &sync.WaitGroup{}
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg)
				return context.WithCancel(ctx)
			},
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewMDNSService,
			mock.Mock[service.BroadcasterServiceInterface],
			mock.Mock[service.SettingServiceInterface],
		),
		fx.Populate(&suite.ctrl),
		fx.Populate(&suite.ctx, &suite.cancel),
		fx.Populate(&suite.eventBus),
		fx.Populate(&suite.mdnsService),
		fx.Populate(&suite.mockBroadcaster),
		fx.Populate(&suite.mockSettings),
	)
	suite.app.RequireStart()
}

func (suite *MDNSServiceTestSuite) TearDownTest() {
	suite.cancel()
	suite.wg.Wait()
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

// defaultSettings returns a Settings value with MDNSRegistration enabled.
func defaultSettings(hostname string, enabled bool) *dto.Settings {
	e := enabled
	return &dto.Settings{
		Hostname:         hostname,
		MDNSRegistration: &e,
	}
}

// TestOnComponentConnected_BroadcastsEnabledNotification verifies that connecting
// a component immediately broadcasts a MdnsRegisterNotification with Enabled=true
// when the setting is true.
func (suite *MDNSServiceTestSuite) TestOnComponentConnected_BroadcastsEnabledNotification() {
	mock.When(suite.mockSettings.Load()).ThenReturn(defaultSettings("myserver", true), nil)

	captured := make(chan any, 1)
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[any]())).
		ThenAnswer(func(args []any) []any {
			captured <- args[0]
			return []any{nil}
		})

	suite.mdnsService.OnComponentConnected(dto.HeloMessage{})

	select {
	case msg := <-captured:
		n, ok := msg.(dto.MdnsRegisterNotification)
		suite.Require().True(ok, "expected MdnsRegisterNotification")
		suite.True(n.Enabled)
		suite.Equal("myserver", n.Hostname)
		suite.Equal(445, n.Port)
	case <-time.After(2 * time.Second):
		suite.Fail("BroadcastMessage was not called within timeout")
	}
}

// TestOnComponentConnected_BroadcastsDisabledNotification verifies that when
// MDNSRegistration is false the broadcast carries Enabled=false.
func (suite *MDNSServiceTestSuite) TestOnComponentConnected_BroadcastsDisabledNotification() {
	mock.When(suite.mockSettings.Load()).ThenReturn(defaultSettings("myserver", false), nil)

	captured := make(chan any, 1)
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[any]())).
		ThenAnswer(func(args []any) []any {
			captured <- args[0]
			return []any{nil}
		})

	suite.mdnsService.OnComponentConnected(dto.HeloMessage{})

	select {
	case msg := <-captured:
		n, ok := msg.(dto.MdnsRegisterNotification)
		suite.Require().True(ok, "expected MdnsRegisterNotification")
		suite.False(n.Enabled)
	case <-time.After(2 * time.Second):
		suite.Fail("BroadcastMessage was not called within timeout")
	}
}

// TestOnComponentDisconnected_NoBroadcast verifies Option B timeout semantics:
// disconnecting the component does NOT send any message.
func (suite *MDNSServiceTestSuite) TestOnComponentDisconnected_NoBroadcast() {
	// First connect so state is "connected"
	mock.When(suite.mockSettings.Load()).ThenReturn(defaultSettings("myserver", true), nil)
	broadcastCalled := false
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[any]())).
		ThenAnswer(func(args []any) []any {
			broadcastCalled = true
			return []any{nil}
		})

	suite.mdnsService.OnComponentConnected(dto.HeloMessage{})
	// Wait a moment for connect broadcast to settle
	time.Sleep(50 * time.Millisecond)
	broadcastCalled = false // reset

	// Now disconnect — should NOT trigger another broadcast
	suite.mdnsService.OnComponentDisconnected()
	time.Sleep(100 * time.Millisecond)

	suite.False(broadcastCalled, "OnComponentDisconnected must not broadcast (Option B)")
}

// TestCleanEvent_ReBroadcastsWhenConnected verifies that a CLEAN ServerProcess
// event causes a re-broadcast when the component is connected.
func (suite *MDNSServiceTestSuite) TestCleanEvent_ReBroadcastsWhenConnected() {
	mock.When(suite.mockSettings.Load()).ThenReturn(defaultSettings("myserver", true), nil)

	broadcastCount := 0
	mu := sync.Mutex{}
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[any]())).
		ThenAnswer(func(args []any) []any {
			mu.Lock()
			broadcastCount++
			mu.Unlock()
			return []any{nil}
		})

	// Connect the component — triggers first broadcast
	suite.mdnsService.OnComponentConnected(dto.HeloMessage{})
	time.Sleep(50 * time.Millisecond)

	// Emit CLEAN server process event
	suite.eventBus.EmitServerProcess(events.ServerProcessEvent{
		Event: events.Event{Type: events.EventTypes.CLEAN},
	})
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	suite.GreaterOrEqual(broadcastCount, 2, "CLEAN event should trigger a second broadcast when connected")
}

// TestCleanEvent_NoReBroadcastWhenDisconnected verifies that a CLEAN event
// does NOT cause a broadcast when no component is connected.
func (suite *MDNSServiceTestSuite) TestCleanEvent_NoReBroadcastWhenDisconnected() {
	mock.When(suite.mockSettings.Load()).ThenReturn(defaultSettings("myserver", true), nil)

	broadcastCalled := false
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[any]())).
		ThenAnswer(func(args []any) []any {
			broadcastCalled = true
			return []any{nil}
		})

	// Do NOT connect — remain disconnected
	suite.eventBus.EmitServerProcess(events.ServerProcessEvent{
		Event: events.Event{Type: events.EventTypes.CLEAN},
	})
	time.Sleep(100 * time.Millisecond)

	suite.False(broadcastCalled, "CLEAN event should not broadcast when no component is connected")
}
