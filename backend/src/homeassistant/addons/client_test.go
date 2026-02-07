package addons

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientAppliesOptions(t *testing.T) {
	mockTransport := httpmock.NewMockTransport()
	called := false
	var lastReq *http.Request
	mockTransport.RegisterResponder(http.MethodGet, "http://example.com/addons/self/info", func(req *http.Request) (*http.Response, error) {
		called = true
		lastReq = req
		return nil, errors.New("network error")
	})
	clientHTTP := &http.Client{Transport: mockTransport}
	editorCalled := false

	client, err := NewClient("http://example.com", WithHTTPClient(clientHTTP), WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		editorCalled = true
		req.Header.Set("X-Test", "true")
		return nil
	}))
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/", client.Server)

	_, err = client.GetSelfAddonInfo(context.Background())
	require.Error(t, err)
	assert.True(t, called)
	require.NotNil(t, lastReq)
	assert.Equal(t, "true", lastReq.Header.Get("X-Test"))
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
