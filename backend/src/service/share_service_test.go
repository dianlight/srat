package service_test

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type ShareServiceSuite struct {
	suite.Suite
	shareService        service.ShareServiceInterface
	exported_share_repo repository.ExportedShareRepositoryInterface
	app                 *fxtest.App
}

func TestShareServiceSuite(t *testing.T) {
	suite.Run(t, new(ShareServiceSuite))
}

func (suite *ShareServiceSuite) SetupTest() {

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
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

				return &sharedResources
			},
			service.NewShareService,
			mock.Mock[service.BroadcasterServiceInterface],
			mock.Mock[repository.ExportedShareRepositoryInterface],
			mock.Mock[repository.MountPointPathRepositoryInterface],
			mock.Mock[repository.SambaUserRepositoryInterface],
		),
		fx.Populate(&suite.shareService),
		fx.Populate(&suite.exported_share_repo),
	)
	suite.app.RequireStart()
}

func (suite *ShareServiceSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *ShareServiceSuite) TestAll() {
	mock.When(suite.exported_share_repo.All()).ThenReturn(&[]dbom.ExportedShare{
		{
			Name: "test",
		},
	}, nil)

	shares, err := suite.shareService.All()

	suite.NoError(err)
	suite.NotNil(shares)
	suite.Len(*shares, 1)
}
