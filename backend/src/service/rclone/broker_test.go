package rclone

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dianlight/srat/config"
)

// fakeBroker spins an in-process implementation of the hosted OAuth broker
// protocol (/v1/start + /v1/session/{id}) and records what SRAT sent.
type fakeBroker struct {
	srv           *httptest.Server
	startBody     brokerStartRequest
	startCalls    int
	sessionCalls  int
	authURL       string
	sessionID     string
	token         BrokerToken
	startStatus   int
	sessionStatus int
	failPayload   bool
}

func newFakeBroker(t *testing.T) *fakeBroker {
	t.Helper()
	b := &fakeBroker{
		authURL:       "https://provider.example/oauth/authorize?client_id=broker-app",
		sessionID:     "sess-1",
		startStatus:   http.StatusOK,
		sessionStatus: http.StatusOK,
		token: BrokerToken{
			TokenJSON:    `{"access_token":"bt","refresh_token":"br","expiry":"2030-01-01T00:00:00Z"}`,
			AccountLabel: "acc-broker",
			ClientID:     "broker-cid",
			ClientSecret: "broker-secret",
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/start", func(w http.ResponseWriter, r *http.Request) {
		b.startCalls++
		if err := json.NewDecoder(r.Body).Decode(&b.startBody); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(b.startStatus)
		if b.startStatus != http.StatusOK {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "provider not supported"})
			return
		}
		_ = json.NewEncoder(w).Encode(BrokerSession{AuthURL: b.authURL, SessionID: b.sessionID})
	})
	mux.HandleFunc("GET /v1/session/", func(w http.ResponseWriter, r *http.Request) {
		b.sessionCalls++
		id := strings.TrimPrefix(r.URL.Path, "/v1/session/")
		if id != b.sessionID {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(b.sessionStatus)
		if b.sessionStatus != http.StatusOK || b.failPayload {
			return
		}
		_ = json.NewEncoder(w).Encode(b.token)
	})
	b.srv = httptest.NewServer(mux)
	t.Cleanup(b.srv.Close)
	return b
}

func TestBrokerBaseURL(t *testing.T) {
	t.Setenv(BrokerBaseURLEnv, "")
	if BrokerAvailable() {
		t.Fatal("broker must be unavailable when the env is unset and no default is baked in")
	}
	t.Setenv(BrokerBaseURLEnv, "https://oauth.example.com/")
	if got := BrokerBaseURL(); got != "https://oauth.example.com" {
		t.Fatalf("trailing slash not trimmed: %q", got)
	}
	if !BrokerAvailable() {
		t.Fatal("broker must be available when the env is set")
	}
}

// TestBrokerBaseURLResolution pins the full resolution order: env override >
// baked release default > disabled; plus the case-insensitive "off" sentinel.
// withAddonOptions points the Supervisor options file at a temporary path
// for the duration of the test and returns a writer for it.
func withAddonOptions(t *testing.T) func(content string) {
	t.Helper()
	saved := config.AddonOptionsFilePath
	path := filepath.Join(t.TempDir(), "options.json")
	config.AddonOptionsFilePath = path
	t.Cleanup(func() { config.AddonOptionsFilePath = saved })
	return func(content string) {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// TestBrokerBaseURLAddonOptions pins the runtime resolution tier: when the
// env is unset/empty, the addon option srat_oauth_broker_url (from the
// Supervisor-mounted /data/options.json) wins over any baked-in default.
func TestBrokerBaseURLAddonOptions(t *testing.T) {
	writeOptions := withAddonOptions(t)
	saved := defaultBrokerURL
	t.Cleanup(func() { defaultBrokerURL = saved })
	defaultBrokerURL = "https://default.example"

	t.Setenv(BrokerBaseURLEnv, "")

	// No options file at all (plain non-addon deployment): default applies.
	if got := BrokerBaseURL(); got != "https://default.example" {
		t.Fatalf("missing file must fall through to default, got %q", got)
	}

	// Option present → runtime value beats the baked default.
	writeOptions(`{"srat_oauth_broker_url":"https://from-option.example/"}`)
	if got := BrokerBaseURL(); got != "https://from-option.example" {
		t.Fatalf("option must win over baked default, got %q", got)
	}

	// Env override still outranks the option.
	t.Setenv(BrokerBaseURLEnv, "https://env.example")
	if got := BrokerBaseURL(); got != "https://env.example" {
		t.Fatalf("env override ignored: %q", got)
	}

	// The off sentinel disables everything.
	t.Setenv(BrokerBaseURLEnv, "OFF")
	if BrokerAvailable() {
		t.Fatal("off sentinel must disable even with a configured option")
	}

	// Malformed JSON is ignored (default applies again).
	t.Setenv(BrokerBaseURLEnv, "")
	writeOptions(`{not-json`)
	if got := BrokerBaseURL(); got != "https://default.example" {
		t.Fatalf("malformed options must be ignored, got %q", got)
	}

	// Non-string values are ignored too.
	writeOptions(`{"srat_oauth_broker_url":42}`)
	if got := BrokerBaseURL(); got != "https://default.example" {
		t.Fatalf("wrong-typed option must be ignored, got %q", got)
	}
}

func TestBrokerBaseURLResolution(t *testing.T) {
	saved := defaultBrokerURL
	t.Cleanup(func() { defaultBrokerURL = saved })

	defaultBrokerURL = ""
	t.Setenv(BrokerBaseURLEnv, "")
	if BrokerAvailable() {
		t.Fatal("no env + no baked default must leave the broker unavailable")
	}

	defaultBrokerURL = "https://default.example"
	if got := BrokerBaseURL(); got != "https://default.example" {
		t.Fatalf("unset env must fall back to the baked default, got %q", got)
	}
	// An empty (or whitespace) value is equivalent to unset.
	t.Setenv(BrokerBaseURLEnv, "   ")
	if got := BrokerBaseURL(); got != "https://default.example" {
		t.Fatalf("empty env must fall back to the baked default, got %q", got)
	}

	t.Setenv(BrokerBaseURLEnv, "https://override.example/")
	if got := BrokerBaseURL(); got != "https://override.example" {
		t.Fatalf("env override ignored: %q", got)
	}

	for _, sentinel := range []string{"off", "OFF"} {
		t.Setenv(BrokerBaseURLEnv, sentinel)
		if BrokerAvailable() {
			t.Fatalf("sentinel %q must disable the hosted flow even with a baked default", sentinel)
		}
	}
}

// TestBrokerAuthHeader verifies SRAT_OAUTH_BROKER_TOKEN travels as a bearer
// header on broker calls and is omitted entirely when unset.
func TestBrokerAuthHeader(t *testing.T) {
	var authHeader string
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BrokerSession{AuthURL: "https://provider.example/a", SessionID: "s1"})
	}))
	defer srv.Close()

	t.Run("token set sends bearer", func(t *testing.T) {
		t.Setenv(BrokerTokenEnv, "tok-123")
		if _, err := BrokerStart(context.Background(), srv.URL, "dropbox", "http://cb"); err != nil {
			t.Fatal(err)
		}
		if authHeader != "Bearer tok-123" {
			t.Fatalf("Authorization = %q, want bearer tok-123", authHeader)
		}
	})

	t.Run("token unset sends nothing", func(t *testing.T) {
		t.Setenv(BrokerTokenEnv, "")
		if _, err := BrokerStart(context.Background(), srv.URL, "dropbox", "http://cb"); err != nil {
			t.Fatal(err)
		}
		if authHeader != "" {
			t.Fatalf("unexpected Authorization header %q", authHeader)
		}
	})
}

