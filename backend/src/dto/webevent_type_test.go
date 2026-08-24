package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestWebEventType_String(t *testing.T) {
	tests := []struct {
		name     string
		event    dto.WebEventType
		expected string
	}{
		{"Hello", dto.WebEventTypes.EVENTHELLO, "hello"},
		{"Updating", dto.WebEventTypes.EVENTUPDATING, "updating"},
		{"Volumes", dto.WebEventTypes.EVENTVOLUMES, "volumes"},
		{"Heartbeat", dto.WebEventTypes.EVENTHEARTBEAT, "heartbeat"},
		{"Shares", dto.WebEventTypes.EVENTSHARES, "shares"},
		{"Dirty Tracker", dto.WebEventTypes.EVENTDIRTYTRACKER, "dirty_data_tracker"},
		{"Smart Test Status", dto.WebEventTypes.EVENTSMARTTESTSTATUS, "smart_test_status"},
		{"Error", dto.WebEventTypes.EVENTERROR, "error"},
		{"Repair Command", dto.WebEventTypes.EVENTREPAIRCOMMAND, "repair_command"},
		{"Problem", dto.WebEventTypes.EVENTPROBLEM, "problem"},
		{"App Config Changed", dto.WebEventTypes.EVENTAPPCONFIGCHANGED, "app_config_changed"},
		{"mDNS Register", dto.WebEventTypes.EVENTMDNSREGISTER, "mdns_register"},
		{"Command Started", dto.WebEventTypes.EVENTCOMMANDSTARTED, "command_started"},
		{"Command Output", dto.WebEventTypes.EVENTCOMMANDOUTPUT, "command_output"},
		{"Command Terminated", dto.WebEventTypes.EVENTCOMMANDTERMINATED, "command_terminated"},
		{"Rclone Task", dto.WebEventTypes.EVENTRCLONETASK, "rclone_task"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.event.String())
		})
	}
}

func TestWebEventType_IsValidEvent_ValidTypes(t *testing.T) {
	tests := []struct {
		name  string
		event any
	}{
		{"Valid Welcome", dto.Welcome{}},
		{"Valid UpdateProgress", dto.UpdateProgress{}},
		{"Valid Disk slice", []*dto.Disk{}},
		{"Valid HealthPing", dto.HealthPing{}},
		{"Valid SharedResource slice", []dto.SharedResource{}},
		{"Valid DataDirtyTracker", dto.DataDirtyTracker{}},
		{"Valid SmartTestStatus", dto.SmartTestStatus{}},
		{"Valid ErrorModel", &dto.ErrorDataModel{}},
		{"Valid RepairCommandMessage", dto.RepairCommandMessage{}},
		{"Valid Problem", dto.Problem{}},
		{"Valid AppConfigChangedNotification", dto.AppConfigChangedNotification{}},
		{"Valid MdnsRegisterNotification", dto.MdnsRegisterNotification{}},
		{"Valid CommandStartedNotification", dto.CommandStartedNotification{}},
		{"Valid CommandOutputNotification", dto.CommandOutputNotification{}},
		{"Valid CommandTerminatedNotification", dto.CommandTerminatedNotification{}},
		{"Valid RcloneTask", dto.RcloneTask{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, dto.WebEventMap.IsValidEvent(tt.event), "expected %T to be valid event", tt.event)
		})
	}
}

func TestWebEventType_IsValidEvent_InvalidTypes(t *testing.T) {
	tests := []struct {
		name  string
		event any
	}{
		{"Invalid string", "not an event"},
		{"Invalid int", 42},
		{"Invalid map", map[string]any{}},
		{"Invalid bool", true},
		{"Invalid User", dto.User{}},
		{"Invalid float64", 3.14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, dto.WebEventMap.IsValidEvent(tt.event), "expected %T to be invalid event", tt.event)
		})
	}
}

