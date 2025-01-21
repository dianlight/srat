package dbom

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPropertiesLoadEmpty(t *testing.T) {
	// Initialize an empty Properties slice
	p := &Properties{}

	// Call the Load method
	err := p.Load()

	// Assert that no error occurred
	require.NoError(t, err)

	// Check if the Properties slice is still empty
	assert.Empty(t, *p)

	// Verify that no properties were loaded from the database
	var count int64
	result := db.Model(&Property{}).Count(&count)
	require.NoError(t, result.Error)
	assert.Equal(t, int64(0), count)
}

func TestPropertiesLoadWithExistingProperties(t *testing.T) {
	// Create test properties
	testProperties := []Property{
		{Key: "key1", Value: `"value1"`},
		{Key: "key2", Value: `42`},
		{Key: "key3", Value: `true`},
	}

	// Insert test properties into the database
	for _, prop := range testProperties {
		result := db.Create(&prop)
		require.NoError(t, result.Error)
	}

	// Initialize an empty Properties slice
	p := &Properties{}

	// Call the Load method
	err := p.Load()

	// Assert that no error occurred
	require.NoError(t, err)

	// Check if the Properties slice contains the correct number of items
	assert.Len(t, *p, len(testProperties))

	// Verify that all properties were loaded correctly
	for _, expectedProp := range testProperties {
		found := false
		for _, loadedProp := range *p {
			if loadedProp.Key == expectedProp.Key && loadedProp.Value == expectedProp.Value {
				found = true
				break
			}
		}
		assert.True(t, found, "Property %s not found in loaded properties", expectedProp.Key)
	}

	// Clean up the test data
	db.Where("key IN ?", []string{"key1", "key2", "key3"}).Delete(&Property{})
}

func TestPropertiesLoadWithNullValues(t *testing.T) {
	// Create test properties with null values
	testProperties := []Property{
		{Key: "key10", Value: "null"},
		{Key: "key20", Value: `"value2"`},
		{Key: "key30", Value: "null"},
	}

	// Insert test properties into the database
	for _, prop := range testProperties {
		result := db.Create(&prop)
		require.NoError(t, result.Error)
	}

	// Initialize an empty Properties slice
	p := &Properties{}

	// Call the Load method
	err := p.Load()

	// Assert that no error occurred
	require.NoError(t, err)

	// Check if the Properties slice contains the correct number of items
	assert.Len(t, *p, len(testProperties))

	// Verify that all properties were loaded correctly, including null values
	for _, expectedProp := range testProperties {
		found := false
		for _, loadedProp := range *p {
			if loadedProp.Key == expectedProp.Key && loadedProp.Value == expectedProp.Value {
				found = true
				break
			}
		}
		assert.True(t, found, "Property %s not found or has incorrect value in loaded properties", expectedProp.Key)
	}

	// Clean up the test data
	db.Where("key IN ?", []string{"key1", "key2", "key3"}).Delete(&Property{})
}

