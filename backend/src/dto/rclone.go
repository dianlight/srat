package dto

import "time"

// Target kinds for rclone links (issue #954, lab feature).
const (
	// RcloneTargetKindVolume marks a mounted volume as local sync target.
	RcloneTargetKindVolume = "volume"
	// RcloneTargetKindHassosData marks the hassos-data partition.
	RcloneTargetKindHassosData = "hassos_data"
)

// Rclone link lifecycle status values.
const (
	// RcloneStatusUnlinked means no OAuth authorization has completed yet.
	RcloneStatusUnlinked = "unlinked"
	// RcloneStatusAuthorized means tokens exist in the managed rclone.conf.
	RcloneStatusAuthorized = "authorized"
	// RcloneStatusError means the last operation failed; see LastSyncMessage.
	RcloneStatusError = "error"
)

// Sync directions for rclone operations.
const (
	// RcloneSyncPush copies local → remote only.
	RcloneSyncPush = "push"
	// RcloneSyncPull copies remote → local only.
	RcloneSyncPull = "pull"
	// RcloneSyncBidi performs a bidirectional (bisync) pass.
	RcloneSyncBidi = "bidi"
)

// RcloneLink represents one cloud-sync link between a SRAT local target and
// an rclone remote. It is exposed read/write through the lab-gated
// /rclone endpoints.
type RcloneLink struct {
	// TargetKind is "volume" or "hassos_data".
	TargetKind string `json:"target_kind"`
	// TargetID is the volume path/root id or "hassos-data".
	TargetID string `json:"target_id"`
	// Provider is the registered driver name (e.g. "dropbox").
	Provider string `json:"provider"`
	// RemotePath is the provider-specific remote directory.
	RemotePath string `json:"remote_path"`
	// Status is "unlinked", "authorized" or "error".
	Status          string     `json:"status"`
	LastSyncAt      *time.Time `json:"last_sync_at,omitempty"`
	LastSyncResult  string     `json:"last_sync_result,omitempty"`
	LastSyncMessage string     `json:"last_sync_message,omitempty"`
	AutoSync        bool       `json:"auto_sync"`
	// ScheduleMinutes is the auto-sync interval; 0 = manual only.
	ScheduleMinutes int `json:"schedule_minutes"`
}

// RcloneConfigField describes one provider credential field the wizard must
// collect (mirrors service/rclone.ConfigField without importing it).
type RcloneConfigField struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// RcloneProviderInfo describes one registered cloud provider.
type RcloneProviderInfo struct {
	// Name is the rclone backend identifier ("dropbox").
	Name string `json:"name"`
	// DisplayName is the human-facing name ("Dropbox").
	DisplayName string `json:"display_name"`
	// ConfigFields lists required credentials/settings.
	ConfigFields []RcloneConfigField `json:"config_fields"`
}

// RcloneProvidersResponse lists available providers for the wizard's first
// step.
type RcloneProvidersResponse struct {
	Providers []RcloneProviderInfo `json:"providers"`
	// LibraryAvailable reports whether this binary embeds librclone
	// (-tags rclonelib). When false the UI shows a build notice instead of
	// the wizard.
	LibraryAvailable bool `json:"library_available"`
	// OauthCallbackPath is the backend path providers must be allowed to
	// redirect to (registered as redirect URI in the provider app console).
	// Clients display it joined with their own origin.
	OauthCallbackPath string `json:"oauth_callback_path"`
	// BrokerAvailable reports whether the optional hosted OAuth broker is
	// configured (SRAT_OAUTH_BROKER_URL). When true, credential fields may
	// be left empty: authorization then runs through SRAT's shared cloud
	// app instead of a user-created provider app.
	BrokerAvailable bool `json:"broker_available"`
	// HaDropboxAvailable reports whether the Home Assistant Dropbox integration
	// has an active OAuth token that can be reused (task 050). When true the
	// wizard enables the "Reuse Dropbox integration auth" option, allowing a
	// server-side-only flow that reuses hass.config_entries data pushed by
	// the SRAT custom component over WebSocket.
	HaDropboxAvailable bool `json:"ha_dropbox_available"`
}

// RcloneLinkRequest creates or updates the non-secret part of a link.
type RcloneLinkRequest struct {
	Provider   string `json:"provider" example:"dropbox"`
	RemotePath string `json:"remote_path" example:"backups/srat"`
	AutoSync   bool   `json:"auto_sync"`
	// ScheduleMinutes is the auto-sync interval; 0 = manual only.
	ScheduleMinutes int               `json:"schedule_minutes,omitempty"`
	Settings        map[string]string `json:"settings,omitempty"` // provider credentials (stored in rclone.conf only)
}

// RcloneAuthStartResponse returns the URL the frontend must open to start
// the provider OAuth flow plus the correlated state token.
type RcloneAuthStartResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
	// RedirectURI echoes the callback endpoint so users can register it in
	// their provider app console.
	RedirectURI string `json:"redirect_uri"`
}

// RcloneDiffEntry represents one differing file between local and remote.
type RcloneDiffEntry struct {
	// Path is relative to the linked root on both sides.
	Path string `json:"path"`
	// DiffType is "local_only", "remote_only" or "changed".
	DiffType      string     `json:"diff_type"`
	LocalSize     *int64     `json:"local_size,omitempty"`
	RemoteSize    *int64     `json:"remote_size,omitempty"`
	LocalModTime  *time.Time `json:"local_mod_time,omitempty"`
	RemoteModTime *time.Time `json:"remote_mod_time,omitempty"`
}

// RcloneDiffResult aggregates the diff between local and remote roots.
type RcloneDiffResult struct {
	Entries []RcloneDiffEntry `json:"entries"`
	// LocalOnly/RemoteOnly/Changed counts for quick badges.
	LocalOnly  int `json:"local_only"`
	RemoteOnly int `json:"remote_only"`
	Changed    int `json:"changed"`
	// Warning is set when the remote listing failed but a (partial)
	// comparison was still produced; consumers should surface it.
	Warning string `json:"warning,omitempty"`
}

// RcloneSyncRequest starts a synchronization job for a link.
type RcloneSyncRequest struct {
	// Direction is "push", "pull" or "bidi".
	Direction string `json:"direction" enum:"push,pull,bidi"`
	// DryRun executes the same passes with rclone dryRun=true: nothing is
	// transferred and the stored link state is left untouched.
	DryRun bool `json:"dry_run,omitempty"`
}

// RcloneTask mirrors FilesystemTask for cloud-sync progress reporting over
// the "rclone_task" WebSocket event.
type RcloneTask struct {
	// TargetKind is "volume" or "hassos_data".
	TargetKind string `json:"target_kind"`

	// TargetID identifies the linked local target.
	TargetID string `json:"target_id"`

	// Operation is the running operation ("diff" or "sync").
	Operation string `json:"operation"`

	// Direction is "push", "pull" or "bidi" (sync only).
	Direction string `json:"direction,omitempty"`

	// Status is the current status ("start", "running", "success", "failure")
	Status string `json:"status"`

	// Message provides additional context about the operation
	Message string `json:"message,omitempty"`

	// Error contains error details if status is "failure"
	Error string `json:"error,omitempty"`

	// Progress is the operation progress percentage (0-100, or 999 for unsupported)
	Progress int `json:"progress,omitempty"`

	// Notes contains progress messages
	Notes []string `json:"notes,omitempty"`

	// Result contains operation result details (for success status)
	Result any `json:"result,omitempty"`
}
