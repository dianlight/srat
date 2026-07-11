package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHDIdleDevice_SuggestionIgnored_FalseIsSerializedToJSON(t *testing.T) {
	device := dto.HDIdleDevice{SuggestionIgnored: false}
	data, err := json.Marshal(device)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"suggestion_ignored":false`,
		"suggestion_ignored:false must be present in JSON (no omitempty)")
}

func TestHDIdleDevice_SuggestionIgnored_TrueIsSerializedToJSON(t *testing.T) {
	device := dto.HDIdleDevice{SuggestionIgnored: true}
	data, err := json.Marshal(device)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"suggestion_ignored":true`)
}

func TestHDIdleDevice_ForceEnabled_FalseIsSerializedToJSON(t *testing.T) {
	device := dto.HDIdleDevice{ForceEnabled: false}
	data, err := json.Marshal(device)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"force_enabled":false`,
		"force_enabled:false must be present in JSON (no omitempty)")
}

func TestHDIdleDevice_ForceEnabled_TrueIsSerializedToJSON(t *testing.T) {
	device := dto.HDIdleDevice{ForceEnabled: true}
	data, err := json.Marshal(device)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"force_enabled":true`)
}

func TestHDIdleDeviceSupport_Supported_FalseIsSerializedToJSON(t *testing.T) {
	support := dto.HDIdleDeviceSupport{Supported: false}
	data, err := json.Marshal(support)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"supported":false`,
		"supported:false must be present in JSON (no omitempty)")
}

func TestHDIdleDeviceSupport_SupportsSCSI_FalseIsSerializedToJSON(t *testing.T) {
	support := dto.HDIdleDeviceSupport{SupportsSCSI: false}
	data, err := json.Marshal(support)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"supports_scsi":false`,
		"supports_scsi:false must be present in JSON (no omitempty)")
}

func TestHDIdleDeviceSupport_SupportsATA_FalseIsSerializedToJSON(t *testing.T) {
	support := dto.HDIdleDeviceSupport{SupportsATA: false}
	data, err := json.Marshal(support)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"supports_ata":false`,
		"supports_ata:false must be present in JSON (no omitempty)")
}

func TestHDIdleDevice_AllBoolFields_FalseRoundTrip(t *testing.T) {
	device := dto.HDIdleDevice{
		SuggestionIgnored: false,
		ForceEnabled:      false,
	}
	data, err := json.Marshal(device)
	require.NoError(t, err)
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"suggestion_ignored":false`)
	assert.Contains(t, jsonStr, `"force_enabled":false`)
	assert.Contains(t, jsonStr, `"supported":false`)
	assert.Contains(t, jsonStr, `"supports_scsi":false`)
	assert.Contains(t, jsonStr, `"supports_ata":false`)
}
