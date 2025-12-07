package dto_test

import (
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

// HAMountUsage Tests
func TestHAMountUsage_Constants(t *testing.T) {
	tests := []struct {
		name     string
		usage    dto.HAMountUsage
		expected string
	}{
		{"None", dto.UsageAsNone, "none"},
		{"Backup", dto.UsageAsBackup, "backup"},
		{"Media", dto.UsageAsMedia, "media"},
		{"Share", dto.UsageAsShare, "share"},
		{"Internal", dto.UsageAsInternal, "internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, dto.HAMountUsage(tt.expected), tt.usage)
		})
	}
}

func TestHAMountUsage_StringConversion(t *testing.T) {
	usage := dto.UsageAsBackup
	assert.Equal(t, "backup", string(usage))
}

// ResolutionIssue Tests
func TestResolutionIssue_AllFields(t *testing.T) {
	now := time.Now()
	issue := dto.ResolutionIssue{
		Type:        "network",
		Context:     "connection_failed",
		Reference:   "REF-001",
		Suggestion:  "Check network configuration",
		Unhealthy:   true,
		Unsupported: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, "network", issue.Type)
	assert.Equal(t, "connection_failed", issue.Context)
	assert.Equal(t, "REF-001", issue.Reference)
	assert.Equal(t, "Check network configuration", issue.Suggestion)
	assert.True(t, issue.Unhealthy)
	assert.False(t, issue.Unsupported)
	assert.Equal(t, now, issue.CreatedAt)
	assert.Equal(t, now, issue.UpdatedAt)
}

func TestResolutionIssue_MinimalFields(t *testing.T) {
	issue := dto.ResolutionIssue{
		Type:    "error",
		Context: "unknown",
	}

	assert.Equal(t, "error", issue.Type)
	assert.Equal(t, "unknown", issue.Context)
	assert.Empty(t, issue.Reference)
	assert.Empty(t, issue.Suggestion)
	assert.False(t, issue.Unhealthy)
	assert.False(t, issue.Unsupported)
}

func TestResolutionIssue_UnhealthyAndUnsupported(t *testing.T) {
	issue := dto.ResolutionIssue{
		Type:        "compatibility",
		Context:     "deprecated_feature",
		Unhealthy:   true,
		Unsupported: true,
	}

	assert.True(t, issue.Unhealthy)
	assert.True(t, issue.Unsupported)
	assert.Equal(t, "compatibility", issue.Type)
	assert.Equal(t, "deprecated_feature", issue.Context)
}

// SmbConf Tests
func TestSmbConf_BasicData(t *testing.T) {
	conf := dto.SmbConf{
		Data: "[global]\nworkgroup = WORKGROUP\n",
	}

	assert.NotEmpty(t, conf.Data)
	assert.Contains(t, conf.Data, "[global]")
	assert.Contains(t, conf.Data, "workgroup")
}

func TestSmbConf_EmptyData(t *testing.T) {
	conf := dto.SmbConf{}

	assert.Empty(t, conf.Data)
}

func TestSmbConf_ComplexConfig(t *testing.T) {
	configData := `[global]
workgroup = WORKGROUP
server string = Samba Server
netbios name = myserver

[share1]
path = /mnt/share1
read only = no
guest ok = yes
`

	conf := dto.SmbConf{Data: configData}

	assert.Contains(t, conf.Data, "[global]")
	assert.Contains(t, conf.Data, "[share1]")
	assert.Contains(t, conf.Data, "path = /mnt/share1")
	assert.Contains(t, conf.Data, "read only = no")
	assert.Contains(t, conf.Data, "guest ok = yes")
}

// ContextState Tests
func TestContextState_AllFields(t *testing.T) {
	startTime := time.Now()
	template := []byte("template data")

	ctx := dto.ContextState{
		AddonIpAddress:  "192.168.1.100",
		ReadOnlyMode:    false,
		ProtectedMode:   true,
		SecureMode:      true,
		HACoreReady:     true,
		UpdateFilePath:  "/tmp/update.tar",
		UpdateChannel:   dto.UpdateChannels.RELEASE,
		SambaConfigFile: "/etc/samba/smb.conf",
		Template:        template,
		DockerInterface: "docker0",
		DockerNet:       "172.17.0.0/16",
		Heartbeat:       30,
		SupervisorURL:   "http://supervisor/api",
		SupervisorToken: "token123",
		DatabasePath:    "/data/db.sqlite",
		StartTime:       startTime,
	}

	assert.Equal(t, "192.168.1.100", ctx.AddonIpAddress)
	assert.False(t, ctx.ReadOnlyMode)
	assert.True(t, ctx.ProtectedMode)
	assert.True(t, ctx.SecureMode)
	assert.True(t, ctx.HACoreReady)
	assert.Equal(t, "/tmp/update.tar", ctx.UpdateFilePath)
	assert.Equal(t, dto.UpdateChannels.RELEASE, ctx.UpdateChannel)
	assert.Equal(t, "/etc/samba/smb.conf", ctx.SambaConfigFile)
	assert.NotNil(t, ctx.Template)
	assert.Equal(t, "docker0", ctx.DockerInterface)
	assert.Equal(t, "172.17.0.0/16", ctx.DockerNet)
	assert.Equal(t, 30, ctx.Heartbeat)
	assert.Equal(t, "http://supervisor/api", ctx.SupervisorURL)
	assert.Equal(t, "token123", ctx.SupervisorToken)
	assert.Equal(t, "/data/db.sqlite", ctx.DatabasePath)
	assert.Equal(t, startTime, ctx.StartTime)
}

