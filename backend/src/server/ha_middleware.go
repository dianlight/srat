package server

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dianlight/srat/homeassistant/ingress"
)

var ingressClient *ingress.ClientWithResponses

// HAMiddleware is a middleware function that ensures requests are coming from
// an authorized Home Assistant environment. It performs the following checks:
//
//  1. Verifies the presence of the "X-Remote-User-Id" header in the request.
//     If the header is missing, it logs an error and responds with an HTTP 401 Unauthorized status.
//
//  2. Validates the remote IP address of the request against a predefined list
//     of allowed IPs. If the IP is not in the allowed list, it logs an error
//     and responds with an HTTP 401 Unauthorized status.
//
// If both checks pass, the middleware adds the "user_id" from the header to
// the request context and forwards the request to the next handler in the chain.
//
// Parameters:
// - next: The next http.Handler to be called if the request passes the middleware checks.
//
// Returns:
// - An http.Handler that wraps the provided handler with the middleware logic.
func HAMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedIPs := []string{"172.30.32.2", "127.0.0.1"}
		user_id := r.Header.Get("X-Remote-User-Id")
		if user_id == "" {
			slog.Error("Not in a HomeAssistant environment!", "header", r.Header)
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
			slog.Error("Unauthorized access from", "IP", ip)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), "user_id", user_id)
		next.ServeHTTP(w, r.WithContext(ctx))
		return

	})
}
