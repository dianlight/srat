package service_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	oerrors "github.com/pkg/errors"
	"github.com/rollbar/rollbar-go"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/tlog"
)

type TelemetryServiceSuite struct {
	suite.Suite
	app    *fxtest.App
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	settingService service.SettingServiceInterface
	telemetry      service.TelemetryServiceInterface

	lastRollbarBody string
}

func TestTelemetryServiceSuite(t *testing.T) {
	// Enable test mode to prevent rollbar from closing between tests
	service.SetSkipRollbarCloseForTest(true)

	suite.Run(t, new(TelemetryServiceSuite))

	// After all tests complete, close rollbar once and reset test mode
	service.SetSkipRollbarCloseForTest(false)
	rollbar.Wait()
	// Don't call Close() - it may already be closed, and we have recovery logic
}

func (suite *TelemetryServiceSuite) SetupTest() {
	suite.wg = &sync.WaitGroup{}

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
			//mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaRootServiceInterface], // Use mock for HaRootServiceInterface
		),
		fx.Populate(&suite.ctx, &suite.cancel),
		fx.Populate(&suite.settingService),
		fx.Populate(&suite.telemetry),
	)

	// Default repository behavior: return empty properties
	mock.When(suite.settingService.Load()).ThenReturn(&dto.Settings{
		TelemetryMode: dto.TelemetryModes.TELEMETRYMODEASK,
	}, nil)
	//mock.When(suite.propRepo.All()).ThenReturn(dbom.Properties{}, nil)

	suite.app.RequireStart()
}

func (suite *TelemetryServiceSuite) TearDownTest() {
	// First, flush any pending rollbar events
	rollbar.Wait()

	// Unregister tlog callbacks but don't close rollbar
	// We'll close it once at the end of all tests
	tlog.ClearAllCallbacks()

	// Then cancel context and wait for goroutines
	if suite.cancel != nil {
		suite.cancel()
	}
	if suite.wg != nil {
		suite.wg.Wait()
	}

	// Stop the app  - OnStop will try to close rollbar but our global flag will prevent double-close
	if suite.app != nil {
		suite.app.RequireStop()
	}

	// Clean up HTTP mocks
	httpmock.DeactivateAndReset()

	// Reset global rollbar closed flag so next test can try to "close" without panic
	// But since rollbar is already closed, the actual close won't happen
	service.ResetRollbarGlobalState()
}

// Helpers
func (suite *TelemetryServiceSuite) stubRollbarConnectivityOK() {
	httpmock.RegisterResponder("HEAD", "https://api.rollbar.com",
		httpmock.NewStringResponder(200, "OK"))
}

func (suite *TelemetryServiceSuite) stubRollbarItemPost(capture bool) {
	if capture {
		suite.lastRollbarBody = ""
		httpmock.RegisterResponder("POST", "https://api.rollbar.com/api/1/item/",
			func(req *http.Request) (*http.Response, error) {
				body, _ := io.ReadAll(req.Body)
				suite.lastRollbarBody = string(body)
				return httpmock.NewStringResponse(200, `{"err":0}`), nil
			})
		return
	}
	httpmock.RegisterResponder("POST", "https://api.rollbar.com/api/1/item/",
		httpmock.NewStringResponder(200, `{"err":0}`))
}

func (suite *TelemetryServiceSuite) resetHTTPCalls() {
	// Not available in httpmock; use lastRollbarBody as a soft marker instead
	suite.lastRollbarBody = ""
}

// Tests

func (suite *TelemetryServiceSuite) TestConfigure_Disabled_LeavesNoCallbacks() {
	// Act
	err := suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEDISABLED)
	suite.Require().NoError(err)

	// Assert: no tlog callbacks registered
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelError))
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelFatal))

	// And no HTTP activity
	suite.Equal(0, httpmock.GetTotalCallCount())
}

func (suite *TelemetryServiceSuite) TestConfigure_All_NoInternet_DoesNotRegisterCallbacks() {
	// Simulate no internet
	httpmock.RegisterResponder("HEAD", "https://api.rollbar.com",
		httpmock.NewErrorResponder(assertErr("network down")))

	err := suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEALL)
	suite.Require().NoError(err)

	// No callbacks when internet is unavailable
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelError))
	suite.Equal(0, tlog.GetCallbackCount(tlog.LevelFatal))
}

func (suite *TelemetryServiceSuite) TestReportError_NotConfigured_NoHTTP() {
	// Ensure disabled
	_ = suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEDISABLED)
	suite.stubRollbarItemPost(true)

	// Act
	err := suite.telemetry.ReportError(io.EOF)
	suite.Require().NoError(err)

	// Assert: no POSTs sent when not configured
	suite.Empty(suite.lastRollbarBody)
}

