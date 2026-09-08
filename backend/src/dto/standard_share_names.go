package dto

import "encoding/json"

// StandardShareNamesMode controls which standard share names are exposed by
// Samba: the legacy names (addons, addon_configs), the new names (local_apps,
// app_configs), or both.
type StandardShareNamesMode string

const (
	// StandardShareNamesModeBoth exposes both legacy and new standard share
	// names. It is the default when the setting is unset.
	StandardShareNamesModeBoth StandardShareNamesMode = "both"
	// StandardShareNamesModeOld exposes only the legacy names.
	StandardShareNamesModeOld StandardShareNamesMode = "old"
	// StandardShareNamesModeNew exposes only the new names.
	StandardShareNamesModeNew StandardShareNamesMode = "new"
)

// MarshalJSON serializes the mode, normalizing the zero value to "both" so
// that a never-configured setting still produces a value valid for the
// "old,new,both" enum schema.
func (m StandardShareNamesMode) MarshalJSON() ([]byte, error) {
	if m == "" {
		m = StandardShareNamesModeBoth
	}
	return json.Marshal(string(m))
}

// IsStandardShareHidden reports whether a standard share name is hidden by
// the mode (issue #1142). Mirrors applyStandardShareNamesPolicy: old hides
// the new names, new hides the legacy names, both (or unset) hides nothing.
// Matching is case-sensitive; non-standard names are never hidden.
func IsStandardShareHidden(name string, mode StandardShareNamesMode) bool {
	switch mode {
	case StandardShareNamesModeOld:
		return name == "local_apps" || name == "app_configs"
	case StandardShareNamesModeNew:
		return name == "addons" || name == "addon_configs"
	default:
		return false
	}
}
