package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	sr "github.com/dianlight/srat/service/rclone"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

// ---------- fakes ----------

// fakeOAuthDriver is registered once per test-binary run so the full
// StartAuth → HandleOAuthCallback flow can be exercised offline.
type fakeOAuthDriver struct{}

func (d *fakeOAuthDriver) Name() string                   { return "fakeoauth_test" }
func (d *fakeOAuthDriver) DisplayName() string            { return "Fake OAuth Test" }
func (d *fakeOAuthDriver) ConfigFields() []sr.ConfigField { return nil }

func (d *fakeOAuthDriver) AuthStart(ctx context.Context, req sr.AuthRequest) (string, error) {
	if req.Settings["client_id"] == "" {
		return "", errors.New("missing client_id")
	}
	return "https://fake.example/authorize?client_id=" + req.Settings["client_id"], nil
}

func (d *fakeOAuthDriver) ExchangeCode(ctx context.Context, req sr.AuthRequest, code string) (*sr.TokenResult, error) {
	if code == "bad-code" {
		return nil, errors.New("invalid code")
	}
	return &sr.TokenResult{TokenJSON: `{"access_token":"tok","refresh_token":"ref"}`, AccountLabel: "acc-1"}, nil
}

var registerFakeOAuthOnce sync.Once

// fakeRcloneRPC implements sr.RcloneRPC with scripted handlers.
type fakeRcloneRPC struct {
	mu       sync.Mutex
	handlers map[string]func(input string) (string, int)
	calls    map[string]int
}

func newFakeRcloneRPC() *fakeRcloneRPC {
	return &fakeRcloneRPC{handlers: map[string]func(string) (string, int){}, calls: map[string]int{}}
}

func (f *fakeRcloneRPC) on(method string, h func(input string) (string, int)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[method] = h
}

func (f *fakeRcloneRPC) callCount(method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[method]
}

func (f *fakeRcloneRPC) Available() bool { return true }

func (f *fakeRcloneRPC) rawCall(method string, input string) (string, int) {
	f.mu.Lock()
	f.calls[method]++
	h, ok := f.handlers[method]
	f.mu.Unlock()
	if !ok {
		return "{}", http.StatusOK
	}
	return h(input)
}

func (f *fakeRcloneRPC) RPC(ctx context.Context, method string, req any, out any) error {
	rawReq, err := json.Marshal(req)
	if err != nil {
		return err
	}
	body, status := f.rawCall(method, string(rawReq))
	if status != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal([]byte(body), &e)
		if e.Error != "" {
			return errors.New(e.Error)
		}
		return errors.New("rpc failed")
	}
	if out == nil || body == "" {
		return nil
	}
	return json.Unmarshal([]byte(body), out)
}

type recordingProblemService struct {
	upserts    []*dto.Problem
	failUpsert bool
}

func (r *recordingProblemService) Upsert(problem *dto.Problem) (*dto.Problem, error) {
	if r.failUpsert {
		return nil, errors.New("problem store unavailable")
	}
	r.upserts = append(r.upserts, problem)
	return problem, nil
}
func (r *recordingProblemService) Dismiss(problemKey string) error             { return nil }
func (r *recordingProblemService) Get(problemKey string) (*dto.Problem, error) { return nil, nil }
func (r *recordingProblemService) List() ([]*dto.Problem, error)               { return r.upserts, nil }
func (r *recordingProblemService) ApplyLifecycle(problemKey string, status dto.ProblemLifecycleStatus, lastError *string) (*dto.Problem, error) {
	return nil, nil
}

// ---------- suite ----------

type RcloneServiceSuite struct {
	suite.Suite
	app        *fxtest.App
	db         *gorm.DB
	bus        events.EventBusInterface
	rc         *fakeRcloneRPC
	problems   *recordingProblemService
	service    service.RcloneServiceInterface
	taskMu     sync.Mutex
	taskEvents []events.RcloneTaskEvent
	cancel     context.CancelFunc
}

func TestRcloneServiceSuite(t *testing.T) {
	registerFakeOAuthOnce.Do(func() { sr.RegisterDriver(&fakeOAuthDriver{}) })
	suite.Run(t, new(RcloneServiceSuite))
}

func (suite *RcloneServiceSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)
	// Pin the pool to a single connection: :memory: schemas live per
	// connection, and the auto-sync goroutine issues queries concurrently
	// with the test goroutine, which would otherwise surface an empty
	// database on the second pooled connection.
	sqlDB, err := db.DB()
	suite.Require().NoError(err)
	sqlDB.SetMaxOpenConns(1)
	suite.Require().NoError(db.AutoMigrate(&dbom.RcloneLink{}))
	suite.db = db

	ctx, cancel := context.WithCancel(context.Background())
	suite.cancel = cancel
	suite.bus = events.NewEventBus(ctx)
	suite.rc = newFakeRcloneRPC()
	suite.problems = &recordingProblemService{}

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *gorm.DB { return db },
			func() context.Context { return ctx },
			func() *dto.ContextState { return &dto.ContextState{} },
			func() events.EventBusInterface { return suite.bus },
			func() sr.RcloneRPC { return suite.rc },
			func() service.ProblemServiceInterface { return suite.problems },
			service.NewRcloneService,
		),
		fx.Populate(&suite.service),
		fx.NopLogger,
	)
	suite.app.RequireStart()

	suite.taskEvents = nil
	unsub := suite.bus.OnRcloneTask(func(_ context.Context, ev events.RcloneTaskEvent) errors.E {
		suite.taskMu.Lock()
		suite.taskEvents = append(suite.taskEvents, ev)
		suite.taskMu.Unlock()
		return nil
	})
	suite.T().Cleanup(unsub)
}

