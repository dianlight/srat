package service_test

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/addons"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type AddonsServiceTestSuite struct {
	suite.Suite
	addonsService    service.AddonsServiceInterface
	mockAddonsClient addons.ClientWithResponsesInterface
	app              *fxtest.App
	ctx              context.Context
	cancel           context.CancelFunc
	wg               *sync.WaitGroup
}

func TestAddonsServiceTestSuite(t *testing.T) {
	suite.Run(t, new(AddonsServiceTestSuite))
}

func (suite *AddonsServiceTestSuite) SetupTest() {
	suite.wg = &sync.WaitGroup{}
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), "wg", suite.wg)
				return context.WithCancel(ctx)
			},
			service.NewAddonsService,
			func() *dto.ContextState {
				return &dto.ContextState{}
			},
			mock.Mock[addons.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.ctx, &suite.cancel),
		fx.Populate(&suite.mockAddonsClient),
		fx.Populate(&suite.addonsService),
	)
	suite.app.RequireStart()
}

func (suite *AddonsServiceTestSuite) TearDownTest() {
	suite.cancel()
	suite.wg.Wait()
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

// --- GetStats Tests ---

func (suite *AddonsServiceTestSuite) TestGetStats_Success() {
	cpu := 55.5
	mem := int(1024 * 1024 * 100) // 100MB
	expectedStats := addons.AddonStatsData{
		CpuPercent:  &cpu,
		MemoryUsage: &mem,
	}
	mockResponse := &addons.GetSelfAddonStatsResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &addons.AddonStatsResponse{
			Data: expectedStats,
		},
	}

	mock.When(suite.mockAddonsClient.GetSelfAddonStatsWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	stats, err := suite.addonsService.GetStats()
	suite.Require().NoError(err)
	suite.Require().NotNil(stats)
	suite.Equal(expectedStats, *stats)
	suite.Equal(cpu, *stats.CpuPercent)
	suite.Equal(mem, *stats.MemoryUsage)
}

func (suite *AddonsServiceTestSuite) TestGetStats_CacheHit() {
	cpu := 55.5
	mem := int(1024 * 1024 * 100) // 100MB
	expectedStats := addons.AddonStatsData{
		CpuPercent:  &cpu,
		MemoryUsage: &mem,
	}
	mockResponse := &addons.GetSelfAddonStatsResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &addons.AddonStatsResponse{
			Data: expectedStats,
		},
	}

	// Expect the client to be called only once.
	mock.When(suite.mockAddonsClient.GetSelfAddonStatsWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	// First call (cache miss)
	stats1, err1 := suite.addonsService.GetStats()
	suite.Require().NoError(err1)
	suite.Require().NotNil(stats1)
	suite.Equal(expectedStats, *stats1)

	// Second call (should be a cache hit)
	stats2, err2 := suite.addonsService.GetStats()
	suite.Require().NoError(err2)
	suite.Require().NotNil(stats2)
	suite.Equal(expectedStats, *stats2)
}

func (suite *AddonsServiceTestSuite) TestGetStats_ClientError() {
	apiError := errors.New("network failure")
	mock.When(suite.mockAddonsClient.GetSelfAddonStatsWithResponse(suite.ctx)).
		ThenReturn(nil, apiError).
		Verify(matchers.Times(1))

	stats, err := suite.addonsService.GetStats()
	suite.Nil(stats)
	suite.Require().Error(err)
	suite.ErrorContains(err, "failed to get addon stats")
	suite.ErrorIs(err, apiError)
}

func (suite *AddonsServiceTestSuite) TestGetStats_Non200Status() {
	mockResponse := &addons.GetSelfAddonStatsResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusNotFound, Status: "Not Found"},
		Body:         []byte("addon not found"),
	}
	mock.When(suite.mockAddonsClient.GetSelfAddonStatsWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	stats, err := suite.addonsService.GetStats()
	suite.Nil(stats)
	suite.Require().Error(err)
	suite.ErrorContains(err, "failed to get addon stats: status 404, body: addon not found")
}

func (suite *AddonsServiceTestSuite) TestGetStats_NilJSONResponse() {
	mockResponse := &addons.GetSelfAddonStatsResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200:      nil,
	}
	mock.When(suite.mockAddonsClient.GetSelfAddonStatsWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	stats, err := suite.addonsService.GetStats()
	suite.Nil(stats)
	suite.Require().Error(err)
	suite.ErrorContains(err, "addon stats not available or data incomplete")
}

