package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListVolumessHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/volumes", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(ListVolumes)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	//t.Log(pretty.Sprint(rr.Body))
	if len(rr.Body.String()) == 0 {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), "[]")
	}

	var volumes dto.BlockInfo
	err2 := json.NewDecoder(rr.Body).Decode(&volumes)
	if err2 != nil {
		t.Errorf("handler error in decode body %v", err2)
	}
	//t.Log(pretty.Sprint(volumes))

	for _, d := range volumes.Partitions {
		if d.Label == "testvolume" {
			return
		}
	}
	t.Error("Test failed: testvolume not found in volumes")

}

var previus_device uint

func TestMountVolumeHandler(t *testing.T) {
	// Check if loop device is available for mounting
	volumes, err := GetVolumesData()
	require.NoError(t, err)

	var mockMountData dbom.MountPointData

	for _, d := range volumes.Partitions {
		if strings.HasPrefix(d.Name, "loop") && d.Label == "_EXT4" {
			mockMountData.Source = "/dev/" + d.Name
			mockMountData.Path = filepath.Join("/mnt", d.Label)
			mockMountData.FSType = d.Type
			mockMountData.Flags = []dto.MounDataFlag{dto.MS_NOATIME}
			previus_device = d.MountPointData.ID
			t.Logf("Selected loop device: %v", mockMountData)
			break
		}
	}
	if mockMountData.Source == "" {
		t.Skip("Test failed: loop device not found for mounting")
		return
	}

	body, _ := json.Marshal(mockMountData)
	requestPath := fmt.Sprintf("/volume/%d/mount", previus_device)
	t.Logf("Request path: %s", requestPath)
	req, err := http.NewRequestWithContext(testContext, "POST", requestPath, bytes.NewBuffer(body))
	require.NoError(t, err)

	// Set up gorilla/mux router
	router := mux.NewRouter()
	router.HandleFunc("/volume/{id}/mount", MountVolume).Methods("POST")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code, "Body %#v", rr.Body.String())

	// Check the response body is what we expect.
	var responseData dbom.MountPointData
	err = json.Unmarshal(rr.Body.Bytes(), &responseData)
	require.NoError(t, err)

	// Verify the response data
	if !strings.HasPrefix(responseData.Path, mockMountData.Path) {
		t.Errorf("Unexpected path in response: got %v want %v", responseData.Path, mockMountData.Path)
	}
	if responseData.FSType != mockMountData.FSType {
		t.Errorf("Unexpected FSType in response: got %v want %v", responseData.FSType, mockMountData.FSType)
	}
	if !reflect.DeepEqual(responseData.Flags, mockMountData.Flags) {
		t.Errorf("Unexpected Flags in response: got %v want %v", responseData.Flags, mockMountData.Flags)
	}
}

func TestUmountVolumeNonExistent(t *testing.T) {
	req, err := http.NewRequestWithContext(testContext, "DELETE", "/volume/999999/mount", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/volume/{id}/mount", UmountVolume).Methods("DELETE")

	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}

	expected := `{"code":404,"error":"MountPoint not found","body":null}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
func TestUmountVolumeSuccess(t *testing.T) {

	if previus_device == 0 {
		t.Skip("Test skip: not prevision mounted volume found")
	}

	require.NotEmpty(t, previus_device, "Test skip: not prevision mounted volume found")

	// Create a request
	req, err := http.NewRequestWithContext(testContext, "DELETE", fmt.Sprintf("/volume/%d/mount", previus_device), nil)
	require.NoError(t, err)

	// Set up gorilla/mux router
	router := mux.NewRouter()
	router.HandleFunc("/volume/{id}/mount", UmountVolume).Methods("DELETE")

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusNoContent, rr.Code, "Body %#v", rr.Body.String())

	// Check that the response body is empty
	assert.Empty(t, rr.Body.String(), "Body should be empty")
}
