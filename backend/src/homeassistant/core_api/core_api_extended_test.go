package core_api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityState_AllFields(t *testing.T) {
	attrs := map[string]interface{}{"friendly_name": "Test Entity", "unit": "°C"}
	ctx := map[string]interface{}{"id": "123", "user_id": "456"}
	domain := "sensor"
	entityID := "sensor.temperature"
	name := "Temperature"
	objectID := "temperature"
	state := "25.5"
	now := time.Now()

	entity := EntityState{
		Attributes:   &attrs,
		Context:      &ctx,
		Domain:       &domain,
		EntityId:     &entityID,
		LastChanged:  &now,
		LastReported: &now,
		LastUpdated:  &now,
		Name:         &name,
		ObjectId:     &objectID,
		State:        &state,
	}

	assert.NotNil(t, entity.Attributes)
	assert.Equal(t, "Test Entity", (*entity.Attributes)["friendly_name"])
	assert.NotNil(t, entity.Context)
	assert.Equal(t, "sensor", *entity.Domain)
	assert.Equal(t, "sensor.temperature", *entity.EntityId)
	assert.NotNil(t, entity.LastChanged)
	assert.Equal(t, "Temperature", *entity.Name)
	assert.Equal(t, "temperature", *entity.ObjectId)
	assert.Equal(t, "25.5", *entity.State)
	assert.NotNil(t, entity.LastReported)
	assert.NotNil(t, entity.LastUpdated)
}

func TestEntityState_ZeroValues(t *testing.T) {
	entity := EntityState{}

	assert.Nil(t, entity.Attributes)
	assert.Nil(t, entity.Context)
	assert.Nil(t, entity.Domain)
	assert.Nil(t, entity.EntityId)
	assert.Nil(t, entity.LastChanged)
	assert.Nil(t, entity.LastReported)
	assert.Nil(t, entity.LastUpdated)
	assert.Nil(t, entity.Name)
	assert.Nil(t, entity.ObjectId)
	assert.Nil(t, entity.State)
}

func TestEntityState_MarshalJSON(t *testing.T) {
	domain := "binary_sensor"
	entityID := "binary_sensor.door"
	state := "on"

	entity := EntityState{
		Domain:   &domain,
		EntityId: &entityID,
		State:    &state,
	}

	data, err := json.Marshal(entity)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "binary_sensor")
	assert.Contains(t, string(data), "binary_sensor.door")
	assert.Contains(t, string(data), "on")
}

func TestEntityState_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"domain": "switch",
		"entity_id": "switch.living_room",
		"state": "off",
		"attributes": {"friendly_name": "Living Room Switch"}
	}`

	var entity EntityState
	err := json.Unmarshal([]byte(jsonData), &entity)
	assert.NoError(t, err)
	assert.Equal(t, "switch", *entity.Domain)
	assert.Equal(t, "switch.living_room", *entity.EntityId)
	assert.Equal(t, "off", *entity.State)
	assert.NotNil(t, entity.Attributes)
}

func TestServiceData_Fields(t *testing.T) {
	message := "Test notification"
	notificationID := "notify_123"
	title := "Test Title"

	data := ServiceData{
		Message:        &message,
		NotificationId: &notificationID,
		Title:          &title,
	}

	assert.Equal(t, "Test notification", *data.Message)
	assert.Equal(t, "notify_123", *data.NotificationId)
	assert.Equal(t, "Test Title", *data.Title)
}

func TestServiceData_AdditionalProperties(t *testing.T) {
	data := ServiceData{}

	// Test Set
	data.Set("custom_field", "custom_value")
	data.Set("priority", 5)

	// Test Get
	value, found := data.Get("custom_field")
	assert.True(t, found)
	assert.Equal(t, "custom_value", value)

	priority, found := data.Get("priority")
	assert.True(t, found)
	assert.Equal(t, 5, priority)

	// Test non-existent field
	_, found = data.Get("nonexistent")
	assert.False(t, found)
}

func TestServiceData_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"message": "Hello",
		"title": "Greeting",
		"notification_id": "greet_1",
		"custom_field": "custom_value",
		"priority": 10
	}`

	var data ServiceData
	err := json.Unmarshal([]byte(jsonData), &data)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", *data.Message)
	assert.Equal(t, "Greeting", *data.Title)
	assert.Equal(t, "greet_1", *data.NotificationId)

	customVal, found := data.Get("custom_field")
	assert.True(t, found)
	assert.Equal(t, "custom_value", customVal)

	priority, found := data.Get("priority")
	assert.True(t, found)
	assert.Equal(t, float64(10), priority)
}

