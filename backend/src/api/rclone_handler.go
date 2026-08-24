package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"go.uber.org/fx"
)

// RcloneHandler exposes the lab-gated cloud-sync endpoints (issue #954).
//
// All routes require Lab Mode (settings.experimental_lab_mode=true) except
// the OAuth callback, which is invoked by the remote provider's browser
// redirect and is protected by the single-use state token instead.
type RcloneHandler struct {
	rcloneService  service.RcloneServiceInterface
	settingService service.SettingServiceInterface
}

type RcloneHandlerParams struct {
	fx.In
	RcloneService  service.RcloneServiceInterface
	SettingService service.SettingServiceInterface
}

func NewRcloneHandler(params RcloneHandlerParams) *RcloneHandler {
	return &RcloneHandler{
		rcloneService:  params.RcloneService,
		settingService: params.SettingService,
	}
}

// RegisterRcloneHandler registers the HTTP handlers for rclone cloud sync.
//
// Routes (lab-gated unless noted):
//   - GET    /rclone/providers                              — registered providers + library availability
//   - GET    /rclone/links                                  — all configured links
//   - GET    /rclone/link/{target_kind}/{target_id}         — one link (404 when absent)
//   - PUT    /rclone/link/{target_kind}/{target_id}         — create/update link configuration
//   - DELETE /rclone/link/{target_kind}/{target_id}         — unlink (removes managed rclone.conf remote)
//   - POST   /rclone/link/{target_kind}/{target_id}/auth/start     — begin provider OAuth flow
//   - POST   /rclone/link/{target_kind}/{target_id}/diff           — compare local vs remote
//   - POST   /rclone/link/{target_kind}/{target_id}/sync           — start push/pull/bidi job
//   - POST   /rclone/link/{target_kind}/{target_id}/abort          — abort running job
//   - GET    /rclone/oauth/callback                         — provider redirect target (state-protected)
func (h *RcloneHandler) RegisterRcloneHandler(api huma.API) {
	tags := huma.OperationTags("volume", "rclone")
	huma.Get(api, "/rclone/providers", h.getProviders, tags)
	huma.Get(api, "/rclone/links", h.listLinks, tags)
	huma.Get(api, "/rclone/link/{target_kind}/{target_id}", h.getLink, tags)
	huma.Put(api, "/rclone/link/{target_kind}/{target_id}", h.putLink, tags)
	huma.Delete(api, "/rclone/link/{target_kind}/{target_id}", h.deleteLink, tags)
	huma.Post(api, "/rclone/link/{target_kind}/{target_id}/auth/start", h.startAuth, tags)
	huma.Post(api, "/rclone/link/{target_kind}/{target_id}/diff", h.diff, tags)
	huma.Post(api, "/rclone/link/{target_kind}/{target_id}/sync", h.sync, tags)
	huma.Post(api, "/rclone/link/{target_kind}/{target_id}/abort", h.abortSync, tags)

	// The OAuth callback returns a tiny HTML auto-close page rendered inside
	// the popup the wizard opened. It is NOT lab-gated: the provider browser
	// redirect cannot carry settings headers; security comes from the
	// single-use state token validated by the service.
	huma.Register(api, huma.Operation{
		OperationID: "rclone-oauth-callback",
		Method:      http.MethodGet,
		Path:        "/rclone/oauth/callback",
		Summary:     "OAuth redirect target for cloud providers",
		Tags:        []string{"volume", "rclone"},
	}, h.oauthCallback)
}

// requireLabMode returns 403 unless settings.experimental_lab_mode is true.
func (h *RcloneHandler) requireLabMode() error {
	settings, err := h.settingService.Load()
	if err != nil {
		return huma.Error500InternalServerError("Failed to read settings", err)
	}
	if settings == nil || !settings.ExperimentalLabMode {
		return huma.Error403Forbidden(
			"Rclone endpoints require Lab Mode (set experimental_lab_mode=true in settings)",
			dto.ErrorLabModeRequired,
		)
	}
	return nil
}

// ---------- handlers ----------

type GetRcloneProvidersOutput struct {
	Body dto.RcloneProvidersResponse
}

func (h *RcloneHandler) getProviders(ctx context.Context, input *struct{}) (*GetRcloneProvidersOutput, error) {
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	out := dto.RcloneProvidersResponse{
		Providers:        h.rcloneService.ListProviders(),
		LibraryAvailable: h.rcloneService.LibraryAvailable(),
	}
	if out.Providers == nil {
		out.Providers = []dto.RcloneProviderInfo{}
	}
	return &GetRcloneProvidersOutput{Body: out}, nil
}

