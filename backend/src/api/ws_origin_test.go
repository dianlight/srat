package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func newWSBrokerForOriginTest(t *testing.T, state *dto.ContextState) *api.WebSocketHandler {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	// Broadcaster/repair/problem/HA services are not touched during upgrade.
	return api.NewWebSocketBroker(api.WebSocketHandlerParams{
		Ctx:   ctx,
		State: state,
	})
}

func wsUpgradeCode(t *testing.T, broker *api.WebSocketHandler, origin string) int {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(broker.HandleWebSocket))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	dialer := websocket.Dialer{}
	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	conn, resp, err := dialer.Dial(wsURL, header)
	if err == nil {
		conn.Close()
		require.NotNil(t, resp)
		return resp.StatusCode
	}
	if resp != nil {
		return resp.StatusCode
	}
	require.Fail(t, "dial failed without response", "err: %v", err)
	return 0
}

func TestWebSocketOriginSecureMode(t *testing.T) {
	secure := &dto.ContextState{
		SecureMode:     true,
		IngressOrigin:  "http://homeassistant.local:8123",
		AllowedOrigins: []string{"https://example.com"},
	}
	broker := newWSBrokerForOriginTest(t, secure)

	require.Equal(t, http.StatusForbidden, wsUpgradeCode(t, broker, "https://evil.example"))
	require.NotEqual(t, http.StatusForbidden, wsUpgradeCode(t, broker, "http://homeassistant.local:8123"))
	require.NotEqual(t, http.StatusForbidden, wsUpgradeCode(t, broker, "https://example.com"))
	// Non-browser clients without Origin still connect.
	require.NotEqual(t, http.StatusForbidden, wsUpgradeCode(t, broker, ""))
}

func TestWebSocketOriginDevModePermissive(t *testing.T) {
	broker := newWSBrokerForOriginTest(t, &dto.ContextState{SecureMode: false})
	require.NotEqual(t, http.StatusForbidden, wsUpgradeCode(t, broker, "https://evil.example"))
}

func TestWebSocketOriginNormalization(t *testing.T) {
	// Whitespace-padded configured origins must match clean request origins,
	// converging with the CORS normalization in dto.TrustedOrigins.
	padded := &dto.ContextState{
		SecureMode:     true,
		IngressOrigin:  "  http://homeassistant.local:8123  ",
		AllowedOrigins: []string{"  https://example.com  "},
	}
	broker := newWSBrokerForOriginTest(t, padded)

	require.NotEqual(t, http.StatusForbidden, wsUpgradeCode(t, broker, "http://homeassistant.local:8123"))
	require.NotEqual(t, http.StatusForbidden, wsUpgradeCode(t, broker, "https://example.com"))
	require.Equal(t, http.StatusForbidden, wsUpgradeCode(t, broker, "https://evil.example"))
}
