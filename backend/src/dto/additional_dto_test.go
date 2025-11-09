package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DataDirtyTracker Tests
func TestDataDirtyTracker_AllFields(t *testing.T) {
	tracker := dto.DataDirtyTracker{
		Shares:   true,
		Users:    false,
		Settings: false,
	}

	assert.True(t, tracker.Shares)
	assert.False(t, tracker.Users)
	assert.False(t, tracker.Settings)
}

func TestDataDirtyTracker_ZeroValues(t *testing.T) {
	tracker := dto.DataDirtyTracker{}

	assert.False(t, tracker.Shares)
	assert.False(t, tracker.Users)
	assert.False(t, tracker.Settings)
}

func TestDataDirtyTracker_JSON(t *testing.T) {
	tracker := dto.DataDirtyTracker{
		Shares:   true,
		Users:    true,
		Settings: true,
	}

	data, err := json.Marshal(tracker)
	require.NoError(t, err)

	var decoded dto.DataDirtyTracker
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, tracker, decoded)
}

// DiskHealth Tests
func TestDiskIOStats_AllFields(t *testing.T) {
	stats := dto.DiskIOStats{
		DeviceName:        "/dev/sda",
		DeviceDescription: "Samsung SSD",
		ReadIOPS:          150.5,
		WriteIOPS:         75.2,
		ReadLatency:       2.5,
		WriteLatency:      3.1,
		SmartData: &dto.SmartInfo{
			DiskType: "SATA",
			Temperature: dto.SmartTempValue{
				Value: 35,
			},
		},
	}

	assert.Equal(t, "/dev/sda", stats.DeviceName)
	assert.Equal(t, "Samsung SSD", stats.DeviceDescription)
	assert.Equal(t, 150.5, stats.ReadIOPS)
	assert.Equal(t, 75.2, stats.WriteIOPS)
	assert.NotNil(t, stats.SmartData)
}

func TestGlobalDiskStats_AllFields(t *testing.T) {
	stats := dto.GlobalDiskStats{
		TotalIOPS:         500.0,
		TotalReadLatency:  10.5,
		TotalWriteLatency: 15.2,
	}

	assert.Equal(t, 500.0, stats.TotalIOPS)
	assert.Equal(t, 10.5, stats.TotalReadLatency)
	assert.Equal(t, 15.2, stats.TotalWriteLatency)
}

func TestPerPartitionInfo_AllFields(t *testing.T) {
	info := dto.PerPartitionInfo{
		Name:          "sda1",
		MountPoint:    "/mnt/data",
		Device:        "/dev/sda1",
		FSType:        "ext4",
		FreeSpace:     1000000000,
		TotalSpace:    2000000000,
		FsckNeeded:    false,
		FsckSupported: true,
	}

	assert.Equal(t, "sda1", info.Name)
	assert.Equal(t, "/mnt/data", info.MountPoint)
	assert.Equal(t, "/dev/sda1", info.Device)
	assert.Equal(t, "ext4", info.FSType)
	assert.Equal(t, uint64(1000000000), info.FreeSpace)
	assert.Equal(t, uint64(2000000000), info.TotalSpace)
	assert.False(t, info.FsckNeeded)
	assert.True(t, info.FsckSupported)
}

func TestDiskHealth_AllFields(t *testing.T) {
	health := dto.DiskHealth{
		Global: dto.GlobalDiskStats{
			TotalIOPS:         500.0,
			TotalReadLatency:  10.5,
			TotalWriteLatency: 15.2,
		},
		PerDiskIO: []dto.DiskIOStats{
			{
				DeviceName:        "/dev/sda",
				DeviceDescription: "Samsung SSD",
				ReadIOPS:          150.5,
				WriteIOPS:         75.2,
			},
		},
		PerPartitionInfo: map[string][]dto.PerPartitionInfo{
			"/dev/sda": {
				{
					Name:       "sda1",
					MountPoint: "/mnt/data",
					Device:     "/dev/sda1",
					FSType:     "ext4",
				},
			},
		},
	}

	assert.Equal(t, 500.0, health.Global.TotalIOPS)
	assert.Len(t, health.PerDiskIO, 1)
	assert.Contains(t, health.PerPartitionInfo, "/dev/sda")
}

// NetworkStats Tests
func TestNicIOStats_AllFields(t *testing.T) {
	stats := dto.NicIOStats{
		DeviceName:      "eth0",
		DeviceMaxSpeed:  1000000000,
		InboundTraffic:  500.5,
		OutboundTraffic: 250.2,
		IP:              "192.168.1.100",
		Netmask:         "255.255.255.0",
	}

	assert.Equal(t, "eth0", stats.DeviceName)
	assert.Equal(t, int64(1000000000), stats.DeviceMaxSpeed)
	assert.Equal(t, 500.5, stats.InboundTraffic)
	assert.Equal(t, 250.2, stats.OutboundTraffic)
	assert.Equal(t, "192.168.1.100", stats.IP)
	assert.Equal(t, "255.255.255.0", stats.Netmask)
}

