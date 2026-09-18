package service_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/apps"
	"github.com/dianlight/srat/homeassistant/discovery"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
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
	if suite.ctx != nil {
		if wg := suite.ctx.Value(ctxkeys.WaitGroup); wg != nil {
			wg.(*sync.WaitGroup).Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

func discoveryEnabledSettings() *dto.Settings {
	return &dto.Settings{EnableHaDiscovery: new(true)}
}

func discoveryDisabledSettings() *dto.Settings {
	return &dto.Settings{EnableHaDiscovery: new(false)}
}

func successDiscoveryResponse(testUUID *openapi_types.UUID) *discovery.CreateDiscoveryServiceResponse {
	return &discovery.CreateDiscoveryServiceResponse{
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
				Uuid: testUUID,
			},
			Result: new(string),
		},
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
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Invoke(func(
			addonsService service.AddonsServiceInterface,
			settingService service.SettingServiceInterface,
			discoveryClient discovery.ClientWithResponsesInterface,
		) {
			mock.When(settingService.Load()).ThenReturn(discoveryEnabledSettings(), nil)
			// Mock addon info
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&apps.AppInfoData{
					Hostname: &hostname,
				}, nil)

			// Mock discovery registration
			mock.When(discoveryClient.CreateDiscoveryServiceWithResponse(
				mock.Any[context.Context](),
				mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody](),
			)).ThenReturn(successDiscoveryResponse(&testUUID), nil)
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
	// The OnStart hook should have called RegisterDiscovery
}

func (suite *HaDiscoveryServiceTestSuite) TestRegisterDiscoverySkipsWhenDisabled() {
	suite.state = &dto.ContextState{
		SupervisorURL:   "http://supervisor",
		SupervisorToken: "test-token",
		HACoreReady:     true,
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg))

	var discoveryClient discovery.ClientWithResponsesInterface
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Populate(&discoveryClient),
		fx.Invoke(func(
			settingService service.SettingServiceInterface,
		) {
			mock.When(settingService.Load()).ThenReturn(discoveryDisabledSettings(), nil)
		}),
	)
	suite.app.RequireStart()
	mock.Verify(discoveryClient, matchers.Times(0)).CreateDiscoveryServiceWithResponse(mock.Any[context.Context](), mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody]())
}

func (suite *HaDiscoveryServiceTestSuite) TestRegisterDiscoverySkipsWhenSettingsLoadFails() {
	suite.state = &dto.ContextState{
		SupervisorURL:   "http://supervisor",
		SupervisorToken: "test-token",
		HACoreReady:     true,
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg))

	var discoveryClient discovery.ClientWithResponsesInterface
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Populate(&discoveryClient),
		fx.Invoke(func(
			settingService service.SettingServiceInterface,
		) {
			mock.When(settingService.Load()).ThenReturn(nil, errors.New("settings load failed"))
		}),
	)
	suite.app.RequireStart()
	mock.Verify(discoveryClient, matchers.Times(0)).CreateDiscoveryServiceWithResponse(mock.Any[context.Context](), mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody]())
}

