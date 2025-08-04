package repository

import (
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&dbom.SambaUser{})
	require.NoError(t, err)

	return db
}

func TestSambaUserRepository_GetAdmin(t *testing.T) {
	t.Run("returns admin user when exists", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSambaUserRepository(db)

		// Create admin user
		adminUser := dbom.SambaUser{
			Username: "admin",
			IsAdmin:  true,
			Password: "password123",
		}
		err := db.Create(&adminUser).Error
		require.NoError(t, err)

		// Create non-admin user
		regularUser := dbom.SambaUser{
			Username: "user1",
			IsAdmin:  false,
			Password: "password456",
		}
		err = db.Create(&regularUser).Error
		require.NoError(t, err)

		// Test GetAdmin
		result, err := repo.GetAdmin()

		assert.NoError(t, err)
		assert.Equal(t, "admin", result.Username)
		assert.True(t, result.IsAdmin)
		assert.Equal(t, "password123", result.Password)
	})

	t.Run("returns ErrRecordNotFound when no admin user exists", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSambaUserRepository(db)

		// Create only non-admin users
		regularUser := dbom.SambaUser{
			Username: "user1",
			IsAdmin:  false,
			Password: "password456",
		}
		err := db.Create(&regularUser).Error
		require.NoError(t, err)

		// Test GetAdmin
		result, err := repo.GetAdmin()

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		assert.Equal(t, dbom.SambaUser{}, result)
	})

	t.Run("returns ErrRecordNotFound when no users exist", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSambaUserRepository(db)

		// Test GetAdmin with empty database
		result, err := repo.GetAdmin()

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		assert.Equal(t, dbom.SambaUser{}, result)
	})

	t.Run("returns first admin user when multiple admin users exist", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSambaUserRepository(db)

		// Create multiple admin users
		admin1 := dbom.SambaUser{
			Username: "admin1",
			IsAdmin:  true,
			Password: "password1",
		}
		err := db.Create(&admin1).Error
		require.NoError(t, err)

		admin2 := dbom.SambaUser{
			Username: "admin2",
			IsAdmin:  true,
			Password: "password2",
		}
		err = db.Create(&admin2).Error
		require.NoError(t, err)

		// Test GetAdmin
		result, err := repo.GetAdmin()

		assert.NoError(t, err)
		assert.True(t, result.IsAdmin)
		// Should return one of the admin users (first one found)
		assert.Contains(t, []string{"admin1", "admin2"}, result.Username)
	})
}