func (suite *AddonsServiceTestSuite) TestGetStats_ClientNotInitialized() {
	var addonsService service.AddonsServiceInterface
	app := fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), "wg", &sync.WaitGroup{})
				return context.WithCancel(ctx)
			},
			service.NewAddonsService,
			func() *dto.ContextState { return &dto.ContextState{} },
			// Provide a nil client explicitly
			func() addons.ClientWithResponsesInterface { return nil },
		),
		fx.Populate(&addonsService),
	)
	app.RequireStart()
	defer app.RequireStop()

	stats, err := addonsService.GetStats()
	suite.Nil(stats)
	suite.Require().Error(err)
	suite.ErrorContains(err, "addons client is not initialized")
}

// --- CheckProtectedMode Tests ---

func (suite *AddonsServiceTestSuite) TestCheckProtectedMode_Success_True() {
	mockResponse := &addons.GetSelfAddonInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &addons.AddonInfoResponse{
			Data: addons.AddonInfoData{
				Protected: pointer.Bool(true),
			},
		},
	}

	mock.When(suite.mockAddonsClient.GetSelfAddonInfoWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	isProtected, err := suite.addonsService.CheckProtectedMode()
	suite.Require().NoError(err)
	suite.True(isProtected)
}

func (suite *AddonsServiceTestSuite) TestCheckProtectedMode_Success_False() {
	mockResponse := &addons.GetSelfAddonInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &addons.AddonInfoResponse{
			Data: addons.AddonInfoData{
				Protected: pointer.Bool(false),
			},
		},
	}

	mock.When(suite.mockAddonsClient.GetSelfAddonInfoWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	isProtected, err := suite.addonsService.CheckProtectedMode()
	suite.Require().NoError(err)
	suite.False(isProtected)
}

func (suite *AddonsServiceTestSuite) TestCheckProtectedMode_CacheHit() {
	mockResponse := &addons.GetSelfAddonInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &addons.AddonInfoResponse{
			Data: addons.AddonInfoData{
				Protected: pointer.Bool(true),
			},
		},
	}

	// Expect the client to be called only once.
	mock.When(suite.mockAddonsClient.GetSelfAddonInfoWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	// First call (cache miss)
	isProtected1, err1 := suite.addonsService.CheckProtectedMode()
	suite.Require().NoError(err1)
	suite.True(isProtected1)

	// Second call (should be a cache hit)
	isProtected2, err2 := suite.addonsService.CheckProtectedMode()
	suite.Require().NoError(err2)
	suite.True(isProtected2)
}

func (suite *AddonsServiceTestSuite) TestCheckProtectedMode_ClientError() {
	apiError := errors.New("network failure")
	mock.When(suite.mockAddonsClient.GetSelfAddonInfoWithResponse(suite.ctx)).
		ThenReturn(nil, apiError).
		Verify(matchers.Times(1))

	isProtected, err := suite.addonsService.CheckProtectedMode()
	suite.False(isProtected)
	suite.Require().Error(err)
	suite.ErrorContains(err, "failed to get addon info")
	suite.ErrorIs(err, apiError)
}

func (suite *AddonsServiceTestSuite) TestCheckProtectedMode_Non200Status() {
	mockResponse := &addons.GetSelfAddonInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusForbidden, Status: "Forbidden"},
		Body:         []byte("access denied"),
	}
	mock.When(suite.mockAddonsClient.GetSelfAddonInfoWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	isProtected, err := suite.addonsService.CheckProtectedMode()
	suite.False(isProtected)
	suite.Require().Error(err)
	suite.ErrorContains(err, "failed to get addon info for 'self': status 403, body: access denied")
}

func (suite *AddonsServiceTestSuite) TestCheckProtectedMode_NilJSONResponse() {
	mockResponse := &addons.GetSelfAddonInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200:      nil,
	}
	mock.When(suite.mockAddonsClient.GetSelfAddonInfoWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	isProtected, err := suite.addonsService.CheckProtectedMode()
	suite.False(isProtected)
	suite.Require().Error(err)
	suite.ErrorContains(err, "protected status not available or data incomplete")
}

func (suite *AddonsServiceTestSuite) TestCheckProtectedMode_NilProtectedField() {
	mockResponse := &addons.GetSelfAddonInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &addons.AddonInfoResponse{
			Data: addons.AddonInfoData{
				Protected: nil, // Field is nil
			},
		},
	}
	mock.When(suite.mockAddonsClient.GetSelfAddonInfoWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	isProtected, err := suite.addonsService.CheckProtectedMode()
	suite.False(isProtected)
	suite.Require().Error(err)
	suite.ErrorContains(err, "protected status not available or data incomplete")
}
