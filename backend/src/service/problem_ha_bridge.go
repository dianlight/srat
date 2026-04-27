package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/tlog"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// ProblemHABridgeInterface is a nominal marker type used by the FX dependency injection container.
// It has no methods; callers never interact with it directly — FX resolves the concrete *ProblemHABridge
// via fx.Invoke so the service lifecycle hooks are registered at startup.
type ProblemHABridgeInterface interface{}

type problemNotificationAction struct {
	dismiss bool
	id      string
	title   string
	message string
}

type ProblemHABridge struct {
	ctx         context.Context
	state       *dto.ContextState
	eventBus    events.EventBusInterface
	haService   HomeAssistantServiceInterface
	broadcaster BroadcasterServiceInterface

	mu    sync.Mutex
	queue []problemNotificationAction
}

type ProblemHABridgeParams struct {
	fx.In
	Ctx         context.Context
	State       *dto.ContextState
	EventBus    events.EventBusInterface
	HAService   HomeAssistantServiceInterface `optional:"true"`
	Broadcaster BroadcasterServiceInterface   `optional:"true"`
}

func NewProblemHABridge(lc fx.Lifecycle, params ProblemHABridgeParams) ProblemHABridgeInterface {
	bridge := &ProblemHABridge{
		ctx:         params.Ctx,
		state:       params.State,
		eventBus:    params.EventBus,
		haService:   params.HAService,
		broadcaster: params.Broadcaster,
		queue:       make([]problemNotificationAction, 0),
	}

	var unsubscribe func()

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			unsubscribe = bridge.eventBus.OnProblem(bridge.handleProblemEvent)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if unsubscribe != nil {
				unsubscribe()
			}
			return nil
		},
	})

	return bridge
}

func (b *ProblemHABridge) handleProblemEvent(ctx context.Context, event events.ProblemEvent) errors.E {
	if event.Problem == nil {
		return nil
	}

	problem := event.Problem
	notificationID, title, message := toNotificationPayload(problem)

	if isTerminalProblemStatus(problem.Status) {
		action := problemNotificationAction{dismiss: true, id: notificationID}
		if !b.canUseHA() {
			b.enqueue(action)
			return nil
		}
		if err := b.flushQueue(); err != nil {
			tlog.WarnContext(ctx, "Failed to flush queued HA problem notifications", "error", err)
		}
		if err := b.haService.DismissPersistentNotification(notificationID); err != nil {
			tlog.WarnContext(ctx, "Failed to dismiss HA persistent notification for problem", "problem_key", problem.ProblemKey, "error", err)
		}
		return nil
	}

	// Routing policy:
	// - Connected + critical/error => HA repair path via WS broadcast (Problem event fan-out).
	// - Connected + warning/info => HA persistent notification.
	// - Disconnected => all severities => HA persistent notification.
	if b.isComponentConnected() && isRepairSeverity(problem.Severity) {
		if b.broadcaster != nil {
			b.broadcaster.BroadcastMessage(*problem)
		}
		return nil
	}

	action := problemNotificationAction{dismiss: false, id: notificationID, title: title, message: message}
	if !b.canUseHA() {
		b.enqueue(action)
		return nil
	}

	if err := b.flushQueue(); err != nil {
		tlog.WarnContext(ctx, "Failed to flush queued HA problem notifications", "error", err)
	}
	if err := b.haService.CreatePersistentNotification(notificationID, title, message); err != nil {
		tlog.WarnContext(ctx, "Failed to create HA persistent notification for problem", "problem_key", problem.ProblemKey, "error", err)
	}

	return nil
}

func (b *ProblemHABridge) enqueue(action problemNotificationAction) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.queue = append(b.queue, action)
}

func (b *ProblemHABridge) flushQueue() error {
	if !b.canUseHA() {
		return nil
	}

	b.mu.Lock()
	queued := make([]problemNotificationAction, len(b.queue))
	copy(queued, b.queue)
	b.queue = b.queue[:0]
	b.mu.Unlock()

	for _, action := range queued {
		if action.dismiss {
			if err := b.haService.DismissPersistentNotification(action.id); err != nil {
				return err
			}
			continue
		}
		if err := b.haService.CreatePersistentNotification(action.id, action.title, action.message); err != nil {
			return err
		}
	}

	return nil
}

func (b *ProblemHABridge) canUseHA() bool {
	return b.haService != nil && b.state != nil && b.state.HACoreReady
}

func (b *ProblemHABridge) isComponentConnected() bool {
	return b.state != nil && b.state.HAWsComponent != nil &&
		b.state.HAWsComponent.Component == dto.HomeAssistantComponentSRAT
}

func isRepairSeverity(severity dto.ProblemSeverity) bool {
	return severity == dto.ProblemSeverities.PROBLEMSEVERITYERROR || severity == dto.ProblemSeverities.PROBLEMSEVERITYCRITICAL
}

func isTerminalProblemStatus(status dto.ProblemLifecycleStatus) bool {
	return status == dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSDISMISSED ||
		status == dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSDELETED ||
		status == dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED
}

func toNotificationPayload(problem *dto.Problem) (id, title, message string) {
	key := strings.TrimSpace(problem.ProblemKey)
	if key == "" {
		key = strings.TrimSpace(problem.Title)
	}
	if key == "" {
		key = "unknown"
	}
	id = fmt.Sprintf("srat_problem_%s", key)

	title = strings.TrimSpace(problem.Title)
	if title == "" {
		title = "SRAT problem"
	}

	message = strings.TrimSpace(problem.Description)
	if message == "" {
		message = title
	}

	return id, title, message
}
