package dto_test

import (
	"encoding/json"
	"testing"
	"time"

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

func TestHDIdleDeviceStatus_OmitZero_ZeroTimestampsAreOmitted(t *testing.T) {
	status := dto.HDIdleDeviceStatus{
		Name:     "sda",
		SpunDown: false,
		// All timestamps zero — must be omitted with omitzero.
	}
	data, err := json.Marshal(status)
	require.NoError(t, err)
	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "last_io_at", "zero LastIOAt must be omitted with omitzero")
	assert.NotContains(t, jsonStr, "spin_down_at", "zero SpinDownAt must be omitted with omitzero")
	assert.NotContains(t, jsonStr, "spin_up_at", "zero SpinUpAt must be omitted with omitzero")
}

func TestHDIdleDeviceStatus_OmitZero_PopulatedTimestampsAreSerialized(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	status := dto.HDIdleDeviceStatus{
		Name:       "sda",
		SpunDown:   true,
		LastIOAt:   now,
		SpinDownAt: now,
		SpinUpAt:   now,
	}
	data, err := json.Marshal(status)
	require.NoError(t, err)
	jsonStr := string(data)
	assert.Contains(t, jsonStr, "last_io_at")
	assert.Contains(t, jsonStr, "spin_down_at")
	assert.Contains(t, jsonStr, "spin_up_at")
	// Round-trip preserves values.
	var decoded dto.HDIdleDeviceStatus
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, now, decoded.LastIOAt)
	assert.Equal(t, now, decoded.SpinDownAt)
	assert.Equal(t, now, decoded.SpinUpAt)
}
