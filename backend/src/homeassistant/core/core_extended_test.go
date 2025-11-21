package core

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoreCheck_Fields(t *testing.T) {
	errors := []string{"error1", "error2"}
	result := "ok"

	check := CoreCheck{
		Errors: &errors,
		Result: &result,
	}

	assert.NotNil(t, check.Errors)
	assert.Len(t, *check.Errors, 2)
	assert.Contains(t, *check.Errors, "error1")
	assert.Equal(t, "ok", *check.Result)
}

func TestCoreCheck_EmptyErrors(t *testing.T) {
	result := "ok"
	check := CoreCheck{
		Result: &result,
	}

	assert.Nil(t, check.Errors)
	assert.Equal(t, "ok", *check.Result)
}

func TestCoreInfo_AllFields(t *testing.T) {
	arch := "amd64"
	audioIn := "default"
	audioOut := "speakers"
	excludeDB := true
	boot := true
	image := "homeassistant/home-assistant:latest"
	ip := "172.17.0.1"
	machine := "raspberrypi4"
	port := 8123
	ssl := false
	updateAvail := true
	version := "2024.1.0"
	versionLatest := "2024.2.0"
	waitBoot := 600
	watchdog := true

	info := CoreInfo{
		Arch:                   &arch,
		AudioInput:             &audioIn,
		AudioOutput:            &audioOut,
		BackupsExcludeDatabase: &excludeDB,
		Boot:                   &boot,
		Image:                  &image,
		IpAddress:              &ip,
		Machine:                &machine,
		Port:                   &port,
		Ssl:                    &ssl,
		UpdateAvailable:        &updateAvail,
		Version:                &version,
		VersionLatest:          &versionLatest,
		WaitBoot:               &waitBoot,
		Watchdog:               &watchdog,
	}

	assert.Equal(t, "amd64", *info.Arch)
	assert.Equal(t, "default", *info.AudioInput)
	assert.Equal(t, "speakers", *info.AudioOutput)
	assert.True(t, *info.BackupsExcludeDatabase)
	assert.True(t, *info.Boot)
	assert.Equal(t, "homeassistant/home-assistant:latest", *info.Image)
	assert.Equal(t, "172.17.0.1", *info.IpAddress)
	assert.Equal(t, "raspberrypi4", *info.Machine)
	assert.Equal(t, 8123, *info.Port)
	assert.False(t, *info.Ssl)
	assert.True(t, *info.UpdateAvailable)
	assert.Equal(t, "2024.1.0", *info.Version)
	assert.Equal(t, "2024.2.0", *info.VersionLatest)
	assert.Equal(t, 600, *info.WaitBoot)
	assert.True(t, *info.Watchdog)
}

func TestCoreInfo_ZeroValues(t *testing.T) {
	info := CoreInfo{}

	assert.Nil(t, info.Arch)
	assert.Nil(t, info.AudioInput)
	assert.Nil(t, info.AudioOutput)
	assert.Nil(t, info.BackupsExcludeDatabase)
	assert.Nil(t, info.Boot)
	assert.Nil(t, info.Image)
	assert.Nil(t, info.IpAddress)
	assert.Nil(t, info.Machine)
	assert.Nil(t, info.Port)
	assert.Nil(t, info.Ssl)
	assert.Nil(t, info.UpdateAvailable)
	assert.Nil(t, info.Version)
	assert.Nil(t, info.VersionLatest)
	assert.Nil(t, info.WaitBoot)
	assert.Nil(t, info.Watchdog)
}

func TestCoreUpdate_Fields(t *testing.T) {
	version := "2024.3.0"
	update := CoreUpdate{
		Version: &version,
	}

	assert.NotNil(t, update.Version)
	assert.Equal(t, "2024.3.0", *update.Version)
}

func TestCoreUpdate_NilVersion(t *testing.T) {
	update := CoreUpdate{}
	assert.Nil(t, update.Version)
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

func TestClient_GetCoreInfoSuccess(t *testing.T) {
	mockClient := &mockResponseHTTPClient{
		statusCode: http.StatusOK,
		body:       `{"data":{"version":"2024.1.0","arch":"amd64"}}`,
	}

	client, err := NewClient("http://core.local", WithHTTPClient(mockClient))
	assert.NoError(t, err)

	resp, err := client.GetCoreInfo(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestClient_ServerURL(t *testing.T) {
	client, err := NewClient("http://core.local")
	assert.NoError(t, err)
	assert.Equal(t, "http://core.local/", client.Server)
}

func TestNewGetCoreInfoRequest(t *testing.T) {
	req, err := NewGetCoreInfoRequest("http://core.local")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodGet, req.Method)
}

func TestNewCheckCoreConfigRequest(t *testing.T) {
	req, err := NewCheckCoreConfigRequest("http://core.local")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
}

func TestNewRestartCoreRequest(t *testing.T) {
	req, err := NewRestartCoreRequest("http://core.local")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
}

func TestNewRebootCoreRequest(t *testing.T) {
	req, err := NewRebootCoreRequest("http://core.local")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
}

func TestNewRepairCoreRequest(t *testing.T) {
	req, err := NewRepairCoreRequest("http://core.local")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
}

func TestCoreInfo_Architectures(t *testing.T) {
	tests := []struct {
		name string
		arch string
	}{
		{"AMD64", "amd64"},
		//	{"ARMv7", "armhf"},
		{"AArch64", "aarch64"},
		//	{"i386", "i386"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := CoreInfo{
				Arch: &tt.arch,
			}
			assert.Equal(t, tt.arch, *info.Arch)
		})
	}
}

func TestCoreInfo_UpdateAvailable(t *testing.T) {
	version := "2024.1.0"
	versionLatest := "2024.2.0"
	updateAvail := true

	info := CoreInfo{
		Version:         &version,
		VersionLatest:   &versionLatest,
		UpdateAvailable: &updateAvail,
	}

	assert.True(t, *info.UpdateAvailable)
	assert.NotEqual(t, *info.Version, *info.VersionLatest)
}
