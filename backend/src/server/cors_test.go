package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func preflight(t *testing.T, state *dto.ContextState, origin string) *httptest.ResponseRecorder {
	t.Helper()
	return preflightWithHeaders(t, state, origin, nil)
}

func preflightWithHeaders(t *testing.T, state *dto.ContextState, origin string, extra http.Header) *httptest.ResponseRecorder {
	t.Helper()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "GET")
	for k, vv := range extra {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	rr := httptest.NewRecorder()
	newCORSHandler(state).Handler(inner).ServeHTTP(rr, req)
	return rr
}

func TestCORSDevModePermissive(t *testing.T) {
	rr := preflight(t, &dto.ContextState{SecureMode: false}, "https://evil.example")
	assert.Equal(t, "https://evil.example", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSSecureModeRestricted(t *testing.T) {
	state := &dto.ContextState{
		SecureMode:     true,
		IngressOrigin:  "http://homeassistant.local:8123",
		AllowedOrigins: []string{"https://example.com"},
	}

	allowed := preflight(t, state, "http://homeassistant.local:8123")
	assert.Equal(t, "http://homeassistant.local:8123", allowed.Header().Get("Access-Control-Allow-Origin"))

	denied := preflight(t, state, "https://evil.example")
	assert.Empty(t, denied.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSSecureModeFailClosed(t *testing.T) {
	rr := preflight(t, &dto.ContextState{SecureMode: true}, "https://evil.example")
	require.NotNil(t, rr)
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSSecureModeIngress(t *testing.T) {
	state := &dto.ContextState{SecureMode: true}
	ingress := http.Header{
		"X-Hass-Source":  []string{"core.ingress"},
		"X-Ingress-Path": []string{"/api/hassio_ingress/abc"},
	}
	rr := preflightWithHeaders(t, state, "http://192.168.0.250:8123", ingress)
	assert.Equal(t, "http://192.168.0.250:8123", rr.Header().Get("Access-Control-Allow-Origin"))
}