func TestContextState_ZeroValues(t *testing.T) {
	ctx := dto.ContextState{}

	assert.Empty(t, ctx.AddonIpAddress)
	assert.False(t, ctx.ReadOnlyMode)
	assert.False(t, ctx.ProtectedMode)
	assert.False(t, ctx.SecureMode)
	assert.False(t, ctx.HACoreReady)
	assert.Empty(t, ctx.UpdateFilePath)
	assert.Nil(t, ctx.Template)
	assert.Zero(t, ctx.Heartbeat)
	assert.Zero(t, ctx.StartTime)
}

func TestContextState_ModesEnabled(t *testing.T) {
	ctx := dto.ContextState{
		ReadOnlyMode:  true,
		ProtectedMode: true,
		SecureMode:    true,
	}

	assert.True(t, ctx.ReadOnlyMode)
	assert.True(t, ctx.ProtectedMode)
	assert.True(t, ctx.SecureMode)
}

func TestContextState_UpdateChannels(t *testing.T) {
	channels := []dto.UpdateChannel{
		dto.UpdateChannels.RELEASE,
		dto.UpdateChannels.PRERELEASE,
		dto.UpdateChannels.DEVELOP,
	}

	for _, channel := range channels {
		ctx := dto.ContextState{
			UpdateChannel: channel,
		}
		assert.Equal(t, channel, ctx.UpdateChannel)
	}
}

func TestContextState_TemplateData(t *testing.T) {
	templateContent := []byte(`
[global]
workgroup = {{ .Workgroup }}
server string = {{ .ServerString }}
`)

	ctx := dto.ContextState{
		Template: templateContent,
	}

	assert.NotNil(t, ctx.Template)
	assert.NotEmpty(t, ctx.Template)
	assert.Contains(t, string(ctx.Template), "{{ .Workgroup }}")
}

// DataDirtyTracker Tests
func TestDataDirtyTracker_AllClean(t *testing.T) {
	tracker := dto.DataDirtyTracker{
		Shares:   false,
		Users:    false,
		Settings: false,
	}

	assert.False(t, tracker.Shares)
	assert.False(t, tracker.Users)
	assert.False(t, tracker.Settings)
}

func TestDataDirtyTracker_AllDirty(t *testing.T) {
	tracker := dto.DataDirtyTracker{
		Shares:   true,
		Users:    true,
		Settings: true,
	}

	assert.True(t, tracker.Shares)
	assert.True(t, tracker.Users)
	assert.True(t, tracker.Settings)
}

func TestDataDirtyTracker_PartialDirty(t *testing.T) {
	tracker := dto.DataDirtyTracker{
		Shares:   true,
		Users:    false,
		Settings: false,
	}

	assert.True(t, tracker.Shares)
	assert.False(t, tracker.Users)
	assert.False(t, tracker.Settings)
}

// NetworkStats Tests
func TestNetworkStats_Empty(t *testing.T) {
	stats := dto.NetworkStats{}

	assert.Empty(t, stats.PerNicIO)
	assert.Zero(t, stats.Global.TotalInboundTraffic)
	assert.Zero(t, stats.Global.TotalOutboundTraffic)
}

func TestNetworkStats_WithData(t *testing.T) {
	stats := dto.NetworkStats{
		PerNicIO: []dto.NicIOStats{
			{
				DeviceName:      "eth0",
				DeviceMaxSpeed:  1000000000,
				InboundTraffic:  1024.5,
				OutboundTraffic: 512.3,
				IP:              "192.168.1.100",
				Netmask:         "255.255.255.0",
			},
		},
		Global: dto.GlobalNicStats{
			TotalInboundTraffic:  2048.7,
			TotalOutboundTraffic: 1024.6,
		},
	}

	assert.Len(t, stats.PerNicIO, 1)
	assert.Equal(t, "eth0", stats.PerNicIO[0].DeviceName)
	assert.Equal(t, int64(1000000000), stats.PerNicIO[0].DeviceMaxSpeed)
	assert.Equal(t, 1024.5, stats.PerNicIO[0].InboundTraffic)
	assert.Equal(t, 512.3, stats.PerNicIO[0].OutboundTraffic)
	assert.Equal(t, "192.168.1.100", stats.PerNicIO[0].IP)
	assert.Equal(t, "255.255.255.0", stats.PerNicIO[0].Netmask)
	assert.Equal(t, 2048.7, stats.Global.TotalInboundTraffic)
	assert.Equal(t, 1024.6, stats.Global.TotalOutboundTraffic)
}

// HealthPing Tests
func TestHealthPing_Alive(t *testing.T) {
	ping := dto.HealthPing{
		Alive:     true,
		AliveTime: 1234567890,
		Uptime:    3600,
	}

	assert.True(t, ping.Alive)
	assert.Equal(t, int64(1234567890), ping.AliveTime)
	assert.Equal(t, int64(3600), ping.Uptime)
}

func TestHealthPing_WithError(t *testing.T) {
	ping := dto.HealthPing{
		Alive:     false,
		LastError: "Connection timeout",
	}

	assert.False(t, ping.Alive)
	assert.Equal(t, "Connection timeout", ping.LastError)
}