func TestGlobalNicStats_AllFields(t *testing.T) {
	stats := dto.GlobalNicStats{
		TotalInboundTraffic:  1500.5,
		TotalOutboundTraffic: 750.2,
	}

	assert.Equal(t, 1500.5, stats.TotalInboundTraffic)
	assert.Equal(t, 750.2, stats.TotalOutboundTraffic)
}

func TestNetworkStats_AllFields(t *testing.T) {
	stats := dto.NetworkStats{
		PerNicIO: []dto.NicIOStats{
			{
				DeviceName:      "eth0",
				DeviceMaxSpeed:  1000000000,
				InboundTraffic:  500.5,
				OutboundTraffic: 250.2,
			},
		},
		Global: dto.GlobalNicStats{
			TotalInboundTraffic:  1500.5,
			TotalOutboundTraffic: 750.2,
		},
	}

	assert.Len(t, stats.PerNicIO, 1)
	assert.Equal(t, 1500.5, stats.Global.TotalInboundTraffic)
}

// ProcessStatus Tests
func TestProcessStatus_AllFields(t *testing.T) {
	createTime := time.Now()
	status := dto.ProcessStatus{
		Pid:           1234,
		Name:          "smbd",
		CreateTime:    createTime,
		CPUPercent:    15.5,
		MemoryPercent: 10.2,
		OpenFiles:     50,
		Connections:   10,
		Status:        []string{"running"},
		IsRunning:     true,
	}

	assert.Equal(t, int32(1234), status.Pid)
	assert.Equal(t, "smbd", status.Name)
	assert.Equal(t, createTime, status.CreateTime)
	assert.Equal(t, 15.5, status.CPUPercent)
	assert.Equal(t, float32(10.2), status.MemoryPercent)
	assert.Equal(t, 50, status.OpenFiles)
	assert.Equal(t, 10, status.Connections)
	assert.Contains(t, status.Status, "running")
	assert.True(t, status.IsRunning)
}

func TestSambaProcessStatus_AllFields(t *testing.T) {
	status := dto.SambaProcessStatus{
		Smbd: dto.ProcessStatus{
			Pid:       1234,
			Name:      "smbd",
			IsRunning: true,
		},
		Nmbd: dto.ProcessStatus{
			Pid:       1235,
			Name:      "nmbd",
			IsRunning: true,
		},
		Wsdd2: dto.ProcessStatus{
			Pid:       1236,
			Name:      "wsdd2",
			IsRunning: false,
		},
		Srat: dto.ProcessStatus{
			Pid:       1237,
			Name:      "srat",
			IsRunning: true,
		},
		Hdidle: dto.ProcessStatus{
			Pid:         -1237, // Negative PID indicates subprocess of srat (PID 1237)
			Name:        "hdidle-monitor",
			IsRunning:   true,
			Connections: 3,
		},
	}

	assert.Equal(t, int32(1234), status.Smbd.Pid)
	assert.Equal(t, "smbd", status.Smbd.Name)
	assert.True(t, status.Smbd.IsRunning)
	assert.False(t, status.Wsdd2.IsRunning)
	assert.Equal(t, int32(-1237), status.Hdidle.Pid)
	assert.Equal(t, "hdidle-monitor", status.Hdidle.Name)
	assert.True(t, status.Hdidle.IsRunning)
	assert.Equal(t, 3, status.Hdidle.Connections)
}

// ReleaseAsset Tests
func TestBinaryAsset_AllFields(t *testing.T) {
	asset := dto.BinaryAsset{
		Name:               "srat-amd64",
		Size:               1024000,
		ID:                 12345,
		BrowserDownloadURL: "https://github.com/releases/v1.0.0/srat-amd64",
	}

	assert.Equal(t, "srat-amd64", asset.Name)
	assert.Equal(t, 1024000, asset.Size)
	assert.Equal(t, int64(12345), asset.ID)
	assert.Equal(t, "https://github.com/releases/v1.0.0/srat-amd64", asset.BrowserDownloadURL)
}

func TestReleaseAsset_AllFields(t *testing.T) {
	release := dto.ReleaseAsset{
		LastRelease: "v1.2.3",
		ArchAsset: dto.BinaryAsset{
			Name: "srat-amd64",
			Size: 1024000,
			ID:   12345,
		},
	}

	assert.Equal(t, "v1.2.3", release.LastRelease)
	assert.Equal(t, "srat-amd64", release.ArchAsset.Name)
	assert.Equal(t, 1024000, release.ArchAsset.Size)
}

func TestUpdateProgress_AllFields(t *testing.T) {
	progress := dto.UpdateProgress{
		ProgressStatus: dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE,
		Progress:       50,
		LastRelease:    "v1.3.0",
		ErrorMessage:   "",
	}

	assert.Equal(t, dto.UpdateProcessStates.UPDATESTATUSUPGRADEAVAILABLE, progress.ProgressStatus)
	assert.Equal(t, 50, progress.Progress)
	assert.Equal(t, "v1.3.0", progress.LastRelease)
	assert.Empty(t, progress.ErrorMessage)
}