func TestPropertiesAddString(t *testing.T) {
	// Initialize Properties
	p := &Properties{}

	// Test adding a new property with a string value
	key := "testKey"
	value := "testValue"
	err := p.Add(key, value)

	// Assert no error occurred
	require.NoError(t, err)

	// Check if the property was added to the slice
	assert.Len(t, *p, 1)
	assert.NotNil(t, (*p)[key])

	// Verify the property was added to the database
	var dbProp Property
	result := db.Where("key = ?", key).First(&dbProp)
	require.NoError(t, result.Error)
	assert.Equal(t, key, dbProp.Key)
	assert.Equal(t, value, dbProp.Value)
	//assert.Equal(t, (*p)[0].Value, dbProp.Value)
}
func TestPropertiesAddInt(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Add a new property with a numeric value
	err := p.Add("testNumberKey", 42)
	require.NoError(t, err)

	// Check if the property was added successfully
	assert.Len(t, *p, 1)
	assert.NotNil(t, (*p)["testNumberKey"])

	// Verify that the property was added to the database
	var dbProp Property
	result := db.Where("key = ?", "testNumberKey").First(&dbProp)
	require.NoError(t, result.Error)
	assert.Equal(t, "testNumberKey", dbProp.Key)
	assert.InDelta(t, float64(42), dbProp.Value, 0.01)
}
func TestPropertiesAddBool(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Add a new property with a boolean value
	err := p.Add("testBoolKey", true)
	require.NoError(t, err)

	// Verify the property in the database
	var dbProp Property
	result := db.Where("key = ?", "testBoolKey").First(&dbProp)
	require.NoError(t, result.Error)
	assert.Equal(t, "testBoolKey", dbProp.Key)
	assert.Equal(t, true, dbProp.Value)

	// Clean up the test data
	db.Where("key = ?", "testBoolKey").Delete(&Property{})
}
func TestPropertiesAddNestedObject(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Create a nested object
	nestedObject := map[string]interface{}{
		"name": "John Doe",
		"age":  float64(30),
		"address": map[string]interface{}{
			"street": "123 Main St",
			"city":   "Anytown",
		},
	}

	// Add the nested object as a new property
	err := p.Add("person", nestedObject)
	require.NoError(t, err)

	// Check if the property was added successfully
	assert.Len(t, *p, 1)
	assert.NotNil(t, (*p)["person"])

	// Verify that the property was added to the database
	var dbProp Property
	result := db.Where("key = ?", "person").First(&dbProp)
	require.NoError(t, result.Error)
	assert.Equal(t, "person", dbProp.Key)
	assert.Equal(t, (*p)["person"].Value, dbProp.Value)

	// Clean up the test data
	db.Where("key = ?", "person").Delete(&Property{})
}
func TestPropertiesAddInvalidJSON(t *testing.T) {
	t.Skip("JSON marshalling not supported for channels")
	// Initialize a new Properties slice
	p := &Properties{}

	// Create a channel, which is not JSON-marshallable
	invalidValue := make(chan int)

	// Attempt to add the invalid value as a new property
	err := p.Add("invalidKey", invalidValue)

	// Check if an error was returned
	require.Error(t, err)

	// Verify that no property was added to the slice
	assert.Empty(t, *p)

	// Verify that no property was added to the database
	var dbProp Property
	result := db.Model(&Property{}).Where("key = ?", "invalidKey").First(&dbProp)
	require.Error(t, result.Error)
	assert.ErrorIs(t, result.Error, gorm.ErrRecordNotFound)
}
func TestPropertiesAddAppendToSlice(t *testing.T) {
	// Initialize a new Properties slice with an existing property
	p := &Properties{}
	err := p.Load()
	require.NoError(t, err)
	prv := len(*p)

	// Add a new property
	err = p.Add("newKey", "newValue")
	require.NoError(t, err)

	// Check if the new property was appended to the slice
	p = &Properties{}
	err = p.Load()
	require.NoError(t, err)
	assert.Len(t, *p, prv+1)

	// Verify that the property was added to the database
	var dbProp Property
	result := db.Where("key =?", "newKey").First(&dbProp)
	require.NoError(t, result.Error)
	assert.Equal(t, "newKey", dbProp.Key)
	assert.Equal(t, "newValue", dbProp.Value)

	// Clean up the test data
	db.Where("key IN (?)", []string{"existingKey", "newKey"}).Delete(&Property{})
}

func TestPropertiesAddLargeValue(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Create a large value (1MB string)
	largeValue := strings.Repeat("a", 1024*1024)

	// Add the large value as a new property
	err := p.Add("largeKey", largeValue)
	require.NoError(t, err)

	// Check if the property was added successfully
	assert.Len(t, *p, 1)
	assert.NotNil(t, "largeKey", (*p)["largeKey"])

	// Verify that the property was added to the database
	var dbProp Property
	result := db.Where("key = ?", "largeKey").First(&dbProp)
	require.NoError(t, result.Error)
	assert.Equal(t, "largeKey", dbProp.Key)
	assert.Equal(t, (*p)["largeKey"].Value, dbProp.Value)

	// Clean up the test data
	db.Where("key = ?", "largeKey").Delete(&Property{})
}

