package dto

import "gitlab.com/tozd/go/errors"

// HaDropboxTokenMessage is sent by the Home Assistant custom component
// (custom_components/srat) over the WebSocket to forward the OAuth token
// already held by the HA Dropbox integration (homeassistant/components/dropbox).
// It reuses hass.config_entries data so the user does not need a second
// authorization when they already authorized Dropbox in HA (task 050).
//
// The token is the rclone-shaped envelope (access_token + refresh_token +
// expiry) that Dropbox issued. When the HA entry used Application Credentials
// (custom app), client_id/client_secret are also forwarded because the
// refresh token is bound to the app that created it. When HA used Cloud
// Account Linking (Nabu Casa proxy), those fields are empty and rclone will
// store only the token.
type HaDropboxTokenMessage struct {
	// Type must be "ha_dropbox_token" (ClientEventTypeHaDropboxToken).
	Type string `json:"type"`
	// TokenJSON is the rclone-shaped token envelope. Required.
	TokenJSON string `json:"token_json"`
	// ClientID/ClientSecret are the app credentials the refresh token is
	// bound to, if known (custom-app entries). Empty for Cloud Linking tokens.
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	// AccountLabel is an optional friendly account identifier.
	AccountLabel string `json:"account_label,omitempty"`
}

// Validate checks required fields.
func (m HaDropboxTokenMessage) Validate() error {
	if m.Type != ClientEventTypes.CLIENTEVENTTYPEHADROPBOXTOKEN.String() {
		return errors.Errorf("invalid type %q (want ha_dropbox_token)", m.Type)
	}
	if m.TokenJSON == "" {
		return errors.New("token_json is required")
	}
	return nil
}
