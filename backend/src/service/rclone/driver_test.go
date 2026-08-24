package rclone

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ---------- test fakes ----------

// fakeDriver is a controllable Driver used to exercise the registry.
type fakeDriver struct {
	name        string
	displayName string
	authURL     string
	token       *TokenResult
	authErr     error
	exchangeErr error
}

func (d *fakeDriver) Name() string                { return d.name }
func (d *fakeDriver) DisplayName() string         { return d.displayName }
func (d *fakeDriver) ConfigFields() []ConfigField { return nil }

func (d *fakeDriver) AuthStart(ctx context.Context, req AuthRequest) (string, error) {
	if d.authErr != nil {
		return "", d.authErr
	}
	return d.authURL + "?state=" + req.State, nil
}

func (d *fakeDriver) ExchangeCode(ctx context.Context, req AuthRequest, code string) (*TokenResult, error) {
	if d.exchangeErr != nil {
		return nil, d.exchangeErr
	}
	return d.token, nil
}

func registerTestDriver(t *testing.T, d Driver) {
	t.Helper()
	RegisterDriver(d)
	t.Cleanup(func() {
		driverMu.Lock()
		delete(driverRegistry, d.Name())
		driverMu.Unlock()
	})
}

// fakeRPC implements RcloneRPC with scripted per-method handlers.
type fakeRPC struct {
	handlers  map[string]func(input string) (string, int)
	calls     map[string]int
	available bool
}

func newFakeRPC(available bool) *fakeRPC {
	return &fakeRPC{handlers: map[string]func(string) (string, int){}, calls: map[string]int{}, available: available}
}

func (f *fakeRPC) on(method string, h func(input string) (string, int)) { f.handlers[method] = h }

func (f *fakeRPC) Available() bool { return f.available }

func (f *fakeRPC) transport(method string, input string) (string, int) {
	f.calls[method]++
	if h, ok := f.handlers[method]; ok {
		return h(input)
	}
	return "{}", http.StatusOK
}

func (f *fakeRPC) RPC(ctx context.Context, method string, req any, out any) error {
	if !f.available {
		return fmt.Errorf("unavailable")
	}
	return CallRaw(ctx, f.transport, method, req, out)
}

// inputFS extracts the "fs" field from an lsjson-style request body.
func inputFS(t *testing.T, input string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		t.Fatalf("unmarshal request %q: %v", input, err)
	}
	fs, _ := m["fs"].(string)
	return fs
}

// ---------- registry tests ----------

func TestRegistry_RegisterDuplicatePanics(t *testing.T) {
	registerTestDriver(t, &fakeDriver{name: "dup_test", displayName: "Dup"})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	RegisterDriver(&fakeDriver{name: "dup_test", displayName: "Dup2"})
}

func TestRegistry_RegisterNilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil driver")
		}
	}()
	var d Driver
	RegisterDriver(d)
}

func TestGetDriver_UnknownReturnsFalse(t *testing.T) {
	if _, ok := GetDriver("no_such_driver_xyz"); ok {
		t.Fatal("unknown driver should not be found")
	}
	if _, ok := GetDriver("dropbox"); !ok {
		t.Fatal("dropbox driver must be registered via init()")
	}
}

func TestListDrivers_SortedByDisplayName(t *testing.T) {
	registerTestDriver(t, &fakeDriver{name: "zzz_test", displayName: "Zebra"})
	registerTestDriver(t, &fakeDriver{name: "aaa_test", displayName: "Aardvark"})
	drivers := ListDrivers()
	if len(drivers) < 3 {
		t.Fatalf("expected at least 3 drivers (dropbox + 2 fakes), got %d", len(drivers))
	}
	for i := 1; i < len(drivers); i++ {
		if drivers[i-1].DisplayName() > drivers[i].DisplayName() {
			t.Fatalf("drivers not sorted: %q before %q", drivers[i-1].DisplayName(), drivers[i].DisplayName())
		}
	}
}

// ---------- dropbox driver tests ----------

func TestDropbox_ConfigFields(t *testing.T) {
	d, ok := GetDriver("dropbox")
	if !ok {
		t.Fatal("dropbox driver missing")
	}
	fields := d.ConfigFields()
	if len(fields) != 2 {
		t.Fatalf("expected 2 config fields, got %d", len(fields))
	}
	if fields[0].Name != fieldClientID || fields[1].Name != fieldClientSecret {
		t.Fatalf("unexpected field names: %v, %v", fields[0].Name, fields[1].Name)
	}
	if !fields[1].Secret {
		t.Fatal("client_secret must be marked secret")
	}
}

func TestDropbox_AuthStart_URLContents(t *testing.T) {
	d := &dropboxDriver{}
	url, err := d.AuthStart(context.Background(), AuthRequest{
		RedirectURI: "http://localhost:8080/api/rclone/oauth/callback",
		State:       "st4te",
		Settings:    map[string]string{"client_id": "cid", "client_secret": "sec"},
	})
	if err != nil {
		t.Fatalf("AuthStart: %v", err)
	}
	for _, want := range []string{
		dropboxAuthorizeURL + "?",
		"client_id=cid",
		"response_type=code",
		"token_access_type=offline",
		"state=st4te",
		"redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fapi%2Frclone%2Foauth%2Fcallback",
	} {
		if !strings.Contains(url, want) {
			t.Errorf("auth url %q missing %q", url, want)
		}
	}
}

