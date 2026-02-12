package resolution

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionResult_Fields(t *testing.T) {
	data := map[string]any{"key": "value"}
	result := Ok

	action := ActionResult{
		Data:   &data,
		Result: &result,
	}

	assert.NotNil(t, action.Data)
	assert.Equal(t, "value", (*action.Data)["key"])
	assert.Equal(t, Ok, *action.Result)
}

func TestActionResult_ZeroValues(t *testing.T) {
	action := ActionResult{}
	assert.Nil(t, action.Data)
	assert.Nil(t, action.Result)
}

func TestCheck_Fields(t *testing.T) {
	enabled := true
	slug := "check_addon_version"

	check := Check{
		Enabled: &enabled,
		Slug:    &slug,
	}

	assert.True(t, *check.Enabled)
	assert.Equal(t, "check_addon_version", *check.Slug)
}

func TestErrorResponse_Fields(t *testing.T) {
	data := map[string]any{"error_code": 500}
	message := "Internal error"

	err := ErrorResponse{
		Data:    &data,
		Message: &message,
	}

	assert.NotNil(t, err.Data)
	assert.Equal(t, 500, (*err.Data)["error_code"])
	assert.Equal(t, "Internal error", *err.Message)
}

func TestIssueContext_Values(t *testing.T) {
	tests := []struct {
		name    string
		context IssueContext
	}{
		{"Addon", IssueContextAddon},
		{"Core", IssueContextCore},
		{"Supervisor", IssueContextSupervisor},
		{"System", IssueContextSystem},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.context)
		})
	}
}

func TestIssue_AllFields(t *testing.T) {
	context := IssueContextAddon
	reference := "addon_slug"
	issueType := "security_issue"
	uuid := types.UUID{}

	issue := Issue{
		Context:   &context,
		Reference: &reference,
		Type:      &issueType,
		Uuid:      &uuid,
	}

	assert.Equal(t, IssueContextAddon, *issue.Context)
	assert.Equal(t, "addon_slug", *issue.Reference)
	assert.Equal(t, "security_issue", *issue.Type)
	assert.NotNil(t, issue.Uuid)
}

func TestIssue_ZeroValues(t *testing.T) {
	issue := Issue{}
	assert.Nil(t, issue.Context)
	assert.Nil(t, issue.Reference)
	assert.Nil(t, issue.Type)
	assert.Nil(t, issue.Uuid)
}

func TestIssue_MarshalJSON(t *testing.T) {
	context := IssueContextCore
	reference := "home_assistant"
	issueType := "update_available"

	issue := Issue{
		Context:   &context,
		Reference: &reference,
		Type:      &issueType,
	}

	data, err := json.Marshal(issue)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "core")
	assert.Contains(t, string(data), "home_assistant")
}

