package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/tlog"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
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

// syncBuffer is a goroutine-safe bytes.Buffer: HTTP access logs are written
// from the server's connection goroutines while assertions run on the test
// goroutine.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// stubLifecycle captures fx hooks so the test can start the server without
// booting a full fx app. OnStop is intentionally never invoked: NewHTTPServer
// pairs wg.Go (auto-Done on return) with an extra wg.Done in OnStop, so
// running both after srv.Close would drive the counter negative.
type stubLifecycle struct {
	hooks []fx.Hook
}

func (l *stubLifecycle) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

// TestNewHTTPServerRequestBodyLogging verifies task 031 at the HTTP logging
// layer: raw request bodies (which may contain Samba user passwords before
// Huma deserializes them past the logfusc.Secret protection) must not reach
// structured logs by default, and must reappear only with SRAT_LOG_BODIES=true.
// The stub POST /user route exercises the real sloghttp middleware chain, not
// the user handler — the leak vector is the middleware.
func TestNewHTTPServerRequestBodyLogging(t *testing.T) {
	const secretPassword = "s3cr3t-hunter2-031"

	testCases := []struct {
		name         string
		logBodiesEnv string
		wantPassword bool
	}{
		{name: "body logging disabled by default", logBodiesEnv: "", wantPassword: false},
		{name: "body logging opt-in via SRAT_LOG_BODIES", logBodiesEnv: "true", wantPassword: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SRAT_LOG_BODIES", tc.logBodiesEnv)

			// Capture all slog output down to TRACE (sloghttp logs at Trace level).
			var logs syncBuffer
			oldDefault := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: tlog.LevelTrace})))
			t.Cleanup(func() { slog.SetDefault(oldDefault) })

			router := mux.NewRouter()
			// Echo stub: reads the body like a real JSON handler (Huma
			// deserialization) so sloghttp has something to capture, and
			// echoes it back to also exercise the response-body path.
			router.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(body)
			}).Methods(http.MethodPost)

			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)

			wg := &sync.WaitGroup{}
			apiCtx := context.WithValue(context.Background(), ctxkeys.WaitGroup, wg)
			lc := &stubLifecycle{}
			srv := NewHTTPServer(lc, router, listener, apiCtx, func() {})
			require.NotNil(t, srv)
			require.Len(t, lc.hooks, 1)
			require.NoError(t, lc.hooks[0].OnStart(context.Background()))

			body := fmt.Sprintf(`{"username":"admin","password":%q}`, secretPassword)
			resp, err := http.Post(fmt.Sprintf("http://%s/user", listener.Addr().String()), "application/json", strings.NewReader(body))
			require.NoError(t, err)
			_ = resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// The access log is a single deferred line written from a connection
			// goroutine after the handler returns: poll for a marker printed
			// after the body attributes (response group) so the assertion
			// cannot race a partially flushed line.
			require.Eventually(t, func() bool {
				return strings.Contains(logs.String(), "status=200")
			}, 5*time.Second, 10*time.Millisecond, "expected the request to appear in access logs")

			if tc.wantPassword {
				assert.Contains(t, logs.String(), secretPassword, "opt-in flag must re-enable body logging")
			} else {
				assert.NotContains(t, logs.String(), secretPassword, "request body must not leak credentials into logs")
			}

			require.NoError(t, srv.Close())
			wg.Wait()
		})
	}
}
