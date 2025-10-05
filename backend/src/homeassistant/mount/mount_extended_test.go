package mount

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

func TestMountType_Values(t *testing.T) {
	tests := []struct {
		name      string
		mountType MountType
	}{
		{"CIFS", Cifs},
		{"NFS", Nfs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.mountType)
		})
	}
}

func TestMountUsage_Values(t *testing.T) {
	tests := []struct {
		name  string
		usage MountUsage
	}{
		{"Backup", Backup},
		{"Media", Media},
		{"Share", Share},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.usage)
		})
	}
}

func TestMount_CIFSMount(t *testing.T) {
	name := "nas_share"
	password := "secretpass"
	port := 445
	readOnly := false
	server := "192.168.1.100"
	share := "public"
	state := "active"
	mountType := Cifs
	usage := Share
	username := "admin"

	mount := Mount{
		Name:     &name,
		Password: &password,
		Port:     &port,
		ReadOnly: &readOnly,
		Server:   &server,
		Share:    &share,
		State:    &state,
		Type:     &mountType,
		Usage:    &usage,
		Username: &username,
	}

	assert.Equal(t, "nas_share", *mount.Name)
	assert.Equal(t, "secretpass", *mount.Password)
	assert.Equal(t, 445, *mount.Port)
	assert.False(t, *mount.ReadOnly)
	assert.Equal(t, "192.168.1.100", *mount.Server)
	assert.Equal(t, "public", *mount.Share)
	assert.Equal(t, "active", *mount.State)
	assert.Equal(t, Cifs, *mount.Type)
	assert.Equal(t, Share, *mount.Usage)
	assert.Equal(t, "admin", *mount.Username)
}

func TestMount_NFSMount(t *testing.T) {
	name := "nfs_backup"
	path := "/mnt/backup"
	port := 2049
	readOnly := true
	server := "nfs.example.com"
	state := "active"
	mountType := Nfs
	usage := Backup

	mount := Mount{
		Name:     &name,
		Path:     &path,
		Port:     &port,
		ReadOnly: &readOnly,
		Server:   &server,
		State:    &state,
		Type:     &mountType,
		Usage:    &usage,
	}

	assert.Equal(t, "nfs_backup", *mount.Name)
	assert.Equal(t, "/mnt/backup", *mount.Path)
	assert.Equal(t, 2049, *mount.Port)
	assert.True(t, *mount.ReadOnly)
	assert.Equal(t, "nfs.example.com", *mount.Server)
	assert.Equal(t, "active", *mount.State)
	assert.Equal(t, Nfs, *mount.Type)
	assert.Equal(t, Backup, *mount.Usage)
	assert.Nil(t, mount.Username) // NFS doesn't use username
	assert.Nil(t, mount.Password) // NFS doesn't use password
	assert.Nil(t, mount.Share)    // NFS doesn't use share
}

func TestMount_ZeroValues(t *testing.T) {
	mount := Mount{}

	assert.Nil(t, mount.Name)
	assert.Nil(t, mount.Password)
	assert.Nil(t, mount.Path)
	assert.Nil(t, mount.Port)
	assert.Nil(t, mount.ReadOnly)
	assert.Nil(t, mount.Server)
	assert.Nil(t, mount.Share)
	assert.Nil(t, mount.State)
	assert.Nil(t, mount.Type)
	assert.Nil(t, mount.Usage)
	assert.Nil(t, mount.Username)
}

func TestMount_MarshalJSON(t *testing.T) {
	name := "test_mount"
	server := "192.168.1.1"
	mountType := Cifs

	mount := Mount{
		Name:   &name,
		Server: &server,
		Type:   &mountType,
	}

	data, err := json.Marshal(mount)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "test_mount")
	assert.Contains(t, string(data), "192.168.1.1")
	assert.Contains(t, string(data), "cifs")
}

