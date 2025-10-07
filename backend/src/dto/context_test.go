package dto_test

import (
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestContextState(t *testing.T) {
	now := time.Now()
	state := dto.ContextState{
		AddonIpAddress:  "172.30.32.1",
		ReadOnlyMode:    false,
		ProtectedMode:   true,
		SecureMode:      true,
		HACoreReady:     true,
		UpdateFilePath:  "/tmp/update.tar",
		SambaConfigFile: "/etc/samba/smb.conf",
		Template:        []byte("template content"),
		DockerInterface: "hassio",
		DockerNet:       "172.30.32.0/23",
		Heartbeat:       60,
		SupervisorURL:   "http://supervisor",
		SupervisorToken: "test-token",
		DatabasePath:    "/data/srat.db",
		StartTime:       now,
	}

	assert.Equal(t, "172.30.32.1", state.AddonIpAddress)
	assert.False(t, state.ReadOnlyMode)
	assert.True(t, state.ProtectedMode)
	assert.True(t, state.SecureMode)
	assert.True(t, state.HACoreReady)
	assert.Equal(t, "/tmp/update.tar", state.UpdateFilePath)
	assert.Equal(t, "/etc/samba/smb.conf", state.SambaConfigFile)
	assert.NotEmpty(t, state.Template)
	assert.Equal(t, "hassio", state.DockerInterface)
	assert.Equal(t, "172.30.32.0/23", state.DockerNet)
	assert.Equal(t, 60, state.Heartbeat)
	assert.Equal(t, "http://supervisor", state.SupervisorURL)
	assert.Equal(t, "test-token", state.SupervisorToken)
	assert.Equal(t, "/data/srat.db", state.DatabasePath)
	assert.Equal(t, now, state.StartTime)
}

func TestDataDirtyTracker(t *testing.T) {
	tracker := dto.DataDirtyTracker{
		Shares:   true,
		Users:    false,
		Volumes:  true,
		Settings: false,
	}

	assert.True(t, tracker.Shares)
	assert.False(t, tracker.Users)
	assert.True(t, tracker.Volumes)
	assert.False(t, tracker.Settings)
}

func TestDataDirtyTrackerAllFalse(t *testing.T) {
	tracker := dto.DataDirtyTracker{}

	assert.False(t, tracker.Shares)
	assert.False(t, tracker.Users)
	assert.False(t, tracker.Volumes)
	assert.False(t, tracker.Settings)
}

func TestDataDirtyTrackerAllTrue(t *testing.T) {
	tracker := dto.DataDirtyTracker{
		Shares:   true,
		Users:    true,
		Volumes:  true,
		Settings: true,
	}

	assert.True(t, tracker.Shares)
	assert.True(t, tracker.Users)
	assert.True(t, tracker.Volumes)
	assert.True(t, tracker.Settings)
}
