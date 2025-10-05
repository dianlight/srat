package host

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionResult_Fields(t *testing.T) {
	data := map[string]interface{}{"reboot": "scheduled"}

	action := ActionResult{
		Data:   &data,
		Result: ActionResultResultOk,
	}

	assert.NotNil(t, action.Data)
	assert.Equal(t, "scheduled", (*action.Data)["reboot"])
	assert.Equal(t, ActionResultResultOk, action.Result)
}

func TestActionResult_EmptyData(t *testing.T) {
	action := ActionResult{
		Result: ActionResultResultOk,
	}

	assert.Nil(t, action.Data)
	assert.Equal(t, ActionResultResultOk, action.Result)
}

func TestErrorResponse_Message(t *testing.T) {
	err := ErrorResponse{
		Message: "Host operation failed",
	}

	assert.Equal(t, "Host operation failed", err.Message)
}

func TestHostInfoData_AllFields(t *testing.T) {
	chassis := "laptop"
	cpe := "cpe:2.3:o:linux:linux_kernel:5.10.0"
	deployment := "production"
	diskFree := float32(100.5)
	diskTotal := float32(500.0)
	diskUsed := float32(399.5)
	features := []string{"reboot", "shutdown", "hostname"}
	hostname := "homeassistant"
	kernel := "5.10.0-8-amd64"
	os := "Debian GNU/Linux 11 (bullseye)"
	timezone := "America/New_York"

	info := HostInfoData{
		Chassis:         &chassis,
		Cpe:             &cpe,
		Deployment:      &deployment,
		DiskFree:        &diskFree,
		DiskTotal:       &diskTotal,
		DiskUsed:        &diskUsed,
		Features:        &features,
		Hostname:        &hostname,
		Kernel:          &kernel,
		OperatingSystem: &os,
		Timezone:        &timezone,
	}

	assert.Equal(t, "laptop", *info.Chassis)
	assert.Equal(t, "cpe:2.3:o:linux:linux_kernel:5.10.0", *info.Cpe)
	assert.Equal(t, "production", *info.Deployment)
	assert.Equal(t, float32(100.5), *info.DiskFree)
	assert.Equal(t, float32(500.0), *info.DiskTotal)
	assert.Equal(t, float32(399.5), *info.DiskUsed)
	assert.NotNil(t, info.Features)
	assert.Contains(t, *info.Features, "reboot")
	assert.Equal(t, "homeassistant", *info.Hostname)
	assert.Equal(t, "5.10.0-8-amd64", *info.Kernel)
	assert.Equal(t, "Debian GNU/Linux 11 (bullseye)", *info.OperatingSystem)
	assert.Equal(t, "America/New_York", *info.Timezone)
}

func TestHostInfoData_ZeroValues(t *testing.T) {
	info := HostInfoData{}

	assert.Nil(t, info.Chassis)
	assert.Nil(t, info.Cpe)
	assert.Nil(t, info.Deployment)
	assert.Nil(t, info.DiskFree)
	assert.Nil(t, info.DiskTotal)
	assert.Nil(t, info.DiskUsed)
	assert.Nil(t, info.Features)
	assert.Nil(t, info.Hostname)
	assert.Nil(t, info.Kernel)
	assert.Nil(t, info.OperatingSystem)
	assert.Nil(t, info.Timezone)
}

func TestHostInfoData_DiskUsageCalculation(t *testing.T) {
	diskFree := float32(100.0)
	diskTotal := float32(500.0)
	diskUsed := float32(400.0)

	info := HostInfoData{
		DiskFree:  &diskFree,
		DiskTotal: &diskTotal,
		DiskUsed:  &diskUsed,
	}

	// Verify disk usage calculation
	expectedUsage := *info.DiskTotal - *info.DiskFree
	assert.InDelta(t, expectedUsage, *info.DiskUsed, 0.1)
}

func TestHostInfoData_MarshalJSON(t *testing.T) {
	hostname := "test-host"
	kernel := "5.15.0"

	info := HostInfoData{
		Hostname: &hostname,
		Kernel:   &kernel,
	}

	data, err := json.Marshal(info)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "test-host")
	assert.Contains(t, string(data), "5.15.0")
}

