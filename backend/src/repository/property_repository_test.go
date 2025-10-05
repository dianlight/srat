package repository

import (
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPropertyTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&dbom.Property{})
	require.NoError(t, err)

	return db
}

func TestPropertyRepository_SetValue(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	err := repo.SetValue("test_key", "test_value")
	require.NoError(t, err)

	// Verify it was saved
	var prop dbom.Property
	dbErr := db.Where("key = ?", "test_key").First(&prop).Error
	require.NoError(t, dbErr)
	assert.Equal(t, "test_key", prop.Key)
	assert.Equal(t, "test_value", prop.Value)
	assert.False(t, prop.Internal)
}

func TestPropertyRepository_SetInternalValue(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	err := repo.SetInternalValue("internal_key", "internal_value")
	require.NoError(t, err)

	// Verify it was saved as internal
	var prop dbom.Property
	dbErr := db.Where("key = ?", "internal_key").First(&prop).Error
	require.NoError(t, dbErr)
	assert.Equal(t, "internal_key", prop.Key)
	assert.Equal(t, "internal_value", prop.Value)
	assert.True(t, prop.Internal)
}

func TestPropertyRepository_Value_Found(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Setup test data
	err := repo.SetValue("test_key", "test_value")
	require.NoError(t, err)

	// Retrieve value
	value, err := repo.Value("test_key", false)
	require.NoError(t, err)
	assert.Equal(t, "test_value", value)
}

func TestPropertyRepository_Value_NotFound(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Try to get non-existent key
	value, err := repo.Value("nonexistent", false)
	assert.Error(t, err)
	assert.Nil(t, value)
}

func TestPropertyRepository_Value_InternalExcluded(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Setup internal property
	err := repo.SetInternalValue("internal_key", "internal_value")
	require.NoError(t, err)

	// Try to get with include_internal = false
	value, err := repo.Value("internal_key", false)
	assert.Error(t, err) // Should not find it
	assert.Nil(t, value)
}

func TestPropertyRepository_Value_InternalIncluded(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Setup internal property
	err := repo.SetInternalValue("internal_key", "internal_value")
	require.NoError(t, err)

	// Try to get with include_internal = true
	value, err := repo.Value("internal_key", true)
	require.NoError(t, err)
	assert.Equal(t, "internal_value", value)
}

func TestPropertyRepository_All_ExcludeInternal(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Setup test data
	err := repo.SetValue("key1", "value1")
	require.NoError(t, err)
	err = repo.SetValue("key2", "value2")
	require.NoError(t, err)
	err = repo.SetInternalValue("internal_key", "internal_value")
	require.NoError(t, err)

	// Get all non-internal properties
	props, err := repo.All(false)
	require.NoError(t, err)
	assert.Len(t, props, 2)
	assert.Contains(t, props, "key1")
	assert.Contains(t, props, "key2")
	assert.NotContains(t, props, "internal_key")
}

func TestPropertyRepository_All_IncludeInternal(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Setup test data
	err := repo.SetValue("key1", "value1")
	require.NoError(t, err)
	err = repo.SetInternalValue("internal_key", "internal_value")
	require.NoError(t, err)

	// Get all properties including internal
	props, err := repo.All(true)
	require.NoError(t, err)
	assert.Len(t, props, 2)
	assert.Contains(t, props, "key1")
	assert.Contains(t, props, "internal_key")
}

func TestPropertyRepository_SaveAll(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Create properties to save
	props := dbom.Properties{
		"key1": {Key: "key1", Value: "value1", Internal: false},
		"key2": {Key: "key2", Value: "value2", Internal: false},
		"key3": {Key: "key3", Value: "value3", Internal: true},
	}

	err := repo.SaveAll(&props)
	require.NoError(t, err)

	// Verify all were saved
	allProps, err := repo.All(true)
	require.NoError(t, err)
	assert.Len(t, allProps, 3)
	assert.Equal(t, "value1", allProps["key1"].Value)
	assert.Equal(t, "value2", allProps["key2"].Value)
	assert.Equal(t, "value3", allProps["key3"].Value)
	assert.True(t, allProps["key3"].Internal)
}

func TestPropertyRepository_UpdateExisting(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Create initial value
	err := repo.SetValue("key1", "initial_value")
	require.NoError(t, err)

	// Update with new value
	err = repo.SetValue("key1", "updated_value")
	require.NoError(t, err)

	// Verify update
	value, err := repo.Value("key1", false)
	require.NoError(t, err)
	assert.Equal(t, "updated_value", value)
}

func TestPropertyRepository_DifferentValueTypes(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string", "key_string", "string_value"},
		{"int", "key_int", 42},
		{"bool", "key_bool", true},
		{"float", "key_float", 3.14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.SetValue(tt.key, tt.value)
			require.NoError(t, err)

			value, err := repo.Value(tt.key, false)
			require.NoError(t, err)
			assert.NotNil(t, value)
		})
	}
}

func TestPropertyRepository_EmptyDatabase(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Get all from empty database
	props, err := repo.All(false)
	require.NoError(t, err)
	assert.Len(t, props, 0)
}