func (suite *RcloneServiceSuite) TearDownTest() {
	if suite.app != nil {
		suite.app.RequireStop()
	}
	if suite.cancel != nil {
		suite.cancel()
	}
}

func (suite *RcloneServiceSuite) recordedTasks() []events.RcloneTaskEvent {
	suite.taskMu.Lock()
	defer suite.taskMu.Unlock()
	out := make([]events.RcloneTaskEvent, len(suite.taskEvents))
	copy(out, suite.taskEvents)
	return out
}

func (suite *RcloneServiceSuite) lastTaskStatus() string {
	tasks := suite.recordedTasks()
	if len(tasks) == 0 {
		return ""
	}
	last := tasks[len(tasks)-1]
	if last.Task == nil {
		return ""
	}
	return last.Task.Status
}

// saveAuthorizedLink persists a link already in authorized state.
func (suite *RcloneServiceSuite) saveAuthorizedLink(kind, id, provider string) {
	suite.NoError(suite.service.SaveLink(dto.RcloneLink{TargetKind: kind, TargetID: id, Provider: provider}))
	suite.NoError(suite.db.Model(&dbom.RcloneLink{}).
		Where("target_kind = ? AND target_id = ?", kind, id).
		Update("status", dto.RcloneStatusAuthorized).Error)
}

// ---------- providers ----------

func (suite *RcloneServiceSuite) TestLibraryAvailable() {
	suite.True(suite.service.LibraryAvailable())
}

func (suite *RcloneServiceSuite) TestListProviders_ContainsDropbox() {
	providers := suite.service.ListProviders()
	found := false
	for _, p := range providers {
		if p.Name == "dropbox" {
			found = true
			suite.Equal("Dropbox", p.DisplayName)
			suite.Len(p.ConfigFields, 2)
		}
	}
	suite.True(found, "dropbox provider expected")
}

// ---------- link CRUD ----------

const (
	testKind = dto.RcloneTargetKindVolume
	testPath = "/tmp/srat-test-vol"
)

func (suite *RcloneServiceSuite) TestSaveGetDeleteRoundtrip() {
	link, err := suite.service.GetLink(testKind, testPath)
	suite.NoError(err)
	suite.Nil(link, "absent link should be nil,nil")

	suite.NoError(suite.service.SaveLink(dto.RcloneLink{
		TargetKind: testKind, TargetID: testPath,
		Provider: "dropbox", RemotePath: "/backup", AutoSync: true, ScheduleMinutes: 30,
	}))

	link, err = suite.service.GetLink(testKind, testPath)
	suite.NoError(err)
	suite.Require().NotNil(link)
	suite.Equal("dropbox", link.Provider)
	suite.Equal("/backup", link.RemotePath)
	suite.True(link.AutoSync)
	suite.Equal(30, link.ScheduleMinutes)
	suite.Equal(dto.RcloneStatusUnlinked, link.Status)

	links, err := suite.service.ListLinks()
	suite.NoError(err)
	suite.Len(links, 1)

	// Update preserves existing status
	suite.NoError(suite.service.SaveLink(dto.RcloneLink{
		TargetKind: testKind, TargetID: testPath,
		Provider: "dropbox", RemotePath: "/backup2",
	}))
	link, _ = suite.service.GetLink(testKind, testPath)
	suite.Equal("/backup2", link.RemotePath)
	suite.Equal(dto.RcloneStatusUnlinked, link.Status)

	suite.NoError(suite.service.DeleteLink(context.Background(), testKind, testPath))
	suite.Equal(1, suite.rc.callCount("config/delete"), "managed remote must be removed")
	link, _ = suite.service.GetLink(testKind, testPath)
	suite.Nil(link)

	// Deleting again is a no-op
	suite.NoError(suite.service.DeleteLink(context.Background(), testKind, testPath))
}

func (suite *RcloneServiceSuite) TestSaveLink_Validation() {
	err := suite.service.SaveLink(dto.RcloneLink{TargetKind: "nope", TargetID: "x", Provider: "dropbox"})
	suite.Require().Error(err)
	suite.Contains(err.Error(), "validate rclone link")

	err = suite.service.SaveLink(dto.RcloneLink{TargetKind: testKind, TargetID: testPath, Provider: "ghost"})
	suite.Error(err)
}

// ---------- local paths ----------

