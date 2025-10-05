package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestIssueSeverity_MarshalJSON(t *testing.T) {
	severity := dto.IssueSeverities.ISSUESEVERITYERROR
	data, err := json.Marshal(severity)
	assert.NoError(t, err)
	assert.Equal(t, `"error"`, string(data))
}

func TestIssueSeverity_UnmarshalJSON(t *testing.T) {
	var severity dto.IssueSeverity
	err := json.Unmarshal([]byte(`"error"`), &severity)
	assert.NoError(t, err)
	assert.Equal(t, dto.IssueSeverities.ISSUESEVERITYERROR, severity)
}

func TestIssueSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		severity dto.IssueSeverity
		expected string
	}{
		{"Error", dto.IssueSeverities.ISSUESEVERITYERROR, "error"},
		{"Warning", dto.IssueSeverities.ISSUESEVERITYWARNING, "warning"},
		{"Info", dto.IssueSeverities.ISSUESEVERITYINFO, "info"},
		{"Success", dto.IssueSeverities.ISSUESEVERITYSUCCESS, "success"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.severity.String())
		})
	}
}

func TestIssueSeverity_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		severity dto.IssueSeverity
		valid    bool
	}{
		{"Error is valid", dto.IssueSeverities.ISSUESEVERITYERROR, true},
		{"Warning is valid", dto.IssueSeverities.ISSUESEVERITYWARNING, true},
		{"Info is valid", dto.IssueSeverities.ISSUESEVERITYINFO, true},
		{"Success is valid", dto.IssueSeverities.ISSUESEVERITYSUCCESS, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.severity.IsValid())
		})
	}
}

func TestIssueSeverity_ParseIssueSeverity(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected dto.IssueSeverity
		hasError bool
	}{
		{"String error", "error", dto.IssueSeverities.ISSUESEVERITYERROR, false},
		{"String warning", "warning", dto.IssueSeverities.ISSUESEVERITYWARNING, false},
		{"Bytes", []byte("info"), dto.IssueSeverities.ISSUESEVERITYINFO, false},
		{"IssueSeverity type", dto.IssueSeverities.ISSUESEVERITYSUCCESS, dto.IssueSeverities.ISSUESEVERITYSUCCESS, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dto.ParseIssueSeverity(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIssueSeverity_MarshalText(t *testing.T) {
	severity := dto.IssueSeverities.ISSUESEVERITYWARNING
	data, err := severity.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, `"warning"`, string(data))
}

func TestIssueSeverity_UnmarshalText(t *testing.T) {
	var severity dto.IssueSeverity
	err := severity.UnmarshalText([]byte("info"))
	assert.NoError(t, err)
	assert.Equal(t, dto.IssueSeverities.ISSUESEVERITYINFO, severity)
}

func TestIssueSeverity_MarshalBinary(t *testing.T) {
	severity := dto.IssueSeverities.ISSUESEVERITYSUCCESS
	data, err := severity.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, `"success"`, string(data))
}

func TestIssueSeverity_UnmarshalBinary(t *testing.T) {
	var severity dto.IssueSeverity
	err := severity.UnmarshalBinary([]byte("error"))
	assert.NoError(t, err)
	assert.Equal(t, dto.IssueSeverities.ISSUESEVERITYERROR, severity)
}

func TestIssueSeverity_Value(t *testing.T) {
	severity := dto.IssueSeverities.ISSUESEVERITYERROR
	val, err := severity.Value()
	assert.NoError(t, err)
	assert.Equal(t, "error", val)
}

func TestIssueSeverity_All(t *testing.T) {
	count := 0
	for range dto.IssueSeverities.All() {
		count++
	}
	assert.Equal(t, 4, count)
}

func TestUpdateChannel_String(t *testing.T) {
	tests := []struct {
		name     string
		channel  dto.UpdateChannel
		expected string
	}{
		{"None", dto.UpdateChannels.NONE, "Release"},
		{"Develop", dto.UpdateChannels.DEVELOP, "Develop"},
		{"Release", dto.UpdateChannels.RELEASE, "None"},
		{"Prerelease", dto.UpdateChannels.PRERELEASE, "Prerelease"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.channel.String())
		})
	}
}

