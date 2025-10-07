package server_test

import (
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/server"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type TestHumaRoute struct{}

func TestNewHumaAPI(t *testing.T) {
	var api huma.API
	app := fxtest.New(t,
		fx.Provide(
			mux.NewRouter,
			func() []server.HumaRoute {
				return []server.HumaRoute{}
			},
			server.NewHumaAPI,
		),
		fx.Populate(&api),
	)
	
	app.RequireStart()
	assert.NotNil(t, api)
	app.RequireStop()
}

func TestAsHumaRoute(t *testing.T) {
	// Test that AsHumaRoute returns an fx annotation
	result := server.AsHumaRoute(func() server.HumaRoute {
		return &TestHumaRoute{}
	})
	
	assert.NotNil(t, result)
}

func TestNewHumaAPIWithRoutes(t *testing.T) {
	var api huma.API
	app := fxtest.New(t,
		fx.Provide(
			mux.NewRouter,
			func() []server.HumaRoute {
				// Return a non-empty route list
				return []server.HumaRoute{&TestHumaRoute{}}
			},
			server.NewHumaAPI,
		),
		fx.Populate(&api),
	)
	
	app.RequireStart()
	assert.NotNil(t, api)
	app.RequireStop()
}
