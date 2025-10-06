package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SambaHandlerSuite struct {
	suite.Suite
	app              *fxtest.App
	handler          *api.SambaHanler
	mockSambaService service.SambaServiceInterface
	mockApiContext   *dto.ContextState
	ctx              context.Context
	cancel           context.CancelFunc
}

func (suite *SambaHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			func() *dto.ContextState {
				return &dto.ContextState{
					ReadOnlyMode:    false,
					Heartbeat:       1,
					DockerInterface: "hassio",
					DockerNet:       "172.30.32.0/23",
				}
			},
			api.NewSambaHanler,
			mock.Mock[service.SambaServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockSambaService),
		fx.Populate(&suite.mockApiContext),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *SambaHandlerSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
		suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
	}
	suite.app.RequireStop()
}

func (suite *SambaHandlerSuite) TestGetSambaStatusSuccess() {
	expectedStatus := &dto.SambaStatus{
		Sessions: map[string]dto.SambaSession{
			"session1": {Username: "testuser", Hostname: "testhost"},
		},
	}

	// Configure mock expectations
	mock.When(suite.mockSambaService.GetSambaStatus()).ThenReturn(expectedStatus, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterSambaHandler(api)

	// Make HTTP request
	resp := api.Get("/samba/status")
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Parse response
	var result dto.SambaStatus
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Len(result.Sessions, 1)
	session, ok := result.Sessions["session1"]
	suite.True(ok)
	suite.Equal("testuser", session.Username)
}

func (suite *SambaHandlerSuite) TestGetSambaStatusError() {
	expectedErr := errors.New("failed to get samba status")

	// Configure mock expectations
	mock.When(suite.mockSambaService.GetSambaStatus()).ThenReturn(nil, expectedErr)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterSambaHandler(api)

	// Make HTTP request
	resp := api.Get("/samba/status")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)
}

func (suite *SambaHandlerSuite) TestApplySambaSuccess() {
	// Configure mock expectations - ApplySamba calls multiple service methods
	mock.When(suite.mockSambaService.WriteSambaConfig()).ThenReturn(nil)
	mock.When(suite.mockSambaService.TestSambaConfig()).ThenReturn(nil)
	mock.When(suite.mockSambaService.RestartSambaService()).ThenReturn(nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterSambaHandler(api)

	// Make HTTP request
	resp := api.Put("/samba/apply", struct{}{})
	suite.Require().Equal(http.StatusNoContent, resp.Code)
}

func (suite *SambaHandlerSuite) TestApplySambaError() {
	expectedErr := errors.New("failed to write samba configuration")

	// Configure mock expectations - fails on first step
	mock.When(suite.mockSambaService.WriteSambaConfig()).ThenReturn(expectedErr)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterSambaHandler(api)

	// Make HTTP request
	resp := api.Put("/samba/apply", struct{}{})
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)
}

func (suite *SambaHandlerSuite) TestGetSambaConfigSuccess() {
	configData := []byte("[global]\nworkgroup = WORKGROUP\nsecurity = user\n")

	// Configure mock expectations
	mock.When(suite.mockSambaService.CreateConfigStream()).ThenReturn(&configData, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterSambaHandler(api)

	// Make HTTP request
	resp := api.Get("/samba/config")
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Parse response
	var result dto.SmbConf
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.NotEmpty(result.Data)
	suite.Contains(result.Data, "WORKGROUP")
	suite.Contains(result.Data, "security")
}

func (suite *SambaHandlerSuite) TestGetSambaConfigError() {
	expectedErr := errors.New("failed to read samba config")

	// Configure mock expectations
	mock.When(suite.mockSambaService.CreateConfigStream()).ThenReturn(nil, expectedErr)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterSambaHandler(api)

	// Make HTTP request
	resp := api.Get("/samba/config")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)
}

func TestSambaHandlerSuite(t *testing.T) {
	suite.Run(t, new(SambaHandlerSuite))
}
