package server

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/danielgtaylor/huma/v2/autopatch"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
)

type HumaRoute interface {
	// HumaRoute(*huma.API) error
}

func NewHumaAPI(v struct {
	fx.In
	Mux    *mux.Router
	Routes []HumaRoute `group:"api_routes"`
}) huma.API {
	config := huma.DefaultConfig("SRAT API", "1.0.0")
	config.Info.Description = "This are samba rest admin API"
	config.Info.Contact = &huma.Contact{}
	config.Info.Contact.Name = "Lucio Tarantino"
	config.Info.Contact.URL = "https://github.com/dianlight"
	config.Info.Contact.Email = "lucio.tarantino@gmail.com"
	config.Info.License = &huma.License{}
	config.Info.License.Name = "Apache 2.0"
	config.Info.License.URL = "http://www.apache.org/licenses/LICENSE-2.0.html"

	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"ApiKeyAuth": {
			Type:        "apiKey",
			Description: "HomeAssistant Supervisor Token",
			In:          "header",
			Name:        "X-Supervisor-Token",
			Scheme:      "bearer",
		},
	}
	config.Security = []map[string][]string{
		{"ApiKeyAuth": {}},
	}

	api := humamux.New(v.Mux, config)
	for _, route := range v.Routes {
		huma.AutoRegister(api, route)
	}
	autopatch.AutoPatch(api)
	return api
}

func AsHumaRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(HumaRoute)),
		fx.ResultTags(`group:"api_routes"`),
	)
}
