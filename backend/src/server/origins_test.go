package server

import (
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestAllowedOrigins(t *testing.T) {
	assert.Nil(t, AllowedOrigins(nil))
	assert.Empty(t, AllowedOrigins(&dto.ContextState{}))

	state := &dto.ContextState{
		IngressOrigin:  "http://homeassistant.local:8123",
		AllowedOrigins: []string{" http://homeassistant.local:8123 ", "https://example.com", "", "https://example.com"},
	}
	assert.Equal(t,
		[]string{"http://homeassistant.local:8123", "https://example.com"},
		AllowedOrigins(state),
	)
}

func TestIsOriginAllowed(t *testing.T) {
	// Dev mode stays permissive.
	assert.True(t, IsOriginAllowed(nil, "https://evil.example"))
	assert.True(t, IsOriginAllowed(&dto.ContextState{SecureMode: false}, "https://evil.example"))

	secure := &dto.ContextState{
		SecureMode:     true,
		IngressOrigin:  "http://homeassistant.local:8123",
		AllowedOrigins: []string{"https://example.com"},
	}
	assert.True(t, IsOriginAllowed(secure, "http://homeassistant.local:8123"))
	assert.True(t, IsOriginAllowed(secure, "https://example.com"))
	// Non-browser clients omit Origin.
	assert.True(t, IsOriginAllowed(secure, ""))
	assert.False(t, IsOriginAllowed(secure, "https://evil.example"))

	// SecureMode with no configured origins fails closed for browser origins.
	failClosed := &dto.ContextState{SecureMode: true}
	assert.False(t, IsOriginAllowed(failClosed, "https://evil.example"))
	assert.True(t, IsOriginAllowed(failClosed, ""))
}

func TestDefaultTrustedPrefixes_CustomNetwork(t *testing.T) {
	prefixes := defaultTrustedPrefixes(&dto.ContextState{
		SupervisorAllowedIPs: []string{"192.168.1.0/24", "10.0.0.5", "  ", "not-an-ip"},
	})
	// 3 defaults + CIDR + single IP; invalid entries ignored.
	assert.Len(t, prefixes, 5)
}

func TestClientIP(t *testing.T) {
	assert.Equal(t, "127.0.0.1", clientIP("127.0.0.1:12345"))
	assert.Equal(t, "172.30.32.2", clientIP("172.30.32.2:8080"))
	assert.Equal(t, "::1", clientIP("[::1]:8080"))
	assert.Equal(t, "10.0.0.5", clientIP("10.0.0.5"))
}
