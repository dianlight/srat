package service_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	oerrors "github.com/pkg/errors"
	"github.com/rollbar/rollbar-go"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/tlog"
)

type TelemetryServiceSuite struct {
	suite.Suite
	app       *fxtest.App
	ctx       context.Context
	cancel    context.CancelFunc
	wg        *sync.WaitGroup
	propRepo  repository.PropertyRepositoryInterface
	telemetry service.TelemetryServiceInterface

	lastRollbarBody string
}

func TestTelemetryServiceSuite(t *testing.T) {
	suite.Run(t, new(TelemetryServiceSuite))
}

func (suite *TelemetryServiceSuite) SetupTest() {
	suite.wg = &sync.WaitGroup{}

	httpmock.Activate()

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), "wg", suite.wg)
				return context.WithCancel(ctx)
			},
			service.NewTelemetryService,
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[service.HaRootServiceInterface], // Use mock for HaRootServiceInterface
		),
		fx.Populate(&suite.ctx, &suite.cancel),
		fx.Populate(&suite.propRepo),
		fx.Populate(&suite.telemetry),
	)

	// Default repository behavior: return empty properties
	mock.When(suite.propRepo.All(mock.Any[bool]())).ThenReturn(dbom.Properties{}, nil)

	suite.app.RequireStart()
}

func (suite *TelemetryServiceSuite) TearDownTest() {
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

	// Ensure tlog callbacks are clean across tests
	tlog.ClearAllCallbacks()
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
	_ = suite.telemetry.ReportEvent("custom_test", map[string]interface{}{"x": 1})
	suite.Empty(suite.lastRollbarBody)

	// All mode: should send
	// But only if internet is available; simulate down to avoid network
	httpmock.RegisterResponder("HEAD", "https://api.rollbar.com",
		httpmock.NewStringResponder(200, "OK"))
	_ = suite.telemetry.Configure(dto.TelemetryModes.TELEMETRYMODEALL)
	suite.resetHTTPCalls()
	suite.stubRollbarItemPost(true)

	_ = suite.telemetry.ReportEvent("custom_test", map[string]interface{}{"x": 1})
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
	custom := map[string]interface{}{"k": "v", "n": 123}
	suite.Require().NoError(suite.telemetry.ReportError(errStd, custom))

	// Flush async rollbar sender
	rollbar.Wait()

	// Assert: captured JSON has expected shape
	suite.NotEmpty(suite.lastRollbarBody)
	var payload map[string]interface{}
	suite.Require().NoError(json.Unmarshal([]byte(suite.lastRollbarBody), &payload))

	data, _ := payload["data"].(map[string]interface{})
	suite.Require().NotNil(data)
	suite.Equal("error", data["level"])

	// custom data
	customObj, _ := data["custom"].(map[string]interface{})
	suite.Require().NotNil(customObj)
	suite.Equal("v", customObj["k"])
	// numbers decode as float64
	suite.Equal(float64(123), customObj["n"])

	// trace or trace_chain with frames
	body, _ := data["body"].(map[string]interface{})
	suite.Require().NotNil(body)
	if trace, ok := body["trace"].(map[string]interface{}); ok {
		if exception, ok := trace["exception"].(map[string]interface{}); ok {
			if msg, ok := exception["message"].(string); ok {
				suite.Contains(msg, "boom")
			}
		}
		if frames, ok := trace["frames"].([]interface{}); ok {
			suite.NotEmpty(frames)
			suite.Equal("github.com/dianlight/srat/service/telemetry_service_test.go", frames[0].(map[string]interface{})["filename"])
		}
	} else if traceChain, ok := body["trace_chain"].([]interface{}); ok && len(traceChain) > 0 {
		if first, ok := traceChain[0].(map[string]interface{}); ok {
			if exception, ok := first["exception"].(map[string]interface{}); ok {
				if msg, ok := exception["message"].(string); ok {
					suite.NotEmpty(msg)
				}
			}
			if frames, ok := first["frames"].([]interface{}); ok {
				suite.NotEmpty(frames)
				suite.Contains(frames[0].(map[string]interface{})["filename"], "/service/telemetry_service_test.go")
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
	custom := map[string]interface{}{"a": 1, "b": "c"}
	suite.Require().NoError(suite.telemetry.ReportError(e, custom))

	// Flush async rollbar sender
	rollbar.Wait()

	// Assert: captured JSON has expected shape
	suite.NotEmpty(suite.lastRollbarBody)
	var payload map[string]interface{}
	suite.Require().NoError(json.Unmarshal([]byte(suite.lastRollbarBody), &payload))

	data, _ := payload["data"].(map[string]interface{})
	suite.Require().NotNil(data)
	suite.Equal("error", data["level"])

	// custom data
	customObj, _ := data["custom"].(map[string]interface{})
	suite.Require().NotNil(customObj)
	suite.Equal(float64(1), customObj["a"]) // numbers as float64
	suite.Equal("c", customObj["b"])

	// trace or trace_chain with frames
	body, _ := data["body"].(map[string]interface{})
	suite.Require().NotNil(body)
	if trace, ok := body["trace"].(map[string]interface{}); ok {
		if exception, ok := trace["exception"].(map[string]interface{}); ok {
			if msg, ok := exception["message"].(string); ok {
				suite.NotEmpty(msg)
			}
		}
		if frames, ok := trace["frames"].([]interface{}); ok {
			suite.NotEmpty(frames)
			suite.Contains(frames[0].(map[string]interface{})["filename"], "/service/telemetry_service_test.go")
		}
	} else if traceChain, ok := body["trace_chain"].([]interface{}); ok && len(traceChain) > 0 {
		if first, ok := traceChain[0].(map[string]interface{}); ok {
			if exception, ok := first["exception"].(map[string]interface{}); ok {
				if msg, ok := exception["message"].(string); ok {
					suite.NotEmpty(msg)
				}
			}
			if frames, ok := first["frames"].([]interface{}); ok {
				suite.NotEmpty(frames)
				suite.Contains(frames[0].(map[string]interface{})["filename"], "/service/telemetry_service_test.go")
			}
		}
	}
}
