package rclone

import (
	"context"
	"sort"
	"sync"
)

// ConfigField describes one piece of provider-specific configuration the
// frontend wizard must collect from the user (e.g. Dropbox app key).
type ConfigField struct {
	Name        string `json:"name"`                  // machine name, e.g. "client_id"
	Label       string `json:"label"`                 // human label for the form
	Description string `json:"description,omitempty"` // helper text
	Secret      bool   `json:"secret,omitempty"`      // render as password input
	Required    bool   `json:"required,omitempty"`
}

// AuthRequest carries everything a driver needs to start an OAuth flow.
type AuthRequest struct {
	// RedirectURI is SRAT's own callback endpoint (absolute URL), reachable
	// through Home Assistant ingress as well because the browser resolves it.
	RedirectURI string
	// State is an opaque anti-CSRF value; SRAT correlates it to the pending
	// link and validates it on callback.
	State string
	// Settings holds user-provided provider credentials (app key/secret…),
	// keyed by ConfigField.Name.
	Settings map[string]string
	// TargetKind/TargetID identify the link this flow belongs to, so the
	// callback can reload the row by primary key without extra correlation
	// columns.
	TargetKind string
	TargetID   string
}

// TokenResult is the outcome of exchanging the OAuth code. TokenJSON must be
// the exact JSON document rclone expects in its remote config "token"
// parameter (provider-shaped access/refresh token envelope).
type TokenResult struct {
	TokenJSON string
	// AccountLabel is a friendly account identifier shown in the UI when the
	// provider can cheaply provide one (e.g. Dropbox account id); optional.
	AccountLabel string
}

// Driver abstracts one cloud provider so new providers can be added without
// touching service or UI logic. Implementations register themselves via
// RegisterDriver (init()) and are looked up by name from link records.
type Driver interface {
	// Name is the rclone backend identifier, e.g. "dropbox".
	Name() string
	// DisplayName is the human-facing provider name.
	DisplayName() string
	// ConfigFields declares what the wizard collects beyond target/path.
	ConfigFields() []ConfigField
	// AuthStart builds the provider authorization URL for the browser.
	AuthStart(ctx context.Context, req AuthRequest) (string, error)
	// ExchangeCode trades the OAuth code for an rclone-compatible token.
	ExchangeCode(ctx context.Context, req AuthRequest, code string) (*TokenResult, error)
}

var (
	driverMu       sync.RWMutex
	driverRegistry = map[string]Driver{}
)

// RegisterDriver adds a provider implementation to the registry. Panics on
// duplicates or nil — a programming error caught at startup.
func RegisterDriver(d Driver) {
	if d == nil || d.Name() == "" {
		panic("rclone: RegisterDriver called with nil driver or empty name")
	}
	driverMu.Lock()
	defer driverMu.Unlock()
	if _, dup := driverRegistry[d.Name()]; dup {
		panic("rclone: duplicate driver registration for " + d.Name())
	}
	driverRegistry[d.Name()] = d
}

// GetDriver returns the registered provider by name (ok=false if unknown).
func GetDriver(name string) (Driver, bool) {
	driverMu.RLock()
	defer driverMu.RUnlock()
	d, ok := driverRegistry[name]
	return d, ok
}

// ListDrivers returns all registered providers sorted by display name for a
// stable UI ordering.
func ListDrivers() []Driver {
	driverMu.RLock()
	defer driverMu.RUnlock()
	out := make([]Driver, 0, len(driverRegistry))
	for _, d := range driverRegistry {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DisplayName() < out[j].DisplayName() })
	return out
}
