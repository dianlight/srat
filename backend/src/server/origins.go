package server

import (
	"github.com/dianlight/srat/dto"
)

// AllowedOrigins returns the deduplicated list of trusted origins from
// ContextState (IngressOrigin plus AllowedOrigins). Empty when nothing is
// configured, which means fail-closed in SecureMode.
func AllowedOrigins(state *dto.ContextState) []string {
	return dto.TrustedOrigins(state)
}

// IsOriginAllowed reports whether an Origin header value is trusted.
// Dev mode (!SecureMode or nil state) stays permissive. Empty origins
// (non-browser clients such as the HA component) are allowed so internal
// traffic is not broken; browsers always send Origin for CORS/WS.
func IsOriginAllowed(state *dto.ContextState, origin string) bool {
	return dto.IsOriginAllowed(state, origin)
}