func TestDropbox_AuthStart_MissingSettings(t *testing.T) {
	d := &dropboxDriver{}
	if _, err := d.AuthStart(context.Background(), AuthRequest{Settings: map[string]string{"client_id": "only"}}); err == nil {
		t.Fatal("expected error when client_secret missing")
	}
}

func TestDropbox_ExchangeCode_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "authorization_code" {
			t.Errorf("grant_type = %q", got)
		}
		if got := r.Form.Get("code"); got != "the-code" {
			t.Errorf("code = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","token_type":"bearer","refresh_token":"rt","expires_in":14400,"account_id":"dbid:X"}`))
	}))
	defer srv.Close()

	old := dropboxTokenURL
	dropboxTokenURL = srv.URL
	defer func() { dropboxTokenURL = old }()

	d := &dropboxDriver{}
	res, err := d.ExchangeCode(context.Background(), AuthRequest{
		RedirectURI: "http://cb",
		Settings:    map[string]string{"client_id": "cid", "client_secret": "sec"},
	}, "the-code")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if res.AccountLabel != "dbid:X" {
		t.Errorf("account label = %q", res.AccountLabel)
	}
	var token map[string]any
	if err := json.Unmarshal([]byte(res.TokenJSON), &token); err != nil {
		t.Fatalf("token json invalid: %v", err)
	}
	if token["access_token"] != "at" || token["refresh_token"] != "rt" {
		t.Errorf("token payload wrong: %v", token)
	}
	expiry, err := time.Parse(time.RFC3339, token["expiry"].(string))
	if err != nil {
		t.Fatalf("expiry not RFC3339: %v", err)
	}
	if time.Until(expiry) <= 0 {
		t.Error("expiry should be in the future")
	}
}

func TestDropbox_ExchangeCode_MissingSettings(t *testing.T) {
	d := &dropboxDriver{}
	if _, err := d.ExchangeCode(context.Background(), AuthRequest{}, "c"); err == nil {
		t.Fatal("expected error without client settings")
	}
}

func TestDropbox_ExchangeCode_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()
	old := dropboxTokenURL
	dropboxTokenURL = srv.URL
	defer func() { dropboxTokenURL = old }()

	d := &dropboxDriver{}
	if _, err := d.ExchangeCode(context.Background(), AuthRequest{
		RedirectURI: "http://cb",
		Settings:    map[string]string{"client_id": "cid", "client_secret": "sec"},
	}, "bad"); err == nil {
		t.Fatal("expected error for 400 response")
	}
}

// ---------- RPC seam tests ----------

func TestCallRaw_DecodesSuccessResponse(t *testing.T) {
	var gotMethod, gotInput string
	transport := func(method string, input string) (string, int) {
		gotMethod, gotInput = method, input
		return `{"version":"1.67.0"}`, http.StatusOK
	}
	var out struct {
		Version string `json:"version"`
	}
	if err := CallRaw(context.Background(), transport, "core/version", map[string]any{"a": 1}, &out); err != nil {
		t.Fatalf("CallRaw: %v", err)
	}
	if out.Version != "1.67.0" {
		t.Errorf("version = %q", out.Version)
	}
	if gotMethod != "core/version" || !strings.Contains(gotInput, `"a":1`) {
		t.Errorf("request not forwarded: %q %q", gotMethod, gotInput)
	}
}

func TestCallRaw_NilOutSkipsDecode(t *testing.T) {
	transport := func(string, string) (string, int) { return ``, http.StatusOK }
	if err := CallRaw(context.Background(), transport, "config/delete", map[string]any{}, nil); err != nil {
		t.Fatalf("CallRaw with nil out: %v", err)
	}
}

func TestCallRaw_ErrorBodyBecomesError(t *testing.T) {
	transport := func(string, string) (string, int) {
		return `{"error":"directory not found"}`, http.StatusNotFound
	}
	err := CallRaw(context.Background(), transport, "operations/lsjson", map[string]any{}, &map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "operations/lsjson") || !strings.Contains(err.Error(), "directory not found") {
		t.Fatalf("expected descriptive error, got %v", err)
	}
}

func TestCallRaw_ErrorStatusWithoutErrorBody(t *testing.T) {
	transport := func(string, string) (string, int) { return `not json`, http.StatusInternalServerError }
	err := CallRaw(context.Background(), transport, "job/status", map[string]any{}, &map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestCall_UnavailableBackend(t *testing.T) {
	rc := newFakeRPC(false)
	if err := Call(context.Background(), rc, "core/version", map[string]any{}, &map[string]any{}); err == nil {
		t.Fatal("expected unavailable-backend error")
	}
}

// Compile-time check that the fake satisfies the exported seam.
var _ RcloneRPC = (*fakeRPC)(nil)

// newSyncScriptedRPC simulates the async job lifecycle of sync/sync +
// job/status + core/stats. The first status poll reports running, the second
// reports finished success.
func newSyncScriptedRPC(statusCalls *atomic.Int64) *fakeRPC {
	rc := newFakeRPC(true)
	rc.on("sync/sync", func(input string) (string, int) {
		return `{"jobid":42}`, http.StatusOK
	})
	rc.on("sync/copy", func(input string) (string, int) {
		return `{"jobid":43}`, http.StatusOK
	})
	rc.on("job/status", func(string) (string, int) {
		n := statusCalls.Add(1)
		if n <= 1 {
			return `{"finished":false,"success":false,"error":""}`, http.StatusOK
		}
		return `{"finished":true,"success":true,"error":""}`, http.StatusOK
	})
	rc.on("core/stats", func(string) (string, int) {
		return `{"bytes":50,"totalBytes":100}`, http.StatusOK
	})
	return rc
}
