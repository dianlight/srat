package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestListSharesHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/shares", nil)
	assert.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(ListShares)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	addon_config := testContext.Value("addon_config").(*config.Config)
	var expectedDto dto.SharedResources
	expectedDto.From(addon_config.Shares)
	expected, jsonError := json.Marshal(expectedDto)
	assert.NoError(t, jsonError)
	assert.Equal(t, string(expected), rr.Body.String())
}

func TestGetShareHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/share/LIBRARY", nil)
	assert.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", GetShare).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	addon_config := testContext.Value("addon_config").(*config.Config)
	var expectedDto dto.SharedResource
	expectedDto.From(addon_config.Shares["LIBRARY"])
	expected, jsonError := json.Marshal(expectedDto)
	assert.NoError(t, jsonError)
	assert.Equal(t, string(expected), rr.Body.String())
}

func TestCreateShareHandler(t *testing.T) {

	share := config.Share{
		Name: "PIPPO",
		Path: "/pippo",
		FS:   "tmpfs",
	}

	jsonBody, jsonError := json.Marshal(share)
	assert.NoError(t, jsonError)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "POST", "/share", strings.NewReader(string(jsonBody)))
	assert.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share", CreateShare).Methods(http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(share)
	assert.NoError(t, jsonError)
	assert.Equal(t, string(expected), rr.Body.String())
}

func TestCreateShareDuplicateHandler(t *testing.T) {

	share := config.Share{
		Name:        "LIBRARY",
		Path:        "/mnt/LIBRARY",
		FS:          "ext4",
		RoUsers:     []string{"rouser"},
		TimeMachine: true,
		Users:       []string{"dianlight"},
		Usage:       "media",
	}

	jsonBody, jsonError := json.Marshal(share)
	assert.NoError(t, jsonError)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "POST", "/share/LIBRARY", strings.NewReader(string(jsonBody)))
	assert.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", CreateShare).Methods(http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusConflict, rr.Code)

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(dto.ResponseError{Error: "Share already exists", Body: share})
	assert.NoError(t, jsonError)
	assert.Equal(t, string(expected), rr.Body.String())
}

func TestUpdateShareHandler(t *testing.T) {

	share := dto.SharedResource{
		Path: "/pippo",
	}

	jsonBody, jsonError := json.Marshal(share)
	assert.NoError(t, jsonError)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/share/LIBRARY", strings.NewReader(string(jsonBody)))
	assert.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", UpdateShare).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	share.FS = "ext4"
	share.Name = "LIBRARY"
	share.RoUsers = []string{"rouser"}
	share.TimeMachine = true
	share.Users = []string{"dianlight"}
	share.Usage = "media"
	expected, jsonError := json.Marshal(share)
	assert.NoError(t, jsonError)
	assert.Equal(t, string(expected), rr.Body.String())
}

func TestDeleteShareHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "DELETE", "/share/LIBRARY", nil)
	assert.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", DeleteShare).Methods(http.MethodDelete)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Refresh shares list anche check that LIBRARY don't exists
	req, err = http.NewRequestWithContext(testContext, "GET", "/shares", nil)
	assert.NoError(t, err)
	rr = httptest.NewRecorder()
	handler := http.HandlerFunc(ListShares)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NotContains(t, rr.Body.String(), "LIBRARY", "LIBRARY share still exists")
}
