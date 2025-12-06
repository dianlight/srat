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

// EventType tests
func TestEventType_String(t *testing.T) {
	tests := []struct {
		name     string
		event    dto.WebEventType
		expected string
	}{
		{"Hello", dto.WebEventTypes.EVENTHELLO, "hello"},
		{"Updating", dto.WebEventTypes.EVENTUPDATING, "updating"},
		{"Volumes", dto.WebEventTypes.EVENTVOLUMES, "volumes"},
		{"Heartbeat", dto.WebEventTypes.EVENTHEARTBEAT, "heartbeat"},
		{"Share", dto.WebEventTypes.EVENTSHARE, "share"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.event.String())
		})
	}
}

func TestEventType_IsValid(t *testing.T) {
	assert.True(t, dto.WebEventTypes.EVENTHELLO.IsValid())
	assert.True(t, dto.WebEventTypes.EVENTUPDATING.IsValid())
	assert.True(t, dto.WebEventTypes.EVENTVOLUMES.IsValid())
	assert.True(t, dto.WebEventTypes.EVENTHEARTBEAT.IsValid())
	assert.True(t, dto.WebEventTypes.EVENTSHARE.IsValid())
}

func TestEventType_MarshalJSON(t *testing.T) {
	event := dto.WebEventTypes.EVENTHELLO
	data, err := json.Marshal(event)
	assert.NoError(t, err)
	assert.Equal(t, `"hello"`, string(data))
}

func TestEventType_UnmarshalJSON(t *testing.T) {
	var event dto.WebEventType
	err := json.Unmarshal([]byte(`"updating"`), &event)
	assert.NoError(t, err)
	assert.Equal(t, dto.WebEventTypes.EVENTUPDATING, event)
}

