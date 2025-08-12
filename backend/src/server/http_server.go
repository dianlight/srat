package server

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/jpillora/overseer"
	"github.com/rs/cors"
	sloghttp "github.com/samber/slog-http"
	"go.uber.org/fx"
)

func NewHTTPServer(
	lc fx.Lifecycle,
	mux *mux.Router,
	state *overseer.State,
	apiContext context.Context,
	cxtClose context.CancelFunc,
) *http.Server {
	handler := sloghttp.NewWithConfig(slog.Default(), sloghttp.Config{
		DefaultLevel:       slog.LevelDebug,
		WithRequestBody:    true,
		WithRequestHeader:  true,
		WithResponseBody:   true,
		WithResponseHeader: true,
		WithUserAgent:      true,
		WithRequestID:      true,
		WithSpanID:         true,
		WithTraceID:        true,
	})(sloghttp.Recovery(mux))
	handler = cors.New(
		cors.Options{
			//AllowedOrigins:   []string{"*"},
			AllowOriginFunc:     func(origin string) bool { return true },
			AllowedMethods:      []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
			AllowedHeaders:      []string{"*"},
			AllowCredentials:    true,
			AllowPrivateNetwork: true,
			MaxAge:              300,
		},
	).Handler(handler)
	srv := &http.Server{
		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      handler,
		ErrorLog:     slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			apiContext.Value("wg").(*sync.WaitGroup).Add(1)
			go func() {
				defer apiContext.Value("wg").(*sync.WaitGroup).Done()
				slog.Debug("Starting HTTP server at", "listener", state.Address, "pid", state.ID)
				if err := srv.Serve(state.Listener); err != nil {
					if err == http.ErrServerClosed {
						slog.Info("HTTP server stopped gracefully")
					} else {
						log.Fatalf("%#+v", err)
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
			apiContext.Value("wg").(*sync.WaitGroup).Done()
			slog.Info("HTTP server stopped")
			return nil
		},
	})
	return srv
}

func NewMuxRouter(apiCtx *dto.ContextState /*, ingressClient ingress.ClientWithResponsesInterface*/) *mux.Router {
	router := mux.NewRouter()
	if apiCtx.SecureMode {
		router.Use(NewHAMiddleware( /*ingressClient*/ ))
	}

	return router
}
