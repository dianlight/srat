package service_test

import (
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/require"
)

func TestAlertEnabledForKey(t *testing.T) {
	oldVersion := config.Version
	t.Cleanup(func() { config.Version = oldVersion })
	config.Version = "0.0.0-dev.0"

	trueVal := new(true)
	falseVal := new(false)

	tests := []struct {
		name     string
		settings *dto.Settings
		key      string
		expected bool
	}{
		{"nil settings fail open", nil, "protected_mode", true},
		{"nil settings unknown key", nil, "something_new", true},
		{"protected enabled", &dto.Settings{AlertProtectedMode: trueVal}, "protected_mode", true},
		{"protected missing defaults enabled", &dto.Settings{}, "protected_mode", true},
		{"protected disabled", &dto.Settings{AlertProtectedMode: falseVal}, "protected_mode", false},
		{"protected key trimmed", &dto.Settings{AlertProtectedMode: falseVal}, "  protected_mode  ", false},
		{"addon enabled", &dto.Settings{AlertAddonConfigChanged: trueVal}, "addon_config_changed", true},
		{"addon disabled", &dto.Settings{AlertAddonConfigChanged: falseVal}, "addon_config_changed", false},
		{
			"custom enabled with lab on",
			&dto.Settings{ExperimentalLabMode: true, AlertCustomComponent: trueVal},
			"custom_component_restart_required",
			true,
		},
		{
			"custom missing enabled with lab on",
			&dto.Settings{ExperimentalLabMode: true, AlertCustomComponent: trueVal},
			"custom_component_missing",
			true,
		},
		{
			"custom disabled with lab on",
			&dto.Settings{ExperimentalLabMode: true, AlertCustomComponent: falseVal},
			"custom_component_restart_required",
			false,
		},
		{
			"custom suppressed with lab off",
			&dto.Settings{ExperimentalLabMode: false, AlertCustomComponent: trueVal},
			"custom_component_restart_required",
			false,
		},
		{
			"custom suppressed with lab off missing toggle",
			&dto.Settings{ExperimentalLabMode: false},
			"custom_component_missing",
			false,
		},
		{"unknown key enabled", &dto.Settings{}, "future_alert_key", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, service.AlertEnabledForKey(tt.settings, tt.key))
		})
	}
}

func TestAlertEnabledForKey_CustomSuppressedInProduction(t *testing.T) {
	oldVersion := config.Version
	t.Cleanup(func() { config.Version = oldVersion })
	config.Version = "1.0.0"

	settings := &dto.Settings{ExperimentalLabMode: true, AlertCustomComponent: new(true)}
	require.False(t, service.AlertEnabledForKey(settings, "custom_component_missing"))
	// Non-custom alerts are unaffected by the production gate.
	require.True(t, service.AlertEnabledForKey(settings, "protected_mode"))
}
