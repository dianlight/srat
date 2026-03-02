// Package ctxkeys defines typed context keys for use with context.WithValue.
// Using an unexported type prevents key collisions between packages and
// enables static analysis to catch misuse.
package ctxkeys

// contextKey is an unexported type for context keys in this package.
// Using a named type prevents collisions with keys from other packages.
type contextKey string

const (
	// WaitGroup is the context key for the *sync.WaitGroup used to track
	// background goroutines throughout the application lifecycle.
	WaitGroup contextKey = "wg"

	// UserID is the context key for the authenticated user ID injected by
	// the HA middleware from the X-Remote-User-Id request header.
	UserID contextKey = "user_id"

	// EventUUID is the context key for the event UUID used to correlate
	// events across the event bus.
	EventUUID contextKey = "event_uuid"
)
