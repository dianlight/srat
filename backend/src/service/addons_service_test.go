package service_test

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/apps"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type AddonsServiceTestSuite struct {
	suite.Suite
	addonsService    service.AddonsServiceInterface
	mockAddonsClient apps.ClientWithResponsesInterface
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
				ctx := context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg)
				return context.WithCancel(ctx)
			},
			service.NewAddonsService,
			func() *dto.ContextState {
				return &dto.ContextState{HACoreReady: true}
			},
			mock.Mock[apps.ClientWithResponsesInterface],
			mock.Mock[service.HaWsServiceInterface],
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

func (suite *AddonsServiceTestSuite) TestGetStats_Success() {
	cpu := 55.5
	mem := int(1024 * 1024 * 100)
	expectedStats := apps.AppStatsData{CpuPercent: &cpu, MemoryUsage: &mem}
	mockResponse := &apps.GetSelfAppStatsResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200:      &apps.AppStatsResponse{Data: expectedStats},
	}

	mock.When(suite.mockAddonsClient.GetSelfAppStatsWithResponse(mock.AnyContext())).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	stats, err := suite.addonsService.GetStats()
	suite.Require().NoError(err)
	suite.Require().NotNil(stats)
	suite.Equal(expectedStats, *stats)
}

func (suite *AddonsServiceTestSuite) TestGetStats_CacheHit() {
	cpu := 55.5
	mem := int(1024 * 1024 * 100)
	expectedStats := apps.AppStatsData{CpuPercent: &cpu, MemoryUsage: &mem}
	mockResponse := &apps.GetSelfAppStatsResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200:      &apps.AppStatsResponse{Data: expectedStats},
	}

	mock.When(suite.mockAddonsClient.GetSelfAppStatsWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	stats1, err1 := suite.addonsService.GetStats()
	suite.Require().NoError(err1)
	suite.Require().NotNil(stats1)

	stats2, err2 := suite.addonsService.GetStats()
	suite.Require().NoError(err2)
	suite.Require().NotNil(stats2)
	suite.Equal(expectedStats, *stats2)
}

func (suite *AddonsServiceTestSuite) TestGetStats_ClientError() {
	apiError := errors.New("network failure")
	mock.When(suite.mockAddonsClient.GetSelfAppStatsWithResponse(suite.ctx)).
		ThenReturn(nil, apiError).
		Verify(matchers.Times(1))

	stats, err := suite.addonsService.GetStats()
	suite.Nil(stats)
	suite.Require().Error(err)
	suite.ErrorContains(err, "failed to get addon stats")
	suite.ErrorIs(err, apiError)
}

func (suite *AddonsServiceTestSuite) TestGetStats_Non200Status() {
	mockResponse := &apps.GetSelfAppStatsResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusNotFound, Status: "Not Found"},
		Body:         []byte("addon not found"),
	}
	mock.When(suite.mockAddonsClient.GetSelfAppStatsWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	stats, err := suite.addonsService.GetStats()
	suite.Nil(stats)
	suite.Require().Error(err)
	suite.ErrorContains(err, "failed to get addon stats: status 404, body: addon not found")
}

func (suite *AddonsServiceTestSuite) TestGetStats_NilJSONResponse() {
	mockResponse := &apps.GetSelfAppStatsResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200:      nil,
	}
	mock.When(suite.mockAddonsClient.GetSelfAppStatsWithResponse(suite.ctx)).
		ThenReturn(mockResponse, nil).
		Verify(matchers.Times(1))

	stats, err := suite.addonsService.GetStats()
	suite.Nil(stats)
	suite.Require().Error(err)
	suite.ErrorContains(err, "addon stats not available or data incomplete")
}

func (suite *AddonsServiceTestSuite) TestGetAppConfigSchema_Success() {
	desc := "Example app"
	longDesc := "Example long description"
	fieldLogLevel := map[string]interface{}{"log_level": "str"}
	fieldEnabled := map[string]interface{}{"enabled": "bool"}
	schema := []*map[string]interface{}{
		&fieldLogLevel,
		&fieldEnabled,
	}

	mock.When(suite.mockAddonsClient.GetAppInfoWithResponse(suite.ctx, "self")).
		ThenReturn(&apps.GetAppInfoResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &apps.AppInfoResponse{Data: apps.AppInfoData{
				Description:     &desc,
				LongDescription: &longDesc,
				Schema:          &schema,
			}},
		}, nil).
		Verify(matchers.Times(1))

	result, err := suite.addonsService.GetAppConfigSchema(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)
	suite.Equal("Example app", result.Description)
	suite.Equal("Example long description", result.LongDescription)
	suite.True(result.RequiresRestart)
	suite.Len(result.Fields, 2)

	fieldConstraints := make(map[string]string)
	for _, field := range result.Fields {
		fieldConstraints[field.Name] = field.Constraint
	}
	suite.Equal("str", fieldConstraints["log_level"])
	suite.Equal("bool", fieldConstraints["enabled"])
}

