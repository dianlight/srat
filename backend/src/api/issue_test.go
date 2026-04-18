package api_test

import (
	"context"
	"sync"
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type IssueHandlerSuite struct {
	suite.Suite
	app     *fxtest.App
	handler *api.IssueAPI
	ctx     context.Context
	cancel  context.CancelFunc
}

func TestIssueHandlerSuite(t *testing.T) {
	suite.Run(t, new(IssueHandlerSuite))
}

func (suite *IssueHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			api.NewIssueAPI,
			mock.Mock[service.IssueReportServiceInterface],
			mock.Mock[service.IssueTemplateServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *IssueHandlerSuite) TearDownTest() {
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
