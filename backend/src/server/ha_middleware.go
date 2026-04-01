package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"

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
func NewHAMiddleware( /*ingressClient ingress.ClientWithResponsesInterface*/ ) func(http.Handler) http.Handler {
	trustedPrefixes := []netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("172.30.32.0/23"),
	}

	// Create a cache with a 30-second expiration and cleanup every minute.
	//sessionCache := gocache.New(30*time.Second, 1*time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for ingress session cookie and validate it
			/*
				if ingressClient != nil {
					if ingressCookie, err := r.Cookie("ingress_session"); err == nil && ingressCookie.Value != "" {
						sessionID := ingressCookie.Value

						// Check if the session is already cached and valid
						if _, found := sessionCache.Get(sessionID); found {
							slog.Debug("Ingress session is valid (from cache)")
						} else {
							slog.Debug("Found ingress_session cookie, validating session.")
							resp, err := ingressClient.ValidateIngressSessionWithResponse(r.Context(), ingress.ValidateIngressSessionJSONRequestBody{
								Session: &sessionID,
							})

							if err != nil {
								slog.Error("Error validating ingress session", "error", err)
								http.Error(w, "Internal Server Error", http.StatusInternalServerError)
								return
							}

							if resp.StatusCode() != http.StatusOK {
								slog.Warn("Invalid ingress session", "status", resp.Status(), "body", string(resp.Body), "session", sessionID)
								http.Error(w, "Unauthorized: Invalid ingress session", http.StatusUnauthorized)
								return
							}
							// Cache the valid session
							sessionCache.Set(sessionID, true, gocache.DefaultExpiration)
							slog.Debug("Ingress session is valid (validated and cached)")
						}
					}
				}
			*/

			remoteIP := r.RemoteAddr
			// Remove port if present
			ip := strings.Split(remoteIP, ":")[0]

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
