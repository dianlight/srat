package dto_test

import (
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestSettings_AllFields(t *testing.T) {
	localMaster := true
	exportStats := true
	smbOverQUIC := false

	settings := dto.Settings{
		Hostname:          "test-host",
		Workgroup:         "WORKGROUP",
		Mountoptions:      []string{"rw", "sync"},
		AllowHost:         []string{"192.168.1.0/24", "10.0.0.0/8"},
		CompatibilityMode: true,
		Interfaces:        []string{"eth0", "wlan0"},
		BindAllInterfaces: false,
		LogLevel:          "info",
		MultiChannel:      true,
		TelemetryMode:     dto.TelemetryModes.TELEMETRYMODEERRORS,
		LocalMaster:       &localMaster,
		ExportStatsToHA:   &exportStats,
		SMBoverQUIC:       &smbOverQUIC,
	}

	assert.Equal(t, "test-host", settings.Hostname)
	assert.Equal(t, "WORKGROUP", settings.Workgroup)
	assert.Len(t, settings.Mountoptions, 2)
	assert.Contains(t, settings.Mountoptions, "rw")
	assert.Contains(t, settings.Mountoptions, "sync")
	assert.Len(t, settings.AllowHost, 2)
	assert.Contains(t, settings.AllowHost, "192.168.1.0/24")
	assert.True(t, settings.CompatibilityMode)
	assert.Len(t, settings.Interfaces, 2)
	assert.False(t, settings.BindAllInterfaces)
	assert.Equal(t, "info", settings.LogLevel)
	assert.True(t, settings.MultiChannel)
	assert.Equal(t, dto.TelemetryModes.TELEMETRYMODEERRORS, settings.TelemetryMode)
	assert.NotNil(t, settings.LocalMaster)
	assert.True(t, *settings.LocalMaster)
	assert.NotNil(t, settings.ExportStatsToHA)
	assert.True(t, *settings.ExportStatsToHA)
	assert.NotNil(t, settings.SMBoverQUIC)
	assert.False(t, *settings.SMBoverQUIC)
}

func TestSettings_ZeroValues(t *testing.T) {
	settings := dto.Settings{}

	assert.Empty(t, settings.Hostname)
	assert.Empty(t, settings.Workgroup)
	assert.Nil(t, settings.Mountoptions)
	assert.Nil(t, settings.AllowHost)
	assert.False(t, settings.CompatibilityMode)
	assert.Nil(t, settings.Interfaces)
	assert.False(t, settings.BindAllInterfaces)
	assert.Empty(t, settings.LogLevel)
	assert.False(t, settings.MultiChannel)
	assert.Nil(t, settings.LocalMaster)
	assert.Nil(t, settings.ExportStatsToHA)
	assert.Nil(t, settings.SMBoverQUIC)
}

func TestSettings_TelemetryModes(t *testing.T) {
	tests := []struct {
		name string
		mode dto.TelemetryMode
	}{
		{"Ask", dto.TelemetryModes.TELEMETRYMODEASK},
		{"All", dto.TelemetryModes.TELEMETRYMODEALL},
		{"Errors", dto.TelemetryModes.TELEMETRYMODEERRORS},
		{"Disabled", dto.TelemetryModes.TELEMETRYMODEDISABLED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := dto.Settings{
				TelemetryMode: tt.mode,
			}
			assert.Equal(t, tt.mode, settings.TelemetryMode)
		})
	}
}

func TestSettings_EmptySlices(t *testing.T) {
	settings := dto.Settings{
		Mountoptions: []string{},
		AllowHost:    []string{},
		Interfaces:   []string{},
	}

	assert.NotNil(t, settings.Mountoptions)
	assert.Empty(t, settings.Mountoptions)
	assert.NotNil(t, settings.AllowHost)
	assert.Empty(t, settings.AllowHost)
	assert.NotNil(t, settings.Interfaces)
	assert.Empty(t, settings.Interfaces)
}

func TestSettings_BooleanPointers(t *testing.T) {
	trueVal := true
	falseVal := false

	settings := dto.Settings{
		LocalMaster:     &trueVal,
		ExportStatsToHA: &falseVal,
		SMBoverQUIC:     &trueVal,
	}

	assert.True(t, *settings.LocalMaster)
	assert.False(t, *settings.ExportStatsToHA)
	assert.True(t, *settings.SMBoverQUIC)
}