type GetRcloneLinksOutput struct {
	Body struct {
		Links []dto.RcloneLink `json:"links"`
	}
}

func (h *RcloneHandler) listLinks(ctx context.Context, input *struct{}) (*GetRcloneLinksOutput, error) {
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	links, err := h.rcloneService.ListLinks()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list rclone links", err)
	}
	if links == nil {
		links = []dto.RcloneLink{}
	}
	out := &GetRcloneLinksOutput{}
	out.Body.Links = links
	return out, nil
}

type GetRcloneLinkOutput struct {
	Body dto.RcloneLink
}

func (h *RcloneHandler) getLink(ctx context.Context, input *struct {
	TargetKind string `path:"target_kind" required:"true" doc:"Link target kind (volume or hassos_data)"`
	TargetID   string `path:"target_id" required:"true" doc:"Link target id (volume path or hassos-data)"`
}) (*GetRcloneLinkOutput, error) {
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	link, err := h.rcloneService.GetLink(input.TargetKind, input.TargetID)
	if err != nil || link == nil {
		// The service signals "no such link" with a nil result and no error,
		// so both cases must answer 404 instead of dereferencing nil.
		return nil, huma.Error404NotFound("Rclone link not found: "+input.TargetKind+"/"+input.TargetID, err)
	}
	return &GetRcloneLinkOutput{Body: *link}, nil
}

type PutRcloneLinkInput struct {
	TargetKind string `path:"target_kind" required:"true" doc:"Link target kind (volume or hassos_data)"`
	TargetID   string `path:"target_id" required:"true" doc:"Link target id (volume path or hassos-data)"`
	Body       dto.RcloneLinkRequest
}

func (h *RcloneHandler) putLink(ctx context.Context, input *PutRcloneLinkInput) (*GetRcloneLinkOutput, error) {
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	err := h.rcloneService.SaveLink(dto.RcloneLink{
		TargetKind:      input.TargetKind,
		TargetID:        input.TargetID,
		Provider:        input.Body.Provider,
		RemotePath:      input.Body.RemotePath,
		Status:          dto.RcloneStatusUnlinked,
		AutoSync:        input.Body.AutoSync,
		ScheduleMinutes: input.Body.ScheduleMinutes,
	})
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid rclone link request", err)
	}
	link, gerr := h.rcloneService.GetLink(input.TargetKind, input.TargetID)
	if gerr != nil || link == nil {
		return nil, huma.Error500InternalServerError("Failed to persist rclone link", gerr)
	}
	return &GetRcloneLinkOutput{Body: *link}, nil
}

// DeleteRcloneLinkOutput carries no body so huma answers 204 No Content.
type DeleteRcloneLinkOutput struct{}

func (h *RcloneHandler) deleteLink(ctx context.Context, input *struct {
	TargetKind string `path:"target_kind" required:"true" doc:"Link target kind (volume or hassos_data)"`
	TargetID   string `path:"target_id" required:"true" doc:"Link target id (volume path or hassos-data)"`
}) (*DeleteRcloneLinkOutput, error) {
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	if err := h.rcloneService.DeleteLink(ctx, input.TargetKind, input.TargetID); err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete rclone link", err)
	}
	return &DeleteRcloneLinkOutput{}, nil
}

type StartRcloneAuthInput struct {
	TargetKind string `path:"target_kind" required:"true" doc:"Link target kind (volume or hassos_data)"`
	TargetID   string `path:"target_id" required:"true" doc:"Link target id (volume path or hassos-data)"`
	// Body carries the provider credentials collected by the wizard.
	// They are forwarded to the driver and persisted ONLY in the managed
	// rclone.conf, never in SQLite.
	Body struct {
		Settings map[string]string `json:"settings"`
	}
}

type StartRcloneAuthOutput struct {
	Body dto.RcloneAuthStartResponse
}

func (h *RcloneHandler) startAuth(ctx context.Context, input *StartRcloneAuthInput) (*StartRcloneAuthOutput, error) {
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	resp, err := h.rcloneService.StartAuth(ctx, input.TargetKind, input.TargetID, input.Body.Settings)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to start authorization flow", err)
	}
	return &StartRcloneAuthOutput{Body: *resp}, nil
}

type RcloneDiffOutput struct {
	Body dto.RcloneDiffResult
}