func (suite *RcloneServiceSuite) TestLocalPath_VolumeMustBeAbsolute() {
	_, err := suite.service.(interface {
		LocalPath(string, string) (string, error)
	}).LocalPath(testKind, "relative/path")
	suite.Error(err)
}

func (suite *RcloneServiceSuite) TestLocalPath_HassosDataEnvOverride() {
	suite.T().Setenv("SRAT_HASSOS_DATA_PATH", "/tmp/fake-hassos-data")
	p, err := suite.service.(interface {
		LocalPath(string, string) (string, error)
	}).LocalPath(dto.RcloneTargetKindHassosData, "")
	suite.NoError(err)
	suite.Equal("/tmp/fake-hassos-data", p)
}

// ---------- OAuth flow ----------

func (suite *RcloneServiceSuite) TestStartAuth_LinkNotFound() {
	_, err := suite.service.StartAuth(context.Background(), testKind, "/never-saved", map[string]string{}, "", "")
	suite.Error(err)
	suite.Contains(err.Error(), "link not found")
}

func (suite *RcloneServiceSuite) TestStartAuth_BuildsURLAndStoresState() {
	suite.NoError(suite.service.SaveLink(dto.RcloneLink{TargetKind: testKind, TargetID: testPath, Provider: "dropbox"}))
	res, err := suite.service.StartAuth(context.Background(), testKind, testPath, map[string]string{"client_id": "cid", "client_secret": "sec"}, "", "")
	suite.NoError(err)
	suite.Require().NotNil(res)
	suite.Contains(res.AuthURL, "https://www.dropbox.com/oauth2/authorize")
	suite.Contains(res.AuthURL, "client_id=cid")
	suite.Contains(res.RedirectURI, "/api/rclone/oauth/callback")
	suite.NotEmpty(res.State)

	var row dbom.RcloneLink
	suite.NoError(suite.db.Where("target_kind = ? AND target_id = ?", testKind, testPath).First(&row).Error)
	suite.Equal(res.State, row.OAuthState)
	suite.Equal(dto.RcloneStatusUnlinked, row.Status)
}

func (suite *RcloneServiceSuite) TestHandleOAuthCallback_UnknownState() {
	_, err := suite.service.HandleOAuthCallback(context.Background(), "code", "bogus-state")
	suite.Error(err)
	suite.Contains(err.Error(), "unknown or expired oauth state")
}

// TestStartAuth_PublicBaseURLPrecedence verifies that the browser-supplied
// public base URL wins over the server-side default when building the OAuth
// redirect URI (required behind Home Assistant ingress, where the
// browser-visible origin differs from the addon-local address).
func (suite *RcloneServiceSuite) TestStartAuth_PublicBaseURLPrecedence() {
	suite.NoError(suite.service.SaveLink(dto.RcloneLink{TargetKind: testKind, TargetID: testPath, Provider: "dropbox"}))

	res, err := suite.service.StartAuth(context.Background(), testKind, testPath,
		map[string]string{"client_id": "cid", "client_secret": "sec"}, "http://ha.example:8123/ingress-prefix", "")
	suite.NoError(err)
	suite.Require().NotNil(res)
	suite.Equal("http://ha.example:8123/ingress-prefix/api/rclone/oauth/callback", res.RedirectURI)

	// Trailing slashes are normalized away.
	res, err = suite.service.StartAuth(context.Background(), testKind, testPath,
		map[string]string{"client_id": "cid", "client_secret": "sec"}, "http://ha.example:8123/", "")
	suite.NoError(err)
	suite.Require().NotNil(res)
	suite.Equal("http://ha.example:8123/api/rclone/oauth/callback", res.RedirectURI)

	// Empty value falls back to the configured base URL (SRAT_PORT env or
	// the localhost:8080 default).
	res, err = suite.service.StartAuth(context.Background(), testKind, testPath,
		map[string]string{"client_id": "cid", "client_secret": "sec"}, "", "")
	suite.NoError(err)
	suite.Require().NotNil(res)
	port := os.Getenv("SRAT_PORT")
	if port == "" {
		port = "8080"
	}
	suite.Equal("http://localhost:"+port+"/api/rclone/oauth/callback", res.RedirectURI)
}

func (suite *RcloneServiceSuite) TestHandleOAuthCallback_HappyPath() {
	suite.NoError(suite.service.SaveLink(dto.RcloneLink{TargetKind: testKind, TargetID: testPath, Provider: "fakeoauth_test"}))
	res, err := suite.service.StartAuth(context.Background(), testKind, testPath, map[string]string{"client_id": "cid"}, "", "")
	suite.NoError(err)

	suite.rc.on("config/create", func(input string) (string, int) {
		var m map[string]any
		suite.NoError(json.Unmarshal([]byte(input), &m))
		suite.Equal("srat_volume__tmp_srat-test-vol", m["name"])
		params := m["parameters"].(map[string]any)
		suite.Equal("fakeoauth_test", params["type"])
		suite.Contains(params["token"], "access_token")
		return "{}", http.StatusOK
	})

	link, err := suite.service.HandleOAuthCallback(context.Background(), "good-code", res.State)
	suite.NoError(err)
	suite.Require().NotNil(link)
	suite.Equal(dto.RcloneStatusAuthorized, link.Status)

	var row dbom.RcloneLink
	suite.NoError(suite.db.Where("target_kind = ? AND target_id = ?", testKind, testPath).First(&row).Error)
	suite.Empty(row.OAuthState, "state token must be cleared after use")

	// State tokens are single-use
	_, err = suite.service.HandleOAuthCallback(context.Background(), "good-code", res.State)
	suite.Error(err)
}

