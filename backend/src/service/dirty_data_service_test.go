package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type DirtyDataServiceTestSuite struct {
	suite.Suite
	dirtyDataService DirtyDataServiceInterface
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
		),
		fx.Populate(&suite.dirtyDataService),
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
	suite.Equal(dto.DataDirtyTracker{}, suite.dirtyDataService.GetDirtyDataTracker())
}

func (suite *DirtyDataServiceTestSuite) TestSetDirtyShares() {
	suite.dirtyDataService.SetDirtyShares()
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	suite.True(tracker.Shares)
	suite.False(tracker.Volumes)
	suite.False(tracker.Users)
	suite.False(tracker.Settings)
	suite.True(suite.dirtyDataService.IsTimerRunning())
}

func (suite *DirtyDataServiceTestSuite) TestSetDirtyVolumes() {
	suite.dirtyDataService.SetDirtyVolumes()
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	suite.False(tracker.Shares)
	suite.True(tracker.Volumes)
	suite.False(tracker.Users)
	suite.False(tracker.Settings)
	suite.True(suite.dirtyDataService.IsTimerRunning())

}

func (suite *DirtyDataServiceTestSuite) TestSetDirtyUsers() {
	suite.dirtyDataService.SetDirtyUsers()
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	suite.False(tracker.Shares)
	suite.False(tracker.Volumes)
	suite.True(tracker.Users)
	suite.False(tracker.Settings)
	suite.True(suite.dirtyDataService.IsTimerRunning())

}

func (suite *DirtyDataServiceTestSuite) TestSetDirtySettings() {
	suite.dirtyDataService.SetDirtySettings()
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	suite.False(tracker.Shares)
	suite.False(tracker.Volumes)
	suite.False(tracker.Users)
	suite.True(tracker.Settings)
	suite.True(suite.dirtyDataService.IsTimerRunning())

}

func (suite *DirtyDataServiceTestSuite) TestResetDirtyStatus() {
	suite.dirtyDataService.SetDirtyShares()
	suite.dirtyDataService.SetDirtyVolumes()
	suite.dirtyDataService.SetDirtyUsers()
	suite.dirtyDataService.SetDirtySettings()

	suite.dirtyDataService.ResetDirtyStatus()
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	suite.False(tracker.Shares)
	suite.False(tracker.Volumes)
	suite.False(tracker.Users)
	suite.False(tracker.Settings)
	suite.False(suite.dirtyDataService.IsTimerRunning())

}

func (suite *DirtyDataServiceTestSuite) TestAddRestartCallback() {
	var callbackCalled bool
	callback := func() error {
		callbackCalled = true
		return nil
	}
	suite.dirtyDataService.AddRestartCallback(callback)
	suite.dirtyDataService.SetDirtySettings()
	time.Sleep(6 * time.Second)
	suite.True(callbackCalled)
	suite.False(suite.dirtyDataService.IsTimerRunning())

}

func (suite *DirtyDataServiceTestSuite) TestTimerResetOnMultipleSetDirty() {
	suite.dirtyDataService.SetDirtyShares()
	time.Sleep(1 * time.Second)
	suite.dirtyDataService.SetDirtyVolumes()

	suite.True(suite.dirtyDataService.IsTimerRunning())
	tracker := suite.dirtyDataService.GetDirtyDataTracker()

	suite.True(tracker.Shares)
	suite.True(tracker.Volumes)

	time.Sleep(6 * time.Second)
	tracker = suite.dirtyDataService.GetDirtyDataTracker()
	suite.False(tracker.Shares)
	suite.False(tracker.Volumes)
	suite.False(suite.dirtyDataService.IsTimerRunning())

}
