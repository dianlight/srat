package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSharedResourceCreation(t *testing.T) {
	tests := []struct {
		name     string
		resource SharedResource
	}{
		{
			name: "Empty resource",
			resource: SharedResource{
				Name: "test-share",
			},
		},
		{
			name: "Resource with basic fields",
			resource: SharedResource{
				Name:     "media-share",
				Disabled: boolPtr(false),
				GuestOk:  boolPtr(true),
			},
		},
		{
			name: "Resource with users",
			resource: SharedResource{
				Name: "user-share",
				Users: []User{
					{Username: "user1", IsAdmin: false},
					{Username: "admin1", IsAdmin: true},
				},
			},
		},
		{
			name: "Resource with time machine enabled",
			resource: SharedResource{
				Name:               "backup-share",
				TimeMachine:        boolPtr(true),
				TimeMachineMaxSize: stringPtr("1TB"),
			},
		},
		{
			name: "Resource with HA mount",
			resource: SharedResource{
				Name:  "ha-share",
				Usage: "backup",
				Status: &SharedResourceStatus{
					IsHAMounted: true,
				},
			},
		},
		{
			name: "Resource with veto files",
			resource: SharedResource{
				Name:      "protected-share",
				VetoFiles: []string{".DS_Store", "Thumbs.db", "desktop.ini"},
			},
		},
		{
			name: "Resource with recycle bin",
			resource: SharedResource{
				Name:       "recycled-share",
				RecycleBin: boolPtr(true),
			},
		},
		{
			name: "Invalid resource",
			resource: SharedResource{
				Name: "invalid-share",
				Status: &SharedResourceStatus{
					IsValid: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.resource.Name)
			if tt.resource.Users != nil {
				assert.NotNil(t, tt.resource.Users)
			}
			if tt.resource.VetoFiles != nil {
				assert.NotNil(t, tt.resource.VetoFiles)
			}
		})
	}
}

func TestSharedResourceReadOnlyUsers(t *testing.T) {
	resource := SharedResource{
		Name: "mixed-access",
		Users: []User{
			{Username: "readwrite1"},
		},
		RoUsers: []User{
			{Username: "readonly1"},
			{Username: "readonly2"},
		},
	}

	assert.Len(t, resource.Users, 1)
	assert.Len(t, resource.RoUsers, 2)
	assert.Equal(t, "readwrite1", resource.Users[0].Username)
	assert.Equal(t, "readonly1", resource.RoUsers[0].Username)
	assert.Equal(t, "readonly2", resource.RoUsers[1].Username)
	assert.Equal(t, "mixed-access", resource.Name)
}

func TestSharedResourceMountPointData(t *testing.T) {
	mountData := &MountPointData{
		Path:     "/mnt/test",
		DeviceId: "/dev/sda1",
		Type:     "HOST",
	}

	resource := SharedResource{
		Name:           "mounted-share",
		MountPointData: mountData,
	}

	assert.NotNil(t, resource.MountPointData)
	assert.Equal(t, "/mnt/test", resource.MountPointData.Path)
	assert.Equal(t, "/dev/sda1", resource.MountPointData.DeviceId)
	assert.Equal(t, "HOST", resource.MountPointData.Type)
	assert.Equal(t, "mounted-share", resource.Name)
}

func TestSharedResourceUsageTypes(t *testing.T) {
	usageTypes := []HAMountUsage{"none", "backup", "media", "share", "internal"}

	for _, usage := range usageTypes {
		t.Run(string(usage), func(t *testing.T) {
			resource := SharedResource{
				Name:  "test-share",
				Usage: usage,
			}
			assert.Equal(t, usage, resource.Usage)
			assert.Equal(t, "test-share", resource.Name)
		})
	}
}

func TestSharedResourcePointerFields(t *testing.T) {
	disabled := true
	timeMachine := false
	recycleBin := true
	guestOk := false

	resource := SharedResource{
		Name:        "pointer-test",
		Disabled:    &disabled,
		TimeMachine: &timeMachine,
		RecycleBin:  &recycleBin,
		GuestOk:     &guestOk,
		Status: &SharedResourceStatus{
			IsHAMounted: true,
			IsValid:     true,
		},
	}

	assert.NotNil(t, resource.Disabled)
	assert.True(t, *resource.Disabled)
	assert.NotNil(t, resource.TimeMachine)
	assert.False(t, *resource.TimeMachine)
	assert.NotNil(t, resource.RecycleBin)
	assert.True(t, *resource.RecycleBin)
	assert.NotNil(t, resource.GuestOk)
	assert.False(t, *resource.GuestOk)
	assert.NotNil(t, resource.Status)
	assert.True(t, resource.Status.IsHAMounted)
	assert.True(t, resource.Status.IsValid)
	assert.Equal(t, "pointer-test", resource.Name)
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}