func (suite *RcloneServiceSuite) TestHandleOAuthCallback_ExchangeFailureMarksError() {
	suite.NoError(suite.service.SaveLink(dto.RcloneLink{TargetKind: testKind, TargetID: testPath, Provider: "fakeoauth_test"}))
	res, err := suite.service.StartAuth(context.Background(), testKind, testPath, map[string]string{"client_id": "cid"}, "", "")
	suite.NoError(err)

	_, err = suite.service.HandleOAuthCallback(context.Background(), "bad-code", res.State)
	suite.Require().Error(err)

	row, _ := suite.service.GetLink(testKind, testPath)
	suite.Equal(dto.RcloneStatusError, row.Status)
}

// ---------- hosted oauth broker (dual-mode flows) ----------

// fakeOAuthBroker is a minimal in-process double of the hosted SRAT OAuth
// broker protocol (/v1/start + /v1/session/{id}) used by the dual-mode
// StartAuth/HandleOAuthCallback tests.
type fakeOAuthBroker struct {
	srv          *httptest.Server
	startCalls   atomic.Int64
	fetchCalls   atomic.Int64
	callbackURL  string
	authURL      string
	sessionID    string
	clientID     string
	clientSecret string
	tokenJSON    string
	fetchStatus  int
}

func newFakeOAuthBroker(t *testing.T) *fakeOAuthBroker {
	t.Helper()
	b := &fakeOAuthBroker{
		authURL:      "https://oauth.example/authorize?client_id=shared-app",
		sessionID:    "sess-42",
		clientID:     "shared-cid",
		clientSecret: "shared-secret",
		tokenJSON:    `{"access_token":"bt","refresh_token":"br","expiry":"2030-01-01T00:00:00Z"}`,
		fetchStatus:  http.StatusOK,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/start", func(w http.ResponseWriter, r *http.Request) {
		b.startCalls.Add(1)
		var body struct {
			Provider        string `json:"provider"`
			SratCallbackURL string `json:"srat_callback_url"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		b.callbackURL = body.SratCallbackURL
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"auth_url": b.authURL, "session_id": b.sessionID})
	})
	mux.HandleFunc("GET /v1/session/", func(w http.ResponseWriter, _ *http.Request) {
		b.fetchCalls.Add(1)
		w.WriteHeader(b.fetchStatus)
		if b.fetchStatus != http.StatusOK {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"token_json":    b.tokenJSON,
			"account_label": "acc-shared",
			"client_id":     b.clientID,
			"client_secret": b.clientSecret,
		})
	})
	b.srv = httptest.NewServer(mux)
	t.Cleanup(b.srv.Close)
	return b
}

// TestStartAuth_BrokerModeWhenNoCredentials covers the full hosted flow:
// empty credentials + configured broker → the auth URL comes from the broker
// session and the callback fetches token plus the broker app credentials
// (librclone needs them to refresh the token offline).
func (suite *RcloneServiceSuite) TestStartAuth_BrokerModeWhenNoCredentials() {
	broker := newFakeOAuthBroker(suite.T())
	suite.T().Setenv(sr.BrokerBaseURLEnv, broker.srv.URL)
	suite.NoError(suite.service.SaveLink(dto.RcloneLink{TargetKind: testKind, TargetID: testPath, Provider: "dropbox"}))

	res, err := suite.service.StartAuth(context.Background(), testKind, testPath, map[string]string{}, "", "")
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Equal(broker.authURL, res.AuthURL, "auth url must come from the broker session")
	suite.EqualValues(1, broker.startCalls.Load())
	// The broker must redirect the browser back to SRAT's own callback
	// carrying the state SRAT generated for this flow.
	expectedCallback := res.RedirectURI + "?state=" + url.QueryEscape(res.State)
	suite.Equal(expectedCallback, broker.callbackURL)

	var gotParams map[string]any
	suite.rc.on("config/create", func(input string) (string, int) {
		var m map[string]any
		suite.Require().NoError(json.Unmarshal([]byte(input), &m))
		gotParams = m["parameters"].(map[string]any)
		return "{}", http.StatusOK
	})

	link, err := suite.service.HandleOAuthCallback(context.Background(), "", res.State)
	suite.Require().NoError(err)
	suite.Require().NotNil(link)
	suite.Equal(dto.RcloneStatusAuthorized, link.Status)
	suite.EqualValues(1, broker.fetchCalls.Load())
	suite.Contains(gotParams["token"], "access_token")
	suite.Equal("shared-cid", gotParams["client_id"], "refresh tokens are bound to the broker app client id")
	suite.Equal("shared-secret", gotParams["client_secret"])
}

// TestStartAuth_CustomAppTakesPrecedenceOverBroker pins the dual-mode rule:
// user-supplied credentials always use the driver's direct flow, even when a
// broker is configured; partial credentials also go to the driver so its own
// validation message reaches the user.
func (suite *RcloneServiceSuite) TestStartAuth_CustomAppTakesPrecedenceOverBroker() {
	broker := newFakeOAuthBroker(suite.T())
	suite.T().Setenv(sr.BrokerBaseURLEnv, broker.srv.URL)
	suite.NoError(suite.service.SaveLink(dto.RcloneLink{TargetKind: testKind, TargetID: testPath, Provider: "dropbox"}))

	res, err := suite.service.StartAuth(context.Background(), testKind, testPath,
		map[string]string{"client_id": "cid", "client_secret": "sec"}, "", "")
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Contains(res.AuthURL, "https://www.dropbox.com/oauth2/authorize")
	suite.Contains(res.AuthURL, "client_id=cid")
	suite.EqualValues(0, broker.startCalls.Load(), "broker must not be contacted in custom-app mode")

	// Partial credentials still mean "user brings their own app": the driver
	// explains what is missing instead of silently switching to the broker.
	_, err = suite.service.StartAuth(context.Background(), testKind, testPath,
		map[string]string{"client_secret": "sec"}, "", "")
	suite.Require().Error(err)
	suite.Contains(err.Error(), "App key")
	suite.EqualValues(0, broker.startCalls.Load())
}

// TestStartAuth_BrokerUnavailableRequiresCredentials preserves the original
// behavior when no broker is configured: credential-less StartAuth fails with
// the driver's guidance.
func (suite *RcloneServiceSuite) TestStartAuth_BrokerUnavailableRequiresCredentials() {
	suite.T().Setenv(sr.BrokerBaseURLEnv, "")
	suite.NoError(suite.service.SaveLink(dto.RcloneLink{TargetKind: testKind, TargetID: testPath, Provider: "dropbox"}))

	res, err := suite.service.StartAuth(context.Background(), testKind, testPath, map[string]string{}, "", "")
	suite.Require().Error(err)
	suite.Nil(res)
	suite.Contains(err.Error(), "App key")
}

// TestStartAuth_ExplicitModes pins the wizard combobox contract: an explicit
// mode always wins over inference — "broker" uses the hosted flow even when
// credentials were typed (and errors when unconfigured), "custom_app" keeps
// requiring the user's own credentials even when a broker exists.
func (suite *RcloneServiceSuite) TestStartAuth_ExplicitModes() {
	broker := newFakeOAuthBroker(suite.T())
	suite.T().Setenv(sr.BrokerBaseURLEnv, broker.srv.URL)
	suite.NoError(suite.service.SaveLink(dto.RcloneLink{TargetKind: testKind, TargetID: testPath, Provider: "dropbox"}))

	// Explicit broker wins over present credentials.
	res, err := suite.service.StartAuth(context.Background(), testKind, testPath,
		map[string]string{"client_id": "cid", "client_secret": "sec"}, "", "broker")
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Equal(broker.authURL, res.AuthURL)
	suite.EqualValues(1, broker.startCalls.Load())

	// Explicit custom app with no credentials surfaces driver guidance.
	_, err = suite.service.StartAuth(context.Background(), testKind, testPath,
		map[string]string{}, "", "custom_app")
	suite.Require().Error(err)
	suite.Contains(err.Error(), "App key")
	suite.EqualValues(1, broker.startCalls.Load(), "explicit custom_app must not touch the broker")

	// Unknown modes are rejected outright.
	_, err = suite.service.StartAuth(context.Background(), testKind, testPath,
		map[string]string{}, "", "carrier_pigeon")
	suite.Require().Error(err)
	suite.Contains(err.Error(), "unknown auth mode")

	// Explicit broker without a configured broker fails with a clear message.
	suite.T().Setenv(sr.BrokerBaseURLEnv, "")
	res, err = suite.service.StartAuth(context.Background(), testKind, testPath, map[string]string{}, "", "broker")
	suite.Require().Error(err)
	suite.Nil(res)
	suite.Contains(err.Error(), "hosted OAuth broker is not available")
}

// TestHandleOAuthCallback_BrokerFetchFailureMarksError verifies that a failed
// single-use token fetch marks the link as error, mirroring the exchange
// failure semantics of custom-app mode.
func (suite *RcloneServiceSuite) TestHandleOAuthCallback_BrokerFetchFailureMarksError() {
	broker := newFakeOAuthBroker(suite.T())
	broker.fetchStatus = http.StatusInternalServerError
	suite.T().Setenv(sr.BrokerBaseURLEnv, broker.srv.URL)
	suite.NoError(suite.service.SaveLink(dto.RcloneLink{TargetKind: testKind, TargetID: testPath, Provider: "dropbox"}))

	res, err := suite.service.StartAuth(context.Background(), testKind, testPath, map[string]string{}, "", "")
	suite.Require().NoError(err)

	_, err = suite.service.HandleOAuthCallback(context.Background(), "", res.State)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "fetch oauth token from broker")

	row, _ := suite.service.GetLink(testKind, testPath)
	suite.Equal(dto.RcloneStatusError, row.Status)
}

// ---------- diff ----------

func lsjsonBody(entries ...map[string]any) string {
	b, _ := json.Marshal(map[string]any{"list": entries})
	return string(b)
}

func entry(path string, size int64, modTime string, isDir bool) map[string]any {
	return map[string]any{"Path": path, "Size": size, "ModTime": modTime, "IsDir": isDir}
}

func (suite *RcloneServiceSuite) TestDiff_NotLinked() {
	_, err := suite.service.Diff(context.Background(), testKind, testPath)
	suite.Error(err)
	suite.Contains(err.Error(), "not linked")
}

func (suite *RcloneServiceSuite) TestDiff_CountsClasses() {
	const diffPath = "/tmp/srat-test-diff"
	suite.saveAuthorizedLink(testKind, diffPath, "dropbox")

	localFS := "srat_volume__tmp_srat-test-diff:tmp/srat-test-diff"
	suite.rc.on("operations/lsjson", func(input string) (string, int) {
		var m map[string]any
		suite.NoError(json.Unmarshal([]byte(input), &m))
		fs := m["fs"].(string)
		switch {
		case fs == diffPath:
			return lsjsonBody(
				entry("a.txt", 10, "2024-01-01T00:00:00Z", false),
				entry("c.txt", 5, "2024-01-01T00:00:00Z", false),
				entry("dir", 0, "2024-01-01T00:00:00Z", true),
			), http.StatusOK
		case fs == localFS:
			return lsjsonBody(
				entry("a.txt", 20, "2024-02-01T00:00:00Z", false),
				entry("b.txt", 7, "2024-01-01T00:00:00Z", false),
			), http.StatusOK
		default:
			return `{"error":"unexpected fs ` + fs + `"}`, http.StatusBadRequest
		}
	})

	res, err := suite.service.Diff(context.Background(), testKind, diffPath)
	suite.NoError(err)
	suite.Require().NotNil(res)
	suite.Equal(1, res.LocalOnly)
	suite.Equal(1, res.RemoteOnly)
	suite.Equal(1, res.Changed)
	suite.Len(res.Entries, 3)

	for _, e := range res.Entries {
		switch e.Path {
		case "a.txt":
			suite.Equal("changed", e.DiffType)
		case "b.txt":
			suite.Equal("remote_only", e.DiffType)
		case "c.txt":
			suite.Equal("local_only", e.DiffType)
		default:
			suite.Failf("unexpected diff entry", "%s (%s)", e.Path, e.DiffType)
		}
	}
}

// TestDiff_RemoteErrorSurfacesWarning verifies that when the remote listing
// fails but the local one succeeds, Diff still returns a (partial) result —
// every file local_only — and sets Warning so consumers know the comparison
// may be misleading. A failing remote must not be indistinguishable from an
// empty one.
func (suite *RcloneServiceSuite) TestDiff_RemoteErrorSurfacesWarning() {
	const diffPath = "/tmp/srat-test-diff-warn"
	suite.saveAuthorizedLink(testKind, diffPath, "dropbox")

	remoteFS := "srat_volume__tmp_srat-test-diff-warn:tmp/srat-test-diff-warn"
	suite.rc.on("operations/lsjson", func(input string) (string, int) {
		var m map[string]any
		suite.NoError(json.Unmarshal([]byte(input), &m))
		fs := m["fs"].(string)
		switch {
		case fs == diffPath:
			return lsjsonBody(entry("a.txt", 10, "2024-01-01T00:00:00Z", false)), http.StatusOK
		case fs == remoteFS:
			return `{"error":"directory not found"}`, http.StatusForbidden
		default:
			return `{"error":"unexpected fs ` + fs + `"}`, http.StatusBadRequest
		}
	})

	res, err := suite.service.Diff(context.Background(), testKind, diffPath)
	suite.NoError(err)
	suite.Require().NotNil(res)
	suite.Equal(1, res.LocalOnly, "all files appear local_only")
	suite.NotEmpty(res.Warning, "the remote failure must be surfaced as a warning")
	suite.Contains(res.Warning, "remote listing failed")
}

// ---------- sync ----------

func (suite *RcloneServiceSuite) installSyncScript(statusCalls *atomic.Int64) {
	suite.rc.on("sync/sync", func(string) (string, int) { return `{"jobid":42}`, http.StatusOK })
	suite.rc.on("job/status", func(string) (string, int) {
		if statusCalls.Add(1) <= 1 {
			return `{"finished":false,"success":false,"error":""}`, http.StatusOK
		}
		return `{"finished":true,"success":true,"error":""}`, http.StatusOK
	})
	suite.rc.on("core/stats", func(string) (string, int) { return `{"bytes":50,"totalBytes":100}`, http.StatusOK })
}

func (suite *RcloneServiceSuite) TestSync_NotLinked() {
	err := suite.service.Sync(testKind, testPath, dto.RcloneSyncPush, false)
	suite.Error(err)
	suite.Contains(err.Error(), "not linked")
}

func (suite *RcloneServiceSuite) TestSync_InvalidDirection() {
	suite.saveAuthorizedLink(testKind, testPath, "dropbox")
	err := suite.service.Sync(testKind, testPath, "sideways", false)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid direction")
}

func (suite *RcloneServiceSuite) TestAbortSync_Idle() {
	suite.Error(suite.service.AbortSync(testKind, testPath))
}

func (suite *RcloneServiceSuite) TestSync_PushHappyPath() {
	suite.saveAuthorizedLink(testKind, testPath, "dropbox")
	statusCalls := &atomic.Int64{}
	suite.installSyncScript(statusCalls)
	suite.rc.on("sync/sync", func(input string) (string, int) {
		var m map[string]any
		suite.NoError(json.Unmarshal([]byte(input), &m))
		suite.Equal(testPath, m["srcFs"])
		suite.Equal(true, m["_async"])
		suite.Contains(m["_group"], "srat/volume/")
		return `{"jobid":42}`, http.StatusOK
	})

	started := time.Now()
	suite.NoError(suite.service.Sync(testKind, testPath, dto.RcloneSyncPush, false))

	suite.Require().Eventually(func() bool {
		return suite.lastTaskStatus() == "success"
	}, 15*time.Second, 50*time.Millisecond, "expected a success task event")

	tasks := suite.recordedTasks()
	suite.GreaterOrEqual(len(tasks), 2, "start + finish (+running) events expected")
	first := tasks[0].Task
	suite.Equal("start", first.Status)
	suite.Equal("sync", first.Operation)
	suite.Equal(dto.RcloneSyncPush, first.Direction)
	last := tasks[len(tasks)-1].Task
	suite.Equal("success", last.Status)
	suite.Equal(100, last.Progress)
	suite.GreaterOrEqual(time.Since(started), 500*time.Millisecond, "polling should have waited at least one tick")

	link, err := suite.service.GetLink(testKind, testPath)
	suite.NoError(err)
	suite.NotNil(link.LastSyncAt)
	suite.Equal("success", link.LastSyncResult)
	suite.Equal(dto.RcloneStatusAuthorized, link.Status)
	suite.Empty(suite.problems.upserts, "no problems on success")
}

func (suite *RcloneServiceSuite) TestSync_FailureRaisesProblem() {
	suite.saveAuthorizedLink(testKind, testPath, "dropbox")
	suite.rc.on("sync/sync", func(string) (string, int) {
		return `{"error":"boom during transfer"}`, http.StatusInternalServerError
	})

	suite.NoError(suite.service.Sync(testKind, testPath, dto.RcloneSyncPush, false))

	suite.Require().Eventually(func() bool {
		return suite.lastTaskStatus() == "failure"
	}, 15*time.Second, 50*time.Millisecond, "expected a failure task event")

	link, err := suite.service.GetLink(testKind, testPath)
	suite.NoError(err)
	suite.Equal("failure", link.LastSyncResult)
	suite.Equal(dto.RcloneStatusError, link.Status)
	suite.Require().Len(suite.problems.upserts, 1)
	suite.Contains(suite.problems.upserts[0].ProblemKey, "rclone_sync_failed_volume__tmp_srat-test-vol")
	suite.NotEmpty(link.LastSyncMessage)
}

// TestSync_DryRunForwardsFlagAndSkipsBookkeeping verifies that a dry run
// passes rclone's dryRun flag through to every pass and leaves the stored
// link row completely untouched on success.
func (suite *RcloneServiceSuite) TestSync_DryRunForwardsFlagAndSkipsBookkeeping() {
	suite.saveAuthorizedLink(testKind, testPath, "dropbox")
	statusCalls := &atomic.Int64{}
	suite.installSyncScript(statusCalls)
	sawDryRun := &atomic.Bool{}
	suite.rc.on("sync/sync", func(input string) (string, int) {
		var m map[string]any
		suite.NoError(json.Unmarshal([]byte(input), &m))
		if v, ok := m["dryRun"].(bool); ok && v {
			sawDryRun.Store(true)
		}
		return `{"jobid":7}`, http.StatusOK
	})

	suite.NoError(suite.service.Sync(testKind, testPath, dto.RcloneSyncPush, true))

	suite.Require().Eventually(func() bool {
		return suite.lastTaskStatus() == "success"
	}, 15*time.Second, 50*time.Millisecond, "expected a success task event")
	suite.True(sawDryRun.Load(), "rclone request must carry dryRun=true")

	link, err := suite.service.GetLink(testKind, testPath)
	suite.NoError(err)
	suite.Nil(link.LastSyncAt, "dry run must not touch last_sync_at")
	suite.Empty(link.LastSyncResult, "dry run must not touch last_sync_result")
	suite.Equal(dto.RcloneStatusAuthorized, link.Status, "dry run must not change status")
	suite.Empty(suite.problems.upserts, "no problems for a successful dry run")
}

// TestSync_DryRunFailureSkipsProblemAndRowUpdate verifies that even a failed
// dry run is side-effect free: no problem upsert, no link-row mutation.
func (suite *RcloneServiceSuite) TestSync_DryRunFailureSkipsProblemAndRowUpdate() {
	suite.saveAuthorizedLink(testKind, testPath, "dropbox")
	suite.rc.on("sync/sync", func(string) (string, int) {
		return `{"error":"boom during transfer"}`, http.StatusInternalServerError
	})

	suite.NoError(suite.service.Sync(testKind, testPath, dto.RcloneSyncPush, true))

	suite.Require().Eventually(func() bool {
		return suite.lastTaskStatus() == "failure"
	}, 15*time.Second, 50*time.Millisecond, "expected a failure task event")

	link, err := suite.service.GetLink(testKind, testPath)
	suite.NoError(err)
	suite.Nil(link.LastSyncAt)
	suite.Empty(link.LastSyncResult)
	suite.Empty(link.LastSyncMessage)
	suite.Equal(dto.RcloneStatusAuthorized, link.Status)
	suite.Empty(suite.problems.upserts, "failed dry runs must not raise problems")
}

// TestSync_ProblemUpsertFailureIsNonFatal verifies that a failure while
// recording the sync problem (best-effort mirroring) neither panics nor
// changes the link's terminal failure state.
func (suite *RcloneServiceSuite) TestSync_ProblemUpsertFailureIsNonFatal() {
	suite.saveAuthorizedLink(testKind, testPath, "dropbox")
	suite.problems.failUpsert = true
	suite.rc.on("sync/sync", func(string) (string, int) {
		return `{"error":"transfer broke"}`, http.StatusInternalServerError
	})

	suite.NoError(suite.service.Sync(testKind, testPath, dto.RcloneSyncPush, false))

	suite.Require().Eventually(func() bool {
		return suite.lastTaskStatus() == "failure"
	}, 15*time.Second, 50*time.Millisecond, "expected a failure task event")

	link, err := suite.service.GetLink(testKind, testPath)
	suite.NoError(err)
	suite.Equal("failure", link.LastSyncResult)
	suite.Empty(suite.problems.upserts, "failed upsert must not record a problem")
}

func (suite *RcloneServiceSuite) TestSync_BidiUsesTwoCopyPasses() {
	const bidPath = "/tmp/srat-test-bidi"
	suite.saveAuthorizedLink(testKind, bidPath, "dropbox")
	copyCalls := &atomic.Int64{}
	suite.rc.on("sync/copy", func(string) (string, int) { copyCalls.Add(1); return `{"jobid":43}`, http.StatusOK })
	statusCalls := &atomic.Int64{}
	suite.rc.on("job/status", func(string) (string, int) {
		if statusCalls.Add(1)%2 == 1 { // every pass: first poll pending, second finished
			return `{"finished":false,"success":false,"error":""}`, http.StatusOK
		}
		return `{"finished":true,"success":true,"error":""}`, http.StatusOK
	})
	suite.rc.on("core/stats", func(string) (string, int) { return `{"bytes":25,"totalBytes":100}`, http.StatusOK })

	suite.NoError(suite.service.Sync(testKind, bidPath, dto.RcloneSyncBidi, false))

	suite.Require().Eventually(func() bool {
		return suite.lastTaskStatus() == "success"
	}, 20*time.Second, 50*time.Millisecond)
	suite.Equal(int64(2), copyCalls.Load(), "bidi performs two non-destructive copy passes")
}

func (suite *RcloneServiceSuite) TestSync_AbortRunningJob() {
	const abPath = "/tmp/srat-test-abort"
	suite.saveAuthorizedLink(testKind, abPath, "dropbox")
	release := make(chan struct{})
	suite.rc.on("sync/sync", func(string) (string, int) { return `{"jobid":99}`, http.StatusOK })
	suite.rc.on("job/status", func(string) (string, int) {
		<-release // block until test releases, simulating a long job
		return `{"finished":true,"success":true,"error":""}`, http.StatusOK
	})
	suite.T().Cleanup(func() { close(release) })

	suite.NoError(suite.service.Sync(testKind, abPath, dto.RcloneSyncPull, false))

	// Busy rejection while the job is parked
	suite.Eventually(func() bool {
		return suite.service.Sync(testKind, abPath, dto.RcloneSyncPull, false) != nil
	}, 5*time.Second, 50*time.Millisecond, "second sync while busy must fail")

	suite.NoError(suite.service.AbortSync(testKind, abPath))
	suite.Require().Eventually(func() bool {
		return suite.lastTaskStatus() == "failure"
	}, 15*time.Second, 50*time.Millisecond, "aborted job should end in failure")
}
