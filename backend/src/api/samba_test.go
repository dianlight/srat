// endpoints_test.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestApplySambaHandler(t *testing.T) {

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "POST", "/samba/apply", nil)
	assert.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/samba/apply", ApplySamba).Methods(http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusNoContent, rr.Code, "Expected status code 204, got %d with Body %#v", rr.Code, rr.Body.String())
}

func checkStringInSMBConfig(testvalue string, expected string, t *testing.T) bool {
	stream, err := createConfigStream(testContext)
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	rexpt := fmt.Sprintf(expected, testvalue)

	m, err := regexp.MatchString(rexpt, string(*stream))
	assert.NoError(t, err)
	assert.True(t, m, "Wrong Match `%s` not found in stream \n%s", rexpt, string(*stream))

	return true
}

func TestCreateConfigStream(t *testing.T) {
	stream, err := createConfigStream(testContext)
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	samba_config_file := testContext.Value("samba_config_file").(*string)
	assert.NotEmpty(t, *samba_config_file)

	fsbyte, err := os.ReadFile(*samba_config_file)
	assert.NoError(t, err)
	assert.Equal(t, len(fsbyte), len(*stream))
}

// check migrate config don't duplicate share

func TestGetSambaProcessStatus(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequestWithContext(testContext, "GET", "/samba/status", nil)
	assert.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/samba/status", GetSambaProcessStatus).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, rr.Code)
}

func TestGetSambaConfig(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequestWithContext(testContext, "GET", "/samba", nil)
	assert.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/samba", GetSambaConfig).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	var responseBody dto.SmbConf
	err = json.Unmarshal(rr.Body.Bytes(), &responseBody)
	assert.NoError(t, err)

	// Create the expected config stream
	expectedStream, err := createConfigStream(testContext)
	assert.NoError(t, err)

	// Create the expected SmbConf
	var expectedSmbConf dto.SmbConf
	expectedSmbConf.From(*expectedStream)

	// Compare the response body with the expected SmbConf
	assert.Equal(t, expectedSmbConf, responseBody)
}
