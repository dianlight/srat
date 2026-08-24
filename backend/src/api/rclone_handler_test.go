package api_test

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	errors "gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// RcloneHandlerSuite exercises the lab-gated rclone HTTP endpoints through
// humatest with mocked services (issue #954).
type RcloneHandlerSuite struct {
	suite.Suite
	app        *fxtest.App
	handler    *api.RcloneHandler
	rcloneSvc  service.RcloneServiceInterface
	settingSvc service.SettingServiceInterface
	ctrl       *matchers.MockController
	ctx        context.Context
	cancel     context.CancelFunc
}

func TestRcloneHandlerSuite(t *testing.T) {
	suite.Run(t, new(RcloneHandlerSuite))
}

func (suite *RcloneHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			api.NewRcloneHandler,
			mock.Mock[service.RcloneServiceInterface],
			mock.Mock[service.SettingServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.rcloneSvc),
		fx.Populate(&suite.settingSvc),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
		fx.Populate(&suite.ctrl),
	)
	suite.app.RequireStart()
}

func (suite *RcloneHandlerSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
	}
	if suite.ctx != nil {
		if wg := suite.ctx.Value("wg"); wg != nil {
			wg.(*sync.WaitGroup).Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

// labMode configures the settings mock for the given mode and returns the
// registered test API.
func (suite *RcloneHandlerSuite) labMode(enabled bool) humatest.TestAPI {
	if enabled {
		mock.When(suite.settingSvc.Load()).ThenReturn(&dto.Settings{ExperimentalLabMode: true}, nil)
	} else {
		mock.When(suite.settingSvc.Load()).ThenReturn(&dto.Settings{}, nil)
	}
	_, testAPI := humatest.New(suite.T())
	suite.handler.RegisterRcloneHandler(testAPI)
	return testAPI
}

func (suite *RcloneHandlerSuite) TestProviders_LabModeRequired() {
	testAPI := suite.labMode(false)
	resp := testAPI.Get("/rclone/providers")
	suite.Equal(http.StatusForbidden, resp.Code)
}

func (suite *RcloneHandlerSuite) TestProviders_LoadError() {
	mock.When(suite.settingSvc.Load()).ThenReturn(nil, errors.New("boom"))
	_, testAPI := humatest.New(suite.T())
	suite.handler.RegisterRcloneHandler(testAPI)
	resp := testAPI.Get("/rclone/providers")
	suite.Equal(http.StatusInternalServerError, resp.Code)
}

func (suite *RcloneHandlerSuite) TestProviders_OK() {
	testAPI := suite.labMode(true)
	mock.When(suite.rcloneSvc.ListProviders()).ThenReturn([]dto.RcloneProviderInfo{})
	mock.When(suite.rcloneSvc.LibraryAvailable()).ThenReturn(true)
	resp := testAPI.Get("/rclone/providers")
	suite.Equal(http.StatusOK, resp.Code)
	suite.Contains(resp.Body.String(), "providers")
}

func (suite *RcloneHandlerSuite) TestListLinks_NilNormalizedToEmptyArray() {
	testAPI := suite.labMode(true)
	mock.When(suite.rcloneSvc.ListLinks()).ThenReturn(nil, nil)
	resp := testAPI.Get("/rclone/links")
	suite.Equal(http.StatusOK, resp.Code)
	suite.Contains(resp.Body.String(), "\"links\":[]")
}

func (suite *RcloneHandlerSuite) TestListLinks_ServiceError() {
	testAPI := suite.labMode(true)
	mock.When(suite.rcloneSvc.ListLinks()).ThenReturn(nil, errors.New("db"))
	resp := testAPI.Get("/rclone/links")
	suite.Equal(http.StatusInternalServerError, resp.Code)
}

func (suite *RcloneHandlerSuite) TestGetLink_FoundAndMissing() {
	testAPI := suite.labMode(true)

	link := new(dto.RcloneLink)
	mock.When(suite.rcloneSvc.GetLink(mock.Exact("volume"), mock.Exact("volx"))).ThenReturn(link, nil)
	resp := testAPI.Get("/rclone/link/volume/volx")
	suite.Equal(http.StatusOK, resp.Code)

	mock.When(suite.rcloneSvc.GetLink(mock.Exact("volume"), mock.Exact("missing"))).
		ThenReturn(nil, errors.New("not found"))
	resp = testAPI.Get("/rclone/link/volume/missing")
	suite.Equal(http.StatusNotFound, resp.Code)
}

// TestGetLink_NilLinkNilErrorIs404 guards against the nil-pointer
// dereference regression: the service reports "no such link" as a nil
// result with no error, and the handler must answer 404 instead of
// panicking on *link.
func (suite *RcloneHandlerSuite) TestGetLink_NilLinkNilErrorIs404() {
	testAPI := suite.labMode(true)
	mock.When(suite.rcloneSvc.GetLink(mock.Exact("volume"), mock.Exact("absent"))).
		ThenReturn(nil, nil)
	resp := testAPI.Get("/rclone/link/volume/absent")
	suite.Equal(http.StatusNotFound, resp.Code)
}

func (suite *RcloneHandlerSuite) TestPutLink_OK() {
	testAPI := suite.labMode(true)
	mock.When(suite.rcloneSvc.SaveLink(mock.Any[dto.RcloneLink]())).ThenReturn(nil)
	saved := new(dto.RcloneLink)
	mock.When(suite.rcloneSvc.GetLink(mock.Exact("volume"), mock.Exact("volx"))).ThenReturn(saved, nil)
	resp := testAPI.Put("/rclone/link/volume/volx", strings.NewReader(`{"provider":"dropbox","remote_path":"bk","auto_sync":true,"schedule_minutes":10}`))
	suite.Equal(http.StatusOK, resp.Code)
}

func (suite *RcloneHandlerSuite) TestPutLink_InvalidMapsTo400() {
	testAPI := suite.labMode(true)
	mock.When(suite.rcloneSvc.SaveLink(mock.Any[dto.RcloneLink]())).ThenReturn(errors.New("invalid"))
	resp := testAPI.Put("/rclone/link/volume/bad", strings.NewReader(`{"provider":"ghost","remote_path":"bk","auto_sync":false}`))
	suite.Equal(http.StatusBadRequest, resp.Code)
}

func (suite *RcloneHandlerSuite) TestDeleteLink_OKAndError() {
	testAPI := suite.labMode(true)

	mock.When(suite.rcloneSvc.DeleteLink(mock.AnyContext(), mock.Exact("volume"), mock.Exact("volx"))).ThenReturn(nil)
	resp := testAPI.Delete("/rclone/link/volume/volx")
	suite.Equal(http.StatusNoContent, resp.Code)

	mock.When(suite.rcloneSvc.DeleteLink(mock.AnyContext(), mock.Any[string](), mock.Any[string]())).ThenReturn(errors.New("busy"))
	resp = testAPI.Delete("/rclone/link/volume/stuck")
	suite.Equal(http.StatusInternalServerError, resp.Code)
}

func (suite *RcloneHandlerSuite) TestStartAuth_OKAndError() {
	testAPI := suite.labMode(true)

	mock.When(suite.rcloneSvc.StartAuth(mock.AnyContext(), mock.Exact("volume"), mock.Exact("volx"), mock.Any[map[string]string]())).
		ThenReturn(new(dto.RcloneAuthStartResponse), nil)
	resp := testAPI.Post("/rclone/link/volume/volx/auth/start", strings.NewReader(`{"settings":{"client_id":"cid"}}`))
	suite.Equal(http.StatusOK, resp.Code)

	mock.When(suite.rcloneSvc.StartAuth(mock.AnyContext(), mock.Any[string](), mock.Any[string](), mock.Any[map[string]string]())).
		ThenReturn(nil, errors.New("link not found"))
	resp = testAPI.Post("/rclone/link/volume/none/auth/start", strings.NewReader(`{"settings":{}}`))
	suite.Equal(http.StatusBadRequest, resp.Code)
}

func (suite *RcloneHandlerSuite) TestDiff_OKAndError() {
	testAPI := suite.labMode(true)

	mock.When(suite.rcloneSvc.Diff(mock.AnyContext(), mock.Exact("volume"), mock.Exact("volx"))).
		ThenReturn(new(dto.RcloneDiffResult), nil)
	resp := testAPI.Post("/rclone/link/volume/volx/diff", strings.NewReader(`{}`))
	suite.Equal(http.StatusOK, resp.Code)

	mock.When(suite.rcloneSvc.Diff(mock.AnyContext(), mock.Any[string](), mock.Any[string]())).ThenReturn(nil, errors.New("no link"))
	resp = testAPI.Post("/rclone/link/volume/none/diff", strings.NewReader(`{}`))
	suite.Equal(http.StatusBadRequest, resp.Code)
}

func (suite *RcloneHandlerSuite) TestSync_DirectionValidationAndConflict() {
	testAPI := suite.labMode(true)

	// Invalid direction is rejected by the OpenAPI enum before the handler.
	resp := testAPI.Post("/rclone/link/volume/volx/sync", strings.NewReader(`{"direction":"sideways"}`))
	suite.Equal(http.StatusUnprocessableEntity, resp.Code)

	mock.When(suite.rcloneSvc.Sync(mock.Exact("volume"), mock.Exact("volx"), mock.Exact("push"), mock.Exact(false))).ThenReturn(nil)
	resp = testAPI.Post("/rclone/link/volume/volx/sync", strings.NewReader(`{"direction":"push"}`))
	suite.Equal(http.StatusNoContent, resp.Code)

	// Dry-run flag is forwarded to the service untouched.
	mock.When(suite.rcloneSvc.Sync(mock.Exact("volume"), mock.Exact("volx"), mock.Exact("push"), mock.Exact(true))).ThenReturn(nil)
	resp = testAPI.Post("/rclone/link/volume/volx/sync", strings.NewReader(`{"direction":"push","dry_run":true}`))
	suite.Equal(http.StatusNoContent, resp.Code)

	mock.When(suite.rcloneSvc.Sync(mock.Any[string](), mock.Any[string](), mock.Any[string](), mock.Any[bool]())).ThenReturn(errors.New("job already running"))
	resp = testAPI.Post("/rclone/link/volume/volx/sync", strings.NewReader(`{"direction":"bidi"}`))
	suite.Equal(http.StatusConflict, resp.Code)
}

func (suite *RcloneHandlerSuite) TestAbortSync_OKAndError() {
	testAPI := suite.labMode(true)

	mock.When(suite.rcloneSvc.AbortSync(mock.Exact("volume"), mock.Exact("volx"))).ThenReturn(nil)
	resp := testAPI.Post("/rclone/link/volume/volx/abort", strings.NewReader(`{}`))
	suite.Equal(http.StatusNoContent, resp.Code)

	mock.When(suite.rcloneSvc.AbortSync(mock.Any[string](), mock.Any[string]())).ThenReturn(errors.New("idle"))
	resp = testAPI.Post("/rclone/link/volume/idle/abort", strings.NewReader(`{}`))
	suite.Equal(http.StatusConflict, resp.Code)
}

func (suite *RcloneHandlerSuite) TestOAuthCallback_NotLabGated() {
	// Settings load fails but the callback must still work: it is invoked by
	// the provider redirect and never carries lab headers.
	mock.When(suite.settingSvc.Load()).ThenReturn(nil, errors.New("no settings"))
	_, testAPI := humatest.New(suite.T())
	suite.handler.RegisterRcloneHandler(testAPI)

	resp := testAPI.Get("/rclone/oauth/callback")
	suite.Equal(http.StatusOK, resp.Code)
	suite.Contains(resp.Header().Get("Content-Type"), "text/html")
	suite.Contains(resp.Body.String(), "Missing code or state parameter")
}

func (suite *RcloneHandlerSuite) TestOAuthCallback_ProviderErrorEscaped() {
	testAPI := suite.labMode(true)
	resp := testAPI.Get("/rclone/oauth/callback?error=denied%3Cscript%3Ealert(1)%3C%2Fscript%3E")
	suite.Equal(http.StatusOK, resp.Code)
	suite.Contains(resp.Body.String(), "Authorization denied:")
	suite.Contains(resp.Body.String(), "&lt;script&gt;")
	suite.NotContains(resp.Body.String(), "<script>alert")
}

func (suite *RcloneHandlerSuite) TestOAuthCallback_FailureAndSuccess() {
	testAPI := suite.labMode(true)

	mock.When(suite.rcloneSvc.HandleOAuthCallback(mock.AnyContext(), mock.Exact("bad-code"), mock.Exact("st"))).
		ThenReturn(nil, errors.New("exchange failed"))
	resp := testAPI.Get("/rclone/oauth/callback?code=bad-code&state=st")
	suite.Equal(http.StatusOK, resp.Code)
	suite.Contains(resp.Body.String(), "Authorization failed:")
	suite.Contains(resp.Body.String(), "exchange failed")

	mock.When(suite.rcloneSvc.HandleOAuthCallback(mock.AnyContext(), mock.Exact("ok"), mock.Exact("st2"))).
		ThenReturn(new(dto.RcloneLink), nil)
	resp = testAPI.Get("/rclone/oauth/callback?code=ok&state=st2")
	suite.Equal(http.StatusOK, resp.Code)
	suite.Contains(resp.Body.String(), "Authorization complete.")
}
