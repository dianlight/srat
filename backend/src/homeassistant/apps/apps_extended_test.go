package apps

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
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

func TestAppInfoDataBoot_Values(t *testing.T) {
	tests := []struct {
		name  string
		value AppInfoDataBoot
	}{
		{"Auto", AppInfoDataBootAuto},
		{"Manual", AppInfoDataBootManual},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value)
		})
	}
}

func TestAppInfoDataStage_Values(t *testing.T) {
	tests := []struct {
		name  string
		value AppInfoDataStage
	}{
		{"Deprecated", AppInfoDataStageDeprecated},
		{"Experimental", AppInfoDataStageExperimental},
		{"Stable", AppInfoDataStageStable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value)
		})
	}
}

func TestAppInfoDataStartup_Values(t *testing.T) {
	tests := []struct {
		name  string
		value AppInfoDataStartup
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

func TestAppInfoDataState_Values(t *testing.T) {
	tests := []struct {
		name  string
		value AppInfoDataState
	}{
		{"Error", AppInfoDataStateError},
		{"Started", AppInfoDataStateStarted},
		{"Stopped", AppInfoDataStateStopped},
		{"Unknown", AppInfoDataStateUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value)
		})
	}
}

func TestAppInfoData_Fields(t *testing.T) {
	arch := []string{"amd64", "aarch64"}
	authApi := true
	autoUpdate := false
	boot := AppInfoDataBootAuto
	desc := "Test App"
	fullAccess := true
	hostname := "test-app"
	icon := true

	data := AppInfoData{
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
	assert.Equal(t, AppInfoDataBootAuto, *data.Boot)
	assert.Equal(t, "Test App", *data.Description)
	assert.True(t, *data.FullAccess)
	assert.Equal(t, "test-app", *data.Hostname)
	assert.True(t, *data.Icon)
}

func TestClient_GetAppInfoSuccess(t *testing.T) {
	mockTransport := httpmock.NewMockTransport()
	infoResponse := httpmock.NewStringResponse(http.StatusOK, `{"result":"ok","data":{"description":"Test app"}}`)
	infoResponse.Header.Set("Content-Type", "application/json")
	mockTransport.RegisterResponder(http.MethodGet, "http://example.com/addons/self/info", httpmock.ResponderFromResponse(infoResponse))
	clientHTTP := &http.Client{Transport: mockTransport}

	client, err := NewClient("http://example.com", WithHTTPClient(clientHTTP))
	assert.NoError(t, err)

	resp, err := client.GetAppInfo(context.Background(), "self")
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

func TestNewGetAppInfoRequest(t *testing.T) {
	req, err := NewGetAppInfoRequest("http://example.com", "self")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodGet, req.Method)
}

func TestNewSetAppOptionsRequest(t *testing.T) {
	req, err := NewSetAppOptionsRequest("http://example.com", "self", AppOptionsRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
}

func TestNewGetSelfAppStatsRequest(t *testing.T) {
	req, err := NewGetSelfAppStatsRequest("http://example.com")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodGet, req.Method)
}

func TestClient_GetAppLogsWithResponse_Success(t *testing.T) {
	tests := []struct {
		name   string
		accept GetAppLogsParamsAccept
	}{
		{
			name:   "AcceptTextPlain",
			accept: GetAppLogsParamsAcceptTextplain,
		},
		{
			name:   "AcceptTextXLog",
			accept: GetAppLogsParamsAcceptTextxLog,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := httpmock.NewMockTransport()
			logResponse := httpmock.NewStringResponse(http.StatusOK, "log line 1\nlog line 2")
			logResponse.Header.Set("Content-Type", "text/plain")
			mockTransport.RegisterMatcherResponder(
				http.MethodGet,
				"http://example.com/addons/self/logs",
				httpmock.HeaderIs("Accept", string(tt.accept)),
				httpmock.ResponderFromResponse(logResponse),
			)
			clientHTTP := &http.Client{Transport: mockTransport}

			client, err := NewClientWithResponses("http://example.com", WithHTTPClient(clientHTTP))
			assert.NoError(t, err)

			resp, err := client.GetAppLogsWithResponse(context.Background(), "self", &GetAppLogsParams{Accept: tt.accept})
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, http.StatusOK, resp.StatusCode())
			assert.Equal(t, "log line 1\nlog line 2", string(resp.Body))
			assert.Nil(t, resp.JSON401)
		})
	}
}

func TestClient_GetAppLogsLatestWithResponse_Success(t *testing.T) {
	tests := []struct {
		name   string
		accept GetAppLogsLatestParamsAccept
	}{
		{
			name:   "AcceptTextPlain",
			accept: Textplain,
		},
		{
			name:   "AcceptTextXLog",
			accept: TextxLog,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := httpmock.NewMockTransport()
			logResponse := httpmock.NewStringResponse(http.StatusOK, "log line 1\nlog line 2")
			logResponse.Header.Set("Content-Type", "text/plain")
			mockTransport.RegisterMatcherResponder(
				http.MethodGet,
				"http://example.com/addons/self/logs/latest?lines=1000",
				httpmock.HeaderIs("Accept", string(tt.accept)),
				httpmock.ResponderFromResponse(logResponse),
			)
			clientHTTP := &http.Client{Transport: mockTransport}

			client, err := NewClientWithResponses("http://example.com", WithHTTPClient(clientHTTP))
			assert.NoError(t, err)

			resp, err := client.GetAppLogsLatestWithResponse(context.Background(), "self", &GetAppLogsLatestParams{
				Lines:  new(1000),
				Accept: tt.accept,
			})
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, http.StatusOK, resp.StatusCode())
			assert.Equal(t, "log line 1\nlog line 2", string(resp.Body))
			assert.Nil(t, resp.JSON401)
		})
	}
}
