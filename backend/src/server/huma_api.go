package server

import (
	"maps"

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
	config.Servers = []*huma.Server{}
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
		},
	}
	config.Security = []map[string][]string{
		{"ApiKeyAuth": {}},
	}

	config.Components.Parameters = map[string]*huma.Param{
		"X-Span-Id": {
			Description: "Unique span ID",
			In:          "header",
			Name:        "X-Span-Id",
			Schema:      &huma.Schema{Type: "string"},
		},
		"X-Trace-Id": {
			Description: "Unique trace ID",
			In:          "header",
			Name:        "X-Trace-Id",
			Schema:      &huma.Schema{Type: "string"},
		},
	}

	config.Components.Headers = map[string]*huma.Header{
		"X-Span-Id": {
			Description: "Unique span ID",
			Schema:      &huma.Schema{Type: "string"},
		},
		"X-Trace-Id": {
			Description: "Unique trace ID",
			Schema:      &huma.Schema{Type: "string"},
		},
	}

	api := humamux.New(v.Mux, config)
	apigroup := huma.NewGroup(api, "/api")
	apigroup.UseSimpleModifier(func(op *huma.Operation) {
		for p := range maps.Values(config.Components.Parameters) {
			op.Parameters = append(op.Parameters, &huma.Param{
				Ref: "#/components/parameters/" + p.Name,
			})
		}
		op.Responses["4xx"] = &huma.Response{
			Content: map[string]*huma.MediaType{
				"application/problem+json": {
					Schema: &huma.Schema{
						Ref: "#/components/schemas/ErrorModel",
					},
				},
			},
			Description: "Error",
		}

		for code := range op.Responses {
			if op.Responses[code].Headers == nil {
				op.Responses[code].Headers = make(map[string]*huma.Header)
			}
			for key := range maps.Keys(config.Components.Headers) {
				op.Responses[code].Headers[key] = &huma.Header{
					Ref: "#/components/headers/" + key,
				}
			}

			//			maps.Copy(op.Responses[code].Headers, config.Components.Headers)
		}
	})

	for _, route := range v.Routes {
		huma.AutoRegister(apigroup, route)
	}
	autopatch.AutoPatch(apigroup)

	return apigroup
}

func AsHumaRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(HumaRoute)),
		fx.ResultTags(`group:"api_routes"`),
	)
}