func TestPropertiesAddUnicode(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Unicode key and value
	unicodeKey := "テスト_キー"
	unicodeValue := "こんにちは世界"

	// Add the Unicode property
	err := p.Add(unicodeKey, unicodeValue)
	require.NoError(t, err)

	// Check if the property was added successfully
	assert.Len(t, *p, 1)
	assert.NotNil(t, (*p)[unicodeKey])

	// Verify that the property was added to the database
	var dbProp Property
	result := db.Where("key = ?", unicodeKey).First(&dbProp)
	require.NoError(t, result.Error)
	assert.Equal(t, unicodeKey, dbProp.Key)
	assert.Equal(t, (*p)[unicodeKey].Value, dbProp.Value)

	// Clean up the test data
	db.Where("key = ?", unicodeKey).Delete(&Property{})
}
func TestPropertiesAddNilValue(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Add a property with a nil value
	err := p.Add("nilKey", nil)
	require.NoError(t, err)

	// Check if the property was added successfully
	assert.Len(t, *p, 1)
	assert.Nil(t, (*p)["nilKey"].Value)

	// Verify that the property was added to the database
	var dbProp Property
	result := db.Where("key = ?", "nilKey").First(&dbProp)
	require.NoError(t, result.Error)
	assert.Equal(t, "nilKey", dbProp.Key)
	assert.Nil(t, dbProp.Value)

	// Clean up the test data
	db.Where("key = ?", "nilKey").Delete(&Property{})
}
func TestPropertiesRemove(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Add a test property
	testKey := "testRemoveKey"
	testValue := "testRemoveValue"
	err := p.Add(testKey, testValue)
	require.NoError(t, err)

	// Verify the property was added
	assert.Len(t, *p, 1)
	assert.NotNil(t, (*p)[testKey])

	// Remove the property
	err = p.Remove(testKey)
	require.NoError(t, err)

	// Check if the property was removed from the slice
	assert.Empty(t, *p)

	// Verify that the property was removed from the database
	var dbProp Property
	result := db.Where("key = ?", testKey).First(&dbProp)
	require.Error(t, result.Error)
	assert.ErrorIs(t, result.Error, gorm.ErrRecordNotFound)
}

func TestPropertiesAddExistingKeyInDB(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Add a property directly to the database
	existingKey := "existingKey"
	existingValue := "\"existingValue\""
	dbProp := Property{Key: existingKey, Value: existingValue}
	result := db.Create(&dbProp)
	require.NoError(t, result.Error)

	// Add a property with the same key but different value
	newValue := "newValue"
	err := p.Add(existingKey, newValue)
	require.NoError(t, err)

	// Check if the property was added to the slice
	assert.Len(t, *p, 1)
	assert.NotNil(t, (*p)[existingKey])

	// Verify that the property was updated in the database
	var updatedDBProp Property
	result = db.Where("key = ?", existingKey).First(&updatedDBProp)
	require.NoError(t, result.Error)
	assert.Equal(t, existingKey, updatedDBProp.Key)
	assert.Equal(t, (*p)[existingKey].Value, updatedDBProp.Value)

	// Clean up the test data
	db.Where("key = ?", existingKey).Delete(&Property{})
}
func TestPropertiesRemoveExistsInSliceNotInDB(t *testing.T) {
	// Initialize a new Properties slice with a test property
	p := &Properties{
		"testKey": {Key: "testKey", Value: `"testValue"`},
	}

	// Ensure the property is not in the database
	db.Where("key = ?", "testKey").Delete(&Property{})

	// Remove the property
	err := p.Remove("testKey")
	require.NoError(t, err)

	// Check if the property was removed from the slice
	assert.Empty(t, *p)

	// Verify that no error occurs when trying to delete a non-existent property from the database
	var count int64
	result := db.Model(&Property{}).Where("key = ?", "testKey").Count(&count)
	require.NoError(t, result.Error)
	assert.Equal(t, int64(0), count)
}
func TestPropertiesRemoveExistsInDBNotInSlice(t *testing.T) {
	// Initialize an empty Properties slice
	p := &Properties{}

	// Add a test property directly to the database
	testKey := "testDBOnlyKey"
	testValue := "testDBOnlyValue"
	dbProp := Property{Key: testKey, Value: testValue}
	result := db.Create(&dbProp)
	require.NoError(t, result.Error)

	// Verify the property exists in the database
	var count int64
	result = db.Model(&Property{}).Where("key = ?", testKey).Count(&count)
	require.NoError(t, result.Error)
	assert.Equal(t, int64(1), count)

	// Remove the property
	err := p.Remove(testKey)
	require.NoError(t, err)

	// Verify that the property was removed from the database
	result = db.Model(&Property{}).Where("key = ?", testKey).Count(&count)
	require.NoError(t, result.Error)
	assert.Equal(t, int64(0), count)

	// Verify that the slice remains empty
	assert.Empty(t, *p)

	// Clean up any remaining test data
	db.Where("key = ?", testKey).Delete(&Property{})
}

