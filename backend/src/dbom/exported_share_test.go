package dbom

import (
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestExportedShareCreation(t *testing.T) {
	now := time.Now()
	share := ExportedShare{
		Name:      "test-share",
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "test-share", share.Name)
	assert.False(t, share.CreatedAt.IsZero())
	assert.False(t, share.UpdatedAt.IsZero())
}

func TestExportedShareWithUsers(t *testing.T) {
	share := ExportedShare{
		Name: "user-share",
		Users: []SambaUser{
			{Username: "user1"},
			{Username: "user2"},
		},
		RoUsers: []SambaUser{
			{Username: "readonly1"},
		},
	}

	assert.Equal(t, "user-share", share.Name)
	assert.Len(t, share.Users, 2)
	assert.Len(t, share.RoUsers, 1)
	assert.Equal(t, "user1", share.Users[0].Username)
	assert.Equal(t, "readonly1", share.RoUsers[0].Username)
}

func TestExportedShareBooleanFields(t *testing.T) {
	disabled := true
	share := ExportedShare{
		Name:        "boolean-test",
		Disabled:    &disabled,
		TimeMachine: true,
		RecycleBin:  true,
		GuestOk:     false,
	}

	assert.Equal(t, "boolean-test", share.Name)
	assert.NotNil(t, share.Disabled)
	assert.True(t, *share.Disabled)
	assert.True(t, share.TimeMachine)
	assert.True(t, share.RecycleBin)
	assert.False(t, share.GuestOk)
}

func TestExportedShareVetoFiles(t *testing.T) {
	vetoFiles := datatypes.JSONSlice[string]{".DS_Store", "Thumbs.db", "desktop.ini"}
	share := ExportedShare{
		Name:      "veto-test",
		VetoFiles: vetoFiles,
	}

	assert.Equal(t, "veto-test", share.Name)
	assert.Len(t, share.VetoFiles, 3)
	assert.Contains(t, share.VetoFiles, ".DS_Store")
	assert.Contains(t, share.VetoFiles, "Thumbs.db")
}

func TestExportedShareTimeMachine(t *testing.T) {
	share := ExportedShare{
		Name:               "timemachine-share",
		TimeMachine:        true,
		TimeMachineMaxSize: "2TB",
	}

	assert.True(t, share.TimeMachine)
	assert.Equal(t, "2TB", share.TimeMachineMaxSize)
	assert.Equal(t, "timemachine-share", share.Name)
}

func TestExportedShareUsageTypes(t *testing.T) {
	usageTypes := []dto.HAMountUsage{"none", "backup", "media", "share", "internal"}

	for _, usage := range usageTypes {
		t.Run(string(usage), func(t *testing.T) {
			share := ExportedShare{
				Name:  "usage-test",
				Usage: usage,
			}
			assert.Equal(t, usage, share.Usage)
			assert.Equal(t, "usage-test", share.Name)
		})
	}
}

func TestExportedShareMountPointData(t *testing.T) {
	share := ExportedShare{
		Name:               "mounted-share",
		MountPointDataPath: "/mnt/data",
	}

	assert.Equal(t, "mounted-share", share.Name)
	assert.Equal(t, "/mnt/data", share.MountPointDataPath)
}

func TestExportedShareEmptyVetoFiles(t *testing.T) {
	share := ExportedShare{
		Name:      "no-veto",
		VetoFiles: datatypes.JSONSlice[string]{},
	}

	assert.Equal(t, "no-veto", share.Name)
	assert.NotNil(t, share.VetoFiles)
	assert.Empty(t, share.VetoFiles)
}

func TestExportedShareRecycleBinDefault(t *testing.T) {
	share := ExportedShare{
		Name:       "recycle-test",
		RecycleBin: false,
	}

	assert.Equal(t, "recycle-test", share.Name)
	assert.False(t, share.RecycleBin)
}

func TestExportedShareGuestOkDefault(t *testing.T) {
	share := ExportedShare{
		Name:    "guest-test",
		GuestOk: false,
	}

	assert.Equal(t, "guest-test", share.Name)
	assert.False(t, share.GuestOk)
}

func TestExportedShareMultipleUsers(t *testing.T) {
	users := []SambaUser{
		{Username: "user1", IsAdmin: false},
		{Username: "user2", IsAdmin: false},
		{Username: "user3", IsAdmin: true},
	}

	roUsers := []SambaUser{
		{Username: "ro1", IsAdmin: false},
		{Username: "ro2", IsAdmin: false},
	}

	share := ExportedShare{
		Name:    "multi-user",
		Users:   users,
		RoUsers: roUsers,
	}

	assert.Equal(t, "multi-user", share.Name)
	assert.Len(t, share.Users, 3)
	assert.Len(t, share.RoUsers, 2)

	// Verify admin status
	assert.False(t, share.Users[0].IsAdmin)
	assert.True(t, share.Users[2].IsAdmin)
}

func TestExportedShareBeforeSave_ValidName(t *testing.T) {
	share := ExportedShare{
		Name: "valid-share",
	}

	// Mock GORM DB (nil is acceptable for this test as we only check name validation)
	err := share.BeforeSave(nil)
	assert.NoError(t, err)
}

func TestExportedShareBeforeSave_EmptyName(t *testing.T) {
	share := ExportedShare{
		Name: "",
	}

	// Mock GORM DB
	err := share.BeforeSave(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid name")
}
