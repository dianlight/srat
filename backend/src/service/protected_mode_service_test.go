// Package service contains internal unit tests for ProtectedModeAlertService.
package service

import (
	"context"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/require"
	tozderrors "gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type stubAlertProblemStore struct {
	upserts   []*dto.Problem
	dismissed []string
	existing  *dto.Problem
}

func (s *stubAlertProblemStore) Upsert(problem *dto.Problem) (*dto.Problem, error) {
	if problem != nil {
		clone := *problem
		s.upserts = append(s.upserts, &clone)
	}
	return problem, nil
}

func (s *stubAlertProblemStore) Dismiss(problemKey string) error {
	s.dismissed = append(s.dismissed, problemKey)
	return nil
}

func (s *stubAlertProblemStore) Get(problemKey string) (*dto.Problem, error) {
	if s.existing != nil && s.existing.ProblemKey == problemKey {
		clone := *s.existing
		return &clone, nil
	}
	return nil, dto.ErrorNotFound
}

func (s *stubAlertProblemStore) List() ([]*dto.Problem, error) { return nil, nil }

func (s *stubAlertProblemStore) ApplyLifecycle(_ string, _ dto.ProblemLifecycleStatus, _ *string) (*dto.Problem, error) {
	return nil, dto.ErrorNotFound
}

type stubAlertSettings struct {
	settings *dto.Settings
	err      tozderrors.E
}

func (s *stubAlertSettings) Load() (*dto.Settings, tozderrors.E) { return s.settings, s.err }

func (s *stubAlertSettings) UpdateSettings(*dto.Settings) tozderrors.E { return nil }

func (s *stubAlertSettings) SetCommandExists(func(cmd []string) bool) {}

func (s *stubAlertSettings) DumpTable() (string, tozderrors.E) { return "", nil }

func newProtectedModeTestService(protected bool, settings *dto.Settings, settingsErr tozderrors.E, existing *dto.Problem) (*ProtectedModeAlertService, *stubAlertProblemStore) {
	store := &stubAlertProblemStore{existing: existing}
	svc := &ProtectedModeAlertService{
		ctx:            context.Background(),
		state:          &dto.ContextState{ProtectedMode: protected},
		problemService: store,
		settingService: &stubAlertSettings{settings: settings, err: settingsErr},
	}
	return svc, store
}

func TestProtectedModeReconcile_RaisesProblemWhenProtected(t *testing.T) {
	svc, store := newProtectedModeTestService(true, &dto.Settings{}, nil, nil)
	svc.Reconcile()

	require.Len(t, store.upserts, 1)
	problem := store.upserts[0]
	require.Equal(t, "protected_mode", problem.ProblemKey)
	require.Equal(t, dto.ProblemSeverities.PROBLEMSEVERITYERROR, problem.Severity)
	require.Equal(t, "protected_mode", problem.TranslationKey)
	require.False(t, problem.IsFixable)
	require.True(t, problem.IsPersistent)
	require.Empty(t, store.dismissed)
}

func TestProtectedModeReconcile_DismissesWhenNotProtected(t *testing.T) {
	svc, store := newProtectedModeTestService(false, &dto.Settings{}, nil, nil)
	svc.Reconcile()

	require.Empty(t, store.upserts)
	require.Equal(t, []string{"protected_mode"}, store.dismissed)
}

func TestProtectedModeReconcile_SkipsIgnoredProblem(t *testing.T) {
	existing := &dto.Problem{ProblemKey: "protected_mode", Ignored: true}
	svc, store := newProtectedModeTestService(true, &dto.Settings{}, nil, existing)
	svc.Reconcile()

	require.Empty(t, store.upserts)
	require.Empty(t, store.dismissed)
}

func TestProtectedModeReconcile_DismissesWhenAlertDisabled(t *testing.T) {
	settings := &dto.Settings{AlertProtectedMode: new(false)}
	svc, store := newProtectedModeTestService(true, settings, nil, nil)
	svc.Reconcile()

	require.Empty(t, store.upserts)
	require.Equal(t, []string{"protected_mode"}, store.dismissed)
}

func TestProtectedModeReconcile_SkipsOnSettingsError(t *testing.T) {
	svc, store := newProtectedModeTestService(true, nil, tozderrors.New("boom"), nil)
	svc.Reconcile()

	require.Empty(t, store.upserts)
	require.Empty(t, store.dismissed)
}

func TestProtectedModeReconcile_NilProblemServiceNoPanic(t *testing.T) {
	svc := &ProtectedModeAlertService{
		ctx:   context.Background(),
		state: &dto.ContextState{ProtectedMode: true},
	}
	require.NotPanics(t, func() { svc.Reconcile() })
}

func TestAlertEnabledHelper(t *testing.T) {
	svc := &HomeAssistantComponentService{}
	require.True(t, svc.alertEnabled("protected_mode"))

	svc.settingService = &stubAlertSettings{settings: &dto.Settings{AlertProtectedMode: new(false)}}
	require.False(t, svc.alertEnabled("protected_mode"))
	require.True(t, svc.alertEnabled("addon_config_changed"))

	svc.settingService = &stubAlertSettings{settings: nil, err: tozderrors.New("boom")}
	require.True(t, svc.alertEnabled("protected_mode"))
}

func TestNewProtectedModeAlertService_StartsAndStops(t *testing.T) {
	app := fxtest.New(t,
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(t) },
			func() context.Context {
				return context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{})
			},
			func() *dto.ContextState { return &dto.ContextState{} },
			events.NewEventBus,
			mock.Mock[ProblemServiceInterface],
			mock.Mock[SettingServiceInterface],
			NewProtectedModeAlertService,
		),
		fx.Invoke(func(ProtectedModeAlertServiceInterface) {}),
	)
	app.RequireStart()
	app.RequireStop()
}
