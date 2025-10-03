package dbom

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSambaUserCreation(t *testing.T) {
	now := time.Now()
	user := SambaUser{
		Username:  "testuser",
		Password:  "password123",
		IsAdmin:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "password123", user.Password)
	assert.False(t, user.IsAdmin)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}

func TestSambaUserAdminFlag(t *testing.T) {
	adminUser := SambaUser{
		Username: "admin",
		Password: "adminpass",
		IsAdmin:  true,
	}

	regularUser := SambaUser{
		Username: "user",
		Password: "userpass",
		IsAdmin:  false,
	}

	assert.True(t, adminUser.IsAdmin)
	assert.False(t, regularUser.IsAdmin)
}

func TestSambaUserWithShares(t *testing.T) {
	rwShares := []ExportedShare{
		{Name: "share1"},
		{Name: "share2"},
	}

	roShares := []ExportedShare{
		{Name: "readonly1"},
	}

	user := SambaUser{
		Username: "shareuser",
		Password: "pass",
		RwShares: rwShares,
		RoShares: roShares,
	}

	assert.Len(t, user.RwShares, 2)
	assert.Len(t, user.RoShares, 1)
	assert.Equal(t, "share1", user.RwShares[0].Name)
	assert.Equal(t, "readonly1", user.RoShares[0].Name)
}

func TestSambaUserEmptyShares(t *testing.T) {
	user := SambaUser{
		Username: "noshares",
		Password: "pass",
	}

	assert.Empty(t, user.RwShares)
	assert.Empty(t, user.RoShares)
}

func TestSambaUserMultipleShares(t *testing.T) {
	rwShares := []ExportedShare{
		{Name: "documents"},
		{Name: "media"},
		{Name: "backup"},
		{Name: "projects"},
	}

	roShares := []ExportedShare{
		{Name: "public"},
		{Name: "archives"},
	}

	user := SambaUser{
		Username: "poweruser",
		Password: "pass123",
		RwShares: rwShares,
		RoShares: roShares,
		IsAdmin:  false,
	}

	assert.Len(t, user.RwShares, 4)
	assert.Len(t, user.RoShares, 2)
	assert.Contains(t, []string{"documents", "media", "backup", "projects"}, user.RwShares[0].Name)
}

func TestSambaUserPasswordUpdate(t *testing.T) {
	user := SambaUser{
		Username: "testuser",
		Password: "oldpass",
	}

	assert.Equal(t, "oldpass", user.Password)

	// Simulate password change
	user.Password = "newpass"
	assert.Equal(t, "newpass", user.Password)
}

func TestSambaUsersSlice(t *testing.T) {
	users := SambaUsers{
		{Username: "user1", IsAdmin: false},
		{Username: "user2", IsAdmin: false},
		{Username: "admin", IsAdmin: true},
	}

	assert.Len(t, users, 3)
	assert.Equal(t, "user1", users[0].Username)
	assert.Equal(t, "admin", users[2].Username)
	assert.True(t, users[2].IsAdmin)
}

func TestSambaUserEmptyPassword(t *testing.T) {
	user := SambaUser{
		Username: "nopass",
		Password: "",
	}

	assert.Empty(t, user.Password)
}

func TestSambaUserTimestamps(t *testing.T) {
	now := time.Now()
	user := SambaUser{
		Username:  "timestamptest",
		Password:  "pass",
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, now.Unix(), user.CreatedAt.Unix())
	assert.Equal(t, now.Unix(), user.UpdatedAt.Unix())
}
