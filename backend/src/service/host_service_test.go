package service_test

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/host"
	"github.com/dianlight/srat/service"
	"github.com/xorcare/pointer"

	// gocache "github.com/patrickmn/go-cache" // Not strictly needed for these tests
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

const (
	testHostname    = "test-ha-host"
	defaultHostname = "homeassistant"
	demoHostname    = "demo"
	// hostnameCacheKeyTest = "hostname" // Matches the constant in host_service.go; not directly used in assertions
)

type HostServiceTestSuite struct {
	suite.Suite
	hostService    service.HostServiceInterface
	mockHostClient host.ClientWithResponsesInterface // Mockio will provide this
	staticConfig   *dto.ContextState
	app            *fxtest.App
	ctx            context.Context
	cancel         context.CancelFunc
}

func TestHostServiceTestSuite(t *testing.T) {
	suite.Run(t, new(HostServiceTestSuite))
}

func (suite *HostServiceTestSuite) SetupTest() {
	suite.staticConfig = &dto.ContextState{
		HACoreReady: true,
	}

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				return context.WithValue(ctx, "wg", &sync.WaitGroup{}), cancel
			},
			func() *dto.ContextState {
				return suite.staticConfig
			},
			mock.Mock[host.ClientWithResponsesInterface],
			//	mock.Mock[repository.PropertyRepositoryInterface], // Provided for fx dependency resolution
			service.NewHostService,
		),
		fx.Populate(&suite.hostService),
		fx.Populate(&suite.mockHostClient),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *HostServiceTestSuite) TearDownTest() {
	//mock.TearDown(suite.T())
	if suite.cancel != nil {
		suite.cancel()
	}
	if suite.app != nil {
		// Wait for app to stop to ensure all background goroutines (if any) complete.
		// suite.ctx.Value("wg").(*sync.WaitGroup).Wait() // If services add to WG
		suite.app.RequireStop()
	}
}

func (suite *HostServiceTestSuite) TestGetHostName_DemoMode_CachesResult() {
	suite.staticConfig.SupervisorURL = "demo"

	// First call
	name, err := suite.hostService.GetHostName()
	suite.Require().NoError(err)
	suite.Equal(demoHostname, name)

	// Second call - should be cached
	name, err = suite.hostService.GetHostName()
	suite.Require().NoError(err)
	suite.Equal(demoHostname, name)

	// Verify host client was not called because GetHostInfoWithResponse is a method on the client.
	// If the client itself was never constructed or used, this check is implicit.
	// To be explicit, if GetHostInfoWithResponse is the specific method:
	mock.Verify(suite.mockHostClient, matchers.Times(0)).GetHostInfoWithResponse(mock.AnyContext())
}

func (suite *HostServiceTestSuite) TestGetHostName_APISuccess_CacheMissThenHit() {
	suite.staticConfig.SupervisorURL = "http://supervisor.local"
	expectedResponse := &host.GetHostInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &host.HostInfoResponse{
			Data: host.HostInfoData{
				Hostname: pointer.String(testHostname),
			},
		},
	}

	mock.When(suite.mockHostClient.GetHostInfoWithResponse(suite.ctx)).
		ThenReturn(expectedResponse, nil).
		Verify(matchers.Times(1))

	// First call (cache miss)
	name, err := suite.hostService.GetHostName()
	suite.Require().NoError(err)
	suite.Equal(testHostname, name)

	// Second call (cache hit)
	name, err = suite.hostService.GetHostName()
	suite.Require().NoError(err)
	suite.Equal(testHostname, name)

	// mock.Verify ensures GetHostInfoWithResponse was called exactly once due to the Times(1) constraint.
}

func (suite *HostServiceTestSuite) TestGetHostName_APIClientError() {
	suite.staticConfig.SupervisorURL = "http://supervisor.local"
	apiError := errors.New("network connection failed")

	// Expect API call for the first GetHostName
	mock.When(suite.mockHostClient.GetHostInfoWithResponse(suite.ctx)).
		ThenReturn(nil, apiError).
		Verify(matchers.Times(2))

	name, err := suite.hostService.GetHostName()
	suite.Require().Error(err)
	suite.ErrorContains(err, "network connection failed")
	suite.Equal(defaultHostname, name) // Default hostname on error

	name, err = suite.hostService.GetHostName()
	suite.Require().Error(err)
	suite.ErrorContains(err, "network connection failed")
	suite.Equal(defaultHostname, name)
}

func (suite *HostServiceTestSuite) TestGetHostName_APINon200Status() {
	suite.staticConfig.SupervisorURL = "http://supervisor.local"
	errorResponse := &host.GetHostInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusServiceUnavailable, Status: "Service Unavailable"},
		Body:         []byte("supervisor is temporarily down"),
	}

	mock.When(suite.mockHostClient.GetHostInfoWithResponse(suite.ctx)).
		ThenReturn(errorResponse, nil).
		Verify(matchers.Times(1))

	name, err := suite.hostService.GetHostName()
	suite.Require().Error(err)
	suite.ErrorContains(err, "Error getting info from ha_Host: 503 supervisor is temporarily down")
	suite.Equal(defaultHostname, name)
}

func (suite *HostServiceTestSuite) TestGetHostName_APINilResponseData_Hostname() {
	suite.staticConfig.SupervisorURL = "http://supervisor.local"
	nilDataResponse := &host.GetHostInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &host.HostInfoResponse{
			Data: host.HostInfoData{
				Hostname: nil,
			},
		},
	}
	mock.When(suite.mockHostClient.GetHostInfoWithResponse(suite.ctx)).
		ThenReturn(nilDataResponse, nil).
		Verify(matchers.Times(1))

	name, err := suite.hostService.GetHostName()
	suite.Require().Error(err)
	suite.ErrorContains(err, "Error getting info from ha_Host: response data is nil")
	suite.Equal(defaultHostname, name)
}

func (suite *HostServiceTestSuite) TestGetHostName_APINilResponseData_JSON200() {
	suite.staticConfig.SupervisorURL = "http://supervisor.local"
	nilDataResponse := &host.GetHostInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200:      nil, // JSON200 field is nil
	}
	mock.When(suite.mockHostClient.GetHostInfoWithResponse(suite.ctx)).
		ThenReturn(nilDataResponse, nil).
		Verify(matchers.Times(1))

	name, err := suite.hostService.GetHostName()
	suite.Require().Error(err)
	suite.ErrorContains(err, "Error getting info from ha_Host: response data is nil")
	suite.Equal(defaultHostname, name)
}
