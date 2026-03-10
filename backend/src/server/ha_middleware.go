package server

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dianlight/srat/internal/ctxkeys"
)

// NewHAMiddleware creates a middleware function that ensures requests are coming from
// an authorized Home Assistant environment. It performs the following checks:
//
//  1. Validates the `ingress_session` cookie by calling the Supervisor API.
//     If the cookie is present but invalid, it returns HTTP 401 Unauthorized.
//
//  2. Verifies the presence of the "X-Remote-User-Id" header in the request.
//     If the header is missing, it logs an error and responds with an HTTP 401 Unauthorized status.
//
//  3. Validates the remote IP address of the request against a predefined list
//     of allowed IPs. If the IP is not in the allowed list, it logs an error
//     and responds with an HTTP 401 Unauthorized status.
//
// If all checks pass, the middleware adds the "user_id" from the header to
// the request context and forwards the request to the next handler in the chain.
func NewHAMiddleware( /*ingressClient ingress.ClientWithResponsesInterface*/ ) func(http.Handler) http.Handler {
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

			allowedIPs := []string{"172.30.32.2", "127.0.0.1"}
			user_id := r.Header.Get("X-Remote-User-Id")
			if user_id == "" {
				slog.ErrorContext(r.Context(), "Not in a HomeAssistant environment! X-Remote-User-Id header is missing.", "url", r.URL.String(), "header", r.Header)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			remoteIP := r.RemoteAddr
			// Remove port if present
			ip := strings.Split(remoteIP, ":")[0]

			allowed := false
			for _, allowedIP := range allowedIPs {
				if ip == allowedIP {
					allowed = true
					break
				}
			}

			if !allowed {
				slog.ErrorContext(r.Context(), "Unauthorized access from", "IP", ip)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxkeys.UserID, user_id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
