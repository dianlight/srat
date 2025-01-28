package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
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
	require.Equal(t, http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	//context_state := (&context.Status{}).FromContext(testContext)
	context_state := StateFromContext(testContext)

	var config config.Config
	err = config.FromContext(testContext)
	require.NoError(t, err)
	var expected dto.Settings
	var conv converter.ConfigToDtoConverterImpl
	err = conv.ConfigToSettings(config, &expected)
	//err = mapper.Map(context.Background(), &expected, config)
	require.NoError(t, err)

	// Check the response body is what we expect.
	var returned dto.Settings
	jsonError := json.Unmarshal(rr.Body.Bytes(), &returned)
	require.NoError(t, jsonError)

	assert.Equal(t, expected, returned)

	assert.False(t, context_state.DataDirtyTracker.Settings)
}

func TestUpdateSettingsHandler(t *testing.T) {
	glc := dto.Settings{
		Workgroup: "pluto&admin",
	}
	//context_state := (&dto.Status{}).FromContext(testContext)
	context_state := StateFromContext(testContext)

	jsonBody, jsonError := json.Marshal(glc)
	require.NoError(t, jsonError)
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/global", strings.NewReader(string(jsonBody)))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/global", UpdateSettings).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var res dto.Settings
	err = json.Unmarshal(rr.Body.Bytes(), &res)
	require.NoError(t, err, "Body %#v", rr.Body.String())

	assert.Equal(t, glc.Workgroup, res.Workgroup)
	assert.EqualValues(t, []string{"10.0.0.0/8", "100.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16", "fe80::/10", "fc00::/7"}, res.AllowHost)
	assert.True(t, context_state.DataDirtyTracker.Settings)

	// Restore original state
	var properties dbom.Properties
	if err := properties.Load(); err != nil {
		t.Fatalf("Failed to load properties: %v", err)
	}
	if err := properties.SetValue("Workgroup", "WORKGROUP"); err != nil {
		t.Fatalf("Failed to add workgroup property: %v", err)
	}
}
