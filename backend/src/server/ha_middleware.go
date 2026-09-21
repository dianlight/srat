package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/ctxkeys"
)

// NewHAMiddleware creates a middleware function that ensures requests are coming from
// an authorized Home Assistant environment. It performs the following checks:
//
//  1. Validates the `ingress_session` cookie by calling the Supervisor API.
//     If the cookie is present but invalid, it returns HTTP 401 Unauthorized.
//
//  2. Verifies the request originates from a trusted Home Assistant IP.
//     Requests from untrusted IPs are rejected with HTTP 401 Unauthorized.
//
//  3. Verifies the presence of the "X-Remote-User-Id" header in the request.
//     When the header is missing but the request comes from a trusted internal
//     Home Assistant IP, the middleware falls back to the synthetic
//     "homeassistant" user ID so internal add-on traffic can still be served.
//     Requests from untrusted IPs without the header are rejected with HTTP 401.
//
// If all checks pass, the middleware adds the "user_id" from the header to
// the request context and forwards the request to the next handler in the chain.
//
// Note: ingress session re-validation against POST /ingress/validate_session
// is intentionally not performed here. That endpoint requires Home Assistant
// user auth and the addon SUPERVISOR_TOKEN lacks permission (verified live on
// HAOS 2026-09-21: 401 with a valid token that returns 200 on
// /supervisor/info); the Supervisor proxy already 401s invalid
// ingress_session cookies itself.
func NewHAMiddleware(state *dto.ContextState) func(http.Handler) http.Handler {
	trustedPrefixes := defaultTrustedPrefixes(state)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r.RemoteAddr)

			allowed := false
			if addr, err := netip.ParseAddr(ip); err == nil {
				for _, prefix := range trustedPrefixes {
					if prefix.Contains(addr) {
						allowed = true
						break
					}
				}
			} else {
				slog.WarnContext(r.Context(), "Failed to parse remote IP address", "ip", ip, "error", err)
			}

			if !allowed {
				slog.ErrorContext(r.Context(), "Unauthorized access from", "IP", ip)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			user_id := r.Header.Get("X-Remote-User-Id")
			if user_id == "" {
				slog.WarnContext(r.Context(), "Trusted Home Assistant request missing X-Remote-User-Id header, defaulting to internal identity", "url", r.URL.String(), "ip", ip)
				user_id = "homeassistant"
			}

			ctx := context.WithValue(r.Context(), ctxkeys.UserID, user_id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// defaultTrustedPrefixes returns loopback plus Supervisor defaults merged with
// extra IPs/CIDRs from ContextState.SupervisorAllowedIPs (already populated
// from --supervisor-allowed-ips / SUPERVISOR_NETWORK).
func defaultTrustedPrefixes(state *dto.ContextState) []netip.Prefix {
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("172.30.32.0/23"),
	}
	if state == nil {
		return prefixes
	}
	for _, entry := range state.SupervisorAllowedIPs {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "/") {
			if prefix, err := netip.ParsePrefix(trimmed); err == nil {
				prefixes = append(prefixes, prefix.Masked())
			} else {
				slog.Warn("Ignoring invalid SupervisorAllowedIPs CIDR", "value", trimmed, "error", err)
			}
			continue
		}
		if addr, err := netip.ParseAddr(trimmed); err == nil {
			prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
		} else {
			slog.Warn("Ignoring invalid SupervisorAllowedIPs IP", "value", trimmed, "error", err)
		}
	}
	return prefixes
}

// clientIP extracts the host part of a RemoteAddr, handling host:port,
// IPv6, and bare IPs.
func clientIP(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	if ip, _, _ := strings.Cut(remoteAddr, ":"); ip != "" && strings.Count(remoteAddr, ":") == 1 {
		return ip
	}
	return remoteAddr
}
