package addons

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

func TestNewClientAppliesOptions(t *testing.T) {
	mock := &mockHTTPClient{}
	editorCalled := false

	client, err := NewClient("http://example.com", WithHTTPClient(mock), WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		editorCalled = true
		req.Header.Set("X-Test", "true")
		return nil
	}))
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/", client.Server)

	_, err = client.GetSelfAddonInfo(context.Background())
	require.Error(t, err)
	assert.True(t, mock.called)
	require.NotNil(t, mock.lastReq)
	assert.Equal(t, "true", mock.lastReq.Header.Get("X-Test"))
	assert.True(t, editorCalled)

	additionalCalled := false
	req, err := NewGetSelfAddonInfoRequest(client.Server)
	require.NoError(t, err)
	err = client.applyEditors(context.Background(), req, []RequestEditorFn{func(ctx context.Context, req *http.Request) error {
		additionalCalled = true
		return nil
	}})
	require.NoError(t, err)
	assert.True(t, additionalCalled)

	overrideClient, err := NewClient("http://example.net", WithBaseURL("http://override.local"))
	require.NoError(t, err)
	assert.Equal(t, "http://override.local/", overrideClient.Server)
}
