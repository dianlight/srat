package service

import (
	"strings"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
)

// Alert problem keys raised as HA repair issues / persistent notifications.
// The key is stable across restarts so a user-side ignore in Home Assistant
// (repairs issue ignore) stays applied when the alert is re-emitted.
const (
	// AlertProblemKeyProtectedMode is raised while the addon runs in protected mode.
	AlertProblemKeyProtectedMode = "protected_mode"
	// AlertProblemKeyAddonConfigChanged is raised when addon options change externally.
	AlertProblemKeyAddonConfigChanged = "addon_config_changed"
	// AlertProblemKeyCustomComponentRestart is raised after custom component install/upgrade/uninstall.
	AlertProblemKeyCustomComponentRestart = "custom_component_restart_required"
	// AlertProblemKeyCustomComponentMissing is raised when the custom component is absent and disconnected.
	AlertProblemKeyCustomComponentMissing = "custom_component_missing"
)

// AlertEnabledForKey reports whether the alert identified by problemKey may be
// raised given the current settings. Unknown keys default to enabled so new
// emitters are not silently muted. Custom-component keys are additionally
// gated by the ha_custom_component lab feature: with lab mode off (or in
// production builds) they are never raised and their settings toggle is
// hidden in the UI.
func AlertEnabledForKey(settings *dto.Settings, problemKey string) bool {
	if settings == nil {
		return true
	}
	switch strings.TrimSpace(problemKey) {
	case AlertProblemKeyProtectedMode:
		return settings.ProtectedModeAlertEnabled()
	case AlertProblemKeyAddonConfigChanged:
		return settings.AddonConfigChangedAlertEnabled()
	case AlertProblemKeyCustomComponentRestart, AlertProblemKeyCustomComponentMissing:
		if !isCustomComponentAlertLabActive(settings) {
			return false
		}
		return settings.CustomComponentAlertEnabled()
	default:
		return true
	}
}

// isCustomComponentAlertLabActive mirrors IsHaCustomComponentLabEnabled for
// already-loaded settings: the alpha feature is omitted in production builds
// and otherwise requires experimental lab mode. Fail-closed on nil settings.
func isCustomComponentAlertLabActive(settings *dto.Settings) bool {
	if config.Environment() == "production" {
		return false
	}
	if settings == nil {
		return false
	}
	return settings.ExperimentalLabMode
}