func TestEventType_ParseEventType(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected dto.WebEventType
		hasError bool
	}{
		{"String hello", "hello", dto.WebEventTypes.EVENTHELLO, false},
		{"String updating", "updating", dto.WebEventTypes.EVENTUPDATING, false},
		{"Bytes", []byte("volumes"), dto.WebEventTypes.EVENTVOLUMES, false},
		{"EventType type", dto.WebEventTypes.EVENTSHARE, dto.WebEventTypes.EVENTSHARE, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dto.ParseWebEventType(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEventType_MarshalText(t *testing.T) {
	event := dto.WebEventTypes.EVENTVOLUMES
	data, err := event.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, `"volumes"`, string(data))
}

func TestEventType_UnmarshalText(t *testing.T) {
	var event dto.WebEventType
	err := event.UnmarshalText([]byte("heartbeat"))
	assert.NoError(t, err)
	assert.Equal(t, dto.WebEventTypes.EVENTHEARTBEAT, event)
}

func TestEventType_MarshalBinary(t *testing.T) {
	event := dto.WebEventTypes.EVENTSHARE
	data, err := event.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, `"share"`, string(data))
}

func TestEventType_UnmarshalBinary(t *testing.T) {
	var event dto.WebEventType
	err := event.UnmarshalBinary([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, dto.WebEventTypes.EVENTHELLO, event)
}

func TestEventType_Value(t *testing.T) {
	event := dto.WebEventTypes.EVENTUPDATING
	val, err := event.Value()
	assert.NoError(t, err)
	assert.Equal(t, "updating", val)
}

func TestEventType_All(t *testing.T) {
	count := 0
	for range dto.WebEventTypes.All() {
		count++
	}
	assert.Equal(t, 6, count)
}

func TestEventType_MarshalYAML(t *testing.T) {
	event := dto.WebEventTypes.EVENTHELLO
	data, err := event.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, []byte("hello"), data)
}

func TestEventType_UnmarshalYAML(t *testing.T) {
	var event dto.WebEventType
	err := event.UnmarshalYAML([]byte("share"))
	assert.NoError(t, err)
	assert.Equal(t, dto.WebEventTypes.EVENTSHARE, event)
}

func TestEventType_Scan(t *testing.T) {
	var event dto.WebEventType
	err := event.Scan("volumes")
	assert.NoError(t, err)
	assert.Equal(t, dto.WebEventTypes.EVENTVOLUMES, event)
}

// UpdateProcessState tests
func TestUpdateProcessState_String(t *testing.T) {
	tests := []struct {
		name     string
		state    dto.UpdateProcessState
		expected string
	}{
		{"Idle", dto.UpdateProcessStates.UPDATESTATUSIDLE, "UpdateStatusIdle"},
		{"Checking", dto.UpdateProcessStates.UPDATESTATUSCHECKING, "UpdateStatusChecking"},
		{"NoUpgrade", dto.UpdateProcessStates.UPDATESTATUSNOUPGRDE, "UpdateStatusNoUpgrde"},
		{"Available", dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE, "UpdateStatusUpgradeAvailable"},
		{"Downloading", dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, "UpdateStatusDownloading"},
		{"Downloaded", dto.UpdateProcessStates.UPDATESTATUSDOWNLOADCOMPLETE, "UpdateStatusDownloadComplete"},
		{"Extracting", dto.UpdateProcessStates.UPDATESTATUSEXTRACTING, "UpdateStatusExtracting"},
		{"Extracted", dto.UpdateProcessStates.UPDATESTATUSEXTRACTCOMPLETE, "UpdateStatusExtractComplete"},
		{"Installing", dto.UpdateProcessStates.UPDATESTATUSINSTALLING, "UpdateStatusInstalling"},
		{"NeedRestart", dto.UpdateProcessStates.UPDATESTATUSINSTALLCOMPLETE, "NeedRestart"},
		{"Error", dto.UpdateProcessStates.UPDATESTATUSERROR, "UpdateStatusError"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestUpdateProcessState_Name(t *testing.T) {
	tests := []struct {
		name     string
		state    dto.UpdateProcessState
		expected string
	}{
		{"Idle", dto.UpdateProcessStates.UPDATESTATUSIDLE, "Idle"},
		{"Checking", dto.UpdateProcessStates.UPDATESTATUSCHECKING, "Checking"},
		{"NoUpgrade", dto.UpdateProcessStates.UPDATESTATUSNOUPGRDE, "NoUpgrade"},
		{"Available", dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE, "Available"},
		{"Downloading", dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, "Downloading"},
		{"Downloaded", dto.UpdateProcessStates.UPDATESTATUSDOWNLOADCOMPLETE, "Downloaded"},
		{"Extracting", dto.UpdateProcessStates.UPDATESTATUSEXTRACTING, "Extractiong"},
		{"Extracted", dto.UpdateProcessStates.UPDATESTATUSEXTRACTCOMPLETE, "Extracted"},
		{"Installing", dto.UpdateProcessStates.UPDATESTATUSINSTALLING, "Installing"},
		{"NeedRestart", dto.UpdateProcessStates.UPDATESTATUSINSTALLCOMPLETE, "(Ready for restart)"},
		{"Error", dto.UpdateProcessStates.UPDATESTATUSERROR, "Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.Name)
		})
	}
}

func TestUpdateProcessState_IsValid(t *testing.T) {
	assert.True(t, dto.UpdateProcessStates.UPDATESTATUSIDLE.IsValid())
	assert.True(t, dto.UpdateProcessStates.UPDATESTATUSCHECKING.IsValid())
	assert.True(t, dto.UpdateProcessStates.UPDATESTATUSNOUPGRDE.IsValid())
	assert.True(t, dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE.IsValid())
	assert.True(t, dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING.IsValid())
	assert.True(t, dto.UpdateProcessStates.UPDATESTATUSDOWNLOADCOMPLETE.IsValid())
	assert.True(t, dto.UpdateProcessStates.UPDATESTATUSEXTRACTING.IsValid())
	assert.True(t, dto.UpdateProcessStates.UPDATESTATUSEXTRACTCOMPLETE.IsValid())
	assert.True(t, dto.UpdateProcessStates.UPDATESTATUSINSTALLING.IsValid())
	assert.True(t, dto.UpdateProcessStates.UPDATESTATUSINSTALLCOMPLETE.IsValid())
	assert.True(t, dto.UpdateProcessStates.UPDATESTATUSERROR.IsValid())
}

func TestUpdateProcessState_MarshalJSON(t *testing.T) {
	state := dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING
	data, err := json.Marshal(state)
	assert.NoError(t, err)
	assert.Equal(t, `"UpdateStatusDownloading"`, string(data))
}

func TestUpdateProcessState_UnmarshalJSON(t *testing.T) {
	var state dto.UpdateProcessState
	err := json.Unmarshal([]byte(`"UpdateStatusInstalling"`), &state)
	assert.NoError(t, err)
	assert.Equal(t, dto.UpdateProcessStates.UPDATESTATUSINSTALLING, state)
}

func TestUpdateProcessState_ParseUpdateProcessState(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected dto.UpdateProcessState
		hasError bool
	}{
		{"String Idle", "UpdateStatusIdle", dto.UpdateProcessStates.UPDATESTATUSIDLE, false},
		{"String Checking", "UpdateStatusChecking", dto.UpdateProcessStates.UPDATESTATUSCHECKING, false},
		{"Bytes", []byte("UpdateStatusError"), dto.UpdateProcessStates.UPDATESTATUSERROR, false},
		{"UpdateProcessState type", dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, dto.UpdateProcessStates.UPDATESTATUSDOWNLOADING, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dto.ParseUpdateProcessState(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestUpdateProcessState_MarshalText(t *testing.T) {
	state := dto.UpdateProcessStates.UPDATESTATUSEXTRACTCOMPLETE
	data, err := state.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, `"UpdateStatusExtractComplete"`, string(data))
}

func TestUpdateProcessState_UnmarshalText(t *testing.T) {
	var state dto.UpdateProcessState
	err := state.UnmarshalText([]byte("UpdateStatusUpgradeAvailable"))
	assert.NoError(t, err)
	assert.Equal(t, dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE, state)
}

func TestUpdateProcessState_MarshalBinary(t *testing.T) {
	state := dto.UpdateProcessStates.UPDATESTATUSNOUPGRDE
	data, err := state.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, `"UpdateStatusNoUpgrde"`, string(data))
}

func TestUpdateProcessState_UnmarshalBinary(t *testing.T) {
	var state dto.UpdateProcessState
	err := state.UnmarshalBinary([]byte("NeedRestart"))
	assert.NoError(t, err)
	assert.Equal(t, dto.UpdateProcessStates.UPDATESTATUSINSTALLCOMPLETE, state)
}

func TestUpdateProcessState_Value(t *testing.T) {
	state := dto.UpdateProcessStates.UPDATESTATUSCHECKING
	val, err := state.Value()
	assert.NoError(t, err)
	assert.Equal(t, "UpdateStatusChecking", val)
}

func TestUpdateProcessState_All(t *testing.T) {
	count := 0
	for range dto.UpdateProcessStates.All() {
		count++
	}
	assert.Equal(t, 11, count)
}

func TestUpdateProcessState_MarshalYAML(t *testing.T) {
	state := dto.UpdateProcessStates.UPDATESTATUSDOWNLOADCOMPLETE
	data, err := state.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, []byte("UpdateStatusDownloadComplete"), data)
}

func TestUpdateProcessState_UnmarshalYAML(t *testing.T) {
	var state dto.UpdateProcessState
	err := state.UnmarshalYAML([]byte("UpdateStatusExtracting"))
	assert.NoError(t, err)
	assert.Equal(t, dto.UpdateProcessStates.UPDATESTATUSEXTRACTING, state)
}

func TestUpdateProcessState_Scan(t *testing.T) {
	var state dto.UpdateProcessState
	err := state.Scan("UpdateStatusInstalling")
	assert.NoError(t, err)
	assert.Equal(t, dto.UpdateProcessStates.UPDATESTATUSINSTALLING, state)
}

// SmartAttributeCode tests
func TestSmartAttributeCode_String(t *testing.T) {
	tests := []struct {
		name     string
		attr     dto.SmartAttributeCode
		expected string
	}{
		{"Undefined", dto.SmartAttributeCodes.SMARTATTRIBUTEUNDEFINED, "Undefined"},
		{"Temperature", dto.SmartAttributeCodes.SMARTATTRTEMPERATURECELSIUS, "Temperature"},
		{"PowerOnHours", dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS, "PowerOnHours"},
		{"PowerCycleCount", dto.SmartAttributeCodes.SMARTATTRPOWERCYCLECOUNT, "PowerCycleCount"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.attr.String())
		})
	}
}

func TestSmartAttributeCode_Fields(t *testing.T) {
	tests := []struct {
		name         string
		attr         dto.SmartAttributeCode
		expectedCode int
		expectedType string
	}{
		{"Undefined", dto.SmartAttributeCodes.SMARTATTRIBUTEUNDEFINED, 0, "Unknown"},
		{"Temperature", dto.SmartAttributeCodes.SMARTATTRTEMPERATURECELSIUS, 194, "Old_age"},
		{"PowerOnHours", dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS, 9, "Old_age"},
		{"PowerCycleCount", dto.SmartAttributeCodes.SMARTATTRPOWERCYCLECOUNT, 12, "Old_age"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedCode, tt.attr.Code)
			assert.Equal(t, tt.expectedType, tt.attr.Type)
		})
	}
}