func TestMount_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"name": "media_share",
		"server": "nas.local",
		"share": "media",
		"type": "cifs",
		"usage": "media",
		"read_only": false,
		"username": "user",
		"password": "pass"
	}`

	var mount Mount
	err := json.Unmarshal([]byte(jsonData), &mount)
	assert.NoError(t, err)
	assert.Equal(t, "media_share", *mount.Name)
	assert.Equal(t, "nas.local", *mount.Server)
	assert.Equal(t, "media", *mount.Share)
	assert.Equal(t, Cifs, *mount.Type)
	assert.Equal(t, Media, *mount.Usage)
	assert.False(t, *mount.ReadOnly)
	assert.Equal(t, "user", *mount.Username)
	assert.Equal(t, "pass", *mount.Password)
}

func TestMount_ReadOnlyMount(t *testing.T) {
	readOnly := true
	mount := Mount{
		ReadOnly: &readOnly,
	}

	assert.True(t, *mount.ReadOnly)
}

func TestMount_States(t *testing.T) {
	tests := []struct {
		name  string
		state string
	}{
		{"Active", "active"},
		{"Failed", "failed"},
		{"Inactive", "inactive"},
		{"Mounting", "mounting"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mount := Mount{
				State: &tt.state,
			}
			assert.Equal(t, tt.state, *mount.State)
		})
	}
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

func TestClient_GetMounts(t *testing.T) {
	mockClient := &mockResponseHTTPClient{
		statusCode: http.StatusOK,
		body:       `{"mounts":[{"name":"test"}]}`,
	}

	client, err := NewClient("http://mount.local", WithHTTPClient(mockClient))
	assert.NoError(t, err)

	resp, err := client.GetMounts(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestClient_ServerURL(t *testing.T) {
	client, err := NewClient("http://mount.local")
	assert.NoError(t, err)
	assert.Equal(t, "http://mount.local/", client.Server)
}

func TestClient_WithBaseURL(t *testing.T) {
	client, err := NewClient("http://default.local", WithBaseURL("http://override.local"))
	assert.NoError(t, err)
	assert.Equal(t, "http://override.local/", client.Server)
}

func TestNewGetMountsRequest(t *testing.T) {
	req, err := NewGetMountsRequest("http://mount.local")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodGet, req.Method)
}

func TestNewCreateMountRequest(t *testing.T) {
	name := "new_mount"
	body := Mount{Name: &name}

	req, err := NewCreateMountRequest("http://mount.local", body)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
}

func TestNewRemoveMountRequest(t *testing.T) {
	req, err := NewRemoveMountRequest("http://mount.local", "mount_name")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodDelete, req.Method)
	assert.Contains(t, req.URL.Path, "mount_name")
}

func TestNewReloadMountRequest(t *testing.T) {
	req, err := NewReloadMountRequest("http://mount.local", "mount_name")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Contains(t, req.URL.Path, "mount_name")
	assert.Contains(t, req.URL.Path, "reload")
}

func TestNewUpdateMountRequest(t *testing.T) {
	name := "updated_mount"
	body := Mount{Name: &name}

	req, err := NewUpdateMountRequest("http://mount.local", "mount_name", body)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodPut, req.Method)
	assert.Contains(t, req.URL.Path, "mount_name")
}

func TestClient_RequestEditorFn(t *testing.T) {
	mockClient := &mockResponseHTTPClient{
		statusCode: http.StatusOK,
		body:       `{"mounts":[]}`,
	}

	called := false
	client, err := NewClient("http://mount.local",
		WithHTTPClient(mockClient),
		WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			called = true
			req.Header.Set("X-Custom", "test")
			return nil
		}),
	)
	require.NoError(t, err)

	_, err = client.GetMounts(context.Background())
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestMount_DefaultPorts(t *testing.T) {
	tests := []struct {
		name      string
		mountType MountType
		port      int
	}{
		{"CIFS default", Cifs, 445},
		{"NFS default", Nfs, 2049},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mount := Mount{
				Type: &tt.mountType,
				Port: &tt.port,
			}
			assert.Equal(t, tt.port, *mount.Port)
		})
	}
}

func TestMount_UsageTypes(t *testing.T) {
	tests := []struct {
		name      string
		usage     MountUsage
		mountType MountType
	}{
		{"Backup with NFS", Backup, Nfs},
		{"Media with CIFS", Media, Cifs},
		{"Share with CIFS", Share, Cifs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mount := Mount{
				Usage: &tt.usage,
				Type:  &tt.mountType,
			}
			assert.Equal(t, tt.usage, *mount.Usage)
			assert.Equal(t, tt.mountType, *mount.Type)
		})
	}
}
