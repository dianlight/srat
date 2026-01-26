package dto_test

import (
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
		{"Valid ErrorModel", dto.ErrorDataModel{}},
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
	}

	for _, key := range expectedKeys {
		assert.Contains(t, dto.WebEventMap, key, "WebEventMap should contain key: %s", key)
	}
}

func TestWebEventMap_Size(t *testing.T) {
	assert.Equal(t, 8, len(dto.WebEventMap), "WebEventMap should contain exactly 8 event types")
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
			event:    dto.ErrorDataModel{},
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
	assert.True(t, dto.WebEventMap.IsValidEvent(dto.ErrorDataModel{}), "empty ErrorModel should be valid")
}
