package rclone

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// dropboxDriver is the first provider implementation (issue #954). It relies
// on the standard Dropbox OAuth2 code flow with offline (refresh) tokens so
// rclone can refresh sessions on its own afterwards. The user supplies their
// own Dropbox App key/secret (created at https://www.dropbox.com/developers/apps
// with redirect URI pointing back to SRAT's callback endpoint).
type dropboxDriver struct{}

const (
	dropboxAuthorizeURL = "https://www.dropbox.com/oauth2/authorize"
	fieldClientID       = "client_id"
	fieldClientSecret   = "client_secret"
)

// dropboxTokenURL is a var (not const) so tests can point it at an
// httptest server instead of the real Dropbox endpoint.
var dropboxTokenURL = "https://api.dropboxapi.com/oauth2/token"

func init() { RegisterDriver(&dropboxDriver{}) }

var _ Driver = (*dropboxDriver)(nil)

func (d *dropboxDriver) Name() string        { return "dropbox" }
func (d *dropboxDriver) DisplayName() string { return "Dropbox" }

func (d *dropboxDriver) ConfigFields() []ConfigField {
	return []ConfigField{
		{
			Name:        fieldClientID,
			Label:       "App key",
			Description: "Dropbox app key from the app console (redirect URI must include SRAT's callback URL)",
			Required:    true,
		},
		{
			Name:        fieldClientSecret,
			Label:       "App secret",
			Description: "Dropbox app secret from the app console",
			Secret:      true,
			Required:    true,
		},
	}
}

// AuthStart builds the Dropbox authorization URL. token_access_type=offline
// makes Dropbox issue a refresh token, required for unattended syncs.
func (d *dropboxDriver) AuthStart(ctx context.Context, req AuthRequest) (string, error) {
	clientID := req.Settings[fieldClientID]
	if clientID == "" || req.Settings[fieldClientSecret] == "" {
		// Unlike plain rclone (which can fall back to its own built-in app
		// credentials with a 127.0.0.1 loopback redirect), SRAT completes the
		// OAuth flow server-side on a custom redirect URI that Dropbox only
		// accepts when it is registered for the app owning the credentials.
		return "", fmt.Errorf(
			"Dropbox App key and App secret are required: create an app at https://www.dropbox.com/developers/apps and register %q as a redirect URI (rclone's built-in default app cannot be reused because SRAT handles the OAuth callback server-side)",
			req.RedirectURI,
		)
	}
	values := url.Values{
		"client_id":         {clientID},
		"response_type":     {"code"},
		"token_access_type": {"offline"},
		"redirect_uri":      {req.RedirectURI},
		"state":             {req.State},
	}
	return dropboxAuthorizeURL + "?" + values.Encode(), nil
}

// dropboxTokenResponse matches both the Dropbox token endpoint response and
// rclone's expected "token" config parameter shape, which stores this JSON
// verbatim.
type dropboxTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	Expiry       string `json:"expiry,omitempty"`
	AccountID    string `json:"account_id,omitempty"`
}

func (d *dropboxDriver) ExchangeCode(ctx context.Context, req AuthRequest, code string) (*TokenResult, error) {
	clientID := req.Settings[fieldClientID]
	clientSecret := req.Settings[fieldClientSecret]
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("dropbox driver requires %s and %s settings", fieldClientID, fieldClientSecret)
	}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {req.RedirectURI},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, dropboxTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build dropbox token request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("dropbox token exchange failed: %w", err)
	}
	defer resp.Body.Close()
	var tokenResp dropboxTokenResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&tokenResp); decodeErr != nil || tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("invalid dropbox token response (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dropbox token exchange returned status %d", resp.StatusCode)
	}
	// rclone expects an RFC3339 expiry timestamp alongside the tokens.
	tokenResp.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).UTC().Format(time.RFC3339)
	tokenJSON, encErr := json.Marshal(tokenResp)
	if encErr != nil {
		return nil, fmt.Errorf("encode dropbox token: %w", encErr)
	}
	return &TokenResult{TokenJSON: string(tokenJSON), AccountLabel: tokenResp.AccountID}, nil
}
