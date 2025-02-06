package server

import (
	"context"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jpillora/overseer"
	"github.com/rs/cors"
	"github.com/ztrue/tracerr"
	"go.uber.org/fx"
)

func NewHTTPServer(lc fx.Lifecycle, mux *mux.Router, state *overseer.State, cxtClose context.CancelFunc) *http.Server {
	handler := cors.New(
		cors.Options{
			//AllowedOrigins:   []string{"*"},
			AllowOriginFunc:  func(origin string) bool { return true },
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
			MaxAge:           300,
		},
	).Handler(mux)
	loggedRouter := handlers.LoggingHandler(os.Stdout, handler)
	srv := &http.Server{
		ReadTimeout: time.Second * 15,
		IdleTimeout: time.Second * 60,
		Handler:     loggedRouter, // Pass our instance of gorilla/mux in.
		//		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
		//			log.Printf("New connection: %s\n", c.RemoteAddr())
		//			ctx = api.StateToContext(&sharedResources, ctx)
		//			return ctx
		//		},
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				slog.Debug("Starting HTTP server at", "listener", state.Address, "pid", state.ID)
				if err := srv.Serve(state.Listener); err != nil {
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

type RouteDetail struct {
	Pattern string
	Method  string
	Handler http.HandlerFunc
}

type Route interface {
	Patterns() []RouteDetail
}

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
	// Static files
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusPermanentRedirect)
	})
	router.PathPrefix("/").Handler(http.FileServerFS(static)).Methods(http.MethodGet)

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

func AsRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Route)),
		fx.ResultTags(`group:"routes"`),
	)
}
