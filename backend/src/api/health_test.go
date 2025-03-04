package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/go-fuego/fuego"
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

	//csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()

	suite.Run(t, csuite)
}

func (suite *HealthHandlerSuite) TestHealthCheckHandler() {

	ctx := fuego.NewMockContextNoBody()

	suite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()
	health := api.NewHealthHandler(testContext, &apiContextState, suite.mockBoradcaster, suite.mockSambaService, suite.dirtyService)

	response, err := health.CheckHealthStatus(ctx)
	suite.Require().NoError(err)
	suite.T().Logf("%v", response)

	if !response.Alive {
		suite.T().Errorf("Expected Alive to be true, got false")
	}
}

func (suite *HealthHandlerSuite) TestHealthEventEmitter() {
	health := api.NewHealthHandler(testContext, &apiContextState, suite.mockBoradcaster, suite.mockSambaService, suite.dirtyService)
	numcal := uint64(0)
	startTime := time.Now()
	suite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).Do((func(data any) {
		msg := data.(*dto.EventMessageEnvelope)
		suite.NotNil(msg.Data)
		if msg.Event != dto.EventHeartbeat {
			suite.T().Errorf("Expected Event to be Heartbeat, got %v", msg.Event)
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

	//t.Log(health)
}
