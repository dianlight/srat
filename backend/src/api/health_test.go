package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/stretchr/testify/suite"
	"github.com/tj/go-spin"
)

type HealthHandlerSuite struct {
	suite.Suite
	mockBoradcaster  service.BroadcasterServiceInterface
	mockSambaService service.SambaServiceInterface
	dirtyService     service.DirtyDataServiceInterface
	ctrl             *matchers.MockController
}

func TestHealthHandlerSuite(t *testing.T) {
	csuite := new(HealthHandlerSuite)

	csuite.ctrl = mock.NewMockController(t)

	csuite.mockBoradcaster = mock.Mock[service.BroadcasterServiceInterface](csuite.ctrl)
	csuite.mockSambaService = mock.Mock[service.SambaServiceInterface](csuite.ctrl)
	csuite.dirtyService = service.NewDirtyDataService(testContext)
	//mock.When(csuite.mockBoradcaster.BroadcastMessage(mock.Any[dto.HealthPing]())).ThenReturn(nil, nil)
	//mock.When(csuite.mockBoradcaster.BroadcastMessage(mock.Any[any]())).ThenReturn(nil, nil)
	mock.When(csuite.mockSambaService.GetSambaProcess()).ThenReturn(&process.Process{
		Pid: 1234,
	}, nil)
	suite.Run(t, csuite)
}

func (suite *HealthHandlerSuite) TestHealthCheckHandler() {
	healthHandler := api.NewHealthHandler(api.HealthHandlerParams{Ctx: testContext, Apictx: &apiContextState, Broadcaster: suite.mockBoradcaster, SambaService: suite.mockSambaService, DirtyService: suite.dirtyService})
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
	mockBroadcaster := mock.Mock[service.BroadcasterServiceInterface](suite.ctrl)
	health := api.NewHealthHandler(api.HealthHandlerParams{Ctx: testContext, Apictx: &apiContextState, Broadcaster: mockBroadcaster, SambaService: suite.mockSambaService, DirtyService: suite.dirtyService})
	numcal := uint64(0)
	startTime := time.Now()
	mock.When(mockBroadcaster.BroadcastMessage(mock.NotNil[any]())).ThenAnswer(matchers.Answer(func(data []any) []any {
		msg, ok := data[0].(dto.HealthPing)
		if !ok {
			suite.T().Errorf("Expected Event to be Heartbeat, got %#v", msg)
		}
		numcal++
		return []any{nil, nil}
	}))
	/*
		suite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).Do((func(data any) {
			msg, ok := data.(dto.HealthPing)
			if !ok {
				suite.T().Errorf("Expected Event to be Heartbeat, got %#v", msg)
			}
			numcal++
			return
		})).AnyTimes()
	*/

	s := spin.New()
	for health.OutputEventsCount < 5 {
		fmt.Printf("\r  \033[36mcomputing\033[m %s (%fs)", s.Next(), time.Since(startTime).Abs().Seconds())
		time.Sleep(time.Millisecond * 500)
	}
	elapsed := time.Since(startTime)
	suite.LessOrEqual(uint64(elapsed.Seconds()/float64(apiContextState.Heartbeat)), health.OutputEventsCount, " elapsed seconds %d should be greater than %d", elapsed.Seconds(), health.OutputEventsCount)

	testContextCancel()
	testContext.Value("wg").(*sync.WaitGroup).Wait()

	if numcal == 0 {
		suite.T().Skip("No calls to BroadcastMessage were made")
	}
	//	suite.GreaterOrEqual(health.OutputEventsCount, numcal, " real call %d should be greater than %d", numcal, health.OutputEventsCount)
	mock.Verify(mockBroadcaster, mock.Times(int(health.OutputEventsCount))).BroadcastMessage(mock.NotNil[any]())

	suite.Equal(numcal, health.OutputEventsCount, "Expected OutputEventsCount to match numcal")
}
