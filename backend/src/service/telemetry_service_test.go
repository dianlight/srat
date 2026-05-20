package service_test

import (
	"context"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	sentry "github.com/getsentry/sentry-go"
	"github.com/jarcoal/httpmock"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	oerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/tlog"
)

// testSentryTransport captures Sentry events without making real HTTP calls.
type testSentryTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
	ch     chan *sentry.Event
}

func newTestSentryTransport() *testSentryTransport {
	return &testSentryTransport{ch: make(chan *sentry.Event, 64)}
}

func (t *testSentryTransport) Configure(_ sentry.ClientOptions) {}

func (t *testSentryTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	t.events = append(t.events, event)
	t.mu.Unlock()
	select {
	case t.ch <- event:
	default:
	}
}

func (t *testSentryTransport) Flush(_ time.Duration) bool { return true }

func (t *testSentryTransport) FlushWithContext(_ context.Context) bool { return true }

func (t *testSentryTransport) Close() {}

// nextEvent waits up to timeout for the next event from the transport channel.
func (t *testSentryTransport) nextEvent(timeout time.Duration) *sentry.Event {
	select {
	case ev := <-t.ch:
		return ev
	case <-time.After(timeout):
		return nil
	}
}

// allEvents returns a snapshot of all captured events.
func (t *testSentryTransport) allEvents() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*sentry.Event, len(t.events))
	copy(out, t.events)
	return out
}

// reset clears captured events.
func (t *testSentryTransport) reset() {
	t.mu.Lock()
	t.events = nil
	t.mu.Unlock()
	for {
		select {
		case <-t.ch:
		default:
			return
		}
	}
}

type TelemetryServiceSuite struct {
	suite.Suite
	app    *fxtest.App
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	settingService service.SettingServiceInterface
	telemetry      service.TelemetryServiceInterface

	transport *testSentryTransport
}

func TestTelemetryServiceSuite(t *testing.T) {
	// Disable sentry.Flush() blocking in tests
	service.SetSkipSentryFlushForTest(true)
	suite.Run(t, new(TelemetryServiceSuite))
	service.SetSkipSentryFlushForTest(false)
}

func (suite *TelemetryServiceSuite) SetupTest() {
	suite.wg = &sync.WaitGroup{}
	suite.transport = newTestSentryTransport()

	httpmock.Activate()

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg)
				return context.WithCancel(ctx)
			},
			func() *dto.ContextState {
				sharedResources := dto.ContextState{}
				sharedResources.DockerInterface = "hassio"
				sharedResources.DockerNet = "172.30.32.0/23"
				var err error
				sharedResources.Template, err = os.ReadFile("../templates/smb.gtpl")
				if err != nil {
					suite.T().Errorf("Cant read template file %s", err)
				}
				sharedResources.DatabasePath = "file::memory:?cache=shared&_pragma=foreign_keys(1)"
				return &sharedResources
			},
			service.NewTelemetryService,
			events.NewEventBus,
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaRootServiceInterface],
		),
		fx.Populate(&suite.ctx, &suite.cancel),
		fx.Populate(&suite.settingService),
		fx.Populate(&suite.telemetry),
	)

	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		TelemetryMode: dto.TelemetryModes.TELEMETRYMODEASK,
	}, nil)

	suite.app.RequireStart()

	if ts, ok := suite.telemetry.(*service.TelemetryService); ok {
		ts.SetTestTransport(suite.transport)
	}
}

func (suite *TelemetryServiceSuite) TearDownTest() {
	tlog.ClearAllCallbacks()

	if suite.cancel != nil {
		suite.cancel()
	}
	if suite.wg != nil {
		suite.wg.Wait()
	}

	if suite.app != nil {
		suite.app.RequireStop()
	}

	httpmock.DeactivateAndReset()
}

func (suite *TelemetryServiceSuite) stubSentryConnectivityOK() {
	httpmock.RegisterResponder("HEAD", "https://sentry.io",
		httpmock.NewStringResponder(200, "OK"))
}

func (suite *TelemetryServiceSuite) stubSentryConnectivityDown() {
	httpmock.RegisterResponder("HEAD", "https://sentry.io",
		httpmock.NewErrorResponder(&mockError{msg: "network down"}))
}

func (suite *TelemetryServiceSuite) TestConfigure_Disabled_LeavesNoCallbacks() {
	err := suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEDISABLED)
	suite.Require().NoError(err)

	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelError))
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelFatal))
}

