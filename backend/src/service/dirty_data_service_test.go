package service_test

import (
	"context"
	"testing"

	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/suite"
)

type DirtyDataServiceSuite struct {
	suite.Suite
	dirtyDataService *service.DirtyDataService
}

func TestDirtyDataServiceSuite(t *testing.T) {
	suite.Run(t, new(DirtyDataServiceSuite))
}

func (suite *DirtyDataServiceSuite) SetupTest() {
	suite.dirtyDataService = service.NewDirtyDataService(context.Background()).(*service.DirtyDataService)
}

func (suite *DirtyDataServiceSuite) TestSetDirtyShares() {
	// Arrange
	suite.dirtyDataService.ResetDirtyStatus()
	// Act
	suite.dirtyDataService.SetDirtyShares()

	// Assert
	suite.True(suite.dirtyDataService.GetDirtyDataTracker().Shares, "Shares property should be set to true")

	// Verify that the timer was started
	suite.True(suite.dirtyDataService.IsTimerRunning(), "Timer should be started")
}
