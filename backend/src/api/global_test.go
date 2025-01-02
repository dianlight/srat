package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dm"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestGetGlobalConfigHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/global", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetGlobalConfig)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var expectedDto = dto.Settings{}
	addon_config := testContext.Value("addon_config").(*config.Config)
	expectedDto.From(addon_config)

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(expectedDto)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}

	assert.Equal(t, rr.Body.String(), string(expected[:]))
	//	if rr.Body.String() != string(expected[:]) {
	//		t.Errorf("handler returned unexpected body: go\n %v want\n %v",
	//			rr.Body.String(), string(expected[:]))
	//	}

	assert.False(t, testContext.Value("data_dirty_tracker").(*dm.DataDirtyTracker).Settings)
}

func TestUpdateGlobalConfigHandler(t *testing.T) {
	glc := dto.Settings{
		Workgroup: "pluto&admin",
	}

	jsonBody, jsonError := json.Marshal(glc)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/global", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/global", UpdateGlobalConfig).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)

	var res dto.Settings
	err = json.Unmarshal(rr.Body.Bytes(), &res)
	if err != nil {
		t.Errorf("Unable to decode JSON %s", err.Error())
	}

	assert.True(t, testContext.Value("data_dirty_tracker").(*dm.DataDirtyTracker).Settings)

	assert.Equal(t, res.Workgroup, glc.Workgroup)
}

func TestUpdateGlobalConfigSameConfigHandler(t *testing.T) {
	var glc = dto.Settings{}
	addon_config := testContext.Value("addon_config").(*config.Config)
	assert.Equal(t, addon_config.Workgroup, "pluto&admin")
	testContext.Value("data_dirty_tracker").(*dm.DataDirtyTracker).Settings = false

	glc.From(addon_config)

	jsonBody, jsonError := json.Marshal(glc)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/global", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/global", UpdateGlobalConfig).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusNoContent)
	assert.False(t, testContext.Value("data_dirty_tracker").(*dm.DataDirtyTracker).Settings)
}
