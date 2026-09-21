package dto_test

import (
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
