package server

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/dianlight/srat/homeassistant/ingress"
	"github.com/gorilla/mux"
	"github.com/jpillora/overseer"
	"github.com/rs/cors"
	sloghttp "github.com/samber/slog-http"
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
	//loggedRouter := handlers.LoggingHandler(os.Stdout, handler)
	loggedRouter := sloghttp.NewWithConfig(slog.Default(), sloghttp.Config{
		DefaultLevel:  slog.LevelDebug,
		WithUserAgent: false,
		WithRequestID: false,
	})(sloghttp.Recovery(handler))
	srv := &http.Server{
		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      loggedRouter,
		ErrorLog:     slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				slog.Debug("Starting HTTP server at", "listener", state.Address, "pid", state.ID)
				if err := srv.Serve(state.Listener); err != nil {
					if err == http.ErrServerClosed {
						slog.Info("HTTP server stopped gracefully")
					} else {
						log.Fatal(fmt.Sprintf("%#+v", err))
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
			//	return errors.WithStack(err)
			//}
			//time.Sleep(15 * time.Second)
			slog.Info("HTTP server stopped")
			return nil
		},
	})
	return srv
}

func NewMuxRouter(hamode bool, ingressClient_ *ingress.ClientWithResponses) *mux.Router {
	router := mux.NewRouter()
	if hamode {
		ingressClient = ingressClient_
		router.Use(HAMiddleware)
	}

	return router
}
