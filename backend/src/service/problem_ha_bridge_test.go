package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type ProblemHABridgeSuite struct {
	suite.Suite
	app    *fxtest.App
	state  *dto.ContextState
	events events.EventBusInterface
	haSvc  service.HomeAssistantServiceInterface
}

func TestProblemHABridgeSuite(t *testing.T) {
	suite.Run(t, new(ProblemHABridgeSuite))
}

func (suite *ProblemHABridgeSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() context.Context {
				return context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{})
			},
			func() *dto.ContextState {
				return &dto.ContextState{HACoreReady: true}
			},
			events.NewEventBus,
			mock.Mock[service.HomeAssistantServiceInterface],
			service.NewProblemHABridge,
		),
		fx.Populate(&suite.state),
		fx.Populate(&suite.events),
		fx.Populate(&suite.haSvc),
		fx.Invoke(func(service.ProblemHABridgeInterface) {}),
	)

	suite.app.RequireStart()
}

func (suite *ProblemHABridgeSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *ProblemHABridgeSuite) TestConnectedWarningCreatesPersistentNotification() {
	mock.When(
		suite.haSvc.CreatePersistentNotification(
			mock.Exact("srat_problem_problem.warning"),
			mock.Exact("Warning title"),
			mock.Exact("Warning body"),
		),
	).ThenReturn(nil)

	suite.state.HAWsComponent = &dto.HomeAssistantComponentConnection{Component: dto.HomeAssistantComponentSRAT}
	suite.events.EmitProblem(events.ProblemEvent{
		Event: events.Event{Type: events.EventTypes.ADD},
		Problem: &dto.Problem{
			ProblemKey:   "problem.warning",
			Title:        "Warning title",
			Description:  "Warning body",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
			Status:       dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
			IsPersistent: true,
		},
	})

	_ = mock.Verify(suite.haSvc, matchers.Times(1)).CreatePersistentNotification(
		mock.Exact("srat_problem_problem.warning"),
		mock.Exact("Warning title"),
		mock.Exact("Warning body"),
	)
}

func (suite *ProblemHABridgeSuite) TestDisconnectedErrorCreatesPersistentNotification() {
	mock.When(
		suite.haSvc.CreatePersistentNotification(
			mock.Exact("srat_problem_problem.error"),
			mock.Exact("Error title"),
			mock.Exact("Error body"),
		),
	).ThenReturn(nil)

	suite.state.HAWsComponent = nil
	suite.events.EmitProblem(events.ProblemEvent{
		Event: events.Event{Type: events.EventTypes.ADD},
		Problem: &dto.Problem{
			ProblemKey:   "problem.error",
			Title:        "Error title",
			Description:  "Error body",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYERROR,
			Status:       dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
			IsPersistent: true,
		},
	})

	_ = mock.Verify(suite.haSvc, matchers.Times(1)).CreatePersistentNotification(
		mock.Exact("srat_problem_problem.error"),
		mock.Exact("Error title"),
		mock.Exact("Error body"),
	)
}

func (suite *ProblemHABridgeSuite) TestConnectedErrorDoesNotCreatePersistentNotification() {
	suite.state.HAWsComponent = &dto.HomeAssistantComponentConnection{Component: dto.HomeAssistantComponentSRAT}
	suite.events.EmitProblem(events.ProblemEvent{
		Event: events.Event{Type: events.EventTypes.ADD},
		Problem: &dto.Problem{
			ProblemKey:   "problem.critical",
			Title:        "Critical title",
			Description:  "Critical body",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYCRITICAL,
			Status:       dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
			IsPersistent: true,
		},
	})

	_ = mock.Verify(suite.haSvc, matchers.Times(0)).CreatePersistentNotification(
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[string](),
	)
}

func (suite *ProblemHABridgeSuite) TestOfflineQueueFlushesWhenHABecomesReady() {
	calls := make(chan string, 4)
	mock.When(
		suite.haSvc.CreatePersistentNotification(
			mock.Any[string](),
			mock.Any[string](),
			mock.Any[string](),
		),
	).ThenAnswer(func(args []any) []any {
		id, _ := args[0].(string)
		calls <- id
		return []any{nil}
	})

	suite.state.HACoreReady = false
	suite.events.EmitProblem(events.ProblemEvent{
		Event: events.Event{Type: events.EventTypes.ADD},
		Problem: &dto.Problem{
			ProblemKey:   "problem.queued",
			Title:        "Queued title",
			Description:  "Queued body",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
			Status:       dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
			IsPersistent: true,
		},
	})

	suite.state.HACoreReady = true
	suite.events.EmitProblem(events.ProblemEvent{
		Event: events.Event{Type: events.EventTypes.ADD},
		Problem: &dto.Problem{
			ProblemKey:   "problem.live",
			Title:        "Live title",
			Description:  "Live body",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
			Status:       dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
			IsPersistent: true,
		},
	})

	suite.Eventually(func() bool {
		return len(calls) >= 2
	}, time.Second, 20*time.Millisecond)
}