func (suite *TelemetryServiceSuite) TestConfigure_All_NoInternet_DoesNotRegisterCallbacks() {
	suite.stubSentryConnectivityDown()

	err := suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEALL)
	suite.Require().NoError(err)

	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelError))
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelFatal))
}

func (suite *TelemetryServiceSuite) TestConfigure_Errors_WithInternet_RegistersCallbacks() {
	suite.stubSentryConnectivityOK()

	err := suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEERRORS)
	suite.Require().NoError(err)

	suite.Equal(1, tlog.GetCallbackCount(tlog.LevelError))
	suite.Equal(1, tlog.GetCallbackCount(tlog.LevelFatal))
}

func (suite *TelemetryServiceSuite) TestReportError_NotConfigured_NoCapturedEvents() {
	_ = suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEDISABLED)
	suite.transport.reset()

	err := suite.telemetry.ReportError(io.EOF)
	suite.Require().NoError(err)

	suite.Empty(suite.transport.allEvents())
}

func (suite *TelemetryServiceSuite) TestReportEvent_OnlyInAllMode() {
	suite.stubSentryConnectivityOK()
	suite.Require().NoError(suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEERRORS))
	suite.transport.reset()

	_ = suite.telemetry.ReportEvent("custom_test", map[string]any{"x": 1})
	ev := suite.transport.nextEvent(100 * time.Millisecond)
	suite.Nil(ev, "Expected no event in Errors mode, but got one")

	suite.Require().NoError(suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEALL))
	suite.transport.reset()

	_ = suite.telemetry.ReportEvent("custom_test", map[string]any{"x": 1})
	ev = suite.transport.nextEvent(500 * time.Millisecond)
	suite.Require().NotNil(ev, "Expected an event in All mode, but none arrived")
	suite.Contains(ev.Message, "custom_test")
}

func (suite *TelemetryServiceSuite) TestIsConnectedToInternet_TrueAndFalse() {
	suite.stubSentryConnectivityOK()
	suite.True(suite.telemetry.IsConnectedToInternet())

	httpmock.Reset()
	suite.stubSentryConnectivityDown()
	suite.False(suite.telemetry.IsConnectedToInternet())
}

func (suite *TelemetryServiceSuite) TestReportError_StandardError_EventCaptured() {
	suite.stubSentryConnectivityOK()
	suite.Require().NoError(suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEERRORS))
	suite.transport.reset()

	errStd := oerrors.Errorf("boom %d", 42)
	custom := map[string]any{"k": "v", "n": 123}
	suite.Require().NoError(suite.telemetry.ReportError(errStd, custom))

	ev := suite.transport.nextEvent(500 * time.Millisecond)
	suite.Require().NotNil(ev, "Expected a Sentry event, but none arrived")
	suite.Require().NotEmpty(ev.Exception)
	suite.Contains(ev.Exception[0].Value, "boom")
}

func (suite *TelemetryServiceSuite) TestReportError_ErrorsE_EventCaptured() {
	suite.stubSentryConnectivityOK()
	suite.Require().NoError(suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEERRORS))
	suite.transport.reset()

	e := errors.Errorf("oops %s", "E")
	custom := map[string]any{"a": 1, "b": "c"}
	suite.Require().NoError(suite.telemetry.ReportError(e, custom))

	ev := suite.transport.nextEvent(500 * time.Millisecond)
	suite.Require().NotNil(ev, "Expected a Sentry event, but none arrived")
	suite.Require().NotEmpty(ev.Exception)
	suite.Contains(ev.Exception[0].Value, "oops")
}

func (suite *TelemetryServiceSuite) TestTlogErrorCallbackCapturesEvent() {
	suite.stubSentryConnectivityOK()
	suite.Require().NoError(suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEERRORS))
	suite.transport.reset()

	logErr := errors.Errorf("callback failure")
	tlog.Error("rolling error", "error", logErr)

	var ev *sentry.Event
	suite.Require().Eventually(func() bool {
		ev = suite.transport.nextEvent(50 * time.Millisecond)
		return ev != nil
	}, 2*time.Second, 20*time.Millisecond, "Expected a Sentry event from tlog.Error, but none arrived")

	suite.Require().NotNil(ev)
	hasException := len(ev.Exception) > 0
	hasMessage := ev.Message != ""
	suite.True(hasException || hasMessage, "Event should have exception or message")
}

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }
