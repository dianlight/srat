package appsetup

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/apps"
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
	"github.com/google/go-github/v90/github"
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
		addonsClient     apps.ClientWithResponsesInterface
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
	if client, ok := addonsClient.(*apps.ClientWithResponses); ok {
		if core, ok := client.ClientInterface.(*apps.Client); ok {
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

// TestProvideCoreDependencies_StartsGraph starts the full core dependency
// graph so the provider closures (logger, ctx, ContextState, DiskMap,
// EventBus, github.Client) and the fx.Invoke blocks (AddonConfigWatcher,
// ProblemHABridge, command executor wiring) actually execute.
func (suite *AppSetupSuite) TestProvideCoreDependencies_StartsGraph() {
	// The github.Client provider requires a non-empty token; provide a dummy
	// one for the duration of the test and restore afterwards.
	origToken := config.GistToken
	config.GistToken = "test-token"
	suite.T().Cleanup(func() { config.GistToken = origToken })

	app := fxtest.New(
		suite.T(),
		ProvideCoreDependencies(suite.params),
		ProvideHAClientDependencies(suite.params),
		ProvideCyclicDependencyWorkaroundOption(),
		// Force construction of the provider closures that nothing else
		// in the graph depends on (logger + github client).
		fx.Invoke(func(*slog.Logger, *github.Client) {}),
	)
	app.RequireStart()
	suite.T().Cleanup(func() { app.RequireStop() })
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