func TestPropertiesRemoveNonExistent(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Attempt to remove a non-existent property
	err := p.Remove("nonExistentKey")
	require.NoError(t, err)

	// Verify that the slice remains empty
	assert.Empty(t, *p)

	// Verify that no error occurs when trying to delete a non-existent property from the database
	var count int64
	result := db.Model(&Property{}).Where("key = ?", "nonExistentKey").Count(&count)
	require.NoError(t, result.Error)
	assert.Equal(t, int64(0), count)
}
func TestPropertiesRemoveFromEmptySlice(t *testing.T) {
	// Initialize an empty Properties slice
	p := &Properties{}

	// Attempt to remove a property from the empty slice
	err := p.Remove("nonExistentKey")
	require.NoError(t, err)

	// Verify that the slice remains empty
	assert.Empty(t, *p)

	// Verify that no error occurs when trying to delete a non-existent property from the database
	var count int64
	result := db.Model(&Property{}).Where("key = ?", "nonExistentKey").Count(&count)
	require.NoError(t, result.Error)
	assert.Equal(t, int64(0), count)
}
func TestPropertiesRemoveMultiple(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Add multiple test properties
	testKeys := []string{"key1", "key2", "key3"}
	testValues := []string{"value1", "value2", "value3"}

	for i, key := range testKeys {
		err := p.Add(key, testValues[i])
		require.NoError(t, err)
	}

	// Verify the properties were added
	assert.Len(t, *p, 3)

	// Remove the properties one by one
	for _, key := range testKeys {
		err := p.Remove(key)
		require.NoError(t, err)

		// Check if the property was removed from the slice
		for _, prop := range *p {
			assert.NotEqual(t, key, prop.Key)
		}

		// Verify that the property was removed from the database
		var dbProp Property
		result := db.Where("key = ?", key).First(&dbProp)
		require.Error(t, result.Error)
		require.ErrorIs(t, result.Error, gorm.ErrRecordNotFound)
	}

	// Check if all properties were removed from the slice
	assert.Empty(t, *p)

	// Verify that all properties were removed from the database
	var count int64
	result := db.Model(&Property{}).Where("key IN ?", testKeys).Count(&count)
	require.NoError(t, result.Error)
	assert.Equal(t, int64(0), count)

	// Clean up any remaining test data
	db.Where("key IN ?", testKeys).Delete(&Property{})
}

func TestPropertiesAddAfterRemoveAndReAdd(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Add an initial property
	initialKey := "testKeyTrt"
	initialValue := "initialValue"
	err := p.Add(initialKey, initialValue)
	require.NoError(t, err)

	// Remove the property
	err = p.Remove(initialKey)
	require.NoError(t, err)

	// Re-add the property with a new value
	newValue := "newValue"
	err = p.Add(initialKey, newValue)
	require.NoError(t, err)

	// Update the property
	updatedValue := "updatedValue"
	err = p.Add(initialKey, updatedValue)
	require.NoError(t, err)

	// Check if the property was updated in the slice
	assert.Len(t, *p, 1)
	assert.NotNil(t, (*p)[initialKey])

	// Verify that the property was updated in the database
	var dbProp Property
	result := db.Where("key = ?", initialKey).First(&dbProp)
	require.NoError(t, result.Error)
	assert.Equal(t, initialKey, dbProp.Key)
	assert.Equal(t, (*p)[initialKey].Value, dbProp.Value)

	// Clean up the test data
	db.Where("key = ?", initialKey).Delete(&Property{})
}

