package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type TelemetryHandlerSuite struct {
	suite.Suite
	app              *fxtest.App
	handler          *api.TelemetryHandler
	mockTelemetrySvc service.TelemetryServiceInterface
	ctx              context.Context
	cancel           context.CancelFunc
}

func TestTelemetryHandlerSuite(t *testing.T) {
	suite.Run(t, new(TelemetryHandlerSuite))
}

func (suite *TelemetryHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			func() *dto.ContextState { return &dto.ContextState{} },
			api.NewTelemetryHandler,
			mock.Mock[service.TelemetryServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockTelemetrySvc),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *TelemetryHandlerSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
		if wg, ok := suite.ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok {
			wg.Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

func (suite *TelemetryHandlerSuite) TestGetTelemetryModes() {
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterTelemetry(apiInst)
	resp := apiInst.Get("/telemetry/modes")
	suite.Require().Equal(http.StatusOK, resp.Code)
	var modes []string
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &modes))
	// Expect canonical string forms
	suite.Subset(modes, []string{"Ask", "All", "Errors", "Disabled"})
}

func (suite *TelemetryHandlerSuite) TestGetInternetConnection() {
	mock.When(suite.mockTelemetrySvc.IsConnectedToInternet()).ThenReturn(true)
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterTelemetry(apiInst)
	resp := apiInst.Get("/telemetry/internet-connection")
	suite.Require().Equal(http.StatusOK, resp.Code)
	var connected bool
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &connected))
	suite.True(connected)
	mock.Verify(suite.mockTelemetrySvc, matchers.Times(1)).IsConnectedToInternet()
}
