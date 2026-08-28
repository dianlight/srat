package rclone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dianlight/srat/config"
)

// The hosted OAuth broker is an optional, centrally operated service that
// owns per-provider OAuth app credentials (client_id/client_secret). It lets
// users authorize a cloud account without creating their own provider app:
//
//	SRAT  --POST /v1/start {provider, srat_callback_url}-->  broker
//	browser --- provider authorization (broker's registered app) ---> provider
//	provider --- code --> broker callback: broker exchanges the code with ITS
//	                     secret and stores {session_id → token} (short TTL,
//	                     single use), then redirects the browser back to
//	                     srat_callback_url (SRAT's own callback + state).
//	SRAT  --GET /v1/session/{session_id}--> broker  (single-use token fetch)
//
// The base URL comes from SRAT_OAUTH_BROKER_URL; when unset every flow falls
// back to the custom-app mode where the user supplies client_id/secret.
//
// Home Assistant parallel (Dropbox):
// HA's own Dropbox integration (homeassistant/components/dropbox, since
// 2026.4) does NOT do pure client-side OAuth either. By default it uses
// Home Assistant Cloud Account Linking (Nabu Casa-operated proxy at
// accounts.home-assistant.io) which holds the client_id/client_secret
// registered at Dropbox on behalf of all users — the same centralized-secret
// pattern as this broker. The browser redirect fans out via
// https://my.home-assistant.io/redirect/oauth (or a Nabu Casa cloudhook when
// the user has a Cloud subscription) back to the local instance. The
// "Application Credentials" path (user creates their own Dropbox app) is the
// bring-your-own-app alternative — identical to SRAT's custom_app mode.
// Third-party addons or custom components cannot register their domain there
// nor reuse HA's Dropbox credentials.
//
// PKCE variant:
// Dropbox supports PKCE (no client_secret, public client_id + code_verifier),
// but the bottleneck is not the secret — it is the pre-registered redirect
// URI, which still requires a public domain we control. PKCE would only turn
// the broker from secret-custodian into a simple relay; it would not
// eliminate the deploy.
//
// Task 050 alternative ("Reuse Dropbox integration auth (planned)" in the
// wizard): since custom_components/srat runs inside the HA process, it can
// read hass.config_entries for the Dropbox integration (token + client
// material the integration already obtained) and forward it to the backend
// over the existing WebSocket channel — no broker, no new app, just reuse of
// the authorization the user already performed in HA. See
// docs/tasks/050_reuse-ha-dropbox-oauth.md.

// BrokerBaseURLEnv is the environment variable holding the hosted OAuth
// broker base URL (e.g. "https://oauth.example.com").
const BrokerBaseURLEnv = "SRAT_OAUTH_BROKER_URL"

// BrokerTokenEnv optionally holds the shared bearer token presented to the
// broker on /v1/start and /v1/session calls.
const BrokerTokenEnv = "SRAT_OAUTH_BROKER_TOKEN"

// brokerOffSentinel explicitly disables the hosted flow even when a default
// URL is compiled into the binary (case-insensitive).
const brokerOffSentinel = "off"

// addonBrokerOptionsKey is the sambanas2 addon option mirrored into the
// Supervisor-mounted options file.
const addonBrokerOptionsKey = "srat_oauth_broker_url"

// defaultBrokerURL is baked into release binaries via:
//
//	-ldflags "-X github.com/dianlight/srat/service/rclone.defaultBrokerURL=https://…"
//
// The source-level default is empty; addon deployments normally resolve the
// URL at runtime from their own options file instead (see below).
var defaultBrokerURL = ""

// brokerHTTPClient bounds broker round-trips; a var so tests can swap it.
var brokerHTTPClient = &http.Client{Timeout: 15 * time.Second}

// brokerURLFromAddonOptions reads the hosted-broker URL from this addon's
// own options file (/data/options.json, written by the Home Assistant
// Supervisor from the addon configuration). Reading happens on every call
// so option edits take effect without an addon restart. Anything unusual —
// missing file (non-addon deployments), malformed JSON, wrong type — simply
// yields "".
func brokerURLFromAddonOptions() string {
	raw, err := os.ReadFile(config.AddonOptionsFilePath)
	if err != nil {
		return ""
	}
	var opts map[string]any
	if json.Unmarshal(raw, &opts) != nil {
		return ""
	}
	v, _ := opts[addonBrokerOptionsKey].(string)
	return strings.TrimSpace(v)
}

