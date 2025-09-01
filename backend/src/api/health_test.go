// /workspaces/srat/backend/src/api/health_test.go
package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
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

type HealthHandlerSuite struct {
	suite.Suite
	api              *api.HealthHanler
	mockBroadcaster  service.BroadcasterServiceInterface
	mockSambaService service.SambaServiceInterface
	mockDirtyService service.DirtyDataServiceInterface
	ctrl             *matchers.MockController
	testAPIContext   *dto.ContextState
	ctx              context.Context
	cancel           context.CancelFunc
	app              *fxtest.App
}

// SetupSuite runs once before the tests in the suite are run
func (suite *HealthHandlerSuite) SetupTest() {

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			api.NewHealthHandler,
			mock.Mock[service.BroadcasterServiceInterface],
			mock.Mock[service.SambaServiceInterface],
			mock.Mock[service.DirtyDataServiceInterface],
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.DiskStatsService],
			mock.Mock[service.NetworkStatsService],
			mock.Mock[service.HaRootServiceInterface],
			func() *dto.ContextState {
				sharedResources := dto.ContextState{}
				sharedResources.ReadOnlyMode = false
				sharedResources.Heartbeat = 1
				sharedResources.DockerInterface = "hassio"
				sharedResources.DockerNet = "172.30.32.0/23"
				var err error
				sharedResources.Template, err = os.ReadFile("../templates/smb.gtpl")
				if err != nil {
					suite.T().Errorf("Cant read template file %s", err)
				}

				return &sharedResources
			},
		),
		fx.Populate(&suite.mockBroadcaster),
		fx.Populate(&suite.mockSambaService),
		fx.Populate(&suite.mockDirtyService),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
		fx.Populate(&suite.testAPIContext),
		fx.Populate(&suite.ctrl),
		fx.Populate(&suite.api),
	)
	suite.app.RequireStart()
}

// TearDownSuite runs once after all tests in the suite have finished
func (suite *HealthHandlerSuite) TearDownTest() {
	suite.cancel()
	suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
	suite.app.RequireStop()
}

func TestHealthHandlerSuite(t *testing.T) {
	suite.Run(t, new(HealthHandlerSuite))
}

func (suite *HealthHandlerSuite) TestHealthCheckHandler() {
	// Mock dependencies for the run goroutine (needed for New)
	mock.When(suite.mockSambaService.GetSambaProcess()).ThenReturn(nil, errors.New("initial mock"))
	mock.When(suite.mockDirtyService.GetDirtyDataTracker()).ThenReturn(dto.DataDirtyTracker{})
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[dto.HealthPing]())).ThenReturn(nil, nil)

	_, testAPI := humatest.New(suite.T())
	suite.api.RegisterVolumeHandlers(testAPI) // Register its handler

	// Update some state for the check
	suite.api.Alive = true
	suite.api.LastError = "test error"

	rr := testAPI.Get("/health")
	suite.Equal(http.StatusOK, rr.Code)

	var response dto.HealthPing
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	suite.Require().NoError(err, "Failed to unmarshal response body")

	// Check if the response matches the handler's state
	suite.Equal(suite.api.HealthPing, response)
	suite.True(response.Alive)
	suite.Equal("test error", response.LastError)
}
