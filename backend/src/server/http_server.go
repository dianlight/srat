package server

import (
	"context"
	"io/fs"
	"log"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
	"github.com/jpillora/overseer"
	"github.com/rs/cors"
	"github.com/ztrue/tracerr"
	"go.uber.org/fx"
)

func NewHTTPServer(lc fx.Lifecycle,
	state *overseer.State,
	cxtClose context.CancelFunc,
	logger *slog.Logger,
	//routes []Route,
	hamode bool,
	static fs.FS,
	routes []Route,
) *fuego.Server {
	/*handler := cors.New(
		cors.Options{
			//AllowedOrigins:   []string{"*"},
			AllowOriginFunc:  func(origin string) bool { return true },
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
			MaxAge:           300,
		},
	).Handler(mux)
	*/
	//loggedRouter := handlers.LoggingHandler(os.Stdout, handler)
	srv := fuego.NewServer(
		fuego.WithListener(state.Listener),
		fuego.WithLogHandler(logger.Handler()),
		fuego.WithGlobalMiddlewares(cors.New(
			cors.Options{
				//AllowedOrigins:   []string{"*"},
				AllowOriginFunc:  func(origin string) bool { return true },
				AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
				AllowedHeaders:   []string{"*"},
				AllowCredentials: true,
				MaxAge:           300,
			},
		).Handler,
		),
		fuego.WithEngineOptions(
			fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
				DisableSwaggerUI: false, // If true, the server will not serve the swagger ui nor the openapi json spec
				DisableLocalSave: false, // If true, the server will not save the openapi json spec locally
				PrettyFormatJSON: true,
				SwaggerURL:       "/swagger",              // URL to serve the swagger ui
				SpecURL:          "/swagger/openapi.json", // URL to serve the openapi json spec
				JSONFilePath:     "src/docs/openapi.json", // Local path to save the openapi json spec
				//UIHandler:        DefaultOpenAPIHandler,   // Custom UI handler
			}),
		),
	)

	srv.OpenAPI.Description().Info.Title = "SRAT API"
	srv.OpenAPI.Description().Info.Description = "This are samba rest admin API"
	srv.OpenAPI.Description().Info.Version = "1.0"
	srv.OpenAPI.Description().Info.Contact = &openapi3.Contact{
		Name:  "Lucio Tarantino",
		Email: "lucio.tarantino@gmail.com",
		URL:   "https://github.com/dianlight/srat",
	}

	srv.OpenAPI.Description().Security = []openapi3.SecurityRequirement{
		{
			"ApiKeyAuth": []string{},
		},
	}
	srv.OpenAPI.Description().Components.SecuritySchemes = openapi3.SecuritySchemes{
		"ApiKeyAuth": &openapi3.SecuritySchemeRef{
			Value: openapi3.NewCSRFSecurityScheme().
				WithName("X-Supervisor-Token").
				WithDescription("HomeAssistant Supervisor Token").
				WithIn("header"),
		},
	}

	if hamode {
		fuego.Use(srv, HAMiddleware)
	}
	for _, route := range routes {
		err := route.Routers(srv)
		if err != nil {
			slog.Warn(err.Error())
		}
	}

	slog.Info("Exposed static", "FS", static)

	_, err := fs.ReadDir(static, "static")
	if err != nil {
		slog.Warn("Static directory not found:", "err", err)
		fuego.Handle(srv, "/", http.FileServerFS(static))
	} else {
		fsRoot, _ := fs.Sub(static, "static")
		fuego.Handle(srv, "/", http.FileServerFS(fsRoot))
	}

	/*
		_, err = fs.ReadDir(static, "docs")
		if err != nil {
			slog.Warn("Docs directory not found:", "err", err)
		} else {
			fuego.Handle(srv, "/docs", http.FileServerFS(static))
		}
	*/

	/*
		router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			template, err := route.GetPathTemplate()
			if err != nil {
				return tracerr.Wrap(err)
			}
			slog.Debug("Route:", "template", template)
			return nil
		})
	*/

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				slog.Debug("Starting HTTP server at", "listener", state.Address, "pid", state.ID)
				if err := srv.Run(); err != nil {
					if err == http.ErrServerClosed {
						slog.Info("HTTP server stopped gracefully")
					} else {
						log.Fatal(tracerr.SprintSourceColor(err))
					}
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			slog.Info("Stopping HTTP server")
			//state.Listener.Close()
			cxtClose()
			//ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			//defer cancel()
			//err := srv.Shutdown(ctx)
			//if err != nil {
			//	return tracerr.Wrap(err)
			//}
			//time.Sleep(15 * time.Second)
			slog.Info("HTTP server stopped")
			return nil
		},
	})
	return srv
}

/*
	type RouteDetail struct {
		Pattern string
		Method  string
		Handler http.HandlerFunc
	}
*/
type Route interface {
	Routers(srv *fuego.Server) error
}

/*
	func NewMuxRouter(routes []Route, hamode bool, static fs.FS) *mux.Router {
		router := mux.NewRouter()
		if hamode {
			router.Use(HAMiddleware)
		}
		for _, route := range routes {
			for _, detail := range route.Patterns() {
				router.Handle(detail.Pattern, detail.Handler).Methods(detail.Method)
			}
		}

		slog.Info("Exposed static", "FS", static)

		_, err := fs.ReadDir(static, "docs")
		if err != nil {
			slog.Warn("Docs directory not found:", "err", err)
		} else {
			router.PathPrefix("/docs").Handler(http.FileServerFS(static)).Methods(http.MethodGet)
		}

		_, err = fs.ReadDir(static, "static")
		if err != nil {
			slog.Warn("Static directory not found:", "err", err)
			router.PathPrefix("/").Handler(http.FileServerFS(static)).Methods(http.MethodGet)
		} else {
			fsRoot, _ := fs.Sub(static, "static")
			router.PathPrefix("/").Handler(http.FileServerFS(fsRoot)).Methods(http.MethodGet)
		}

		router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			template, err := route.GetPathTemplate()
			if err != nil {
				return tracerr.Wrap(err)
			}
			slog.Debug("Route:", "template", template)
			return nil
		})

		return router
	}
*/
func AsRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Route)),
		fx.ResultTags(`group:"routes"`),
	)
}
