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
	app         *fxtest.App
	state       *dto.ContextState
	events      events.EventBusInterface
	haSvc       service.HomeAssistantServiceInterface
	settingSvc  service.SettingServiceInterface
	broadcaster service.BroadcasterServiceInterface
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
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.BroadcasterServiceInterface],
			service.NewProblemHABridge,
		),
		fx.Populate(&suite.state),
		fx.Populate(&suite.events),
		fx.Populate(&suite.haSvc),
		fx.Populate(&suite.settingSvc),
		fx.Populate(&suite.broadcaster),
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
		Type: events.EventTypes.ADD,
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
		Type: events.EventTypes.ADD,
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
		Type: events.EventTypes.ADD,
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

func (suite *ProblemHABridgeSuite) TestCustomComponentProblemSuppressedWhenLabNotActive() {
	// Suite provides no SettingService, so the lab gate is fail-closed off.
	suite.state.HAWsComponent = nil
	suite.events.EmitProblem(events.ProblemEvent{
		Type: events.EventTypes.ADD,
		Problem: &dto.Problem{
			ProblemKey:   "custom_component_missing",
			Title:        "Home Assistant SRAT custom component missing and disconnected",
			Description:  "missing",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
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
		Type: events.EventTypes.ADD,
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
		Type: events.EventTypes.ADD,
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

func (suite *ProblemHABridgeSuite) TestRemoveEventDismissesNotification() {
	dismissed := make(chan string, 1)
	mock.When(
		suite.haSvc.DismissPersistentNotification(mock.Any[string]()),
	).ThenAnswer(func(args []any) []any {
		id, _ := args[0].(string)
		dismissed <- id
		return []any{nil}
	})

	suite.state.HAWsComponent = nil
	suite.events.EmitProblem(events.ProblemEvent{
		Type: events.EventTypes.REMOVE,
		Problem: &dto.Problem{
			ProblemKey:   "problem.removed",
			Title:        "Removed title",
			Description:  "Removed body",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
			Status:       dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
			IsPersistent: true,
		},
	})

	suite.Eventually(func() bool {
		select {
		case id := <-dismissed:
			return id == "srat_problem_problem.removed"
		default:
			return false
		}
	}, time.Second, 20*time.Millisecond)
	_ = mock.Verify(suite.haSvc, matchers.Times(0)).CreatePersistentNotification(
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[string](),
	)
}

func (suite *ProblemHABridgeSuite) TestIgnoredProblemSuppressesNotification() {
	dismissed := make(chan string, 1)
	mock.When(
		suite.haSvc.DismissPersistentNotification(mock.Any[string]()),
	).ThenAnswer(func(args []any) []any {
		id, _ := args[0].(string)
		dismissed <- id
		return []any{nil}
	})

	suite.state.HAWsComponent = nil
	suite.events.EmitProblem(events.ProblemEvent{
		Type: events.EventTypes.UPDATE,
		Problem: &dto.Problem{
			ProblemKey:   "problem.ignored",
			Title:        "Ignored title",
			Description:  "Ignored body",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYERROR,
			Status:       dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSIGNORED,
			Ignored:      true,
			IsPersistent: true,
		},
	})

	suite.Eventually(func() bool {
		select {
		case id := <-dismissed:
			return id == "srat_problem_problem.ignored"
		default:
			return false
		}
	}, time.Second, 20*time.Millisecond)
	_ = mock.Verify(suite.haSvc, matchers.Times(0)).CreatePersistentNotification(
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[string](),
	)
}

func (suite *ProblemHABridgeSuite) TestDisabledAlertSuppressesNotification() {
	mock.When(suite.settingSvc.Load()).ThenReturn(
		&dto.Settings{AlertProtectedMode: new(false)},
		nil,
	)
	dismissed := make(chan string, 1)
	mock.When(
		suite.haSvc.DismissPersistentNotification(mock.Any[string]()),
	).ThenAnswer(func(args []any) []any {
		id, _ := args[0].(string)
		dismissed <- id
		return []any{nil}
	})

	suite.state.HAWsComponent = nil
	suite.events.EmitProblem(events.ProblemEvent{
		Type: events.EventTypes.ADD,
		Problem: &dto.Problem{
			ProblemKey:   "protected_mode",
			Title:        "Addon in Protected Mode",
			Description:  "Protected body",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYERROR,
			Status:       dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
			IsPersistent: true,
		},
	})

	suite.Eventually(func() bool {
		select {
		case id := <-dismissed:
			return id == "srat_problem_protected_mode"
		default:
			return false
		}
	}, time.Second, 20*time.Millisecond)
	_ = mock.Verify(suite.haSvc, matchers.Times(0)).CreatePersistentNotification(
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[string](),
	)
}

func (suite *ProblemHABridgeSuite) TestTerminalStatusDismissesNotification() {
	dismissed := make(chan string, 1)
	mock.When(
		suite.haSvc.DismissPersistentNotification(mock.Any[string]()),
	).ThenAnswer(func(args []any) []any {
		id, _ := args[0].(string)
		dismissed <- id
		return []any{nil}
	})

	suite.state.HAWsComponent = nil
	suite.events.EmitProblem(events.ProblemEvent{
		Type: events.EventTypes.UPDATE,
		Problem: &dto.Problem{
			ProblemKey:   "problem.fixed",
			Title:        "Fixed title",
			Description:  "Fixed body",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
			Status:       dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED,
			IsPersistent: true,
		},
	})

	suite.Eventually(func() bool {
		select {
		case id := <-dismissed:
			return id == "srat_problem_problem.fixed"
		default:
			return false
		}
	}, time.Second, 20*time.Millisecond)
	_ = mock.Verify(suite.haSvc, matchers.Times(0)).CreatePersistentNotification(
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[string](),
	)
}

func (suite *ProblemHABridgeSuite) TestConnectedErrorBroadcastsRepair() {
	broadcasted := make(chan string, 1)
	mock.When(
		suite.broadcaster.BroadcastMessage(mock.Any[dto.Problem]()),
	).ThenAnswer(func(args []any) []any {
		problem, _ := args[0].(dto.Problem)
		broadcasted <- problem.ProblemKey
		return []any{args[0]}
	})

	suite.state.HAWsComponent = &dto.HomeAssistantComponentConnection{Component: dto.HomeAssistantComponentSRAT}
	suite.events.EmitProblem(events.ProblemEvent{
		Type: events.EventTypes.ADD,
		Problem: &dto.Problem{
			ProblemKey:   "problem.repair",
			Title:        "Repair title",
			Description:  "Repair body",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYERROR,
			Status:       dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
			IsPersistent: true,
		},
	})

	suite.Eventually(func() bool {
		select {
		case key := <-broadcasted:
			return key == "problem.repair"
		default:
			return false
		}
	}, time.Second, 20*time.Millisecond)
	_ = mock.Verify(suite.haSvc, matchers.Times(0)).CreatePersistentNotification(
		mock.Any[string](),
		mock.Any[string](),
		mock.Any[string](),
	)
}

func (suite *ProblemHABridgeSuite) TestOfflineCreateEnqueuedAndFlushed() {
	created := make(chan string, 2)
	mock.When(
		suite.haSvc.CreatePersistentNotification(
			mock.Any[string](),
			mock.Any[string](),
			mock.Any[string](),
		),
	).ThenAnswer(func(args []any) []any {
		id, _ := args[0].(string)
		created <- id
		return []any{nil}
	})

	suite.state.HAWsComponent = nil
	suite.state.HACoreReady = false
	suite.events.EmitProblem(events.ProblemEvent{
		Type: events.EventTypes.ADD,
		Problem: &dto.Problem{
			ProblemKey:   "problem.offline",
			Title:        "Offline title",
			Description:  "Offline body",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
			Status:       dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
			IsPersistent: true,
		},
	})

	// While HA is not ready nothing is created yet.
	suite.Never(func() bool {
		select {
		case <-created:
			return true
		default:
			return false
		}
	}, 100*time.Millisecond, 20*time.Millisecond)

	// Once HA is ready the queued notification flushes.
	suite.state.HACoreReady = true
	suite.events.EmitProblem(events.ProblemEvent{
		Type: events.EventTypes.ADD,
		Problem: &dto.Problem{
			ProblemKey:   "problem.offline2",
			Title:        "Offline title 2",
			Description:  "Offline body 2",
			Severity:     dto.ProblemSeverities.PROBLEMSEVERITYWARNING,
			Status:       dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
			IsPersistent: true,
		},
	})

	suite.Eventually(func() bool {
		return len(created) >= 2
	}, time.Second, 20*time.Millisecond)
}
