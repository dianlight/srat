// endpoints_test.go
package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ztrue/tracerr"
)

func TestCreateConfigStream(t *testing.T) {
	stream, err := api.CreateConfigStream(testContext)
	require.NoError(t, err, tracerr.SprintSourceColor(err))
	assert.NotNil(t, stream)

	//ctx := testContext.Value("context_state").(*dto.Status)
	ctx := api.StateFromContext(testContext)
	assert.NotEmpty(t, ctx)

	//samba_config_file := testContext.Value("samba_config_file").(*string)
	assert.NotEmpty(t, ctx.SambaConfigFile)

	fsbyte, err := os.ReadFile(ctx.SambaConfigFile)
	require.NoError(t, err)

	var re = regexp.MustCompile(`(?m)^\[([^[]+)\]\n(?:^[^[].*\n+)+`)

	var result = make(map[string]string)
	//t.Log(fmt.Sprintf("%s", *stream))
	for _, match := range re.FindAllStringSubmatch(string(*stream), -1) {
		result[match[1]] = strings.TrimSpace(match[0])
	}

	var expected = make(map[string]string)
	for _, match := range re.FindAllStringSubmatch(string(fsbyte), -1) {
		expected[match[1]] = strings.TrimSpace(match[0])
	}

	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}
	assert.Len(t, keys, len(expected), result)
	m1 := regexp.MustCompile(`/\*(.*)\*/`)

	for k, v := range result {
		//assert.EqualValues(t, expected[k], v)
		var elines = strings.Split(expected[k], "\n")
		var lines = strings.Split(v, "\n")

		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "# DEBUG:") && strings.HasPrefix(strings.TrimSpace(elines[i]), "# DEBUG:") {
				continue
			}
			low := i - 5
			if low < 5 {
				low = 5
			}
			hight := low + 10
			if hight > len(lines) {
				hight = len(lines)
			}

			require.Greater(t, len(lines), i, "Premature End of file reached")
			if logv := m1.FindStringSubmatch(line); len(logv) > 1 {
				t.Logf("%d: %s", i, logv[1])
				line = m1.ReplaceAllString(line, "")
			}

			require.EqualValues(t, strings.TrimSpace(elines[i]), strings.TrimSpace(line), "On Section [%s] Line:%d\n%d:\n%s\n%d:", k, i, low, strings.Join(lines[low:hight], "\n"), hight)
		}

	}
}
func TestApplySambaHandler(t *testing.T) {
	t.Skip("Not yed necessary for now, we need to mock the smbd process and create a test config file")

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "POST", "/samba/apply", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/samba/apply", api.ApplySamba).Methods(http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusNoContent, rr.Code, "Expected status code 204, got %d with Body %#v", rr.Code, rr.Body.String())
}

func checkStringInSMBConfig(testvalue string, expected string, t *testing.T) bool {
	stream, err := api.CreateConfigStream(testContext)
	require.NoError(t, err)
	assert.NotNil(t, stream)

	rexpt := fmt.Sprintf(expected, testvalue)

	m, err := regexp.MatchString(rexpt, string(*stream))
	require.NoError(t, err)
	assert.True(t, m, "Wrong Match `%s` not found in stream \n%s", rexpt, string(*stream))

	return true
}

// check migrate config don't duplicate share

/*
func TestGetSambaProcessStatus(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequestWithContext(testContext, "GET", "/samba/status", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/samba/status", GetSambaProcessStatus).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, rr.Code, "Expected status code 200 or 404, got %d with Body %s", rr.Code, rr.Body.String())
}
*/

func TestGetSambaConfig(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequestWithContext(testContext, "GET", "/samba", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/samba", api.GetSambaConfig).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	var responseBody dto.SmbConf
	err = json.Unmarshal(rr.Body.Bytes(), &responseBody)
	require.NoError(t, err)

	// Create the expected config stream
	expectedStream, err := api.CreateConfigStream(testContext)
	require.NoError(t, err)

	// Compare the response body with the expected SmbConf
	assert.Equal(t, string(*expectedStream), responseBody.Data)
}