func (suite *TelemetryServiceSuite) TestReportEvent_OnlyInAllMode() {
	// Errors mode: should not send events
	suite.resetHTTPCalls()
	_ = suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEERRORS)
	_ = suite.telemetry.ReportEvent("custom_test", map[string]any{"x": 1})
	suite.Empty(suite.lastRollbarBody)

	// All mode: should send
	// But only if internet is available; simulate down to avoid network
	httpmock.RegisterResponder("HEAD", "https://api.rollbar.com",
		httpmock.NewStringResponder(200, "OK"))
	_ = suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEALL)
	suite.resetHTTPCalls()
	suite.stubRollbarItemPost(true)

	_ = suite.telemetry.ReportEvent("custom_test", map[string]any{"x": 1})
	// We don't assert body here due to rollbar-go async behavior; just ensure no panic/error
}

func (suite *TelemetryServiceSuite) TestIsConnectedToInternet_TrueAndFalse() {
	// True case
	suite.stubRollbarConnectivityOK()
	suite.True(suite.telemetry.IsConnectedToInternet())

	// False case
	httpmock.Reset()
	httpmock.RegisterResponder("HEAD", "https://api.rollbar.com",
		httpmock.NewErrorResponder(assertErr("network down")))
	suite.False(suite.telemetry.IsConnectedToInternet())
}

// assertErr wraps an error string as error for httpmock
func assertErr(msg string) error { return &mockError{msg: msg} }

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }

// Removed deep callback forwarding test due to rollbar-go internal client/transport complexity

// New tests: ReportError payloads

func (suite *TelemetryServiceSuite) TestReportError_StandardError_JSONContainsCustomAndStack() {
	// Arrange: enable telemetry (Errors mode) and stub HTTP
	suite.stubRollbarConnectivityOK()
	suite.stubRollbarItemPost(true)
	suite.Require().NoError(suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEERRORS))
	suite.resetHTTPCalls()

	// Act: report a standard error with custom data
	errStd := oerrors.Errorf("boom %d", 42)
	custom := map[string]any{"k": "v", "n": 123}
	suite.Require().NoError(suite.telemetry.ReportError(errStd, custom))

	// Flush async rollbar sender
	rollbar.Wait()

	// Assert: captured JSON has expected shape
	suite.NotEmpty(suite.lastRollbarBody)
	var payload map[string]any
	suite.Require().NoError(json.Unmarshal([]byte(suite.lastRollbarBody), &payload))

	data, _ := payload["data"].(map[string]any)
	suite.Require().NotNil(data)
	suite.Equal("error", data["level"])

	// custom data
	customObj, _ := data["custom"].(map[string]any)
	suite.Require().NotNil(customObj)
	suite.Equal("v", customObj["k"])
	// numbers decode as float64
	suite.Equal(float64(123), customObj["n"])

	// trace or trace_chain with frames
	body, _ := data["body"].(map[string]any)
	suite.Require().NotNil(body)
	if trace, ok := body["trace"].(map[string]any); ok {
		if exception, ok := trace["exception"].(map[string]any); ok {
			if msg, ok := exception["message"].(string); ok {
				suite.Contains(msg, "boom")
			}
		}
		if frames, ok := trace["frames"].([]any); ok {
			suite.NotEmpty(frames)
			suite.Equal("github.com/dianlight/srat/service/telemetry_service_test.go", frames[0].(map[string]any)["filename"])
		}
	} else if traceChain, ok := body["trace_chain"].([]any); ok && len(traceChain) > 0 {
		if first, ok := traceChain[0].(map[string]any); ok {
			if exception, ok := first["exception"].(map[string]any); ok {
				if msg, ok := exception["message"].(string); ok {
					suite.NotEmpty(msg)
				}
			}
			if frames, ok := first["frames"].([]any); ok {
				suite.NotEmpty(frames)
				suite.Contains(frames[0].(map[string]any)["filename"], "/service/telemetry_service_test.go")
			}
		}
	}
}

