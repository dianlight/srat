package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/suite"
	"github.com/tj/go-spin"
	gomock "go.uber.org/mock/gomock"
)

type HealthHandlerSuite struct {
	suite.Suite
	mockBoradcaster  *MockBroadcasterServiceInterface
	mockSambaService *MockSambaServiceInterface
	dirtyService     service.DirtyDataServiceInterface
}

func TestHealthHandlerSuite(t *testing.T) {
	csuite := new(HealthHandlerSuite)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	csuite.mockBoradcaster = NewMockBroadcasterServiceInterface(ctrl)
	csuite.mockSambaService = NewMockSambaServiceInterface(ctrl)
	csuite.dirtyService = service.NewDirtyDataService(testContext)
	csuite.mockSambaService.EXPECT().GetSambaProcess().AnyTimes()
	csuite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()

	//csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()

	suite.Run(t, csuite)
}

func (suite *HealthHandlerSuite) TestHealthCheckHandler() {
	healthHandler := api.NewHealthHandler(testContext, &apiContextState, suite.mockBoradcaster, suite.mockSambaService, suite.dirtyService)
	_, api := humatest.New(suite.T())
	healthHandler.RegisterVolumeHandlers(api)

	rr := api.Get("/health")
	suite.Equal(http.StatusOK, rr.Code)

	var response dto.HealthPing
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		suite.T().Errorf("Failed to unmarshal response body: %v", err)
	}

	if !response.Alive {
		suite.T().Errorf("Expected Alive to be true, got false")
	}
}

func (suite *HealthHandlerSuite) TestHealthEventEmitter() {
	health := api.NewHealthHandler(testContext, &apiContextState, suite.mockBoradcaster, suite.mockSambaService, suite.dirtyService)
	numcal := uint64(0)
	startTime := time.Now()
	suite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).Do((func(data any) {
		msg, ok := data.(dto.HealthPing)
		if !ok {
			suite.T().Errorf("Expected Event to be Heartbeat, got %#v", msg)
		}
		numcal++
		return
	})).AnyTimes()
	s := spin.New()
	for health.OutputEventsCount < 3 {
		fmt.Printf("\r  \033[36mcomputing\033[m %s (%fs)", s.Next(), time.Since(startTime).Abs().Seconds())
		time.Sleep(time.Millisecond * 500)
	}
	elapsed := time.Since(startTime)
	suite.LessOrEqual(uint64(elapsed.Seconds()/float64(apiContextState.Heartbeat)), health.OutputEventsCount, " elapsed seconds %d should be greater than %d", elapsed.Seconds(), numcal)
	if numcal != 0 {
		suite.Equal(health.OutputEventsCount, numcal)
	}
}
