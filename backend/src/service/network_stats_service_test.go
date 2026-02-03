package service_test

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// NetworkStatsServiceSuite contains unit tests for network_stats_service.go
type NetworkStatsServiceSuite struct {
	suite.Suite
	app  *fxtest.App
	ns   service.NetworkStatsService
	ctx  context.Context
	ctrl *matchers.MockController
	//propRepoMock repository.PropertyRepositoryInterface
}

func TestNetworkStatsServiceSuite(t *testing.T) {
	suite.Run(t, new(NetworkStatsServiceSuite))
}

func (suite *NetworkStatsServiceSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				var wg sync.WaitGroup
				ctx := context.WithValue(context.Background(), "wg", &wg)
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
			dbom.NewDB,
			events.NewEventBus,
			service.NewNetworkStatsService,
			service.NewSettingService,
		),
		fx.Populate(&suite.ctrl),
		fx.Populate(&suite.ctx),
		//fx.Populate(&suite.propRepoMock),
		fx.Populate(&suite.ns),
	)
	suite.app.RequireStart()
}

func (suite *NetworkStatsServiceSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *NetworkStatsServiceSuite) TestGetNetworkStatsNotInitialized() {
	stats, err := suite.ns.GetNetworkStats()
	suite.Require().NotNil(stats)
	suite.Require().NoError(err)
}
