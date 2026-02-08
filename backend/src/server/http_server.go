package server

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/tlog"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	sloghttp "github.com/samber/slog-http"
	"go.uber.org/fx"
)

func NewHTTPServer(
	lc fx.Lifecycle,
	mux *mux.Router,
	listener net.Listener,
	apiContext context.Context,
	cxtClose context.CancelFunc,
) *http.Server {
	sloghttp.RequestIDKey = "X-Request-Id"
	sloghttp.SpanIDKey = "X-Span-Id"
	sloghttp.TraceIDKey = "X-Trace-Id"
	handler := sloghttp.NewWithConfig(slog.Default(), sloghttp.Config{
		DefaultLevel:       tlog.LevelTrace,
		WithRequestBody:    false,
		WithRequestHeader:  true,
		WithResponseBody:   false,
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
				slog.Debug("Starting HTTP server at", "listener", listener.Addr(), "pid", os.Getpid())
				if err := srv.Serve(listener); err != nil {
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
			slog.InfoContext(ctx, "Stopping HTTP server")
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
			slog.InfoContext(ctx, "HTTP server stopped")
			return nil
		},
	})
	return srv
}

func NewMuxRouter(apiCtx *dto.ContextState, wsh *api.WebSocketHandler) *mux.Router {
	router := mux.NewRouter()
	if apiCtx.SecureMode {
		router.Use(NewHAMiddleware( /*ingressClient*/ ))
	}

	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	wsh.RegisterWs(router)
	return router
}
