package service

import (
	"context"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/tlog"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// ProtectedModeAlertServiceInterface is a nominal marker type used by the FX dependency injection container.
// It has no methods; callers never interact with it directly — FX resolves the concrete
// *ProtectedModeAlertService via fx.Invoke so the service lifecycle hooks are registered at startup.
type ProtectedModeAlertServiceInterface any

// ProtectedModeAlertService raises a persistent protected_mode problem while
// the addon runs in protected mode so it surfaces in the dashboard and in
// Home Assistant (repair issue when the custom component is connected,
// persistent notification otherwise).
//
// The alert honors the Alerts settings category and permanent ignores:
//   - alert disabled in settings → any existing problem is dismissed and no
//     new one is raised until re-enabled.
//   - problem ignored (dashboard Ignore, HA repair ignore lifecycle, or HA
//     repairs UI ignore with a stable issue id) → never re-raised.
//   - dismissing the problem (DELETE) deletes the row and re-arms the alert,
//     so it is raised again on the next reconcile while protected mode is on.
type ProtectedModeAlertService struct {
	ctx            context.Context
	state          *dto.ContextState
	eventBus       events.EventBusInterface
	problemService ProblemServiceInterface
	settingService SettingServiceInterface
}

type ProtectedModeAlertServiceParams struct {
	fx.In
	Ctx            context.Context
	State          *dto.ContextState
	EventBus       events.EventBusInterface
	ProblemService ProblemServiceInterface
	SettingService SettingServiceInterface
}

func NewProtectedModeAlertService(lc fx.Lifecycle, params ProtectedModeAlertServiceParams) ProtectedModeAlertServiceInterface {
	svc := &ProtectedModeAlertService{
		ctx:            params.Ctx,
		state:          params.State,
		eventBus:       params.EventBus,
		problemService: params.ProblemService,
		settingService: params.SettingService,
	}

	unsubscribe := params.EventBus.OnSetting(func(ctx context.Context, event events.SettingEvent) errors.E {
		svc.Reconcile()
		return nil
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			svc.Reconcile()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if unsubscribe != nil {
				unsubscribe()
			}
			return nil
		},
	})

	return svc
}

// Reconcile aligns the protected_mode problem with the current runtime state,
// alert settings and stored ignore. It is safe to call repeatedly.
func (s *ProtectedModeAlertService) Reconcile() {
	if s.problemService == nil {
		return
	}

	if s.state == nil || !s.state.ProtectedMode {
		// Protected mode off: the condition is gone, drop any stale problem.
		if err := s.problemService.Dismiss(AlertProblemKeyProtectedMode); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			tlog.WarnContext(s.ctx, "protected_mode alert: could not dismiss stale problem", "error", err)
		}
		return
	}

	settings, err := s.settingService.Load()
	if err != nil {
		tlog.WarnContext(s.ctx, "protected_mode alert: could not load settings, skipping reconcile", "error", err)
		return
	}

	if !AlertEnabledForKey(settings, AlertProblemKeyProtectedMode) {
		// Alert disabled in the Alerts settings category: dismiss and stay silent.
		if err := s.problemService.Dismiss(AlertProblemKeyProtectedMode); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			tlog.WarnContext(s.ctx, "protected_mode alert: could not dismiss disabled problem", "error", err)
		}
		return
	}

	if existing, err := s.problemService.Get(AlertProblemKeyProtectedMode); err == nil && existing != nil && existing.Ignored {
		// Permanent ignore: never raise again until dismissed/re-enabled.
		return
	}

	_, upsertErr := s.problemService.Upsert(&dto.Problem{
		ProblemKey:  AlertProblemKeyProtectedMode,
		Title:       "Addon in Protected Mode",
		Description: "The addon is currently in protected mode. In this mode, no disks can be mounted to prevent unauthorized access. To disable protected mode, navigate to the addon settings in your Home Assistant interface and toggle the protected mode option off. Ensure you understand the security implications before disabling.",
		Severity:    dto.ProblemSeverities.PROBLEMSEVERITYERROR,
		Status:      dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
		// Error severity routes to the HA repair path (ignorable repair
		// issue) when the custom component is connected; otherwise the
		// bridge falls back to a persistent notification.
		TranslationKey: AlertProblemKeyProtectedMode,
		IsFixable:      false,
		IsPersistent:   true,
	})
	if upsertErr != nil {
		tlog.WarnContext(s.ctx, "protected_mode alert: could not upsert problem", "error", upsertErr)
	}
}
