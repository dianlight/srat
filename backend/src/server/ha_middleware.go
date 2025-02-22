package server

import (
	"context"
	"log"
	"net/http"
	"os"
)

// HAMiddleware is a middleware function for handling HomeAssistant authentication.
// It checks for the presence and validity of the X-Supervisor-Token in the request header.
//
// Parameters:
//   - next: The next http.Handler in the chain to be called if authentication is successful.
//
// Returns:
//   - http.Handler: A new http.Handler that wraps the authentication logic around the next handler.
//     If authentication fails, it returns a 401 Unauthorized status.
//     If successful, it adds the token to the request context and calls the next handler.
func HAMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("X-Supervisor-Token")
		if tokenString == "" {
			log.Printf("Not in a HomeAssistant environment!")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if tokenString != os.Getenv("SUPERVISOR_TOKEN") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "auth_token", tokenString)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
