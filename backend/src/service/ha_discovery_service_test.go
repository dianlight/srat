package service_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/addons"
	"github.com/dianlight/srat/homeassistant/discovery"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/internal/ctxkeys"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type HaDiscoveryServiceTestSuite struct {
	suite.Suite
	haDiscoveryService service.HaDiscoveryServiceInterface
	state              *dto.ContextState
	app                *fxtest.App
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 *sync.WaitGroup
	server             *httptest.Server
}

func TestHaDiscoveryServiceTestSuite(t *testing.T) {
	suite.Run(t, new(HaDiscoveryServiceTestSuite))
}

func (suite *HaDiscoveryServiceTestSuite) TearDownTest() {
	if suite.server != nil {
		suite.server.Close()
	}
	if suite.cancel != nil {
		suite.cancel()
	}
}

func (suite *HaDiscoveryServiceTestSuite) TestRegisterDiscoverySuccess() {
	hostname := "test-addon-host"
	testUUID := openapi_types.UUID{}
	_ = testUUID.UnmarshalText([]byte("550e8400-e29b-41d4-a716-446655440000"))

	suite.state = &dto.ContextState{
		SupervisorURL:   "http://supervisor",
		SupervisorToken: "test-token",
		HACoreReady:     true,
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Invoke(func(
			addonsService service.AddonsServiceInterface,
			discoveryClient discovery.ClientWithResponsesInterface,
		) {
			// Mock addon info
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&addons.AddonInfoData{
					Hostname: &hostname,
				}, nil)

			// Mock discovery registration
			mock.When(discoveryClient.CreateDiscoveryServiceWithResponse(
				mock.Any[context.Context](),
				mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody](),
			)).ThenReturn(&discovery.CreateDiscoveryServiceResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &struct {
					Data *struct {
						Uuid *openapi_types.UUID `json:"uuid,omitempty"`
					} `json:"data,omitempty"`
					Result *string `json:"result,omitempty"`
				}{
					Data: &struct {
						Uuid *openapi_types.UUID `json:"uuid,omitempty"`
					}{
						Uuid: &testUUID,
					},
					Result: new(string),
				},
			}, nil)
		}),
	)
	suite.app.RequireStart()
	// The OnStart hook should have called RegisterDiscovery
}

func (suite *HaDiscoveryServiceTestSuite) TestRegisterDiscoverySkipsInDemoMode() {
	suite.state = &dto.ContextState{
		SupervisorURL:   "demo",
		SupervisorToken: "test-token",
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
	)
	suite.app.RequireStart()
	// Should skip without error
}

func (suite *HaDiscoveryServiceTestSuite) TestRegisterDiscoverySkipsWithEmptySupervisorURL() {
	suite.state = &dto.ContextState{
		SupervisorURL:   "",
		SupervisorToken: "test-token",
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
	)
	suite.app.RequireStart()
}

func (suite *HaDiscoveryServiceTestSuite) TestRegisterDiscoverySkipsWithNoToken() {
	suite.state = &dto.ContextState{
		SupervisorURL:   "http://supervisor",
		SupervisorToken: "",
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
	)
	suite.app.RequireStart()
}

func (suite *HaDiscoveryServiceTestSuite) TestRegisterDiscoveryFallsBackToAddonIP() {
	testUUID := openapi_types.UUID{}
	_ = testUUID.UnmarshalText([]byte("550e8400-e29b-41d4-a716-446655440000"))

	suite.state = &dto.ContextState{
		SupervisorURL:   "http://supervisor",
		SupervisorToken: "test-token",
		AddonIpAddress:  "172.30.32.1",
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Invoke(func(
			addonsService service.AddonsServiceInterface,
			discoveryClient discovery.ClientWithResponsesInterface,
		) {
			// Return nil hostname — forces fallback to AddonIpAddress
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&addons.AddonInfoData{}, nil)

			// Mock discovery registration
			mock.When(discoveryClient.CreateDiscoveryServiceWithResponse(
				mock.Any[context.Context](),
				mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody](),
			)).ThenReturn(&discovery.CreateDiscoveryServiceResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &struct {
					Data *struct {
						Uuid *openapi_types.UUID `json:"uuid,omitempty"`
					} `json:"data,omitempty"`
					Result *string `json:"result,omitempty"`
				}{
					Data: &struct {
						Uuid *openapi_types.UUID `json:"uuid,omitempty"`
					}{
						Uuid: &testUUID,
					},
					Result: new(string),
				},
			}, nil)
		}),
	)
	suite.app.RequireStart()
}

func (suite *HaDiscoveryServiceTestSuite) TestRegisterDiscoveryHandlesHTTPError() {
	hostname := "test-addon-host"

	suite.state = &dto.ContextState{
		SupervisorURL:   "http://supervisor",
		SupervisorToken: "test-token",
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Invoke(func(
			addonsService service.AddonsServiceInterface,
			discoveryClient discovery.ClientWithResponsesInterface,
		) {
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&addons.AddonInfoData{
					Hostname: &hostname,
				}, nil)

			// Mock HTTP error
			mock.When(discoveryClient.CreateDiscoveryServiceWithResponse(
				mock.Any[context.Context](),
				mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody](),
			)).ThenReturn(&discovery.CreateDiscoveryServiceResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusForbidden},
				Body:         []byte("Add-ons must list services they provide via discovery in their config!"),
			}, nil)
		}),
	)
	// Should start successfully — discovery errors are non-fatal
	suite.app.RequireStart()
}

func (suite *HaDiscoveryServiceTestSuite) TestUnregisterDiscoverySuccess() {
	hostname := "test-addon-host"
	testUUID := openapi_types.UUID{}
	_ = testUUID.UnmarshalText([]byte("550e8400-e29b-41d4-a716-446655440000"))

	suite.state = &dto.ContextState{
		SupervisorURL:   "http://supervisor",
		SupervisorToken: "test-token",
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Invoke(func(
			addonsService service.AddonsServiceInterface,
			discoveryClient discovery.ClientWithResponsesInterface,
		) {
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&addons.AddonInfoData{
					Hostname: &hostname,
				}, nil)

			// Mock registration
			mock.When(discoveryClient.CreateDiscoveryServiceWithResponse(
				mock.Any[context.Context](),
				mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody](),
			)).ThenReturn(&discovery.CreateDiscoveryServiceResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &struct {
					Data *struct {
						Uuid *openapi_types.UUID `json:"uuid,omitempty"`
					} `json:"data,omitempty"`
					Result *string `json:"result,omitempty"`
				}{
					Data: &struct {
						Uuid *openapi_types.UUID `json:"uuid,omitempty"`
					}{
						Uuid: &testUUID,
					},
					Result: new(string),
				},
			}, nil)

			// Mock unregistration
			mock.When(discoveryClient.DeleteDiscoveryServiceWithResponse(
				mock.Any[context.Context](),
				mock.Any[openapi_types.UUID](),
			)).ThenReturn(&discovery.DeleteDiscoveryServiceResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &discovery.SimpleOkResponse{
					Result: new(string),
				},
			}, nil)
		}),
	)
	suite.app.RequireStart()
	suite.app.RequireStop()
}
