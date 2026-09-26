package dto

import (
	"strings"
)

// NormalizeOrigin trims surrounding whitespace from an origin value so
// independently configured sources (flags, env vars, addon options) compare
// consistently across the CORS and WebSocket enforcement points.
func NormalizeOrigin(origin string) string {
	return strings.TrimSpace(origin)
}

// TrustedOrigins returns the deduplicated list of configured trusted origins
// (IngressOrigin plus AllowedOrigins). Empty when nothing is configured,
// which means fail-closed in SecureMode.
func TrustedOrigins(state *ContextState) []string {
	if state == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(origin string) {
		trimmed := NormalizeOrigin(origin)
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
func IsOriginAllowed(state *ContextState, origin string) bool {
	if state == nil || !state.SecureMode {
		return true
	}
	if NormalizeOrigin(origin) == "" {
		return true
	}
	normalized := NormalizeOrigin(origin)
	for _, allowed := range TrustedOrigins(state) {
		if normalized == allowed {
			return true
		}
	}
	return false
}
