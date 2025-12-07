package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CustomTime Tests
func TestCustomTime_UnmarshalJSON_SmbstatusFormat(t *testing.T) {
	jsonData := `"2025-06-28T15:04:28.288225+0200"`
	var ct dto.CustomTime

	err := json.Unmarshal([]byte(jsonData), &ct)
	require.NoError(t, err)

	assert.Equal(t, 2025, ct.Year())
	assert.Equal(t, time.June, ct.Month())
	assert.Equal(t, 28, ct.Day())
	assert.Equal(t, 15, ct.Hour())
	assert.Equal(t, 4, ct.Minute())
	assert.Equal(t, 28, ct.Second())
}

func TestCustomTime_UnmarshalJSON_RFC3339Fallback(t *testing.T) {
	jsonData := `"2025-06-28T15:04:28Z"`
	var ct dto.CustomTime

	err := json.Unmarshal([]byte(jsonData), &ct)
	require.NoError(t, err)

	assert.Equal(t, 2025, ct.Year())
	assert.Equal(t, time.June, ct.Month())
	assert.Equal(t, 28, ct.Day())
}

func TestCustomTime_UnmarshalJSON_Invalid(t *testing.T) {
	jsonData := `"invalid-date-format"`
	var ct dto.CustomTime

	err := json.Unmarshal([]byte(jsonData), &ct)
	assert.Error(t, err)
}

// SambaServerID Tests
func TestSambaServerID_AllFields(t *testing.T) {
	serverID := dto.SambaServerID{
		PID:      "12345",
		TaskID:   "task-001",
		VNN:      "vnn-01",
		UniqueID: "unique-abc123",
	}

	assert.Equal(t, "12345", serverID.PID)
	assert.Equal(t, "task-001", serverID.TaskID)
	assert.Equal(t, "vnn-01", serverID.VNN)
	assert.Equal(t, "unique-abc123", serverID.UniqueID)
}

// SambaSession Tests
func TestSambaSession_AllFields(t *testing.T) {
	now := time.Now()
	ct := dto.CustomTime{Time: now}

	session := dto.SambaSession{
		SessionID:      "session-123",
		ServerID:       dto.SambaServerID{PID: "1234"},
		UserID:         1000,
		GroupID:        1000,
		Username:       "testuser",
		Groupname:      "testgroup",
		CreationTime:   ct,
		AuthTime:       ct,
		RemoteMachine:  "192.168.1.100",
		Hostname:       "client-pc",
		SessionDialect: "SMB3_11",
	}

	session.Encryption.Cipher = "AES-128-GCM"
	session.Encryption.Degree = "full"
	session.Signing.Cipher = "AES-128-CMAC"
	session.Signing.Degree = "partial"

	assert.Equal(t, "session-123", session.SessionID)
	assert.Equal(t, "testuser", session.Username)
	assert.Equal(t, "testgroup", session.Groupname)
	assert.Equal(t, uint64(1000), session.UserID)
	assert.Equal(t, uint64(1000), session.GroupID)
	assert.Equal(t, "AES-128-GCM", session.Encryption.Cipher)
	assert.Equal(t, "full", session.Encryption.Degree)
	assert.Equal(t, "AES-128-CMAC", session.Signing.Cipher)
	assert.Equal(t, "partial", session.Signing.Degree)
	assert.Equal(t, now, session.CreationTime.Time)
	assert.Equal(t, now, session.AuthTime.Time)
	assert.Equal(t, "192.168.1.100", session.RemoteMachine)
	assert.Equal(t, "client-pc", session.Hostname)
	assert.Equal(t, "SMB3_11", session.SessionDialect)
	assert.Equal(t, "1234", session.ServerID.PID)
	assert.Empty(t, session.Channels)

}

func TestSambaSession_Channels(t *testing.T) {
	now := time.Now()
	ct := dto.CustomTime{Time: now}

	session := dto.SambaSession{
		SessionID: "session-123",
		Channels: map[string]struct {
			ChannelID     string         `json:"channel_id"`
			CreationTime  dto.CustomTime `json:"creation_time"`
			LocalAddress  string         `json:"local_address"`
			RemoteAddress string         `json:"remote_address"`
		}{
			"channel-1": {
				ChannelID:     "channel-1",
				CreationTime:  ct,
				LocalAddress:  "192.168.1.1:445",
				RemoteAddress: "192.168.1.100:51234",
			},
		},
	}

	assert.Len(t, session.Channels, 1)
	assert.Contains(t, session.Channels, "channel-1")
	assert.Equal(t, "channel-1", session.Channels["channel-1"].ChannelID)
	assert.Equal(t, now, session.Channels["channel-1"].CreationTime.Time)
	assert.Equal(t, "192.168.1.1:445", session.Channels["channel-1"].LocalAddress)
	assert.Equal(t, "192.168.1.100:51234", session.Channels["channel-1"].RemoteAddress)
	assert.Equal(t, "session-123", session.SessionID)
	assert.Len(t, session.Channels, 1)

}

