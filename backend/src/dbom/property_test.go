package dbom

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestPropertyCreation(t *testing.T) {
	now := time.Now()
	prop := Property{
		Key:       "test-key",
		Value:     "test-value",
		Internal:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "test-key", prop.Key)
	assert.Equal(t, "test-value", prop.Value)
	assert.False(t, prop.Internal)
	assert.False(t, prop.CreatedAt.IsZero())
	assert.False(t, prop.UpdatedAt.IsZero())
}

func TestPropertyInternalFlag(t *testing.T) {
	internalProp := Property{
		Key:      "internal-key",
		Value:    "internal-value",
		Internal: true,
	}

	externalProp := Property{
		Key:      "external-key",
		Value:    "external-value",
		Internal: false,
	}

	assert.True(t, internalProp.Internal)
	assert.False(t, externalProp.Internal)
	assert.Equal(t, "internal-key", internalProp.Key)
	assert.Equal(t, "external-key", externalProp.Key)
	assert.Equal(t, "internal-value", internalProp.Value)
	assert.Equal(t, "external-value", externalProp.Value)
}

func TestPropertyValueTypes(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string value", "str-key", "string value"},
		{"int value", "int-key", 42},
		{"bool value", "bool-key", true},
		{"float value", "float-key", 3.14},
		{"slice value", "slice-key", []string{"a", "b", "c"}},
		{"map value", "map-key", map[string]string{"k1": "v1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prop := Property{
				Key:   tt.key,
				Value: tt.value,
			}
			assert.Equal(t, tt.key, prop.Key)
			assert.Equal(t, tt.value, prop.Value)
		})
	}
}

func TestPropertiesGet(t *testing.T) {
	props := Properties{
		"key1": {Key: "key1", Value: "value1"},
		"key2": {Key: "key2", Value: "value2"},
	}

	prop, err := props.Get("key1")
	assert.NoError(t, err)
	assert.NotNil(t, prop)
	assert.Equal(t, "key1", prop.Key)
	assert.Equal(t, "value1", prop.Value)
}

func TestPropertiesGetNotFound(t *testing.T) {
	props := Properties{
		"key1": {Key: "key1", Value: "value1"},
	}

	prop, err := props.Get("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, prop)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestPropertiesGetValue(t *testing.T) {
	props := Properties{
		"key1": {Key: "key1", Value: "value1"},
		"key2": {Key: "key2", Value: 123},
	}

	val, err := props.GetValue("key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)

	val2, err := props.GetValue("key2")
	assert.NoError(t, err)
	assert.Equal(t, 123, val2)
}

func TestPropertiesGetValueNotFound(t *testing.T) {
	props := Properties{
		"key1": {Key: "key1", Value: "value1"},
	}

	val, err := props.GetValue("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, val)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestPropertiesAddInternalValue(t *testing.T) {
	props := Properties{}

	err := props.AddInternalValue("new-key", "new-value")
	assert.NoError(t, err)

	prop, err := props.Get("new-key")
	assert.NoError(t, err)
	assert.NotNil(t, prop)
	assert.Equal(t, "new-key", prop.Key)
	assert.Equal(t, "new-value", prop.Value)
	assert.True(t, prop.Internal)
}

func TestPropertiesAddInternalValueUpdate(t *testing.T) {
	props := Properties{
		"existing": {Key: "existing", Value: "old-value", Internal: false},
	}

	err := props.AddInternalValue("existing", "new-value")
	assert.NoError(t, err)

	prop, err := props.Get("existing")
	assert.NoError(t, err)
	assert.Equal(t, "new-value", prop.Value)
	assert.True(t, prop.Internal)
}

func TestPropertiesMultipleValues(t *testing.T) {
	props := Properties{
		"key1": {Key: "key1", Value: "value1"},
		"key2": {Key: "key2", Value: 42},
		"key3": {Key: "key3", Value: true},
	}

	assert.Len(t, props, 3)

	val1, err := props.GetValue("key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", val1)

	val2, err := props.GetValue("key2")
	assert.NoError(t, err)
	assert.Equal(t, 42, val2)

	val3, err := props.GetValue("key3")
	assert.NoError(t, err)
	assert.Equal(t, true, val3)
}

func TestPropertiesEmptyMap(t *testing.T) {
	props := Properties{}

	_, err := props.Get("any-key")
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestPropertyDefaultInternal(t *testing.T) {
	prop := Property{
		Key:   "default-test",
		Value: "value",
	}

	// Internal should be false by default in Go (zero value for bool)
	assert.False(t, prop.Internal)
	assert.Equal(t, "default-test", prop.Key)
	assert.Equal(t, "value", prop.Value)
}