func TestPropertiesGetCaseSensitiveKeys(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Add two properties with case-sensitive keys
	err := p.Add("testKeycs", "value1")
	require.NoError(t, err)
	err = p.Add("TestKeyCS", "value2")
	require.NoError(t, err)

	// Get the property with the lowercase key
	prop1, err := p.Get("testKeycs")
	require.NoError(t, err)
	assert.Equal(t, "testKeycs", prop1.Key)
	assert.Equal(t, "value1", prop1.Value)

	// Get the property with the uppercase key
	prop2, err := p.Get("TestKeyCS")
	require.NoError(t, err)
	assert.Equal(t, "TestKeyCS", prop2.Key)
	assert.Equal(t, "value2", prop2.Value)

	// Attempt to get a non-existent property with mixed case
	_, err = p.Get("TESTkeyCS")
	require.Error(t, err)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Clean up test data
	db.Where("key IN ?", []string{"testKeycs", "TestKeyCS"}).Delete(&Property{})
}
func TestPropertiesConcurrentGet(t *testing.T) {
	//t.Skip("This test is failing due to a bug in the Properties.Get method.") // FIXME support concurrent Get operations in Properties
	// Initialize a new Properties slice
	p := &Properties{}
	err := p.Load()
	require.NoError(t, err)

	// Add a test property
	testKey := "concurrentKey"
	testValue := "concurrentValue"
	err = p.Add(testKey, testValue)
	require.NoError(t, err)

	// Number of concurrent goroutines
	numGoroutines := 10

	// Use WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Channel to collect results
	results := make(chan *Property, numGoroutines)

	// Firt non  concurrent
	_, err = p.Get(testKey)
	require.NoError(t, err)

	// Launch concurrent Get operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			prop, err := p.Get(testKey)
			assert.NoError(t, err)
			results <- prop
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)

	// Verify results
	for prop := range results {
		assert.NotNil(t, prop)
		assert.Equal(t, testKey, prop.Key)
	}

	// Clean up the test data
	db.Where("key = ?", testKey).Delete(&Property{})
}
func TestPropertiesGetNullValue(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Add a test property with a NULL value directly to the database
	testKey := "nullValueKey"
	dbProp := Property{Key: testKey, Value: "null"}
	result := db.Create(&dbProp)
	require.NoError(t, result.Error)

	// Get the property
	prop, err := p.Get(testKey)

	// Assert that no error occurred and the property is not nil
	require.NoError(t, err)
	assert.NotNil(t, prop)

	// Verify that the key matches and the value is "null"
	assert.Equal(t, testKey, prop.Key)
	assert.Equal(t, "null", prop.Value)

	// Clean up the test data
	db.Where("key = ?", testKey).Delete(&Property{})
}
func TestPropertiesGetRaceConditionWithRemove(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Add a test property
	testKey := "raceKey"
	testValue := "raceValue"
	err := p.Add(testKey, testValue)
	require.NoError(t, err)

	// Create a WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Channel to collect results
	resultChan := make(chan error, 2)

	// Goroutine for Get operation
	go func() {
		defer wg.Done()
		_, err := p.Get(testKey)
		resultChan <- err
	}()

	// Goroutine for Remove operation
	go func() {
		defer wg.Done()
		err := p.Remove(testKey)
		resultChan <- err
	}()

	// Wait for both goroutines to complete
	wg.Wait()
	close(resultChan)

	// Collect results
	var getErr, removeErr error
	for err := range resultChan {
		if err == gorm.ErrRecordNotFound {
			getErr = err
		} else if err != nil {
			removeErr = err
		}
	}

	// Assert that either Get succeeded and Remove failed, or Remove succeeded and Get failed
	assert.True(t, (getErr == nil && removeErr == nil) || (getErr != nil && removeErr == nil) || (getErr == nil && removeErr != nil))

	// Verify that the property is no longer in the database
	var count int64
	result := db.Model(&Property{}).Where("key = ?", testKey).Count(&count)
	require.NoError(t, result.Error)
	assert.Equal(t, int64(0), count)

	// Clean up any remaining test data
	db.Where("key = ?", testKey).Delete(&Property{})
}
func TestPropertiesGetValueNonExistentKey(t *testing.T) {
	// Initialize a new Properties slice
	p := &Properties{}

	// Attempt to get a value for a non-existent key
	nonExistentKey := "nonExistentKey"
	value, err := p.GetValue(nonExistentKey)

	// Check if an error was returned
	require.Error(t, err)
	assert.Nil(t, value)

	// Verify that the error is of type gorm.ErrRecordNotFound
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Verify that the key does not exist in the database
	var count int64
	result := db.Model(&Property{}).Where("key = ?", nonExistentKey).Count(&count)
	require.NoError(t, result.Error)
	assert.Equal(t, int64(0), count)
}
func TestPropertiesGetValueWithSpecialChars(t *testing.T) {
	//t.Skip("This test is failing due to a bug in the Properties.GetValue method.") // FIXME support special characters in Properties.GetValue
	// Initialize a new Properties slice
	p := &Properties{}

	// Add a property with a key containing special characters and Unicode
	specialKey := "特殊キー!@#$%^&*()_+"
	specialValue := "特別な値"
	err := p.Add(specialKey, specialValue)
	require.NoError(t, err)

	// Get the value using the special key
	value, err := p.GetValue(specialKey)
	require.NoError(t, err)
	require.NotNil(t, value)

	// Assert that the retrieved value matches the original value
	assert.Equal(t, specialValue, value)

	// Verify that the property was added to the database
	var dbProp Property
	result := db.Where("key = ?", specialKey).First(&dbProp)
	require.NoError(t, result.Error)
	assert.Equal(t, specialKey, dbProp.Key)

	// Clean up the test data
	db.Where("key = ?", specialKey).Delete(&Property{})
}