func TestBrokerStart(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path posts provider and callback url", func(t *testing.T) {
		b := newFakeBroker(t)
		callback := "http://srat.example/api/rclone/oauth/callback?state=st%3D1"
		sess, err := BrokerStart(ctx, b.srv.URL, "dropbox", callback)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sess.AuthURL != b.authURL || sess.SessionID != b.sessionID {
			t.Fatalf("unexpected session %+v", sess)
		}
		if b.startCalls != 1 {
			t.Fatalf("start calls = %d, want 1", b.startCalls)
		}
		if b.startBody.Provider != "dropbox" || b.startBody.SratCallbackURL != callback {
			t.Fatalf("unexpected start body %+v", b.startBody)
		}
	})

	t.Run("error status surfaces broker message", func(t *testing.T) {
		b := newFakeBroker(t)
		b.startStatus = http.StatusBadRequest
		_, err := BrokerStart(ctx, b.srv.URL, "nope", "http://cb")
		if err == nil || !strings.Contains(err.Error(), "provider not supported") {
			t.Fatalf("expected broker message in error, got %v", err)
		}
	})

	t.Run("invalid payload rejected", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"auth_url":""}`))
		}))
		defer srv.Close()
		if _, err := BrokerStart(ctx, srv.URL, "dropbox", "http://cb"); err == nil {
			t.Fatal("expected error for missing session fields")
		}
	})
}

func TestBrokerFetchToken(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path returns token and client credentials", func(t *testing.T) {
		b := newFakeBroker(t)
		tok, err := BrokerFetchToken(ctx, b.srv.URL, b.sessionID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tok.TokenJSON != b.token.TokenJSON || tok.ClientID != "broker-cid" || tok.ClientSecret != "broker-secret" {
			t.Fatalf("unexpected token %+v", tok)
		}
		if b.sessionCalls != 1 {
			t.Fatalf("session calls = %d, want 1", b.sessionCalls)
		}
	})

	t.Run("unknown session maps to expiry message", func(t *testing.T) {
		b := newFakeBroker(t)
		_, err := BrokerFetchToken(ctx, b.srv.URL, "other-session")
		if err == nil || !strings.Contains(err.Error(), "expired or already used") {
			t.Fatalf("expected expiry error, got %v", err)
		}
	})

	t.Run("missing token json rejected", func(t *testing.T) {
		b := newFakeBroker(t)
		b.failPayload = true
		if _, err := BrokerFetchToken(ctx, b.srv.URL, b.sessionID); err == nil {
			t.Fatal("expected error for empty token payload")
		}
	})

	t.Run("server error surfaces status", func(t *testing.T) {
		b := newFakeBroker(t)
		b.sessionStatus = http.StatusInternalServerError
		_, err := BrokerFetchToken(ctx, b.srv.URL, b.sessionID)
		if err == nil || !strings.Contains(err.Error(), "status 500") {
			t.Fatalf("expected status error, got %v", err)
		}
	})

	t.Run("session id is path escaped", func(t *testing.T) {
		b := newFakeBroker(t)
		b.sessionID = "a/b c"
		tok, err := BrokerFetchToken(ctx, b.srv.URL, b.sessionID)
		if err != nil {
			t.Fatalf("escaped session id must round-trip intact: %v", err)
		}
		if tok.TokenJSON != b.token.TokenJSON {
			t.Fatalf("unexpected token %+v", tok)
		}
	})
}
