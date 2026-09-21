package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestNewMuxRouter(t *testing.T) {
	apiCtx := &dto.ContextState{
		SecureMode: false,
	}

	// Create a minimal WebSocketHandler for testing
	// We can pass nil since we're just testing router creation
	router := NewMuxRouter(apiCtx, nil)

	assert.NotNil(t, router)
	assert.IsType(t, &mux.Router{}, router)
}

func TestNewMuxRouterWithSecureMode(t *testing.T) {
	apiCtx := &dto.ContextState{
		SecureMode: true,
	}

	router := NewMuxRouter(apiCtx, nil)

	assert.NotNil(t, router)
	assert.IsType(t, &mux.Router{}, router)
}

func TestNewMuxRouterWithoutSecureMode(t *testing.T) {
	apiCtx := &dto.ContextState{
		SecureMode: false,
	}

	router := NewMuxRouter(apiCtx, nil)

	assert.NotNil(t, router)
	assert.IsType(t, &mux.Router{}, router)
}

type fakeLifecycle struct {
	hooks []fx.Hook
}

func (f *fakeLifecycle) Append(h fx.Hook) { f.hooks = append(f.hooks, h) }

func TestNewHTTPServerLifecycle(t *testing.T) {
	router := mux.NewRouter()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	wg := &sync.WaitGroup{}
	apiCtx, cancel := context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, wg))

	lc := &fakeLifecycle{}
	srv := NewHTTPServer(lc, router, listener, apiCtx, cancel, &dto.ContextState{})
	require.NotNil(t, srv)
	require.Len(t, lc.hooks, 1)
	require.NotNil(t, lc.hooks[0].OnStart)
	require.NotNil(t, lc.hooks[0].OnStop)

	// Balance the WaitGroup accounting (OnStart would Add via wg.Go).
	wg.Add(1)
	require.NoError(t, lc.hooks[0].OnStop(context.Background()))
	select {
	case <-apiCtx.Done():
	default:
		assert.Fail(t, "OnStop must cancel the server context")
	}
}

func TestNewHTTPServerCORSEnforcement(t *testing.T) {
	for _, tc := range []struct {
		name       string
		state      *dto.ContextState
		origin     string
		wantHeader bool
	}{
		{"dev permissive", &dto.ContextState{SecureMode: false}, "https://evil.example", true},
		{"secure allowed", &dto.ContextState{SecureMode: true, IngressOrigin: "http://homeassistant.local:8123"}, "http://homeassistant.local:8123", true},
		{"secure denied", &dto.ContextState{SecureMode: true, IngressOrigin: "http://homeassistant.local:8123"}, "https://evil.example", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := mux.NewRouter()
			router.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer listener.Close()

			wg := &sync.WaitGroup{}
			apiCtx, cancel := context.WithCancel(context.WithValue(context.Background(), ctxkeys.WaitGroup, wg))
			defer cancel()

			var lc fx.Lifecycle
			app := fxtest.New(t, fx.Invoke(func(l fx.Lifecycle) { lc = l }))
			defer app.RequireStop()
			require.NotNil(t, lc)

			srv := NewHTTPServer(lc, router, listener, apiCtx, cancel, tc.state)
			require.NotNil(t, srv)
			require.NotNil(t, srv.Handler)

			req := httptest.NewRequest(http.MethodOptions, "/", nil)
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Access-Control-Request-Method", "GET")
			rr := httptest.NewRecorder()
			srv.Handler.ServeHTTP(rr, req)
			if tc.wantHeader {
				assert.Equal(t, tc.origin, rr.Header().Get("Access-Control-Allow-Origin"))
			} else {
				assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}
