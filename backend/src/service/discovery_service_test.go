package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/addons"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type DiscoveryServiceTestSuite struct {
	suite.Suite
	discoveryService service.DiscoveryServiceInterface
	state            *dto.ContextState
	app              *fxtest.App
	ctx              context.Context
	cancel           context.CancelFunc
	wg               *sync.WaitGroup
	server           *httptest.Server
}

func TestDiscoveryServiceTestSuite(t *testing.T) {
	suite.Run(t, new(DiscoveryServiceTestSuite))
}

func (suite *DiscoveryServiceTestSuite) TearDownTest() {
	if suite.server != nil {
		suite.server.Close()
	}
	if suite.cancel != nil {
		suite.cancel()
	}
}

func (suite *DiscoveryServiceTestSuite) TestRegisterDiscoverySuccess() {
	// Mock Supervisor discovery API
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suite.Equal("/discovery", r.URL.Path)
		suite.Equal("POST", r.Method)
		suite.Contains(r.Header.Get("Authorization"), "Bearer test-token")
		suite.Equal("application/json", r.Header.Get("Content-Type"))

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		suite.Equal("srat", body["service"])
		config := body["config"].(map[string]any)
		suite.Equal("test-addon-host", config["host"])
		suite.Equal(float64(8099), config["port"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"result": "ok",
			"data":   map[string]string{"uuid": "test-uuid-1234"},
		})
	}))

	suite.state = &dto.ContextState{
		SupervisorURL:   suite.server.URL,
		SupervisorToken: "test-token",
		HACoreReady:     true,
	}

	hostname := "test-addon-host"
	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), "wg", suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
		),
		fx.Populate(&suite.discoveryService),
		fx.Invoke(func(addonsService service.AddonsServiceInterface) {
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&addons.AddonInfoData{
					Hostname: &hostname,
				}, nil)
		}),
	)
	suite.app.RequireStart()
	// The OnStart hook should have called RegisterDiscovery
	// We verified the call via the HTTP handler assertions above
}

func (suite *DiscoveryServiceTestSuite) TestRegisterDiscoverySkipsInDemoMode() {
	suite.state = &dto.ContextState{
		SupervisorURL:   "demo",
		SupervisorToken: "test-token",
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), "wg", suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
		),
		fx.Populate(&suite.discoveryService),
	)
	suite.app.RequireStart()
	// Should skip without error — no HTTP server needed
}

func (suite *DiscoveryServiceTestSuite) TestRegisterDiscoverySkipsWithEmptySupervisorURL() {
	suite.state = &dto.ContextState{
		SupervisorURL:   "",
		SupervisorToken: "test-token",
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), "wg", suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
		),
		fx.Populate(&suite.discoveryService),
	)
	suite.app.RequireStart()
}

func (suite *DiscoveryServiceTestSuite) TestRegisterDiscoverySkipsWithNoToken() {
	suite.state = &dto.ContextState{
		SupervisorURL:   "http://supervisor",
		SupervisorToken: "",
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), "wg", suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
		),
		fx.Populate(&suite.discoveryService),
	)
	suite.app.RequireStart()
}

func (suite *DiscoveryServiceTestSuite) TestRegisterDiscoveryFallsBackToAddonIP() {
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		config := body["config"].(map[string]any)
		suite.Equal("172.30.32.1", config["host"]) // Falls back to AddonIpAddress

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"result": "ok",
			"data":   map[string]string{"uuid": "test-uuid-5678"},
		})
	}))

	suite.state = &dto.ContextState{
		SupervisorURL:   suite.server.URL,
		SupervisorToken: "test-token",
		AddonIpAddress:  "172.30.32.1",
	}

	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), "wg", suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
		),
		fx.Populate(&suite.discoveryService),
		fx.Invoke(func(addonsService service.AddonsServiceInterface) {
			// Return nil hostname — forces fallback to AddonIpAddress
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&addons.AddonInfoData{}, nil)
		}),
	)
	suite.app.RequireStart()
}

func (suite *DiscoveryServiceTestSuite) TestRegisterDiscoveryHandlesHTTPError() {
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"result":  "error",
			"message": "Add-ons must list services they provide via discovery in their config!",
		})
	}))

	suite.state = &dto.ContextState{
		SupervisorURL:   suite.server.URL,
		SupervisorToken: "test-token",
	}

	hostname := "test-addon-host"
	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), "wg", suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
		),
		fx.Populate(&suite.discoveryService),
		fx.Invoke(func(addonsService service.AddonsServiceInterface) {
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&addons.AddonInfoData{
					Hostname: &hostname,
				}, nil)
		}),
	)
	// Should start successfully — discovery errors are non-fatal
	suite.app.RequireStart()
}

func (suite *DiscoveryServiceTestSuite) TestUnregisterDiscoverySuccess() {
	callCount := 0
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"result": "ok",
				"data":   map[string]string{"uuid": "test-uuid-delete"},
			})
		} else if r.Method == "DELETE" {
			suite.Equal("/discovery/test-uuid-delete", r.URL.Path)
			suite.Contains(r.Header.Get("Authorization"), "Bearer test-token")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"result": "ok",
			})
		}
	}))

	suite.state = &dto.ContextState{
		SupervisorURL:   suite.server.URL,
		SupervisorToken: "test-token",
	}

	hostname := "test-addon-host"
	suite.wg = &sync.WaitGroup{}
	suite.ctx, suite.cancel = context.WithCancel(context.WithValue(context.Background(), "wg", suite.wg))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return suite.ctx, suite.cancel
			},
			func() *dto.ContextState { return suite.state },
			service.NewDiscoveryService,
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			mock.Mock[service.AddonsServiceInterface],
			mock.Mock[service.HaWsServiceInterface],
		),
		fx.Populate(&suite.discoveryService),
		fx.Invoke(func(addonsService service.AddonsServiceInterface) {
			mock.When(addonsService.GetInfo(mock.Any[context.Context]())).
				ThenReturn(&addons.AddonInfoData{
					Hostname: &hostname,
				}, nil)
		}),
	)
	suite.app.RequireStart()
	suite.app.RequireStop()
	suite.Equal(2, callCount) // POST + DELETE
}