func TestSmartAttributeCode_IsValid(t *testing.T) {
	assert.True(t, dto.SmartAttributeCodes.SMARTATTRIBUTEUNDEFINED.IsValid())
	assert.True(t, dto.SmartAttributeCodes.SMARTATTRTEMPERATURECELSIUS.IsValid())
	assert.True(t, dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS.IsValid())
	assert.True(t, dto.SmartAttributeCodes.SMARTATTRPOWERCYCLECOUNT.IsValid())
}

func TestSmartAttributeCode_MarshalJSON(t *testing.T) {
	attr := dto.SmartAttributeCodes.SMARTATTRTEMPERATURECELSIUS
	data, err := json.Marshal(attr)
	assert.NoError(t, err)
	assert.Equal(t, `"Temperature"`, string(data))
}

func TestSmartAttributeCode_UnmarshalJSON(t *testing.T) {
	var attr dto.SmartAttributeCode
	err := json.Unmarshal([]byte(`"PowerOnHours"`), &attr)
	assert.NoError(t, err)
	assert.Equal(t, dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS, attr)
}

func TestSmartAttributeCode_ParseSmartAttributeCode(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected dto.SmartAttributeCode
		hasError bool
	}{
		{"String Undefined", "Undefined", dto.SmartAttributeCodes.SMARTATTRIBUTEUNDEFINED, false},
		{"String Temperature", "Temperature", dto.SmartAttributeCodes.SMARTATTRTEMPERATURECELSIUS, false},
		{"Bytes", []byte("PowerCycleCount"), dto.SmartAttributeCodes.SMARTATTRPOWERCYCLECOUNT, false},
		{"SmartAttributeCode type", dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS, dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dto.ParseSmartAttributeCode(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSmartAttributeCode_MarshalText(t *testing.T) {
	attr := dto.SmartAttributeCodes.SMARTATTRPOWERCYCLECOUNT
	data, err := attr.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, `"PowerCycleCount"`, string(data))
}

func TestSmartAttributeCode_UnmarshalText(t *testing.T) {
	var attr dto.SmartAttributeCode
	err := attr.UnmarshalText([]byte("Temperature"))
	assert.NoError(t, err)
	assert.Equal(t, dto.SmartAttributeCodes.SMARTATTRTEMPERATURECELSIUS, attr)
}

func TestSmartAttributeCode_MarshalBinary(t *testing.T) {
	attr := dto.SmartAttributeCodes.SMARTATTRIBUTEUNDEFINED
	data, err := attr.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, `"Undefined"`, string(data))
}

func TestSmartAttributeCode_UnmarshalBinary(t *testing.T) {
	var attr dto.SmartAttributeCode
	err := attr.UnmarshalBinary([]byte("PowerOnHours"))
	assert.NoError(t, err)
	assert.Equal(t, dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS, attr)
}

func TestSmartAttributeCode_Value(t *testing.T) {
	attr := dto.SmartAttributeCodes.SMARTATTRTEMPERATURECELSIUS
	val, err := attr.Value()
	assert.NoError(t, err)
	assert.Equal(t, "Temperature", val)
}

func TestSmartAttributeCode_All(t *testing.T) {
	count := 0
	for range dto.SmartAttributeCodes.All() {
		count++
	}
	assert.Equal(t, 4, count)
}

func TestSmartAttributeCode_MarshalYAML(t *testing.T) {
	attr := dto.SmartAttributeCodes.SMARTATTRPOWERONHOURS
	data, err := attr.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, []byte("PowerOnHours"), data)
}

func TestSmartAttributeCode_UnmarshalYAML(t *testing.T) {
	var attr dto.SmartAttributeCode
	err := attr.UnmarshalYAML([]byte("PowerCycleCount"))
	assert.NoError(t, err)
	assert.Equal(t, dto.SmartAttributeCodes.SMARTATTRPOWERCYCLECOUNT, attr)
}

func TestSmartAttributeCode_Scan(t *testing.T) {
	var attr dto.SmartAttributeCode
	err := attr.Scan("Temperature")
	assert.NoError(t, err)
	assert.Equal(t, dto.SmartAttributeCodes.SMARTATTRTEMPERATURECELSIUS, attr)
}