func TestServiceData_UnmarshalJSON_Error(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
	}{
		{"Invalid JSON", `{"message": invalid}`},
		{"Invalid message type", `{"message": 123}`},
		{"Invalid title type", `{"title": true}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data ServiceData
			err := json.Unmarshal([]byte(tt.jsonData), &data)
			assert.Error(t, err)
		})
	}
}

func TestServiceData_MarshalJSON(t *testing.T) {
	message := "Test"
	title := "Title"
	data := ServiceData{
		Message: &message,
		Title:   &title,
	}
	data.Set("extra", "value")

	jsonData, err := json.Marshal(data)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "Test")
	assert.Contains(t, string(jsonData), "Title")
}

func TestServiceResult_Fields(t *testing.T) {
	ctx := map[string]interface{}{"id": "abc"}
	domain := "notify"
	service := "persistent_notification"
	serviceData := map[string]interface{}{"message": "test"}

	result := ServiceResult{
		Context:     &ctx,
		Domain:      &domain,
		Service:     &service,
		ServiceData: &serviceData,
	}

	assert.NotNil(t, result.Context)
	assert.Equal(t, "notify", *result.Domain)
	assert.Equal(t, "persistent_notification", *result.Service)
	assert.NotNil(t, result.ServiceData)
	assert.Equal(t, "test", (*result.ServiceData)["message"])
}

func TestServiceResult_ZeroValues(t *testing.T) {
	result := ServiceResult{}

	assert.Nil(t, result.Context)
	assert.Nil(t, result.Domain)
	assert.Nil(t, result.Service)
	assert.Nil(t, result.ServiceData)
}

type mockResponseHTTPClient struct {
	statusCode int
	body       string
}

func (m *mockResponseHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader(m.body)),
		Header:     make(http.Header),
	}, nil
}

func TestClient_GetApi(t *testing.T) {
	mockClient := &mockResponseHTTPClient{
		statusCode: http.StatusOK,
		body:       `{"message":"API running"}`,
	}

	client, err := NewClient("http://api.local", WithHTTPClient(mockClient))
	assert.NoError(t, err)

	resp, err := client.GetApi(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestClient_ServerURL(t *testing.T) {
	client, err := NewClient("http://api.local")
	assert.NoError(t, err)
	assert.Equal(t, "http://api.local/", client.Server)
}

func TestClient_WithBaseURL(t *testing.T) {
	client, err := NewClient("http://default.local", WithBaseURL("http://override.local"))
	assert.NoError(t, err)
	assert.Equal(t, "http://override.local/", client.Server)
}

func TestNewGetApiRequest(t *testing.T) {
	req, err := NewGetApiRequest("http://api.local")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodGet, req.Method)
}

func TestNewGetEntityStateRequest(t *testing.T) {
	req, err := NewGetEntityStateRequest("http://api.local", "sensor.temperature")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodGet, req.Method)
	assert.Contains(t, req.URL.Path, "sensor.temperature")
}

func TestNewCallServiceRequest(t *testing.T) {
	message := "test"
	body := ServiceData{Message: &message}

	req, err := NewCallServiceRequest("http://api.local", "notify", "persistent_notification", body)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Contains(t, req.URL.Path, "notify")
	assert.Contains(t, req.URL.Path, "persistent_notification")
}

func TestNewPostEntityStateRequest(t *testing.T) {
	state := "on"
	body := EntityState{State: &state}

	req, err := NewPostEntityStateRequest("http://api.local", "sensor.test", body)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Contains(t, req.URL.Path, "sensor.test")
}

func TestEntityState_TimeFields(t *testing.T) {
	now := time.Now()
	later := now.Add(5 * time.Minute)

	entity := EntityState{
		LastChanged:  &now,
		LastReported: &later,
		LastUpdated:  &now,
	}

	assert.NotNil(t, entity.LastChanged)
	assert.NotNil(t, entity.LastReported)
	assert.NotNil(t, entity.LastUpdated)
	assert.True(t, entity.LastReported.After(*entity.LastChanged))
}

func TestServiceData_EmptyAdditionalProperties(t *testing.T) {
	data := ServiceData{}

	_, found := data.Get("any_field")
	assert.False(t, found)

	data.Set("field1", "value1")
	assert.NotNil(t, data.AdditionalProperties)

	val, found := data.Get("field1")
	assert.True(t, found)
	assert.Equal(t, "value1", val)
}

func TestClient_RequestEditorFn(t *testing.T) {
	mockClient := &mockResponseHTTPClient{
		statusCode: http.StatusOK,
		body:       `[]`,
	}

	called := false
	client, err := NewClient("http://api.local",
		WithHTTPClient(mockClient),
		WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			called = true
			req.Header.Set("X-Custom", "test")
			return nil
		}),
	)
	require.NoError(t, err)

	_, err = client.GetApi(context.Background())
	assert.NoError(t, err)
	assert.True(t, called)
}
