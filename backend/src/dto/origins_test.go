package dto_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeOrigin(t *testing.T) {
	assert.Equal(t, "https://example.com", dto.NormalizeOrigin("  https://example.com  "))
	assert.Empty(t, dto.NormalizeOrigin("   "))
}

func TestTrustedOrigins(t *testing.T) {
	assert.Nil(t, dto.TrustedOrigins(nil))
	assert.Empty(t, dto.TrustedOrigins(&dto.ContextState{}))
	assert.Equal(t,
		[]string{"http://homeassistant.local:8123", "https://example.com"},
		dto.TrustedOrigins(&dto.ContextState{
			IngressOrigin:  " http://homeassistant.local:8123 ",
			AllowedOrigins: []string{"https://example.com", "", "https://example.com"},
		}),
	)
}

func TestDtoIsOriginAllowed(t *testing.T) {
	assert.True(t, dto.IsOriginAllowed(nil, "https://evil.example"))
	assert.True(t, dto.IsOriginAllowed(&dto.ContextState{}, "https://evil.example"))

	secure := &dto.ContextState{
		SecureMode:     true,
		IngressOrigin:  "  http://homeassistant.local:8123  ",
		AllowedOrigins: []string{"https://example.com"},
	}
	assert.True(t, dto.IsOriginAllowed(secure, "http://homeassistant.local:8123"))
	assert.True(t, dto.IsOriginAllowed(secure, "  https://example.com  "))
	assert.True(t, dto.IsOriginAllowed(secure, ""))
	assert.False(t, dto.IsOriginAllowed(secure, "https://evil.example"))
	assert.False(t, dto.IsOriginAllowed(&dto.ContextState{SecureMode: true}, "https://evil.example"))
}

func TestIsHAIngressRequest(t *testing.T) {
	assert.False(t, dto.IsHAIngressRequest(nil))
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	assert.False(t, dto.IsHAIngressRequest(req))
	req.Header.Set("X-Ingress-Path", "/api/hassio_ingress/IVi3D5gB5yvVwQVSU6sJEa8CgEoFrSuLZ58liwJy3v0")
	assert.True(t, dto.IsHAIngressRequest(req))

	req2 := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req2.Header.Set("X-Hass-Source", "core.ingress")
	assert.True(t, dto.IsHAIngressRequest(req2))
}

func TestOriginMatchesHost(t *testing.T) {
	assert.True(t, dto.OriginMatchesHost("http://192.168.0.250:8123", "192.168.0.250:8123"))
	assert.True(t, dto.OriginMatchesHost("http://homeassistant.local:8123", "homeassistant.local:8123"))
	assert.False(t, dto.OriginMatchesHost("https://evil.example", "192.168.0.250:8123"))
	assert.False(t, dto.OriginMatchesHost("", "192.168.0.250:8123"))
	assert.False(t, dto.OriginMatchesHost("http://192.168.0.250:8123", ""))
	// Hostname comparison is case-insensitive; default ports match explicit ones.
	assert.True(t, dto.OriginMatchesHost("http://HomeAssistant.local:8123", "homeassistant.local:8123"))
	assert.True(t, dto.OriginMatchesHost("https://example.com", "example.com"))
	assert.True(t, dto.OriginMatchesHost("http://example.com:80", "example.com"))
	assert.False(t, dto.OriginMatchesHost("http://example.com:8123", "example.com:8124"))
	// IPv6 and malformed inputs.
	assert.True(t, dto.OriginMatchesHost("http://[::1]:8123", "[::1]:8123"))
	assert.False(t, dto.OriginMatchesHost("::not-a-url", "example.com"))
}

func TestIsOriginAllowedForRequestIngress(t *testing.T) {
	// Regression: HA ingress serves the UI from a user-specific URL
	// (IP, hostname, Nabu Casa) that cannot be enumerated in the static
	// allowlist. Requests carry X-Hass-Source/X-Ingress-Path and
	// Origin == Host.
	secure := &dto.ContextState{SecureMode: true}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Host = "192.168.0.250:8123"
	req.Header.Set("Origin", "http://192.168.0.250:8123")
	req.Header.Set("X-Hass-Source", "core.ingress")
	req.Header.Set("X-Ingress-Path", "/api/hassio_ingress/IVi3D5gB5yvVwQVSU6sJEa8CgEoFrSuLZ58liwJy3v0")
	req.Header.Set("X-Forwarded-Host", "192.168.0.250:8123")
	assert.True(t, dto.IsOriginAllowedForRequest(secure, req))

	// Same-host without ingress headers also passes.
	sameHost := httptest.NewRequest(http.MethodGet, "/ws", nil)
	sameHost.Host = "192.168.0.250:8123"
	sameHost.Header.Set("Origin", "http://192.168.0.250:8123")
	assert.True(t, dto.IsOriginAllowedForRequest(secure, sameHost))

	// Cross-origin without ingress headers still fails closed.
	evil := httptest.NewRequest(http.MethodGet, "/ws", nil)
	evil.Host = "192.168.0.250:8123"
	evil.Header.Set("Origin", "https://evil.example")
	assert.False(t, dto.IsOriginAllowedForRequest(secure, evil))
}
