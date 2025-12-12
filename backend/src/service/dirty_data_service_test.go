package service

import (
	"context"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
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

func (suite *DirtyDataServiceTestSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
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
	suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
	suite.app.RequireStop()
}

func (suite *DirtyDataServiceTestSuite) TestNewDirtyDataService() {
	suite.NotNil(suite.dirtyDataService)
	suite.Equal(dto.DataDirtyTracker{
		Shares:   true,
		Users:    true,
		Settings: true,
	}, suite.dirtyDataService.GetDirtyDataTracker())
}

func (suite *DirtyDataServiceTestSuite) TestSetDirtyShares() {
	suite.dirtyDataService.ResetDirtyDataTracker()
	suite.eventBus.EmitShare(events.ShareEvent{Share: &dto.SharedResource{Name: "testshare"}})
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
