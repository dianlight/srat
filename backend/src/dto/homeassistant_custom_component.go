package dto

import "time"

const (
	// DefaultCustomComponentsPath is the default Home Assistant custom-components root.
	DefaultCustomComponentsPath = "/homeassistant/custom_components/"
	// CustomComponentSRATName is the folder name for SRAT custom component.
	CustomComponentSRATName = "srat"
	// HomeAssistantComponentMissingIssueTitle is emitted when the SRAT custom
	// component is both missing on disk and disconnected from websocket.
	HomeAssistantComponentMissingIssueTitle = "Home Assistant SRAT custom component missing and disconnected"
	// HomeAssistantComponentMissingIssueResolutionLink opens the frontend guided
	// resolution flow for custom-component install/repair.
	HomeAssistantComponentMissingIssueResolutionLink = "srat://settings/homeassistant/custom-component/install"
)

// HomeAssistantCustomComponentStatus describes the current backend view of the
// SRAT Home Assistant custom component installation and websocket connection.
type HomeAssistantCustomComponentStatus struct {
	Component        string     `json:"component"`
	InstallPath      string     `json:"install_path"`
	ManifestPath     string     `json:"manifest_path"`
	Installed        bool       `json:"installed"`
	InstalledVersion *string    `json:"installed_version,omitempty"`
	LatestVersion    *string    `json:"latest_version,omitempty"`
	Connected        bool       `json:"connected"`
	ConnectedVersion *string    `json:"connected_version,omitempty"`
	CanInstall       bool       `json:"can_install"`
	CanUpgrade       bool       `json:"can_upgrade"`
	CanUninstall     bool       `json:"can_uninstall"`
	ConnectedAt      *time.Time `json:"connected_at,omitempty"`
	HAVersion        *string    `json:"ha_version,omitempty"`
	EntryID          *string    `json:"entry_id,omitempty"`
}
