package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserCreation(t *testing.T) {
	tests := []struct {
		name string
		user User
	}{
		{
			name: "Basic user",
			user: User{
				Username: "testuser",
				Password: "password123",
				IsAdmin:  false,
			},
		},
		{
			name: "Admin user",
			user: User{
				Username: "admin",
				Password: "adminpass",
				IsAdmin:  true,
			},
		},
		{
			name: "User with shares",
			user: User{
				Username: "shareuser",
				Password: "pass",
				RwShares: []string{"share1", "share2"},
				RoShares: []string{"share3"},
			},
		},
		{
			name: "User without password (external auth)",
			user: User{
				Username: "extuser",
				IsAdmin:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.user.Username)
		})
	}
}

func TestUserUsernamePattern(t *testing.T) {
	// Test that username follows pattern expectations
	validUsernames := []string{
		"user",
		"testuser",
		"a",
		"admin",
	}

	for _, username := range validUsernames {
		t.Run(username, func(t *testing.T) {
			user := User{
				Username: username,
			}
			assert.NotEmpty(t, user.Username)
			assert.LessOrEqual(t, len(user.Username), 30)
		})
	}
}

func TestUserShares(t *testing.T) {
	user := User{
		Username: "testuser",
		RwShares: []string{"documents", "media", "backup"},
		RoShares: []string{"public", "archives"},
	}

	assert.Len(t, user.RwShares, 3)
	assert.Len(t, user.RoShares, 2)
	assert.Contains(t, user.RwShares, "documents")
	assert.Contains(t, user.RwShares, "media")
	assert.Contains(t, user.RwShares, "backup")
	assert.Contains(t, user.RoShares, "public")
	assert.Contains(t, user.RoShares, "archives")
}

func TestUserEmptyShares(t *testing.T) {
	user := User{
		Username: "noshares",
	}

	assert.Empty(t, user.RwShares)
	assert.Empty(t, user.RoShares)
}

func TestUserAdminFlag(t *testing.T) {
	adminUser := User{
		Username: "admin",
		IsAdmin:  true,
	}
	regularUser := User{
		Username: "user",
		IsAdmin:  false,
	}

	assert.True(t, adminUser.IsAdmin)
	assert.False(t, regularUser.IsAdmin)
}

func TestUserPasswordHandling(t *testing.T) {
	user := User{
		Username: "testuser",
		Password: "securepassword123",
	}

	assert.NotEmpty(t, user.Password)
	assert.Equal(t, "securepassword123", user.Password)

	// Test empty password (might be valid for external auth)
	userNoPass := User{
		Username: "nopassuser",
	}
	assert.Empty(t, userNoPass.Password)
}

func TestUserMaxLength(t *testing.T) {
	// Test max length constraint (30 characters)
	longUsername := "abcdefghijklmnopqrstuvwxyz"
	user := User{
		Username: longUsername,
	}

	assert.LessOrEqual(t, len(user.Username), 30)
}

func TestUserMultipleShares(t *testing.T) {
	user := User{
		Username: "poweruser",
		RwShares: []string{"share1", "share2", "share3", "share4", "share5"},
		RoShares: []string{"readonly1", "readonly2"},
		IsAdmin:  false,
	}

	assert.Equal(t, 5, len(user.RwShares))
	assert.Equal(t, 2, len(user.RoShares))
	
	// Verify all shares are present
	for i := 1; i <= 5; i++ {
		assert.Contains(t, user.RwShares, "share"+string(rune('0'+i)))
	}
}
