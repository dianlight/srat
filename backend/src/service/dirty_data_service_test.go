package service

import (
	"context"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type DirtyDataServiceTestSuite struct {
	suite.Suite
	dirtyDataService DirtyDataServiceInterface
	ctx              context.Context
}

func TestDirtyDataServiceTestSuite(t *testing.T) {
	suite.Run(t, new(DirtyDataServiceTestSuite))
}

func (suite *DirtyDataServiceTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.dirtyDataService = NewDirtyDataService(suite.ctx)
}

func (suite *DirtyDataServiceTestSuite) TestNewDirtyDataService() {
	assert.NotNil(suite.T(), suite.dirtyDataService)
	assert.Equal(suite.T(), dto.DataDirtyTracker{}, suite.dirtyDataService.GetDirtyDataTracker())
}

func (suite *DirtyDataServiceTestSuite) TestSetDirtyShares() {
	suite.dirtyDataService.SetDirtyShares()
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	assert.True(suite.T(), tracker.Shares)
	assert.False(suite.T(), tracker.Volumes)
	assert.False(suite.T(), tracker.Users)
	assert.False(suite.T(), tracker.Settings)
	assert.True(suite.T(), suite.dirtyDataService.IsTimerRunning())
}

func (suite *DirtyDataServiceTestSuite) TestSetDirtyVolumes() {
	suite.dirtyDataService.SetDirtyVolumes()
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	assert.False(suite.T(), tracker.Shares)
	assert.True(suite.T(), tracker.Volumes)
	assert.False(suite.T(), tracker.Users)
	assert.False(suite.T(), tracker.Settings)
	assert.True(suite.T(), suite.dirtyDataService.IsTimerRunning())

}

func (suite *DirtyDataServiceTestSuite) TestSetDirtyUsers() {
	suite.dirtyDataService.SetDirtyUsers()
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	assert.False(suite.T(), tracker.Shares)
	assert.False(suite.T(), tracker.Volumes)
	assert.True(suite.T(), tracker.Users)
	assert.False(suite.T(), tracker.Settings)
	assert.True(suite.T(), suite.dirtyDataService.IsTimerRunning())

}

func (suite *DirtyDataServiceTestSuite) TestSetDirtySettings() {
	suite.dirtyDataService.SetDirtySettings()
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	assert.False(suite.T(), tracker.Shares)
	assert.False(suite.T(), tracker.Volumes)
	assert.False(suite.T(), tracker.Users)
	assert.True(suite.T(), tracker.Settings)
	assert.True(suite.T(), suite.dirtyDataService.IsTimerRunning())

}

func (suite *DirtyDataServiceTestSuite) TestResetDirtyStatus() {
	suite.dirtyDataService.SetDirtyShares()
	suite.dirtyDataService.SetDirtyVolumes()
	suite.dirtyDataService.SetDirtyUsers()
	suite.dirtyDataService.SetDirtySettings()

	suite.dirtyDataService.ResetDirtyStatus()
	tracker := suite.dirtyDataService.GetDirtyDataTracker()
	assert.False(suite.T(), tracker.Shares)
	assert.False(suite.T(), tracker.Volumes)
	assert.False(suite.T(), tracker.Users)
	assert.False(suite.T(), tracker.Settings)
	assert.False(suite.T(), suite.dirtyDataService.IsTimerRunning())

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
	assert.True(suite.T(), callbackCalled)
	assert.False(suite.T(), suite.dirtyDataService.IsTimerRunning())

}

func (suite *DirtyDataServiceTestSuite) TestTimerResetOnMultipleSetDirty() {
	suite.dirtyDataService.SetDirtyShares()
	time.Sleep(1 * time.Second)
	suite.dirtyDataService.SetDirtyVolumes()

	assert.True(suite.T(), suite.dirtyDataService.IsTimerRunning())
	tracker := suite.dirtyDataService.GetDirtyDataTracker()

	assert.True(suite.T(), tracker.Shares)
	assert.True(suite.T(), tracker.Volumes)

	time.Sleep(6 * time.Second)
	tracker = suite.dirtyDataService.GetDirtyDataTracker()
	assert.False(suite.T(), tracker.Shares)
	assert.False(suite.T(), tracker.Volumes)
	assert.False(suite.T(), suite.dirtyDataService.IsTimerRunning())

}
