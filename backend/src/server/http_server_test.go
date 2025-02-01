package server_test

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.uber.org/fx"
)

func NewHTTPTestServer(lc fx.Lifecycle, mux *mux.Router) *http.Server {
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
		Addr: ":8080",
		// Good practice to set timeouts to avoid Slowloris attacks.
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
			go func() error {
				ln, err := net.Listen("tcp", srv.Addr)
				if err != nil {
					return err
				}
				slog.Debug("Starting HTTP server at", "address", srv.Addr)
				if err := srv.Serve(ln); err != nil {
					log.Fatal(err)
				}
				return nil
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
