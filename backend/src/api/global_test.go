package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSettingsHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/global", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetSettings)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	context_state := (&dto.ContextState{}).FromContext(testContext)

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(context_state.Settings)
	require.NoError(t, jsonError)

	assert.Equal(t, rr.Body.String(), string(expected[:]))

	assert.False(t, context_state.DataDirtyTracker.Settings)
}

func TestUpdateSettingsHandler(t *testing.T) {
	glc := dto.Settings{
		Workgroup: "pluto&admin",
	}
	context_state := (&dto.ContextState{}).FromContext(testContext)

	jsonBody, jsonError := json.Marshal(glc)
	require.NoError(t, jsonError)
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/global", strings.NewReader(string(jsonBody)))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/global", UpdateSettings).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var res dto.Settings
	err = json.Unmarshal(rr.Body.Bytes(), &res)
	require.NoError(t, err, "Body %#v", rr.Body.String())

	assert.Equal(t, res.Workgroup, glc.Workgroup)
	assert.True(t, context_state.DataDirtyTracker.Settings)

}

func TestUpdateSettinsSameConfigHandler(t *testing.T) {
	context_state := (&dto.ContextState{}).FromContext(testContext)
	context_state.DataDirtyTracker.Settings = false

	jsonBody, jsonError := json.Marshal(context_state.Settings)
	require.NoError(t, jsonError)
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/global", strings.NewReader(string(jsonBody)))
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/global", UpdateSettings).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.False(t, context_state.DataDirtyTracker.Settings)
}

func TestPersistConfig(t *testing.T) { //TODO: Test also the real persistence logic
	context_state := (&dto.ContextState{}).FromContext(testContext)
	context_state.DataDirtyTracker.Settings = true
	context_state.DataDirtyTracker.Users = true
	context_state.DataDirtyTracker.Shares = true
	context_state.DataDirtyTracker.Volumes = true

	// Create a request to pass to our handler
	req, err := http.NewRequestWithContext(testContext, "PUT", "/config", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PersistAllConfig)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check if DirtySectionState flags are set to false
	assert.False(t, context_state.DataDirtyTracker.Settings)
	assert.False(t, context_state.DataDirtyTracker.Users)
	assert.False(t, context_state.DataDirtyTracker.Shares)
	assert.False(t, context_state.DataDirtyTracker.Volumes)

}

func TestRollbackConfig(t *testing.T) {
	t.Skip("TODO: Test also the real rollback logic")
	context_state := (&dto.ContextState{}).FromContext(testContext)
	context_state.DataDirtyTracker.Settings = true
	context_state.DataDirtyTracker.Users = true
	context_state.DataDirtyTracker.Shares = true
	context_state.DataDirtyTracker.Volumes = true

	var tmpWRG = "WORKGROUP" //context_state.Settings.Workgroup
	context_state.Settings.Workgroup = "pluto&admin_rb"

	// Create a request to pass to our handler
	req, err := http.NewRequestWithContext(testContext, "DELETE", "/config", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(RollbackConfig)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check if DirtySectionState flags are set to false

	assert.Equal(t, context_state.Settings.Workgroup, tmpWRG) // rollback to original workgroup
	assert.False(t, context_state.DataDirtyTracker.Settings)
	//assert.False(t, data_dirty_tracker.Users)
	//assert.False(t, data_dirty_tracker.Shares)
	//assert.False(t, data_dirty_tracker.Volumes)

}
