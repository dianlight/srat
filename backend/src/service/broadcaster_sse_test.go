package service

import (
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestBroadcasterService_shouldSkipSSEEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    any
		expected bool
	}{
		{
			name:     "SambaStatus should be skipped",
			event:    dto.SambaStatus{},
			expected: true,
		},
		{
			name:     "SambaStatus pointer should be skipped",
			event:    &dto.SambaStatus{},
			expected: true,
		},
		{
			name:     "SambaProcessStatus should be skipped",
			event:    dto.SambaProcessStatus{},
			expected: true,
		},
		{
			name:     "SambaProcessStatus pointer should be skipped",
			event:    &dto.SambaProcessStatus{},
			expected: true,
		},
		{
			name:     "HealthPing should not be skipped",
			event:    dto.HealthPing{},
			expected: false,
		},
		{
			name:     "Disk slice should not be skipped",
			event:    []dto.Disk{},
			expected: false,
		},
		{
			name:     "String should not be skipped",
			event:    "test message",
			expected: false,
		},
	}

	broadcaster := &BroadcasterService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := broadcaster.shouldSkipSSEEvent(tt.event)
			assert.Equal(t, tt.expected, result, "shouldSkipSSEEvent() = %v, want %v", result, tt.expected)
		})
	}
}