func TestWebEventMap_ContainsAllEventTypes(t *testing.T) {
	expectedKeys := []string{
		"hello",
		"updating",
		"volumes",
		"heartbeat",
		"shares",
		"dirty_data_tracker",
		"smart_test_status",
		"error",
		"repair_command",
		"problem",
		"app_config_changed",
		"mdns_register",
		"command_started",
		"command_output",
		"command_terminated",
		"filesystem_task",
		"rclone_task",
	}

	for _, key := range expectedKeys {
		assert.Contains(t, dto.WebEventMap, key, "WebEventMap should contain key: %s", key)
	}
}

func TestWebEventMap_Size(t *testing.T) {
	assert.Len(t, dto.WebEventMap, 17, "WebEventMap should contain exactly 17 event types")
}

func TestWebEventType_IsValidEvent_WithConcreteTypes(t *testing.T) {
	tests := []struct {
		name     string
		eventKey string
		event    any
		expected bool
	}{
		{
			name:     "Welcome with hello key",
			eventKey: dto.WebEventTypes.EVENTHELLO.String(),
			event:    dto.Welcome{},
			expected: true,
		},
		{
			name:     "UpdateProgress with updating key",
			eventKey: dto.WebEventTypes.EVENTUPDATING.String(),
			event:    dto.UpdateProgress{},
			expected: true,
		},
		{
			name:     "Disk slice with volumes key",
			eventKey: dto.WebEventTypes.EVENTVOLUMES.String(),
			event:    []*dto.Disk{},
			expected: true,
		},
		{
			name:     "ErrorModel with error key",
			eventKey: dto.WebEventTypes.EVENTERROR.String(),
			event:    &dto.ErrorDataModel{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify key exists in WebEventMap
			assert.Contains(t, dto.WebEventMap, tt.eventKey)
			// Verify event is valid
			assert.Equal(t, tt.expected, dto.WebEventMap.IsValidEvent(tt.event))
		})
	}
}

func TestWebEventType_EmptyValues(t *testing.T) {

	// Test with nil
	assert.False(t, dto.WebEventMap.IsValidEvent(nil), "nil should not be a valid event")

	// Test with empty struct instances
	assert.True(t, dto.WebEventMap.IsValidEvent(dto.Welcome{}), "empty Welcome should be valid")
	assert.True(t, dto.WebEventMap.IsValidEvent(dto.HealthPing{}), "empty HealthPing should be valid")
	assert.True(t, dto.WebEventMap.IsValidEvent(&dto.ErrorDataModel{}), "empty ErrorModel should be valid")
}

func TestWebEventType_RcloneTaskEnum(t *testing.T) {
	rcloneTask := dto.WebEventTypes.EVENTRCLONETASK

	// String / names map entry
	assert.Equal(t, "rclone_task", rcloneTask.String())

	// Validity map entry
	assert.True(t, rcloneTask.IsValid())

	// Parse by canonical name
	parsed, err := dto.ParseWebEventType("rclone_task")
	assert.NoError(t, err)
	assert.Equal(t, rcloneTask, parsed)

	// Parse by ordinal value
	parsedByNumber, err := dto.ParseWebEventType(16)
	assert.NoError(t, err)
	assert.Equal(t, rcloneTask, parsedByNumber)

	// Parse of unknown values yields the invalid sentinel (no error by design)
	unknown, err := dto.ParseWebEventType("not_an_event")
	assert.NoError(t, err)
	assert.False(t, unknown.IsValid())

	// allSlice / All include the new value
	assert.Contains(t, dto.WebEventTypes.All(), rcloneTask)

	// ExhaustiveWebEventTypes visits every enum value exactly once
	var seen []dto.WebEventType
	dto.ExhaustiveWebEventTypes(func(e dto.WebEventType) { seen = append(seen, e) })
	assert.Len(t, seen, 17)
	assert.Contains(t, seen, rcloneTask)

	// MarshalJSON/UnmarshalJSON round trip
	data, err := json.Marshal(rcloneTask)
	assert.NoError(t, err)
	assert.JSONEq(t, `"rclone_task"`, string(data))

	var unmarshalled dto.WebEventType
	err = json.Unmarshal(data, &unmarshalled)
	assert.NoError(t, err)
	assert.Equal(t, rcloneTask, unmarshalled)
}
