package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	//csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()

	suite.Run(t, csuite)
}

func (suite *HealthHandlerSuite) TestHealthCheckHandler() {

	suite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()
	req, err := http.NewRequestWithContext(testContext, "GET", "/health", nil)
	if err != nil {
		suite.T().Fatal(err)
	}

	rr := httptest.NewRecorder()
	health := api.NewHealthHandler(testContext, &apiContextState, suite.mockBoradcaster, suite.mockSambaService, suite.dirtyService)
	handler := http.HandlerFunc(health.HealthCheckHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		suite.T().Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		suite.T().Errorf("handler returned wrong content type: got %v want %v",
			contentType, expectedContentType)
	}

	var response dto.HealthPing
	err = json.Unmarshal(rr.Body.Bytes(), &response)
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
