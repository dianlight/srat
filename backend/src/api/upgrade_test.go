package api_test

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type UpgradeHandlerSuite struct {
	suite.Suite
	app                *fxtest.App
	handler            *api.UpgradeHanler
	mockUpgradeService service.UpgradeServiceInterface
	mockBroadcaster    service.BroadcasterServiceInterface
	ctx                context.Context
	cancel             context.CancelFunc
}

func TestUpgradeHandlerSuite(t *testing.T) { suite.Run(t, new(UpgradeHandlerSuite)) }

func (suite *UpgradeHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			func() *dto.ContextState { return &dto.ContextState{} },
			api.NewUpgradeHanler,
			mock.Mock[service.UpgradeServiceInterface],
			mock.Mock[service.BroadcasterServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockUpgradeService),
		fx.Populate(&suite.mockBroadcaster),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *UpgradeHandlerSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
		if wg, ok := suite.ctx.Value("wg").(*sync.WaitGroup); ok {
			wg.Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

func (suite *UpgradeHandlerSuite) TestGetUpdateInfoSuccess() {
	asset := &dto.ReleaseAsset{LastRelease: "v1.2.3"}
	mock.When(suite.mockUpgradeService.GetUpgradeReleaseAsset()).ThenReturn(asset, nil)
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterUpgradeHanler(apiInst)
	resp := apiInst.Get("/update")
	suite.Require().Equal(http.StatusOK, resp.Code)
	suite.Contains(resp.Body.String(), "v1.2.3")
	mock.Verify(suite.mockUpgradeService, matchers.Times(1)).GetUpgradeReleaseAsset()
}

func (suite *UpgradeHandlerSuite) TestGetUpdateInfoNoUpdate() {
	mock.When(suite.mockUpgradeService.GetUpgradeReleaseAsset()).ThenReturn(nil, errors.WithStack(dto.ErrorNoUpdateAvailable))
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterUpgradeHanler(apiInst)
	resp := apiInst.Get("/update")
	suite.Require().Equal(http.StatusNoContent, resp.Code, "body", resp.Body.String())
	mock.Verify(suite.mockUpgradeService, matchers.Times(1)).GetUpgradeReleaseAsset()
}
