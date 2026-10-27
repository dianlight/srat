package dto

import (
	"net"
	"net/http"
	"net/url"
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

// IsHAIngressRequest reports whether the request arrived via the Home
// Assistant Supervisor ingress proxy. HA sets X-Hass-Source: core.ingress
// and X-Ingress-Path on proxied requests (see log: both present on /ws
// ingress hits). The HA middleware already restricts SecureMode traffic to
// the Supervisor network, and the Supervisor proxy 401s invalid
// ingress_session cookies before forwarding, so ingress arrivals are trusted
// even when the browser Origin (user's HA URL: IP, hostname, Nabu Casa, …)
// is not in the static allowlist.
func IsHAIngressRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	h := r.Header
	if strings.TrimSpace(h.Get("X-Ingress-Path")) != "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(h.Get("X-Hass-Source")), "core.ingress")
}

// OriginMatchesHost reports whether an Origin value targets the same host
// the request was sent to (Host or X-Forwarded-Host). This covers HA ingress
// and direct LAN access where Origin == HA URL (e.g.
// http://192.168.0.250:8123) but no static allowlist entry exists.
func OriginMatchesHost(origin, host string) bool {
	origin = NormalizeOrigin(origin)
	host = NormalizeOrigin(host)
	if origin == "" || host == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	// Compare hostnames case-insensitively; ports must match when both are explicit.
	originHost := strings.ToLower(u.Hostname())
	reqHost := strings.ToLower(hostnameOnly(host))
	if originHost != reqHost {
		return false
	}
	originPort := u.Port()
	reqPort := portOnly(host, u.Scheme)
	if originPort == "" || reqPort == "" {
		return true
	}
	return originPort == reqPort
}

// IsOriginAllowedForRequest extends IsOriginAllowed with request-aware
// ingress and same-host fallbacks. Callers with *http.Request (WebSocket
// CheckOrigin, CORS AllowOriginRequestFunc) must use this instead of the
// origin-only check so HA ingress keeps working without enumerating every
// possible HA URL in the static allowlist.
func IsOriginAllowedForRequest(state *ContextState, r *http.Request) bool {
	if state == nil || !state.SecureMode {
		return true
	}
	var origin string
	if r != nil {
		origin = r.Header.Get("Origin")
	}
	if IsOriginAllowed(state, origin) {
		return true
	}
	if IsHAIngressRequest(r) {
		return true
	}
	if r != nil && NormalizeOrigin(origin) != "" {
		if OriginMatchesHost(origin, r.Host) {
			return true
		}
		if fwdHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); fwdHost != "" {
			// X-Forwarded-Host may be a comma-separated chain; the outermost (client-facing) entry is first.
			if first, _, _ := strings.Cut(fwdHost, ","); OriginMatchesHost(origin, strings.TrimSpace(first)) {
				return true
			}
		}
	}
	return false
}

func hostnameOnly(hostport string) string {
	if h, _, err := net.SplitHostPort(strings.TrimSpace(hostport)); err == nil {
		return h
	}
	// Bare IP or hostname without port, possibly bracketed IPv6.
	trimmed := strings.TrimSpace(hostport)
	trimmed = strings.TrimPrefix(trimmed, "[")
	if i := strings.LastIndex(trimmed, "]"); i >= 0 {
		trimmed = trimmed[:i]
	}
	// A single colon denotes host:port (not IPv6 which has 2+ colons).
	if strings.Count(trimmed, ":") == 1 {
		if h, _, ok := strings.Cut(trimmed, ":"); ok {
			return h
		}
	}
	return trimmed
}

func portOnly(hostport, scheme string) string {
	if _, p, err := net.SplitHostPort(strings.TrimSpace(hostport)); err == nil {
		return p
	}
	trimmed := strings.TrimSpace(hostport)
	if strings.Count(trimmed, ":") == 1 {
		if _, p, ok := strings.Cut(trimmed, ":"); ok {
			return strings.TrimSpace(p)
		}
	}
	return defaultPort(scheme)
}

func defaultPort(scheme string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "https", "wss":
		return "443"
	case "http", "ws":
		return "80"
	default:
		return ""
	}
}
