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
			fx.Annotate(
				func() bool { return false },
				fx.ResultTags(`name:"ha_mode"`),
			),
			api.NewHealthHandler,
			mock.Mock[service.BroadcasterServiceInterface],
			mock.Mock[service.SambaServiceInterface],
			mock.Mock[service.DirtyDataServiceInterface],
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

func (suite *HealthHandlerSuite) TestNewHealthHandler_Singleton() {
	params := api.HealthHandlerParams{
		Ctx:          suite.ctx,
		Apictx:       suite.testAPIContext,
		Broadcaster:  suite.mockBroadcaster,
		SambaService: suite.mockSambaService,
		DirtyService: suite.mockDirtyService,
		HaMode:       false,
	}

	// Mock dependencies for the run goroutine
	mock.When(suite.mockSambaService.GetSambaProcess()).ThenReturn(nil, errors.New("initial mock"))
	mock.When(suite.mockDirtyService.GetDirtyDataTracker()).ThenReturn(dto.DataDirtyTracker{})
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[dto.HealthPing]())).ThenReturn(nil, nil)

	handler2 := api.NewHealthHandler(params)
	suite.Require().NotNil(handler2)

	// Assert that the same instance is returned
	suite.Same(suite.api, handler2, "NewHealthHandler should return a singleton instance")
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
	suite.api.BuildVersion = "v1.2.3"

	rr := testAPI.Get("/health")
	suite.Equal(http.StatusOK, rr.Code)

	var response dto.HealthPing
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	suite.Require().NoError(err, "Failed to unmarshal response body")

	// Check if the response matches the handler's state
	suite.Equal(suite.api.HealthPing, response)
	suite.True(response.Alive)
	suite.Equal("test error", response.LastError)
	suite.Equal("v1.2.3", response.BuildVersion)
}

func (suite *HealthHandlerSuite) TestEventEmitter_Success() {
	suite.T().Skip("Skipping test due to unimplemented mock behavior")

	// Mock dependencies for the run goroutine (needed for New)
	mock.When(suite.mockSambaService.GetSambaProcess()).ThenReturn(nil, errors.New("initial mock"))
	mock.When(suite.mockDirtyService.GetDirtyDataTracker()).ThenReturn(dto.DataDirtyTracker{})
	//mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[dto.HealthPing]())).ThenReturn(nil, nil).Verify(mock.Times(1))

	pingData := dto.HealthPing{Alive: true, BuildVersion: "emit-test"}

	// Expect the broadcast call
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[any]())).ThenReturn(nil, nil).Verify(mock.Times(1))

	err := suite.api.EventEmitter(pingData)

	suite.NoError(err)

	//suite.cancel()
	//suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
}

func (suite *HealthHandlerSuite) TestEventEmitter_Error() {
	suite.T().Skip("Skipping test due to unimplemented mock behavior")
	// Mock dependencies for the run goroutine (needed for New)
	mock.When(suite.mockSambaService.GetSambaProcess()).ThenReturn(nil, errors.New("initial mock"))
	mock.When(suite.mockDirtyService.GetDirtyDataTracker()).ThenReturn(dto.DataDirtyTracker{})
	//mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[dto.HealthPing]())).ThenReturn(nil, nil)

	initialCount := suite.api.OutputEventsCount

	pingData := dto.HealthPing{Alive: true, BuildVersion: "emit-test-fail"}
	expectedErr := errors.New("broadcast failed")

	// Expect the broadcast call to fail
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Equal(pingData))).ThenReturn(nil, expectedErr).Verify(mock.Times(1))

	err := suite.api.EventEmitter(pingData)

	suite.Error(err)
	suite.True(errors.Is(err, expectedErr), "Expected wrapped broadcast error")
	suite.Equal(initialCount, suite.api.OutputEventsCount, "OutputEventsCount should not increment on failed emit")

	suite.cancel()
	suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
}

/*
func (suite *HealthHandlerSuite) TestRun_PeriodicCallsAndCancellation() {
	// Use a slightly longer heartbeat for this test to avoid race conditions
	suite.testAPIContext.Heartbeat = 100 // Milliseconds
	params := api.HealthHandlerParams{
		Ctx:          suite.ctx,
		Apictx:       suite.testAPIContext,
		Broadcaster:  suite.mockBroadcaster,
		SambaService: suite.mockSambaService,
		DirtyService: suite.mockDirtyService,
		HaMode:       false,
	}
	interleave := time.Duration(suite.testAPIContext.Heartbeat) * time.Millisecond

	// --- Mocking ---
	// Allow any number of calls during the run loop
	mock.When(suite.mockSambaService.GetSambaProcess()).ThenReturn(nil, nil)
	mock.When(suite.mockDirtyService.GetDirtyDataTracker()).ThenReturn(dto.DataDirtyTracker{Settings: true}) // Return some dirty state
	mock.When(suite.mockBroadcaster.BroadcastMessage(mock.Any[dto.HealthPing]())).ThenAnswer(
		matchers.Answer(func(args []any) []any {
			// We can inspect the message here if needed
			// msg := args[0].(dto.HealthPing)
			// suite.T().Logf("Broadcasting: %+v", msg)
			return []any{nil, nil}
		}),
	)

	// --- Execution ---
	handler := api.NewHealthHandler(params) // This starts the run() goroutine

	// Let it run for a few cycles
	runCycles := 3
	time.Sleep(interleave*time.Duration(runCycles) + interleave/2) // Sleep for slightly more than N cycles

	// --- Verification (before cancellation) ---
	// Verify mocks were called approximately 'runCycles' times
	// Use AtLeast because timing can be tricky
	mock.Verify(suite.mockSambaService, mock.Times(runCycles)).GetSambaProcess()
	mock.Verify(suite.mockDirtyService, mock.Times(runCycles)).GetDirtyDataTracker()
	mock.Verify(suite.mockBroadcaster, mock.Times(runCycles)).BroadcastMessage(mock.Any[dto.HealthPing]())
	suite.GreaterOrEqual(handler.OutputEventsCount, uint64(runCycles), "Should have emitted at least %d events", runCycles)

	// Reset heartbeat for other tests
	suite.testAPIContext.Heartbeat = 1
}
*/
