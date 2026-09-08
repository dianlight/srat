package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsStandardShareHidden(t *testing.T) {
	tests := []struct {
		name     string
		share    string
		mode     dto.StandardShareNamesMode
		expected bool
	}{
		{"old hides local_apps", "local_apps", dto.StandardShareNamesModeOld, true},
		{"old hides app_configs", "app_configs", dto.StandardShareNamesModeOld, true},
		{"old keeps addons", "addons", dto.StandardShareNamesModeOld, false},
		{"old keeps addon_configs", "addon_configs", dto.StandardShareNamesModeOld, false},
		{"new hides addons", "addons", dto.StandardShareNamesModeNew, true},
		{"new hides addon_configs", "addon_configs", dto.StandardShareNamesModeNew, true},
		{"new keeps local_apps", "local_apps", dto.StandardShareNamesModeNew, false},
		{"new keeps app_configs", "app_configs", dto.StandardShareNamesModeNew, false},
		{"both hides nothing", "addons", dto.StandardShareNamesModeBoth, false},
		{"both hides nothing new", "local_apps", dto.StandardShareNamesModeBoth, false},
		{"empty hides nothing", "addons", "", false},
		{"empty hides nothing new", "app_configs", "", false},
		{"non-standard never hidden old", "media", dto.StandardShareNamesModeOld, false},
		{"non-standard never hidden new", "share", dto.StandardShareNamesModeNew, false},
		{"uppercase untouched", "ADDONS", dto.StandardShareNamesModeNew, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, dto.IsStandardShareHidden(tt.share, tt.mode))
		})
	}
}

func TestSharedResourceStatusIsHiddenSerialization(t *testing.T) {
	status := dto.SharedResourceStatus{IsValid: true, IsHidden: false}
	data, err := json.Marshal(status)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"is_hidden":false`)

	hidden := dto.SharedResourceStatus{IsValid: true, IsHidden: true}
	data, err = json.Marshal(hidden)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"is_hidden":true`)
}