// SystemCapabilities Tests
func TestSystemCapabilities_AllSupported(t *testing.T) {
	caps := dto.SystemCapabilities{
		SupportsQUIC:           true,
		HasKernelModule:        true,
		SambaVersion:           "4.23.1",
		SambaVersionSufficient: true,
		UnsupportedReason:      "",
	}

	assert.True(t, caps.SupportsQUIC)
	assert.True(t, caps.HasKernelModule)
	assert.Equal(t, "4.23.1", caps.SambaVersion)
	assert.True(t, caps.SambaVersionSufficient)
	assert.Empty(t, caps.UnsupportedReason)
}

func TestSystemCapabilities_NotSupported(t *testing.T) {
	caps := dto.SystemCapabilities{
		SupportsQUIC:           false,
		HasKernelModule:        false,
		SambaVersion:           "4.20.0",
		SambaVersionSufficient: false,
		UnsupportedReason:      "Samba version too old",
	}

	assert.False(t, caps.SupportsQUIC)
	assert.False(t, caps.HasKernelModule)
	assert.False(t, caps.SambaVersionSufficient)
	assert.NotEmpty(t, caps.UnsupportedReason)
}

// User Tests
func TestUser_AllFields(t *testing.T) {
	user := dto.User{
		Username: "testuser",
		Password: "secret123",
		IsAdmin:  true,
		RwShares: []string{"share1", "share2"},
		RoShares: []string{"share3"},
	}

	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "secret123", user.Password)
	assert.True(t, user.IsAdmin)
	assert.Len(t, user.RwShares, 2)
	assert.Len(t, user.RoShares, 1)
}

func TestUser_MinimalFields(t *testing.T) {
	user := dto.User{
		Username: "guest",
	}

	assert.Equal(t, "guest", user.Username)
	assert.Empty(t, user.Password)
	assert.False(t, user.IsAdmin)
	assert.Nil(t, user.RwShares)
	assert.Nil(t, user.RoShares)
}

// Welcome Tests
func TestWelcome_AllFields(t *testing.T) {
	machineID := "abc123"
	welcome := dto.Welcome{
		Message:         "Welcome to SRAT",
		ActiveClients:   5,
		SupportedEvents: []dto.WebEventType{dto.WebEventTypes.EVENTHELLO, dto.WebEventTypes.EVENTVOLUMES},
		UpdateChannel:   "Release",
		MachineId:       &machineID,
		BuildVersion:    "1.2.3",
		SecureMode:      true,
		ProtectedMode:   false,
		ReadOnly:        false,
		StartTime:       time.Now().Unix(),
	}

	assert.Equal(t, "Welcome to SRAT", welcome.Message)
	assert.Equal(t, int32(5), welcome.ActiveClients)
	assert.Len(t, welcome.SupportedEvents, 2)
	assert.Equal(t, "Release", welcome.UpdateChannel)
	assert.NotNil(t, welcome.MachineId)
	assert.Equal(t, "abc123", *welcome.MachineId)
	assert.True(t, welcome.SecureMode)
	assert.Positive(t, welcome.StartTime)
}

func TestWelcome_NilMachineID(t *testing.T) {
	welcome := dto.Welcome{
		Message:       "Welcome",
		ActiveClients: 0,
		MachineId:     nil,
	}

	assert.Nil(t, welcome.MachineId)
}

// HealthPing Tests
func TestHealthPing_AllFields(t *testing.T) {
	health := dto.HealthPing{
		Alive:     true,
		AliveTime: time.Now().Unix(),
		SambaProcessStatus: dto.SambaProcessStatus{
			Smbd: dto.ProcessStatus{
				Pid:       1234,
				IsRunning: true,
			},
		},
		LastError: "",
		Dirty: dto.DataDirtyTracker{
			Shares: true,
		},
		LastRelease: dto.ReleaseAsset{
			LastRelease: "v1.0.0",
		},
		DiskHealth: &dto.DiskHealth{
			Global: dto.GlobalDiskStats{
				TotalIOPS: 100.0,
			},
		},
		NetworkHealth: &dto.NetworkStats{
			Global: dto.GlobalNicStats{
				TotalInboundTraffic: 500.0,
			},
		},
		Uptime: 3600,
	}

	assert.True(t, health.Alive)
	assert.Positive(t, health.AliveTime)
	assert.True(t, health.SambaProcessStatus.Smbd.IsRunning)
	assert.Empty(t, health.LastError)
	assert.True(t, health.Dirty.Shares)
	assert.NotNil(t, health.DiskHealth)
	assert.NotNil(t, health.NetworkHealth)
	assert.Equal(t, int64(3600), health.Uptime)
}

func TestHealthPing_NotAlive(t *testing.T) {
	health := dto.HealthPing{
		Alive:     false,
		LastError: "Connection timeout",
	}

	assert.False(t, health.Alive)
	assert.NotEmpty(t, health.LastError)
}
