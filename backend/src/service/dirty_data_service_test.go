package service

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type DirtyDataServiceTestSuite struct {
	suite.Suite
	dirtyDataService DirtyDataServiceInterface
	eventBus         events.EventBusInterface
	ctx              context.Context
	cancel           context.CancelFunc
	app              *fxtest.App
}

func TestDirtyDataServiceTestSuite(t *testing.T) {
	suite.Run(t, new(DirtyDataServiceTestSuite))
}

func TestAppConfigOptionKeys(t *testing.T) {
	tests := []struct {
		name     string
		config   *dto.AppConfigUpdateRequest
		expected []string
	}{
		{"nil config", nil, nil},
		{"nil options", &dto.AppConfigUpdateRequest{}, nil},
		{"sorted keys", &dto.AppConfigUpdateRequest{Options: map[string]any{"workgroup": "w", "password": "p", "log_level": "d"}}, []string{"log_level", "password", "workgroup"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, appConfigOptionKeys(tt.config))
		})
	}
}

func (suite *DirtyDataServiceTestSuite) TestIsCleanInitialState() {
	// After reset, all flags are false and timer is nil, so IsClean should be true
	suite.dirtyDataService.ResetDirtyDataTracker()
	suite.True(suite.dirtyDataService.IsClean())
}

func (suite *DirtyDataServiceTestSuite) TestIsCleanWithDirtyShares() {
	suite.dirtyDataService.ResetDirtyDataTracker()
	_ = suite.eventBus.EmitShare(events.ShareEvent{Share: &dto.SharedResource{Name: "testshare"}})
	suite.False(suite.dirtyDataService.IsClean())
}

func (suite *DirtyDataServiceTestSuite) TestIsCleanWithDirtyUsers() {
	suite.dirtyDataService.ResetDirtyDataTracker()
	suite.eventBus.EmitUser(events.UserEvent{User: &dto.User{}})
	suite.False(suite.dirtyDataService.IsClean())
}

func (suite *DirtyDataServiceTestSuite) TestIsCleanWithDirtySettings() {
	suite.dirtyDataService.ResetDirtyDataTracker()
	suite.eventBus.EmitSetting(events.SettingEvent{Setting: &dto.Settings{}})
	suite.False(suite.dirtyDataService.IsClean())
}

func (suite *DirtyDataServiceTestSuite) TestIsCleanWithMultipleDirtyFlags() {
	suite.dirtyDataService.ResetDirtyDataTracker()
	_ = suite.eventBus.EmitShare(events.ShareEvent{Share: &dto.SharedResource{Name: "testshare"}})
	suite.eventBus.EmitUser(events.UserEvent{User: &dto.User{}})
	suite.False(suite.dirtyDataService.IsClean())
}

func (suite *DirtyDataServiceTestSuite) TestIsCleanAfterReset() {
	suite.dirtyDataService.ResetDirtyDataTracker()
	_ = suite.eventBus.EmitShare(events.ShareEvent{Share: &dto.SharedResource{Name: "testshare"}})
	suite.False(suite.dirtyDataService.IsClean())
	suite.dirtyDataService.ResetDirtyDataTracker()
	suite.True(suite.dirtyDataService.IsClean())
}

func (suite *DirtyDataServiceTestSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{}))
			},
			NewDirtyDataService,
			events.NewEventBus,
		),
		fx.Populate(&suite.dirtyDataService),
		fx.Populate(&suite.eventBus),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *DirtyDataServiceTestSuite) TearDownTest() {
	suite.cancel()
	suite.ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup).Wait()
	suite.app.RequireStop()
}

func (suite *DirtyDataServiceTestSuite) TestNewDirtyDataService() {
	suite.NotNil(suite.dirtyDataService)
	suite.Equal(dto.DataDirtyTracker{
		Shares:    true,
		Users:     true,
		Settings:  true,
		AppConfig: false,
	}, suite.dirtyDataService.GetDirtyDataTracker())
}

func (suite *DirtyDataServiceTestSuite) TestSetDirtyShares() {
	suite.dirtyDataService.ResetDirtyDataTracker()
	_ = suite.eventBus.EmitShare(events.ShareEvent{Share: &dto.SharedResource{Name: "testshare"}})
	//time.Sleep(500 * time.Millisecond)
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	suite.True(tracker.Shares)
	suite.False(tracker.Users)
	suite.False(tracker.Settings)
	suite.True(suite.dirtyDataService.IsTimerRunning())
}