func TestUpdateChannel_IsValid(t *testing.T) {
	assert.True(t, dto.UpdateChannels.NONE.IsValid())
	assert.True(t, dto.UpdateChannels.DEVELOP.IsValid())
	assert.True(t, dto.UpdateChannels.RELEASE.IsValid())
	assert.True(t, dto.UpdateChannels.PRERELEASE.IsValid())
}

func TestUpdateChannel_MarshalJSON(t *testing.T) {
	channel := dto.UpdateChannels.RELEASE
	data, err := json.Marshal(channel)
	assert.NoError(t, err)
	assert.Equal(t, `"None"`, string(data))
}

func TestUpdateChannel_UnmarshalJSON(t *testing.T) {
	var channel dto.UpdateChannel
	err := json.Unmarshal([]byte(`"Develop"`), &channel)
	assert.NoError(t, err)
	assert.Equal(t, dto.UpdateChannels.DEVELOP, channel)
}

func TestTelemetryMode_String(t *testing.T) {
	tests := []struct {
		name     string
		mode     dto.TelemetryMode
		expected string
	}{
		{"Ask", dto.TelemetryModes.TELEMETRYMODEASK, "Ask"},
		{"All", dto.TelemetryModes.TELEMETRYMODEALL, "All"},
		{"Errors", dto.TelemetryModes.TELEMETRYMODEERRORS, "Errors"},
		{"Disabled", dto.TelemetryModes.TELEMETRYMODEDISABLED, "Disabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.mode.String())
		})
	}
}

func TestTelemetryMode_IsValid(t *testing.T) {
	assert.True(t, dto.TelemetryModes.TELEMETRYMODEASK.IsValid())
	assert.True(t, dto.TelemetryModes.TELEMETRYMODEALL.IsValid())
	assert.True(t, dto.TelemetryModes.TELEMETRYMODEERRORS.IsValid())
	assert.True(t, dto.TelemetryModes.TELEMETRYMODEDISABLED.IsValid())
}

func TestTelemetryMode_MarshalJSON(t *testing.T) {
	mode := dto.TelemetryModes.TELEMETRYMODEERRORS
	data, err := json.Marshal(mode)
	assert.NoError(t, err)
	assert.Equal(t, `"Errors"`, string(data))
}

func TestTelemetryMode_UnmarshalJSON(t *testing.T) {
	var mode dto.TelemetryMode
	err := json.Unmarshal([]byte(`"All"`), &mode)
	assert.NoError(t, err)
	assert.Equal(t, dto.TelemetryModes.TELEMETRYMODEALL, mode)
}

func TestTimeMachineSupport_String(t *testing.T) {
	tests := []struct {
		name     string
		support  dto.TimeMachineSupport
		expected string
	}{
		{"Unsupported", dto.TimeMachineSupports.UNSUPPORTED, "unsupported"},
		{"Supported", dto.TimeMachineSupports.SUPPORTED, "supported"},
		{"Experimental", dto.TimeMachineSupports.EXPERIMENTAL, "experimental"},
		{"Unknown", dto.TimeMachineSupports.UNKNOWN, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.support.String())
		})
	}
}

func TestTimeMachineSupport_IsValid(t *testing.T) {
	assert.True(t, dto.TimeMachineSupports.UNSUPPORTED.IsValid())
	assert.True(t, dto.TimeMachineSupports.SUPPORTED.IsValid())
	assert.True(t, dto.TimeMachineSupports.EXPERIMENTAL.IsValid())
	assert.True(t, dto.TimeMachineSupports.UNKNOWN.IsValid())
}

func TestTimeMachineSupport_MarshalJSON(t *testing.T) {
	support := dto.TimeMachineSupports.SUPPORTED
	data, err := json.Marshal(support)
	assert.NoError(t, err)
	assert.Equal(t, `"supported"`, string(data))
}

func TestTimeMachineSupport_UnmarshalJSON(t *testing.T) {
	var support dto.TimeMachineSupport
	err := json.Unmarshal([]byte(`"experimental"`), &support)
	assert.NoError(t, err)
	assert.Equal(t, dto.TimeMachineSupports.EXPERIMENTAL, support)
}
