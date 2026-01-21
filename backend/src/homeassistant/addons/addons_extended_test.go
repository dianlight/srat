package addons

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xorcare/pointer"
)

func TestActionResult_Fields(t *testing.T) {
	data := map[string]any{"key": "value"}
	result := ActionResult{
		Data:   &data,
		Result: ActionResultResultOk,
	}

	assert.NotNil(t, result.Data)
	assert.Equal(t, ActionResultResultOk, result.Result)
	assert.Equal(t, "value", (*result.Data)["key"])
}

func TestActionResult_EmptyData(t *testing.T) {
	result := ActionResult{
		Result: ActionResultResultOk,
	}

	assert.Nil(t, result.Data)
	assert.Equal(t, ActionResultResultOk, result.Result)
}

func TestAddonInfoDataBoot_Values(t *testing.T) {
	tests := []struct {
		name  string
		value AddonInfoDataBoot
	}{
		{"Auto", AddonInfoDataBootAuto},
		{"Manual", AddonInfoDataBootManual},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value)
		})
	}
}

func TestAddonInfoDataStage_Values(t *testing.T) {
	tests := []struct {
		name  string
		value AddonInfoDataStage
	}{
		{"Deprecated", Deprecated},
		{"Experimental", Experimental},
		{"Stable", Stable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value)
		})
	}
}

func TestAddonInfoDataStartup_Values(t *testing.T) {
	tests := []struct {
		name  string
		value AddonInfoDataStartup
	}{
		{"Application", Application},
		{"Initialize", Initialize},
		{"Once", Once},
		{"Services", Services},
		{"System", System},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value)
		})
	}
}

func TestAddonInfoDataState_Values(t *testing.T) {
	tests := []struct {
		name  string
		value AddonInfoDataState
	}{
		{"Error", Error},
		{"Started", Started},
		{"Stopped", Stopped},
		{"Unknown", Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value)
		})
	}
}

func TestAddonInfoData_Fields(t *testing.T) {
	arch := []string{"amd64", "aarch64"}
	authApi := true
	autoUpdate := false
	boot := AddonInfoDataBootAuto
	desc := "Test Addon"
	fullAccess := true
	hostname := "test-addon"
	icon := true

	data := AddonInfoData{
		Arch:        &arch,
		AuthApi:     &authApi,
		AutoUpdate:  &autoUpdate,
		Boot:        &boot,
		Description: &desc,
		FullAccess:  &fullAccess,
		Hostname:    &hostname,
		Icon:        &icon,
	}

	assert.NotNil(t, data.Arch)
	assert.Len(t, *data.Arch, 2)
	assert.True(t, *data.AuthApi)
	assert.False(t, *data.AutoUpdate)
	assert.Equal(t, AddonInfoDataBootAuto, *data.Boot)
	assert.Equal(t, "Test Addon", *data.Description)
	assert.True(t, *data.FullAccess)
	assert.Equal(t, "test-addon", *data.Hostname)
	assert.True(t, *data.Icon)
}

type mockResponseHTTPClient struct {
	statusCode int
	body       string
}

func (m *mockResponseHTTPClient) Do(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader(m.body)),
		Header:     make(http.Header),
	}
	// Default to plain text for log endpoints
	resp.Header.Set("Content-Type", "text/plain")
	return resp, nil
}

func TestClient_GetSelfAddonInfoSuccess(t *testing.T) {
	mockClient := &mockResponseHTTPClient{
		statusCode: http.StatusOK,
		body:       `{"result":"ok","data":{"description":"Test addon"}}`,
	}

	client, err := NewClient("http://example.com", WithHTTPClient(mockClient))
	assert.NoError(t, err)

	resp, err := client.GetSelfAddonInfo(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestClient_ServerURL(t *testing.T) {
	client, err := NewClient("http://example.com")
	assert.NoError(t, err)
	assert.Equal(t, "http://example.com/", client.Server)
}

func TestClient_WithBaseURL(t *testing.T) {
	client, err := NewClient("http://default.com", WithBaseURL("http://override.com"))
	assert.NoError(t, err)
	assert.Equal(t, "http://override.com/", client.Server)
}

func TestNewGetSelfAddonInfoRequest(t *testing.T) {
	req, err := NewGetSelfAddonInfoRequest("http://example.com")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodGet, req.Method)
}

func TestNewGetSelfAddonOptionsRequest(t *testing.T) {
	req, err := NewSetSelfAddonOptionsRequest("http://example.com", AddonOptionsRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
}

func TestNewGetSelfAddonStatsRequest(t *testing.T) {
	req, err := NewGetSelfAddonStatsRequest("http://example.com")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodGet, req.Method)
}

func TestClient_GetSelfAddonLogsWithResponse_Success(t *testing.T) {
	mockClient := &mockResponseHTTPClient{
		statusCode: http.StatusOK,
		body:       "log line 1\nlog line 2",
	}

	client, err := NewClientWithResponses("http://example.com", WithHTTPClient(mockClient))
	assert.NoError(t, err)

	resp, err := client.GetSelfAddonLogsWithResponse(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, "log line 1\nlog line 2", string(resp.Body))
	assert.Nil(t, resp.JSON401)
}

func TestClient_GetSelfAddonLogsLeatestWithResponse_Success(t *testing.T) {
	mockClient := &mockResponseHTTPClient{
		statusCode: http.StatusOK,
		body:       "log line 1\nlog line 2",
	}

	client, err := NewClientWithResponses("http://example.com", WithHTTPClient(mockClient))
	assert.NoError(t, err)

	resp, err := client.GetSelfAddonLogsLatestWithResponse(context.Background(), &GetSelfAddonLogsLatestParams{
		Lines: pointer.Int(1000),
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, "log line 1\nlog line 2", string(resp.Body))
	assert.Nil(t, resp.JSON401)
}