// BrokerBaseURL returns the effective broker base URL with any trailing
// slash removed, or "" when no broker is available. Resolution order:
//
//  1. SRAT_OAUTH_BROKER_URL set to "off" → disabled
//  2. SRAT_OAUTH_BROKER_URL set to a non-empty value → that value
//  3. addon option srat_oauth_broker_url (non-empty) → that value
//  4. build-time defaultBrokerURL (may itself be empty)
func BrokerBaseURL() string {
	envValue := os.Getenv(BrokerBaseURLEnv)
	switch strings.ToLower(strings.TrimSpace(envValue)) {
	case brokerOffSentinel:
		return ""
	case "":
	default:
		return strings.TrimSuffix(strings.TrimSpace(envValue), "/")
	}
	if v := brokerURLFromAddonOptions(); v != "" {
		return strings.TrimSuffix(v, "/")
	}
	return strings.TrimSuffix(defaultBrokerURL, "/")
}

// BrokerAvailable reports whether a hosted OAuth broker is configured.
func BrokerAvailable() bool { return BrokerBaseURL() != "" }

// setBrokerAuth attaches the optional bearer credential to an outgoing
// broker request.
func setBrokerAuth(req *http.Request) {
	if tok := strings.TrimSpace(os.Getenv(BrokerTokenEnv)); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
}

// BrokerSession is the outcome of starting a hosted authorization flow.
type BrokerSession struct {
	// AuthURL sends the user's browser to the provider authorization page
	// (pre-built by the broker with its own app credentials).
	AuthURL string `json:"auth_url"`
	// SessionID correlates the flow; SRAT presents it once to fetch the
	// resulting token after the broker redirects the browser back.
	SessionID string `json:"session_id"`
}

// BrokerToken is the single-use payload handed over by the broker after the
// user completed authorization.
type BrokerToken struct {
	// TokenJSON is the rclone-shaped provider token envelope (same contract
	// as TokenResult.TokenJSON).
	TokenJSON string `json:"token_json"`
	// AccountLabel is an optional friendly account identifier.
	AccountLabel string `json:"account_label,omitempty"`
	// ClientID/ClientSecret are the broker app credentials the refresh token
	// is bound to. librclone needs them to refresh offline, so they transit
	// once to this addon over TLS; they are not shipped in the binary.
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

type brokerStartRequest struct {
	Provider        string `json:"provider"`
	SratCallbackURL string `json:"srat_callback_url"`
}

type brokerErrorResponse struct {
	Error string `json:"error"`
}

// BrokerStart opens a hosted authorization session for provider. The broker
// redirects the browser to sratCallbackURL (which must already carry SRAT's
// state query parameter) once the token exchange completed.
func BrokerStart(ctx context.Context, baseURL, provider, sratCallbackURL string) (*BrokerSession, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("no oauth broker configured (%s is unset)", BrokerBaseURLEnv)
	}
	body, err := json.Marshal(brokerStartRequest{Provider: provider, SratCallbackURL: sratCallbackURL})
	if err != nil {
		return nil, fmt.Errorf("encode oauth broker request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/start", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build oauth broker request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	setBrokerAuth(httpReq)
	resp, err := brokerHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("oauth broker unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg := readBrokerError(resp)
		return nil, fmt.Errorf("oauth broker rejected start (status %d): %s", resp.StatusCode, msg)
	}
	var session BrokerSession
	if decodeErr := json.NewDecoder(resp.Body).Decode(&session); decodeErr != nil || session.AuthURL == "" || session.SessionID == "" {
		return nil, fmt.Errorf("invalid oauth broker start response (status %d)", resp.StatusCode)
	}
	return &session, nil
}

// BrokerFetchToken retrieves the stored token for a session. The fetch is
// single-use server-side: a second call (or one after the TTL) fails.
func BrokerFetchToken(ctx context.Context, baseURL, sessionID string) (*BrokerToken, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("no oauth broker configured (%s is unset)", BrokerBaseURLEnv)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/session/"+url.PathEscape(sessionID), nil)
	if err != nil {
		return nil, fmt.Errorf("build oauth broker request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	setBrokerAuth(httpReq)
	resp, err := brokerHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("oauth broker unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("oauth broker session expired or already used")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oauth broker returned status %d: %s", resp.StatusCode, readBrokerError(resp))
	}
	var token BrokerToken
	if decodeErr := json.NewDecoder(resp.Body).Decode(&token); decodeErr != nil || token.TokenJSON == "" {
		return nil, fmt.Errorf("invalid oauth broker session response (status %d)", resp.StatusCode)
	}
	return &token, nil
}

// readBrokerError extracts the broker's error message when present.
func readBrokerError(resp *http.Response) string {
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 512))
	if err != nil || len(raw) == 0 {
		return "no details"
	}
	var e brokerErrorResponse
	if json.Unmarshal(raw, &e) == nil && e.Error != "" {
		return e.Error
	}
	return strings.TrimSpace(string(raw))
}