func (suite *HaDiscoveryServiceTestSuite) TestSettingEventDisableUnregistersDiscovery() {
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

	var eventBus events.EventBusInterface
	var discoveryClient discovery.ClientWithResponsesInterface
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Populate(&eventBus),
		fx.Populate(&discoveryClient),
		fx.Invoke(func(
			addonsService service.AddonsServiceInterface,
			settingService service.SettingServiceInterface,
			client discovery.ClientWithResponsesInterface,
		) {
			mock.When(settingService.Load()).ThenReturn(discoveryEnabledSettings(), nil)
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&apps.AppInfoData{
					Hostname: &hostname,
				}, nil)
			mock.When(client.CreateDiscoveryServiceWithResponse(
				mock.Any[context.Context](),
				mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody](),
			)).ThenReturn(successDiscoveryResponse(&testUUID), nil)
			mock.When(client.DeleteDiscoveryServiceWithResponse(
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
	eventBus.EmitSetting(events.SettingEvent{Setting: discoveryDisabledSettings()})
	mock.Verify(discoveryClient, matchers.Times(1)).DeleteDiscoveryServiceWithResponse(mock.Any[context.Context](), mock.Any[openapi_types.UUID]())
}

func (suite *HaDiscoveryServiceTestSuite) TestSettingEventEnableRegistersDiscovery() {
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

	var eventBus events.EventBusInterface
	var discoveryClient discovery.ClientWithResponsesInterface
	var settingService service.SettingServiceInterface
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Populate(&eventBus),
		fx.Populate(&discoveryClient),
		fx.Populate(&settingService),
		fx.Invoke(func(
			addonsService service.AddonsServiceInterface,
			settings service.SettingServiceInterface,
			client discovery.ClientWithResponsesInterface,
		) {
			mock.When(settings.Load()).ThenReturn(discoveryDisabledSettings(), nil)
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&apps.AppInfoData{
					Hostname: &hostname,
				}, nil)
			mock.When(client.CreateDiscoveryServiceWithResponse(
				mock.Any[context.Context](),
				mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody](),
			)).ThenReturn(successDiscoveryResponse(&testUUID), nil)
			mock.When(client.DeleteDiscoveryServiceWithResponse(
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
	mock.Verify(discoveryClient, matchers.Times(0)).CreateDiscoveryServiceWithResponse(mock.Any[context.Context](), mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody]())
	eventBus.EmitSetting(events.SettingEvent{Setting: discoveryEnabledSettings()})
	mock.Verify(discoveryClient, matchers.Times(1)).CreateDiscoveryServiceWithResponse(mock.Any[context.Context](), mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody]())
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
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Invoke(func(
			settingService service.SettingServiceInterface,
		) {
			mock.When(settingService.Load()).ThenReturn(discoveryEnabledSettings(), nil)
		}),
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
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Invoke(func(
			settingService service.SettingServiceInterface,
		) {
			mock.When(settingService.Load()).ThenReturn(discoveryEnabledSettings(), nil)
		}),
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
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Invoke(func(
			settingService service.SettingServiceInterface,
		) {
			mock.When(settingService.Load()).ThenReturn(discoveryEnabledSettings(), nil)
		}),
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
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Invoke(func(
			addonsService service.AddonsServiceInterface,
			settingService service.SettingServiceInterface,
			discoveryClient discovery.ClientWithResponsesInterface,
		) {
			mock.When(settingService.Load()).ThenReturn(discoveryEnabledSettings(), nil)
			// Return nil hostname — forces fallback to AddonIpAddress
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&apps.AppInfoData{}, nil)

			// Mock discovery registration
			mock.When(discoveryClient.CreateDiscoveryServiceWithResponse(
				mock.Any[context.Context](),
				mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody](),
			)).ThenReturn(successDiscoveryResponse(&testUUID), nil)
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
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Invoke(func(
			addonsService service.AddonsServiceInterface,
			settingService service.SettingServiceInterface,
			discoveryClient discovery.ClientWithResponsesInterface,
		) {
			mock.When(settingService.Load()).ThenReturn(discoveryEnabledSettings(), nil)
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&apps.AppInfoData{
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
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewHaDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.SettingServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[discovery.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.haDiscoveryService),
		fx.Invoke(func(
			addonsService service.AddonsServiceInterface,
			settingService service.SettingServiceInterface,
			discoveryClient discovery.ClientWithResponsesInterface,
		) {
			mock.When(settingService.Load()).ThenReturn(discoveryEnabledSettings(), nil)
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&apps.AppInfoData{
					Hostname: &hostname,
				}, nil)

			// Mock registration
			mock.When(discoveryClient.CreateDiscoveryServiceWithResponse(
				mock.Any[context.Context](),
				mock.Any[discovery.CreateDiscoveryServiceJSONRequestBody](),
			)).ThenReturn(successDiscoveryResponse(&testUUID), nil)

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
