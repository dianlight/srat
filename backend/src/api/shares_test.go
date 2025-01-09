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
	"github.com/stretchr/testify/require"
)

func TestListSharesHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/shares", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(ListShares)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code, "Body %#v", rr.Body.String())

	// Check the response body is what we expect.
	//context_state := (&dto.ContextState{}).FromContext(testContext)
	resultsDto := dto.SharedResources{}
	jsonError := json.Unmarshal(rr.Body.Bytes(), &resultsDto)
	require.NoError(t, jsonError, "Body %#v", rr.Body.String())
	assert.NotEmpty(t, resultsDto)
	var config config.Config
	config.FromContext(testContext)
	assert.EqualValues(t, resultsDto, config.Shares)

	for _, sdto := range resultsDto {
		assert.NotEmpty(t, sdto.Path)
		if sdto.DeviceId == nil {
			assert.NoDirExists(t, sdto.Path, "DeviceId is false but %s exists", sdto.Path)
		} else {
			assert.DirExists(t, sdto.Path, "DeviceId is true but %s doesn't exist", sdto.Path)
		}
		if sdto.Invalid {
			assert.NoDirExists(t, sdto.Path, "Share is invalid  but %s exists", sdto.Path)
		}
	}

}

func TestGetShareHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/share/LIBRARY", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", GetShare).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	//context_state := (&dto.ContextState{}).FromContext(testContext)
	resultShare := dto.SharedResource{}
	jsonError := json.Unmarshal(rr.Body.Bytes(), &resultShare)
	require.NoError(t, jsonError)

	var config config.Config
	config.FromContext(testContext)
	assert.Equal(t, config.Shares["LIBRARY"], resultShare)

	assert.EqualValues(t, config.Shares["LIBRARY"], resultShare)
}

func TestCreateShareHandler(t *testing.T) {

	share := dto.SharedResource{
		Name: "PIPPO",
		Path: "/pippo",
		FS:   "tmpfs",
	}

	jsonBody, jsonError := json.Marshal(share)
	require.NoError(t, jsonError)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "POST", "/share", strings.NewReader(string(jsonBody)))
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share", CreateShare).Methods(http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(share)
	require.NoError(t, jsonError)
	assert.Equal(t, string(expected), rr.Body.String())
}

func TestCreateShareDuplicateHandler(t *testing.T) {

	share := dto.SharedResource{
		Name:        "LIBRARY",
		Path:        "/mnt/LIBRARY",
		FS:          "ext4",
		RoUsers:     []string{"rouser"},
		TimeMachine: true,
		Users:       []string{"dianlight"},
		Usage:       "media",
	}

	jsonBody, jsonError := json.Marshal(share)
	require.NoError(t, jsonError)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "POST", "/share/LIBRARY", strings.NewReader(string(jsonBody)))
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", CreateShare).Methods(http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusConflict, rr.Code)

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(dto.ResponseError{Error: "Share already exists", Body: share})
	require.NoError(t, jsonError)
	assert.Equal(t, string(expected)[:len(expected)-3], rr.Body.String()[:len(expected)-3])
}

func TestUpdateShareHandler(t *testing.T) {

	share := dto.SharedResource{
		Path: "/pippo",
	}

	jsonBody, jsonError := json.Marshal(share)
	require.NoError(t, jsonError)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/share/LIBRARY", strings.NewReader(string(jsonBody)))
	require.NoError(t, err)

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
	require.NoError(t, jsonError)
	assert.Equal(t, string(expected)[:len(expected)-3], rr.Body.String()[:len(expected)-3])
}

func TestDeleteShareHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "DELETE", "/share/LIBRARY", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", DeleteShare).Methods(http.MethodDelete)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Refresh shares list anche check that LIBRARY don't exists
	req, err = http.NewRequestWithContext(testContext, "GET", "/shares", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()
	handler := http.HandlerFunc(ListShares)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NotContains(t, rr.Body.String(), "LIBRARY", "LIBRARY share still exists")
}
