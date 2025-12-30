package dto_test

import (
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

func TestMountPointData_Fields(t *testing.T) {
	diskLabel := "Data"
	diskSerial := "S123456"
	diskSize := uint64(1000000000)
	fstype := "ext4"
	invalidError := "mount error"
	warnings := "warning message"
	isToMount := true
	isWriteSupported := true
	tmSupport := dto.TimeMachineSupports.SUPPORTED

	mountData := dto.MountPointData{
		DiskLabel:  &diskLabel,
		DiskSerial: &diskSerial,
		DiskSize:   &diskSize,
		Path:       "/mnt/data",
		//PathHash:           "hash123",
		Type:               "HOST",
		FSType:             &fstype,
		DeviceId:           "/dev/sda1",
		IsMounted:          true,
		IsInvalid:          false,
		IsToMountAtStartup: &isToMount,
		IsWriteSupported:   &isWriteSupported,
		TimeMachineSupport: &tmSupport,
		InvalidError:       &invalidError,
		Warnings:           &warnings,
	}

	assert.Equal(t, diskLabel, *mountData.DiskLabel)
	assert.Equal(t, diskSerial, *mountData.DiskSerial)
	assert.Equal(t, diskSize, *mountData.DiskSize)
	assert.Equal(t, "/mnt/data", mountData.Path)
	//assert.Equal(t, "hash123", mountData.PathHash)
	assert.Equal(t, "HOST", mountData.Type)
	assert.Equal(t, fstype, *mountData.FSType)
	assert.Equal(t, "/dev/sda1", mountData.DeviceId)
	assert.True(t, mountData.IsMounted)
	assert.False(t, mountData.IsInvalid)
	assert.True(t, *mountData.IsToMountAtStartup)
	assert.True(t, *mountData.IsWriteSupported)
	assert.Equal(t, tmSupport, *mountData.TimeMachineSupport)
	assert.Equal(t, invalidError, *mountData.InvalidError)
	assert.Equal(t, warnings, *mountData.Warnings)
}

func TestMountPointData_ZeroValues(t *testing.T) {
	mountData := dto.MountPointData{}

	assert.Nil(t, mountData.DiskLabel)
	assert.Nil(t, mountData.DiskSerial)
	assert.Nil(t, mountData.DiskSize)
	assert.Empty(t, mountData.Path)
	//assert.Empty(t, mountData.PathHash)
	assert.Empty(t, mountData.Type)
	assert.Nil(t, mountData.FSType)
	assert.Empty(t, mountData.DeviceId)
	assert.False(t, mountData.IsMounted)
	assert.False(t, mountData.IsInvalid)
	assert.Nil(t, mountData.IsToMountAtStartup)
	assert.Nil(t, mountData.IsWriteSupported)
	assert.Nil(t, mountData.TimeMachineSupport)
	assert.Nil(t, mountData.InvalidError)
	assert.Nil(t, mountData.Warnings)
	assert.Nil(t, mountData.Share)
}

func TestMountPointData_WithPartition(t *testing.T) {
	partitionName := "sda1"
	partition := dto.Partition{
		LegacyDeviceName: &partitionName,
	}

	mountData := dto.MountPointData{
		Path:      "/mnt/data",
		Partition: &partition,
	}

	assert.NotNil(t, mountData.Partition)
	assert.Equal(t, "/mnt/data", mountData.Path)
	assert.Equal(t, partitionName, *mountData.Partition.LegacyDeviceName)
}

func TestMountPointData_WithShares(t *testing.T) {
	shareName := "share1"
	share := dto.SharedResource{
		Name: shareName,
	}

	mountData := dto.MountPointData{
		Path:  "/mnt/data",
		Share: &share,
	}

	assert.NotNil(t, mountData.Share)
	assert.Equal(t, "/mnt/data", mountData.Path)
	assert.Equal(t, shareName, mountData.Share.Name)
}

func TestMountPointData_Types(t *testing.T) {
	tests := []struct {
		name       string
		mountType  string
		devicePath string
	}{
		{"Host Mount", "HOST", "/dev/sda1"},
		{"Addon Mount", "ADDON", "/dev/sdb1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mountData := dto.MountPointData{
				Type:     tt.mountType,
				DeviceId: tt.devicePath,
			}
			assert.Equal(t, tt.mountType, mountData.Type)
			assert.Equal(t, tt.devicePath, mountData.DeviceId)
		})
	}
}

func TestMountPointData_InvalidState(t *testing.T) {
	errorMsg := "mount failed"
	mountData := dto.MountPointData{
		Path:         "/mnt/data",
		IsInvalid:    true,
		InvalidError: &errorMsg,
	}

	assert.True(t, mountData.IsInvalid)
	assert.Equal(t, "/mnt/data", mountData.Path)
	assert.NotNil(t, mountData.InvalidError)
	assert.Equal(t, errorMsg, *mountData.InvalidError)
}

func TestMountPointData_TimeMachineSupport(t *testing.T) {
	tests := []struct {
		name    string
		support dto.TimeMachineSupport
	}{
		{"Unsupported", dto.TimeMachineSupports.UNSUPPORTED},
		{"Supported", dto.TimeMachineSupports.SUPPORTED},
		{"Experimental", dto.TimeMachineSupports.EXPERIMENTAL},
		{"Unknown", dto.TimeMachineSupports.UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mountData := dto.MountPointData{
				TimeMachineSupport: &tt.support,
			}
			assert.Equal(t, tt.support, *mountData.TimeMachineSupport)
		})
	}
}
