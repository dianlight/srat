package repository

/*
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
}

func TestPropertyRepository_Value_Found(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Setup test data
	err := repo.SetValue("test_key", "test_value")
	require.NoError(t, err)

	// Retrieve value
	value, err := repo.Value("test_key")
	require.NoError(t, err)
	assert.Equal(t, "test_value", value)
}

func TestPropertyRepository_Value_NotFound(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Try to get non-existent key
	value, err := repo.Value("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, value)
}

func TestPropertyRepository_SaveAll(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Create properties to save
	props := dbom.Properties{
		"key1": {Key: "key1", Value: "value1"},
		"key2": {Key: "key2", Value: "value2"},
		"key3": {Key: "key3", Value: "value3"},
	}

	err := repo.SaveAll(&props)
	require.NoError(t, err)

	// Verify all were saved
	allProps, err := repo.All()
	require.NoError(t, err)
	assert.Len(t, allProps, 3)
	assert.Equal(t, "value1", allProps["key1"].Value)
	assert.Equal(t, "value2", allProps["key2"].Value)
	assert.Equal(t, "value3", allProps["key3"].Value)
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
	value, err := repo.Value("key1")
	require.NoError(t, err)
	assert.Equal(t, "updated_value", value)
}

func TestPropertyRepository_DifferentValueTypes(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	tests := []struct {
		name  string
		key   string
		value any
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

			value, err := repo.Value(tt.key)
			require.NoError(t, err)
			assert.NotNil(t, value)
		})
	}
}

func TestPropertyRepository_EmptyDatabase(t *testing.T) {
	db := setupPropertyTestDB(t)
	repo := NewPropertyRepositoryRepository(db)

	// Get all from empty database
	props, err := repo.All()
	require.NoError(t, err)
	assert.Empty(t, props)
}
*/