func (suite *DirtyDataServiceTestSuite) TestSetDirtyUsers() {
	suite.dirtyDataService.ResetDirtyDataTracker()
	suite.eventBus.EmitUser(events.UserEvent{User: &dto.User{}})
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	suite.False(tracker.Shares)
	suite.True(tracker.Users)
	suite.False(tracker.Settings)
	suite.True(suite.dirtyDataService.IsTimerRunning())

}

func (suite *DirtyDataServiceTestSuite) TestSetDirtySettings() {
	suite.dirtyDataService.ResetDirtyDataTracker()
	suite.eventBus.EmitSetting(events.SettingEvent{Setting: &dto.Settings{}})
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	suite.False(tracker.Shares)
	suite.False(tracker.Users)
	suite.True(tracker.Settings)
	suite.True(suite.dirtyDataService.IsTimerRunning())

}

func (suite *DirtyDataServiceTestSuite) TestSetDirtyAppConfig() {
	suite.dirtyDataService.ResetDirtyDataTracker()
	suite.eventBus.EmitAppConfig(events.AppConfigEvent{Config: &dto.AppConfigUpdateRequest{Options: map[string]any{"log_level": "debug"}}})

	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	suite.False(tracker.Shares)
	suite.False(tracker.Users)
	suite.False(tracker.Settings)
	suite.True(tracker.AppConfig)
	suite.False(suite.dirtyDataService.IsTimerRunning())
	suite.False(suite.dirtyDataService.IsClean())
}

// TestSetDirtyAppConfigDoesNotLogOptionValues verifies the task 031 audit
// fix: AppConfig options arrive as an arbitrary user-controlled map that may
// contain credentials (e.g. the addon password), so only the changed option
// names may reach the logs — never the values. The bus dispatches
// synchronously, so no polling is needed after EmitAppConfig.
func (suite *DirtyDataServiceTestSuite) TestSetDirtyAppConfigDoesNotLogOptionValues() {
	var buf bytes.Buffer
	oldDefault := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	suite.T().Cleanup(func() { slog.SetDefault(oldDefault) })

	const secretValue = "s3cr3t-appconfig-031"
	suite.eventBus.EmitAppConfig(events.AppConfigEvent{Config: &dto.AppConfigUpdateRequest{
		Options: map[string]any{"password": secretValue, "log_level": "debug"},
	}})

	logs := buf.String()
	suite.NotContains(logs, secretValue, "addon option values must not reach logs")
	suite.Contains(logs, "password", "changed option names should still be logged for debuggability")
}

func (suite *DirtyDataServiceTestSuite) TestResetDirtyStatus_EmitsCleanTracker() {
	suite.dirtyDataService.ResetDirtyDataTracker()
	suite.eventBus.EmitUser(events.UserEvent{User: &dto.User{}})

	var captured []events.DirtyDataEvent
	unsub := suite.eventBus.OnDirtyData(func(_ context.Context, e events.DirtyDataEvent) errors.E {
		captured = append(captured, e)
		return nil
	})
	defer unsub()

	suite.dirtyDataService.ResetDirtyDataTracker()

	suite.NotEmpty(captured)
	last := captured[len(captured)-1]
	suite.Equal(events.EventTypes.CLEAN, last.Type)
	suite.Equal(dto.DataDirtyTracker{}, last.DataDirtyTracker)
	suite.True(suite.dirtyDataService.IsClean())
	suite.False(suite.dirtyDataService.IsTimerRunning())
}

func (suite *DirtyDataServiceTestSuite) TestResetDirtyStatus_NilEventBusDoesNotPanic() {
	svc := &DirtyDataService{
		ctx:              suite.ctx,
		dataDirtyTracker: dto.DataDirtyTracker{Users: true},
		eventBus:         nil,
	}
	suite.NotPanics(func() {
		svc.ResetDirtyDataTracker()
	})
	suite.Equal(dto.DataDirtyTracker{}, svc.dataDirtyTracker)
	suite.True(svc.IsClean())
}
