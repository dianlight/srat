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
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type VolumeHandlerSuite struct {
	suite.Suite
	app           *fxtest.App
	handler       *api.VolumeHandler
	mockVolumeSvc service.VolumeServiceInterface
	mockShareSvc  service.ShareServiceInterface
	mockDirtySvc  service.DirtyDataServiceInterface
	ctx           context.Context
	cancel        context.CancelFunc
}

func TestVolumeHandlerSuite(t *testing.T) { suite.Run(t, new(VolumeHandlerSuite)) }

func (suite *VolumeHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			func() *dto.ContextState { return &dto.ContextState{} },
			api.NewVolumeHandler,
			mock.Mock[service.VolumeServiceInterface],
			mock.Mock[service.ShareServiceInterface],
			mock.Mock[service.DirtyDataServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockVolumeSvc),
		fx.Populate(&suite.mockShareSvc),
		fx.Populate(&suite.mockDirtySvc),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *VolumeHandlerSuite) TearDownTest() {
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

func (suite *VolumeHandlerSuite) TestListVolumesSuccess() {
	id1, id2 := "sda", "sdb"
	vols := &[]dto.Disk{{Id: &id1}, {Id: &id2}}
	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(vols, nil)
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterVolumeHandlers(apiInst)
	resp := apiInst.Get("/volumes")
	suite.Require().Equal(http.StatusOK, resp.Code)
	var out []dto.Disk
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Len(out, 2)
	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).GetVolumesData()
}

func (suite *VolumeHandlerSuite) TestListVolumesError() {
	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(nil, errors.New("db"))
	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterVolumeHandlers(apiInst)
	resp := apiInst.Get("/volumes")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)
	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).GetVolumesData()
}