// SambaTcon Tests
func TestSambaTcon_AllFields(t *testing.T) {
	now := time.Now()
	ct := dto.CustomTime{Time: now}

	tcon := dto.SambaTcon{
		TconID:      "tcon-456",
		SessionID:   "session-123",
		Share:       "share1",
		Device:      "A:",
		Service:     "samba-service",
		ServerID:    dto.SambaServerID{PID: "1234"},
		Machine:     "client-pc",
		ConnectedAt: ct,
	}

	tcon.Encryption.Cipher = "AES-256-GCM"
	tcon.Encryption.Degree = "full"
	tcon.Signing.Cipher = "AES-256-CMAC"
	tcon.Signing.Degree = "partial"

	assert.Equal(t, "tcon-456", tcon.TconID)
	assert.Equal(t, "client-pc", tcon.Machine)
	assert.Equal(t, now, tcon.ConnectedAt.Time)
	assert.Equal(t, "samba-service", tcon.Service)
	assert.Equal(t, "1234", tcon.ServerID.PID)
	assert.Equal(t, "session-123", tcon.SessionID)
	assert.Equal(t, "share1", tcon.Share)
	assert.Equal(t, "A:", tcon.Device)
	assert.Equal(t, "AES-256-GCM", tcon.Encryption.Cipher)
	assert.Equal(t, "full", tcon.Encryption.Degree)
	assert.Equal(t, "AES-256-CMAC", tcon.Signing.Cipher)
	assert.Equal(t, "partial", tcon.Signing.Degree)
}

// SambaStatus Tests
func TestSambaStatus_AllFields(t *testing.T) {
	now := time.Now()
	ct := dto.CustomTime{Time: now}

	status := dto.SambaStatus{
		Timestamp: ct,
		Version:   "4.18.0",
		SmbConf:   "/etc/samba/smb.conf",
		Sessions: map[string]dto.SambaSession{
			"session-1": {
				SessionID: "session-1",
				Username:  "user1",
			},
		},
		Tcons: map[string]dto.SambaTcon{
			"tcon-1": {
				TconID: "tcon-1",
				Share:  "share1",
			},
		},
	}

	assert.Equal(t, "4.18.0", status.Version)
	assert.Equal(t, "/etc/samba/smb.conf", status.SmbConf)
	assert.Equal(t, now, status.Timestamp.Time)
	assert.Len(t, status.Sessions, 1)
	assert.Len(t, status.Tcons, 1)
	assert.Contains(t, status.Sessions, "session-1")
	assert.Contains(t, status.Tcons, "tcon-1")
}

func TestSambaStatus_JSON(t *testing.T) {
	jsonData := `{
		"timestamp": "2025-06-28T15:04:28.288225+0200",
		"version": "4.18.0",
		"smb_conf": "/etc/samba/smb.conf",
		"sessions": {
			"session-1": {
				"session_id": "session-1",
				"server_id": {
					"pid": "1234",
					"task_id": "0",
					"vnn": "0",
					"unique_id": "abc123"
				},
				"uid": 1000,
				"gid": 1000,
				"username": "testuser",
				"groupname": "testgroup",
				"creation_time": "2025-06-28T15:04:28.288225+0200",
				"auth_time": "2025-06-28T15:04:28.288225+0200",
				"remote_machine": "192.168.1.100",
				"hostname": "client-pc",
				"session_dialect": "SMB3_11",
				"encryption": {
					"cipher": "AES-128-GCM",
					"degree": "full"
				},
				"signing": {
					"cipher": "AES-128-CMAC",
					"degree": "partial"
				},
				"channels": {}
			}
		},
		"tcons": {}
	}`

	var status dto.SambaStatus
	err := json.Unmarshal([]byte(jsonData), &status)
	require.NoError(t, err)

	assert.Equal(t, "4.18.0", status.Version)
	assert.Len(t, status.Sessions, 1)
	assert.Equal(t, "testuser", status.Sessions["session-1"].Username)
	assert.Equal(t, "AES-128-GCM", status.Sessions["session-1"].Encryption.Cipher)
	assert.Equal(t, "AES-128-CMAC", status.Sessions["session-1"].Signing.Cipher)
}