func (suite *AddonsServiceTestSuite) TestGetAppConfigSchema_DescriptorItems() {
	descriptorAutoUpdate := map[string]interface{}{
		"name":     "auto_update",
		"type":     "bool",
		"optional": true,
	}
	descriptorLogLevel := map[string]interface{}{
		"name":    "log_level",
		"type":    "str",
		"options": []interface{}{"trace", "debug", "info"},
	}
	schema := []*map[string]interface{}{
		&descriptorAutoUpdate,
		&descriptorLogLevel,
	}

	mock.When(suite.mockAddonsClient.GetAppInfoWithResponse(suite.ctx, "self")).
		ThenReturn(&apps.GetAppInfoResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &apps.AppInfoResponse{Data: apps.AppInfoData{
				Schema: &schema,
			}},
		}, nil).
		Verify(matchers.Times(1))

	result, err := suite.addonsService.GetAppConfigSchema(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	type fieldExpectation struct {
		constraint string
		optional   bool
		options    []string
	}
	fieldMap := make(map[string]fieldExpectation)
	for _, field := range result.Fields {
		fieldMap[field.Name] = fieldExpectation{
			constraint: field.Constraint,
			optional:   field.Optional,
			options:    field.Options,
		}
	}

	suite.Contains(fieldMap, "auto_update")
	suite.Equal("bool", fieldMap["auto_update"].constraint)
	suite.True(fieldMap["auto_update"].optional)

	suite.Contains(fieldMap, "log_level")
	suite.Equal("str", fieldMap["log_level"].constraint)
	suite.ElementsMatch([]string{"trace", "debug", "info"}, fieldMap["log_level"].options)
}

func (suite *AddonsServiceTestSuite) TestGetAppConfigSchema_FieldDescriptionsFromTranslations() {
	fieldEnabled := map[string]interface{}{"enabled": "bool"}
	fieldLogLevel := map[string]interface{}{"log_level": "str"}
	schema := []*map[string]interface{}{
		&fieldEnabled,
		&fieldLogLevel,
	}

	translations := map[string]interface{}{
		"en": map[string]interface{}{
			"configuration": map[string]interface{}{
				"enabled":   "Enable or disable the feature",
				"log_level": "Application logging level",
			},
		},
	}

	mock.When(suite.mockAddonsClient.GetAppInfoWithResponse(suite.ctx, "self")).
		ThenReturn(&apps.GetAppInfoResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &apps.AppInfoResponse{Data: apps.AppInfoData{
				Schema:       &schema,
				Translations: &translations,
			}},
		}, nil).
		Verify(matchers.Times(1))

	result, err := suite.addonsService.GetAppConfigSchema(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	fieldDescriptions := make(map[string]string)
	for _, field := range result.Fields {
		fieldDescriptions[field.Name] = field.Description
	}

	suite.Equal("Enable or disable the feature", fieldDescriptions["enabled"])
	suite.Equal("Application logging level", fieldDescriptions["log_level"])
}

func (suite *AddonsServiceTestSuite) TestSetAppConfig_Success() {
	options := map[string]any{"log_level": "debug", "enabled": true}
	request := apps.SetAppOptionsJSONRequestBody{Options: &options}

	mock.When(suite.mockAddonsClient.SetAppOptionsWithResponse(suite.ctx, "self", request)).
		ThenReturn(&apps.SetAppOptionsResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		}, nil).
		Verify(matchers.Times(1))

	err := suite.addonsService.SetAppConfig(suite.ctx, options)
	suite.Require().NoError(err)
}

func (suite *AddonsServiceTestSuite) TestGetAppConfig_Success() {
	options := map[string]interface{}{"log_level": "info"}
	runtimeConfig := apps.AppOptionsConfigData{"rendered": true}

	mock.When(suite.mockAddonsClient.GetAppInfoWithResponse(suite.ctx, "self")).
		ThenReturn(&apps.GetAppInfoResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &apps.AppInfoResponse{Data: apps.AppInfoData{
				Options: &options,
			}},
		}, nil)

	mock.When(suite.mockAddonsClient.GetAppOptionsConfigWithResponse(suite.ctx, "self")).
		ThenReturn(&apps.GetAppOptionsConfigResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &apps.AppOptionsConfigResponse{
				Data: runtimeConfig,
			},
		}, nil)

	result, err := suite.addonsService.GetAppConfig(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)
	suite.Equal("info", result.Options["log_level"])
	suite.Equal(true, result.RuntimeConfig["rendered"])
	suite.True(result.RequiresRestart)
}

func (suite *AddonsServiceTestSuite) TestGetStats_ClientNotInitialized() {
	var addonsService service.AddonsServiceInterface
	app := fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{})
				return context.WithCancel(ctx)
			},
			service.NewAddonsService,
			mock.Mock[service.HaWsServiceInterface],
			func() *dto.ContextState {
				return &dto.ContextState{HACoreReady: true}
			},
			func() apps.ClientWithResponsesInterface { return nil },
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