func (h *RcloneHandler) diff(ctx context.Context, input *struct {
	TargetKind string `path:"target_kind" required:"true" doc:"Link target kind (volume or hassos_data)"`
	TargetID   string `path:"target_id" required:"true" doc:"Link target id (volume path or hassos-data)"`
}) (*RcloneDiffOutput, error) {
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	result, err := h.rcloneService.Diff(ctx, input.TargetKind, input.TargetID)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to compute diff", err)
	}
	return &RcloneDiffOutput{Body: *result}, nil
}

// RcloneSyncOutput has no body: progress arrives over the "rclone_task"
// WebSocket event, mirroring filesystem check semantics.
type RcloneSyncOutput struct{}

func (h *RcloneHandler) sync(ctx context.Context, input *struct {
	TargetKind string `path:"target_kind" required:"true" doc:"Link target kind (volume or hassos_data)"`
	TargetID   string `path:"target_id" required:"true" doc:"Link target id (volume path or hassos-data)"`
	Body       dto.RcloneSyncRequest
}) (*RcloneSyncOutput, error) {
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	// Direction validity is enforced by the OpenAPI enum on
	// dto.RcloneSyncRequest; huma answers 422 for anything else.
	// DryRun=true runs rclone with dryRun enabled and skips link
	// bookkeeping entirely (see RcloneService.Sync).
	if err := h.rcloneService.Sync(input.TargetKind, input.TargetID, input.Body.Direction, input.Body.DryRun); err != nil {
		return nil, huma.Error409Conflict("Cannot start synchronization", err)
	}
	return &RcloneSyncOutput{}, nil
}

// RcloneAbortOutput has no body; 204 confirms an abort request was issued.
type RcloneAbortOutput struct{}

func (h *RcloneHandler) abortSync(ctx context.Context, input *struct {
	TargetKind string `path:"target_kind" required:"true" doc:"Link target kind (volume or hassos_data)"`
	TargetID   string `path:"target_id" required:"true" doc:"Link target id (volume path or hassos-data)"`
}) (*RcloneAbortOutput, error) {
	if err := h.requireLabMode(); err != nil {
		return nil, err
	}
	if err := h.rcloneService.AbortSync(input.TargetKind, input.TargetID); err != nil {
		return nil, huma.Error409Conflict("No abortable synchronization job", err)
	}
	return &RcloneAbortOutput{}, nil
}

// rcloneCallbackOutput uses huma's raw-response pattern (Body as a writer
// function) so the callback can serve a small HTML page instead of JSON.
type rcloneCallbackOutput struct {
	Body func(huma.Context)
}

// oauthCallback renders the popup-closing page after Dropbox (or another
// provider) redirects back with the authorization code. Errors are shown as
// plain text in the same page so the user can copy them from the popup.
func (h *RcloneHandler) oauthCallback(ctx context.Context, input *struct {
	Code  string `query:"code" required:"false" doc:"Provider authorization code"`
	State string `query:"state" required:"false" doc:"Anti-CSRF state token"`
	Error string `query:"error" required:"false" doc:"Provider-reported denial reason"`
}) (*rcloneCallbackOutput, error) {
	var page string
	switch {
	case input.Error != "":
		page = callbackPage("&#10060; Authorization denied: " + htmlEsc(input.Error))
	case input.Code == "" || input.State == "":
		page = callbackPage("&#10060; Missing code or state parameter")
	default:
		if _, err := h.rcloneService.HandleOAuthCallback(ctx, input.Code, input.State); err != nil {
			page = callbackPage("&#10060; Authorization failed: " + htmlEsc(err.Error()))
		} else {
			// Success: auto-close the wizard popup.
			page = `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>SRAT Cloud Sync</title></head>
<body style="font-family:sans-serif;text-align:center;padding-top:3em;">
<p>&#9989; Authorization complete.</p>
<script>window.close()</script>
<p>You can close this window.</p>
</body>
</html>`
		}
	}
	return &rcloneCallbackOutput{Body: func(c huma.Context) {
		c.SetHeader("Content-Type", "text/html; charset=utf-8")
		_, _ = c.BodyWriter().Write([]byte(page))
	}}, nil
}

// callbackPage builds the shared error page shown inside the OAuth popup.
func callbackPage(messageHTML string) string {
	return `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>SRAT Cloud Sync</title></head>
<body style="font-family:sans-serif;text-align:center;padding-top:3em;">
<p>` + messageHTML + `</p>
<p>You can close this window.</p>
</body>
</html>`
}

// htmlEsc performs minimal HTML escaping for untrusted error strings.
func htmlEsc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&#34;")
	return r.Replace(s)
}
