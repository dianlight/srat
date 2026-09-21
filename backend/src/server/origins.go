package server

import (
	"strings"

	"github.com/dianlight/srat/dto"
)

// AllowedOrigins returns the deduplicated list of trusted origins from
// ContextState (IngressOrigin plus AllowedOrigins). Empty when nothing is
// configured, which means fail-closed in SecureMode.
func AllowedOrigins(state *dto.ContextState) []string {
	if state == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(origin string) {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	add(state.IngressOrigin)
	for _, origin := range state.AllowedOrigins {
		add(origin)
	}
	return out
}

// IsOriginAllowed reports whether an Origin header value is trusted.
// Dev mode (!SecureMode or nil state) stays permissive. Empty origins
// (non-browser clients such as the HA component) are allowed so internal
// traffic is not broken; browsers always send Origin for CORS/WS.
func IsOriginAllowed(state *dto.ContextState, origin string) bool {
	if state == nil || !state.SecureMode {
		return true
	}
	if strings.TrimSpace(origin) == "" {
		return true
	}
	for _, allowed := range AllowedOrigins(state) {
		if origin == allowed {
			return true
		}
	}
	return false
}
