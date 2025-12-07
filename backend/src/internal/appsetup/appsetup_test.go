package appsetup

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/addons"
	"github.com/dianlight/srat/homeassistant/core_api"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/homeassistant/host"
	"github.com/dianlight/srat/homeassistant/ingress"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/homeassistant/resolution"
	"github.com/dianlight/srat/homeassistant/root"
	"github.com/dianlight/srat/homeassistant/websocket"
	"github.com/dianlight/srat/internal"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type AppSetupSuite struct {
	suite.Suite
	params BaseAppParams
	logger *slog.Logger
}

func (suite *AppSetupSuite) SetupTest() {
	ctx, cancel := context.WithCancel(context.Background())
	suite.params = BaseAppParams{
		Ctx:      ctx,
		CancelFn: cancel,
		StaticConfig: &dto.ContextState{
			SupervisorURL:   "http://example.org",
			SupervisorToken: "token",
			DatabasePath:    filepath.Join(suite.T().TempDir(), "test.db"),
		},
	}
	suite.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (suite *AppSetupSuite) TearDownTest() {
	if suite.params.CancelFn != nil {
		suite.params.CancelFn()
	}
}

func TestAppSetupSuite(t *testing.T) {
	suite.Run(t, new(AppSetupSuite))
}

func (suite *AppSetupSuite) TestNewFXLoggerOption() {
	app := fxtest.New(
		suite.T(),
		fx.Provide(func() *slog.Logger { return suite.logger }),
		NewFXLoggerOption(),
	)

	app.RequireStart()
	app.RequireStop()
}

func (suite *AppSetupSuite) TestProvideHAClientDependencies() {
	var (
		addonsClient     addons.ClientWithResponsesInterface
		hardwareClient   hardware.ClientWithResponsesInterface
		mountClient      mount.ClientWithResponsesInterface
		hostClient       host.ClientWithResponsesInterface
		resolutionClient resolution.ClientWithResponsesInterface
		coreAPIClient    core_api.ClientWithResponsesInterface
		rootClient       root.ClientWithResponsesInterface
		ingressClient    ingress.ClientWithResponsesInterface
		websocketClient  websocket.ClientInterface
	)

	app := fxtest.New(
		suite.T(),
		ProvideHAClientDependencies(suite.params),
		fx.Populate(
			&addonsClient,
			&hardwareClient,
			&mountClient,
			&hostClient,
			&resolutionClient,
			&coreAPIClient,
			&rootClient,
			&ingressClient,
			&websocketClient,
		),
	)
	app.RequireStart()
	suite.T().Cleanup(func() { app.RequireStop() })

	suite.Require().NotNil(addonsClient)
	suite.Require().NotNil(hardwareClient)
	suite.Require().NotNil(mountClient)
	suite.Require().NotNil(hostClient)
	suite.Require().NotNil(resolutionClient)
	suite.Require().NotNil(coreAPIClient)
	suite.Require().NotNil(rootClient)
	suite.Require().NotNil(ingressClient)
	if client, ok := addonsClient.(*addons.ClientWithResponses); ok {
		if core, ok := client.ClientInterface.(*addons.Client); ok {
			suite.Equal("http://example.org/", core.Server)
		} else {
			suite.T().Fatalf("unexpected addons client interface type %T", client.ClientInterface)
		}
	} else {
		suite.T().Fatalf("unexpected addons client type %T", addonsClient)
	}

	suite.Require().NotNil(websocketClient)
}

func (suite *AppSetupSuite) TestProvideCoreDependenciesReturnsOption() {
	option := ProvideCoreDependencies(suite.params)
	suite.Require().NotNil(option)
}

func (suite *AppSetupSuite) TestProvideCyclicDependencyWorkaroundOption() {
	var shareService service.ShareServiceInterface
	var supervisorService service.SupervisorServiceInterface

	app := fxtest.New(
		suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.ShareServiceInterface],
			mock.Mock[service.SupervisorServiceInterface],
		),
		ProvideCyclicDependencyWorkaroundOption(),
		fx.Populate(&shareService, &supervisorService),
	)

	app.RequireStart()
	app.RequireStop()

	//mock.Verify(shareService, matchers.Times(1)).SetSupervisorService(supervisorService)
}

func (suite *AppSetupSuite) TestProvideFrontendOption() {
	original := internal.Frontend
	internal.Frontend = nil
	suite.T().Cleanup(func() { internal.Frontend = original })

	var fs http.FileSystem
	app := fxtest.New(
		suite.T(),
		ProvideFrontendOption(),
		fx.Populate(&fs),
	)

	app.RequireStart()
	suite.Require().NotNil(fs)
	app.RequireStop()
}