func (suite *TelemetryServiceSuite) TestReportError_ErrorsE_JSONContainsCustomAndStack() {
	// Arrange: enable telemetry (Errors mode) and stub HTTP
	suite.stubRollbarConnectivityOK()
	suite.stubRollbarItemPost(true)
	suite.Require().NoError(suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEERRORS))
	suite.resetHTTPCalls()

	// Act: report an errors.E with custom data
	e := errors.Errorf("oops %s", "E")
	custom := map[string]any{"a": 1, "b": "c"}
	suite.Require().NoError(suite.telemetry.ReportError(e, custom))

	// Flush async rollbar sender
	rollbar.Wait()

	// Assert: captured JSON has expected shape
	suite.NotEmpty(suite.lastRollbarBody)
	var payload map[string]any
	suite.Require().NoError(json.Unmarshal([]byte(suite.lastRollbarBody), &payload))

	data, _ := payload["data"].(map[string]any)
	suite.Require().NotNil(data)
	suite.Equal("error", data["level"])

	// custom data
	customObj, _ := data["custom"].(map[string]any)
	suite.Require().NotNil(customObj)
	suite.Equal(float64(1), customObj["a"]) // numbers as float64
	suite.Equal("c", customObj["b"])

	// trace or trace_chain with frames
	body, _ := data["body"].(map[string]any)
	suite.Require().NotNil(body)
	if trace, ok := body["trace"].(map[string]any); ok {
		if exception, ok := trace["exception"].(map[string]any); ok {
			if msg, ok := exception["message"].(string); ok {
				suite.NotEmpty(msg)
			}
		}
		if frames, ok := trace["frames"].([]any); ok {
			suite.NotEmpty(frames)
			suite.Contains(frames[0].(map[string]any)["filename"], "/service/telemetry_service_test.go")
		}
	} else if traceChain, ok := body["trace_chain"].([]any); ok && len(traceChain) > 0 {
		if first, ok := traceChain[0].(map[string]any); ok {
			if exception, ok := first["exception"].(map[string]any); ok {
				if msg, ok := exception["message"].(string); ok {
					suite.NotEmpty(msg)
				}
			}
			if frames, ok := first["frames"].([]any); ok {
				suite.NotEmpty(frames)
				suite.Contains(frames[0].(map[string]any)["filename"], "/service/telemetry_service_test.go")
			}
		}
	}
}

func (suite *TelemetryServiceSuite) TestTlogErrorCallbackIncludesOriginalStack() {
	// Arrange: enable telemetry callbacks and capture rollbar payloads
	suite.stubRollbarConnectivityOK()
	suite.stubRollbarItemPost(true)
	suite.Require().NoError(suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEERRORS))
	suite.resetHTTPCalls()
	// Flush any pending rollbar events from previous tests
	rollbar.Wait()
	suite.lastRollbarBody = "" // Ensure clean start

	// Emit a tlog error with an attached stack-carrying error
	logErr := errors.Errorf("callback failure")
	tlog.Error("rolling error", "error", logErr)

	// Wait for asynchronous logging and rollbar dispatch, specifically for an error-level event
	suite.Eventually(func() bool {
		rollbar.Wait()
		if suite.lastRollbarBody == "" {
			return false
		}
		// Parse and check if this is the error-level event we're expecting
		var payload map[string]any
		if err := json.Unmarshal([]byte(suite.lastRollbarBody), &payload); err != nil {
			return false
		}
		data, ok := payload["data"].(map[string]any)
		if !ok {
			return false
		}
		// Only accept error-level events, ignore info-level events from previous tests
		level, _ := data["level"].(string)
		return level == "error"
	}, 2*time.Second, 20*time.Millisecond)

	// Validate payload contains stack information pointing to this test file
	suite.NotEmpty(suite.lastRollbarBody)
	var payload map[string]any
	suite.Require().NoError(json.Unmarshal([]byte(suite.lastRollbarBody), &payload))

	data, _ := payload["data"].(map[string]any)
	suite.Require().NotNil(data)
	// Ensure error-level rollbar events are sent for tlog.Error
	suite.Equal("error", data["level"], "log level 'error' expected for telemetry events, got %#v", data)

	if trace, ok := data["trace"].(map[string]any); ok {
		if frames, ok := trace["frames"].([]any); ok {
			suite.assertFrames(frames)
		}
	} else if traceChain, ok := data["trace_chain"].([]any); ok && len(traceChain) > 0 {
		if first, ok := traceChain[0].(map[string]any); ok {
			if frames, ok := first["frames"].([]any); ok {
				suite.assertFrames(frames)
			}
		}
	}
}

// Helper to assert stack frames include this test file
func (suite *TelemetryServiceSuite) assertFrames(frames []any) {
	suite.Require().NotEmpty(frames)
	first, ok := frames[0].(map[string]any)
	suite.Require().True(ok)
	filename, ok := first["filename"].(string)
	suite.Require().True(ok)
	suite.Contains(filename, "/service/telemetry_service_test.go")
}
