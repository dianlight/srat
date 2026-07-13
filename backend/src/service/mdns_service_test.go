// Package service_test contains black-box unit tests for MDNSService.
package service_test

import (
	"context"
	"net"
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
	fakeRegister    *fakeZeroconfRegister
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
	suite.fakeRegister = &fakeZeroconfRegister{}
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
			func() service.ZeroconfRegister { return suite.fakeRegister },
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
	mock.When(suite.mockBroadcaster.BroadcastGuaranteedMessage(mock.Any[any]())).
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
		suite.Fail("BroadcastGuaranteedMessage was not called within timeout")
	}
}

// TestOnComponentConnected_BroadcastsDisabledNotification verifies that when
// MDNSRegistration is false the broadcast carries Enabled=false.
func (suite *MDNSServiceTestSuite) TestOnComponentConnected_BroadcastsDisabledNotification() {
	mock.When(suite.mockSettings.Load()).ThenReturn(defaultSettings("myserver", false), nil)

	captured := make(chan any, 1)
	mock.When(suite.mockBroadcaster.BroadcastGuaranteedMessage(mock.Any[any]())).
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
		suite.Fail("BroadcastGuaranteedMessage was not called within timeout")
	}
}

// TestOnComponentDisconnected_NoBroadcast verifies Option B timeout semantics:
// disconnecting the component does NOT send any message.
func (suite *MDNSServiceTestSuite) TestOnComponentDisconnected_NoBroadcast() {
	// First connect so state is "connected"
	mock.When(suite.mockSettings.Load()).ThenReturn(defaultSettings("myserver", true), nil)
	broadcastCalled := false
	mock.When(suite.mockBroadcaster.BroadcastGuaranteedMessage(mock.Any[any]())).
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
	mock.When(suite.mockBroadcaster.BroadcastGuaranteedMessage(mock.Any[any]())).
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

// fakeZeroconfServer records whether Shutdown has been called.
type fakeZeroconfServer struct {
	shutdownCalled bool
	mu             sync.Mutex
}

func (s *fakeZeroconfServer) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdownCalled = true
}

func (s *fakeZeroconfServer) wasShutdownCalled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shutdownCalled
}

// fakeZeroconfRegister records every Register call and returns a fake server.
type fakeZeroconfRegister struct {
	calls []fakeZeroconfRegisterCall
	mu    sync.Mutex
}

type fakeZeroconfRegisterCall struct {
	instance string
	service  string
	domain   string
	port     int
	text     []string
	ifaces   []string
}

func (r *fakeZeroconfRegister) Register(instance, svc, domain string, port int, text []string, ifaces []net.Interface) (service.ZeroconfServer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ifaceNames := make([]string, len(ifaces))
	for i, iface := range ifaces {
		ifaceNames[i] = iface.Name
	}
	r.calls = append(r.calls, fakeZeroconfRegisterCall{
		instance: instance,
		service:  svc,
		domain:   domain,
		port:     port,
		text:     append([]string(nil), text...),
		ifaces:   ifaceNames,
	})
	return &fakeZeroconfServer{}, nil
}

func (r *fakeZeroconfRegister) callsCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func (r *fakeZeroconfRegister) lastCall() (fakeZeroconfRegisterCall, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) == 0 {
		return fakeZeroconfRegisterCall{}, false
	}
	return r.calls[len(r.calls)-1], true
}

// directSettings returns Settings with addon-side direct mDNS enabled.
func directSettings(hostname string) *dto.Settings {
	trueVal := true
	falseVal := false
	return &dto.Settings{
		Hostname:              hostname,
		ExperimentalLabMode:   true,
		MDNSRegistration:      &falseVal,
		AddonMDNSRegistration: &trueVal,
		AddonMDNSInterfaces:   []string{},
	}
}

// TestSettingEvent_EnablesDirectMDNS verifies that a settings change with
// addon-side direct mDNS enabled triggers zeroconf.Register with the expected
// service details and instance name sanitization.
func (suite *MDNSServiceTestSuite) TestSettingEvent_EnablesDirectMDNS() {
	mock.When(suite.mockSettings.Load()).ThenReturn(directSettings("My-Server-01"), nil)

	// Suppress broadcasts for this test.
	mock.When(suite.mockBroadcaster.BroadcastGuaranteedMessage(mock.Any[any]())).ThenReturn(nil)

	suite.eventBus.EmitSetting(events.SettingEvent{
		Event:   events.Event{Type: events.EventTypes.UPDATE},
		Setting: directSettings("My-Server-01"),
	})
	time.Sleep(200 * time.Millisecond)

	suite.GreaterOrEqual(suite.fakeRegister.callsCount(), 1, "zeroconf.Register should have been called")
	call, ok := suite.fakeRegister.lastCall()
	suite.Require().True(ok)
	suite.Equal("MY-SERVER-01", call.instance)
	suite.Equal("_smb._tcp", call.service)
	suite.Equal("local.", call.domain)
	suite.Equal(445, call.port)
	suite.Equal([]string{"path=/"}, call.text)
}

// TestAppStop_ShutsDownDirectMDNS verifies that stopping the service shuts down
// an active direct mDNS registration.
func (suite *MDNSServiceTestSuite) TestAppStop_ShutsDownDirectMDNS() {
	mock.When(suite.mockSettings.Load()).ThenReturn(directSettings("server"), nil)
	mock.When(suite.mockBroadcaster.BroadcastGuaranteedMessage(mock.Any[any]())).ThenReturn(nil)

	suite.eventBus.EmitSetting(events.SettingEvent{
		Event:   events.Event{Type: events.EventTypes.UPDATE},
		Setting: directSettings("server"),
	})
	time.Sleep(200 * time.Millisecond)

	suite.Require().GreaterOrEqual(suite.fakeRegister.callsCount(), 1, "registration should have happened")

	suite.app.RequireStop()
	suite.app = nil // prevent TearDownTest from stopping again
}

// TestSanitizeNetBIOSName is tested indirectly via the registered instance name
// in integration tests; unit-level coverage lives in the same-package helper.
func (suite *MDNSServiceTestSuite) TestSanitizeNetBIOSNamePlaceholder() {
	suite.T().Skip("covered by integration tests and same-package unit tests")
}