func TestIssue_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"context": "supervisor",
		"reference": "supervisor_update",
		"type": "update_required"
	}`

	var issue Issue
	err := json.Unmarshal([]byte(jsonData), &issue)
	assert.NoError(t, err)
	assert.Equal(t, IssueContextSupervisor, *issue.Context)
	assert.Equal(t, "supervisor_update", *issue.Reference)
	assert.Equal(t, "update_required", *issue.Type)
}

func TestSuggestionContext_Values(t *testing.T) {
	tests := []struct {
		name    string
		context SuggestionContext
	}{
		{"Addon", SuggestionContextAddon},
		{"Core", SuggestionContextCore},
		{"Supervisor", SuggestionContextSupervisor},
		{"System", SuggestionContextSystem},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.context)
		})
	}
}

func TestSuggestion_AllFields(t *testing.T) {
	auto := true
	context := SuggestionContextSystem
	reference := "system_component"
	suggestionType := "restart"
	uuid := types.UUID{}

	suggestion := Suggestion{
		Auto:      &auto,
		Context:   &context,
		Reference: &reference,
		Type:      &suggestionType,
		Uuid:      &uuid,
	}

	assert.True(t, *suggestion.Auto)
	assert.Equal(t, SuggestionContextSystem, *suggestion.Context)
	assert.Equal(t, "system_component", *suggestion.Reference)
	assert.Equal(t, "restart", *suggestion.Type)
	assert.NotNil(t, suggestion.Uuid)
}

func TestSuggestion_ManualAction(t *testing.T) {
	auto := false
	suggestionType := "manual_intervention"

	suggestion := Suggestion{
		Auto: &auto,
		Type: &suggestionType,
	}

	assert.False(t, *suggestion.Auto)
	assert.Equal(t, "manual_intervention", *suggestion.Type)
}

func TestResolutionInfo_AllFields(t *testing.T) {
	checks := []Check{{Enabled: ptrBool(true)}}
	issues := []Issue{{Type: ptrString("test_issue")}}
	suggestions := []Suggestion{{Auto: ptrBool(true)}}
	unhealthy := []string{"component1", "component2"}
	unsupported := []string{"old_addon"}

	info := ResolutionInfo{
		Checks:      &checks,
		Issues:      &issues,
		Suggestions: &suggestions,
		Unhealthy:   &unhealthy,
		Unsupported: &unsupported,
	}

	assert.NotNil(t, info.Checks)
	assert.Len(t, *info.Checks, 1)
	assert.NotNil(t, info.Issues)
	assert.Len(t, *info.Issues, 1)
	assert.NotNil(t, info.Suggestions)
	assert.Len(t, *info.Suggestions, 1)
	assert.NotNil(t, info.Unhealthy)
	assert.Contains(t, *info.Unhealthy, "component1")
	assert.NotNil(t, info.Unsupported)
	assert.Contains(t, *info.Unsupported, "old_addon")
}

func TestResolutionInfo_ZeroValues(t *testing.T) {
	info := ResolutionInfo{}
	assert.Nil(t, info.Checks)
	assert.Nil(t, info.Issues)
	assert.Nil(t, info.Suggestions)
	assert.Nil(t, info.Unhealthy)
	assert.Nil(t, info.Unsupported)
}

func TestResolutionInfo_EmptyLists(t *testing.T) {
	checks := []Check{}
	issues := []Issue{}
	suggestions := []Suggestion{}

	info := ResolutionInfo{
		Checks:      &checks,
		Issues:      &issues,
		Suggestions: &suggestions,
	}

	assert.NotNil(t, info.Checks)
	assert.Empty(t, *info.Checks)
	assert.NotNil(t, info.Issues)
	assert.Empty(t, *info.Issues)
	assert.NotNil(t, info.Suggestions)
	assert.Empty(t, *info.Suggestions)
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

func TestClient_GetResolutionInfo(t *testing.T) {
	mockClient := &mockResponseHTTPClient{
		statusCode: http.StatusOK,
		body:       `{"issues":[],"suggestions":[],"checks":[]}`,
	}

	client, err := NewClient("http://resolution.local", WithHTTPClient(mockClient))
	assert.NoError(t, err)

	resp, err := client.GetResolutionInfo(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestClient_ServerURL(t *testing.T) {
	client, err := NewClient("http://resolution.local")
	assert.NoError(t, err)
	assert.Equal(t, "http://resolution.local/", client.Server)
}

func TestClient_WithBaseURL(t *testing.T) {
	client, err := NewClient("http://default.local", WithBaseURL("http://override.local"))
	assert.NoError(t, err)
	assert.Equal(t, "http://override.local/", client.Server)
}

func TestNewGetResolutionInfoRequest(t *testing.T) {
	req, err := NewGetResolutionInfoRequest("http://resolution.local")
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, http.MethodGet, req.Method)
}

func TestClient_RequestEditorFn(t *testing.T) {
	mockClient := &mockResponseHTTPClient{
		statusCode: http.StatusOK,
		body:       `{"issues":[]}`,
	}

	called := false
	client, err := NewClient("http://resolution.local",
		WithHTTPClient(mockClient),
		WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			called = true
			req.Header.Set("X-Custom", "test")
			return nil
		}),
	)
	require.NoError(t, err)

	_, err = client.GetResolutionInfo(context.Background())
	assert.NoError(t, err)
	assert.True(t, called)
}

// Helper functions
func ptrBool(b bool) *bool {
	return &b
}

func ptrString(s string) *string {
	return &s
}
