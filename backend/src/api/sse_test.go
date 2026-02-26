package api_test

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SseHandlerSuite struct {
	suite.Suite
	app             *fxtest.App
	broker          api.BrokerInterface
	mockBroadcaster service.BroadcasterServiceInterface
	ctx             context.Context
	cancel          context.CancelFunc
}

func TestSseHandlerSuite(t *testing.T) { suite.Run(t, new(SseHandlerSuite)) }

func (suite *SseHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			api.NewSSEBroker,
			mock.Mock[service.BroadcasterServiceInterface],
		),
		fx.Populate(&suite.broker),
		fx.Populate(&suite.mockBroadcaster),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *SseHandlerSuite) TearDownTest() {
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

func (suite *SseHandlerSuite) TestRegisterSseEndpoint() {
	// No stub needed; mock will record the invocation. Call the endpoint.
	_, apiInst := humatest.New(suite.T())
	suite.broker.RegisterSse(apiInst)

	resp := apiInst.Get("/sse")
	// The humatest GET to /sse should return 200 OK (handler registers endpoint)
	suite.Require().Equal(http.StatusOK, resp.Code)
	mock.Verify(suite.mockBroadcaster, matchers.Times(1)).ProcessHttpChannel(mock.Any[sse.Sender]())
}