func TestHostInfoData_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"hostname": "my-homeassistant",
		"kernel": "5.10.0",
		"operating_system": "Home Assistant OS",
		"disk_total": 32.0,
		"disk_free": 10.5
	}`

	var info HostInfoData
	err := json.Unmarshal([]byte(jsonData), &info)
	assert.NoError(t, err)
	assert.Equal(t, "my-homeassistant", *info.Hostname)
	assert.Equal(t, "5.10.0", *info.Kernel)
	assert.Equal(t, "Home Assistant OS", *info.OperatingSystem)
	assert.Equal(t, float32(32.0), *info.DiskTotal)
	assert.Equal(t, float32(10.5), *info.DiskFree)
}

func TestHostInfoResponse_Fields(t *testing.T) {
	hostname := "test"
	data := HostInfoData{
		Hostname: &hostname,
	}

	response := HostInfoResponse{
		Data:   data,
		Result: HostInfoResponseResultOk,
	}

	assert.Equal(t, "test", *response.Data.Hostname)
	assert.Equal(t, HostInfoResponseResultOk, response.Result)
}

func TestHostOptionsRequest_Hostname(t *testing.T) {
	req := HostOptionsRequest{
		Hostname: "new-hostname",
	}

	assert.Equal(t, "new-hostname", req.Hostname)
}

func TestHostOptionsRequest_MarshalJSON(t *testing.T) {
	req := HostOptionsRequest{
		Hostname: "updated-host",
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "updated-host")
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

func TestClient_GetHostInfo(t *testing.T) {
	mockClient := &mockResponseHTTPClient{
		statusCode: http.StatusOK,
		body:       `{"result":"ok","data":{"hostname":"test"}}`,
	}

	client, err := NewClient("http://host.local", WithHTTPClient(mockClient))
	assert.NoError(t, err)

	resp, err := client.GetHostInfo(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestClient_ServerURL(t *testing.T) {
	client, err := NewClient("http://host.local")
	assert.NoError(t, err)
	assert.Equal(t, "http://host.local/", client.Server)
}

func TestClient_WithBaseURL(t *testing.T) {
	client, err := NewClient("http://default.local", WithBaseURL("http://override.local"))
	assert.NoError(t, err)
	assert.Equal(t, "http://override.local/", client.Server)
}

func TestNewGetHostInfoRequest(t *testing.T) {
	req, err := NewGetHostInfoRequest("http://host.local")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodGet, req.Method)
}

func TestNewRebootHostRequest(t *testing.T) {
	req, err := NewRebootHostRequest("http://host.local")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
}

func TestNewShutdownHostRequest(t *testing.T) {
	req, err := NewShutdownHostRequest("http://host.local")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
}

func TestNewSetHostOptionsRequest(t *testing.T) {
	body := HostOptionsRequest{Hostname: "new-host"}
	req, err := NewSetHostOptionsRequest("http://host.local", body)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
}

func TestClient_RequestEditorFn(t *testing.T) {
	mockClient := &mockResponseHTTPClient{
		statusCode: http.StatusOK,
		body:       `{"result":"ok","data":{}}`,
	}

	called := false
	client, err := NewClient("http://host.local",
		WithHTTPClient(mockClient),
		WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			called = true
			req.Header.Set("X-Custom", "test")
			return nil
		}),
	)
	require.NoError(t, err)

	_, err = client.GetHostInfo(context.Background())
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestHostInfoData_ChassisTypes(t *testing.T) {
	tests := []struct {
		name    string
		chassis string
	}{
		{"Desktop", "desktop"},
		{"Laptop", "laptop"},
		{"Server", "server"},
		{"VM", "vm"},
		{"Container", "container"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := HostInfoData{
				Chassis: &tt.chassis,
			}
			assert.Equal(t, tt.chassis, *info.Chassis)
		})
	}
}

func TestHostInfoData_Features(t *testing.T) {
	features := []string{"reboot", "shutdown", "hostname", "services", "network"}

	info := HostInfoData{
		Features: &features,
	}

	assert.Len(t, *info.Features, 5)
	assert.Contains(t, *info.Features, "reboot")
	assert.Contains(t, *info.Features, "shutdown")
	assert.Contains(t, *info.Features, "hostname")
}
