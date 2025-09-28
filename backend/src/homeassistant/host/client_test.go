package host

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHTTPClient struct {
	called  bool
	lastReq *http.Request
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.called = true
	m.lastReq = req
	return nil, errors.New("network error")
}

func TestClientOptions(t *testing.T) {
	mock := &mockHTTPClient{}
	editorCalled := false

	client, err := NewClient("http://host.local", WithHTTPClient(mock), WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		editorCalled = true
		req.Header.Set("X-Host", "1")
		return nil
	}))
	require.NoError(t, err)
	assert.Equal(t, "http://host.local/", client.Server)

	_, err = client.GetHostInfo(context.Background())
	require.Error(t, err)
	assert.True(t, mock.called)
	require.NotNil(t, mock.lastReq)
	assert.Equal(t, "1", mock.lastReq.Header.Get("X-Host"))
	assert.True(t, editorCalled)

	additionalCalled := false
	req, err := NewGetHostInfoRequest(client.Server)
	require.NoError(t, err)
	err = client.applyEditors(context.Background(), req, []RequestEditorFn{func(ctx context.Context, req *http.Request) error {
		additionalCalled = true
		return nil
	}})
	require.NoError(t, err)
	assert.True(t, additionalCalled)

	overrideClient, err := NewClient("http://host.local", WithBaseURL("http://override.local"))
	require.NoError(t, err)
	assert.Equal(t, "http://override.local/", overrideClient.Server)
}
