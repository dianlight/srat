// endpoints_test.go
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/gorilla/mux"
	"github.com/kr/pretty"
)

func TestListVolumessHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/volumes", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(listVolumes)

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

	var volumes BlockInfo
	err2 := json.NewDecoder(rr.Body).Decode(&volumes)
	if err2 != nil {
		t.Errorf("handler error in decode body %v", err2)
	}
	t.Log(pretty.Sprint(volumes))

	for _, d := range volumes.Partitions {
		if d.Label == "testvolume" {
			return
		}
	}
	t.Error("Test failed: testvolume not found in volumes")

}

var previus_device string

func TestMountVolumeHandler(t *testing.T) {
	// Check if loop device is available for mounting
	volumes, err := GetVolumesData()
	if err != nil {
		t.Fatalf("Error fetching volumes: %v", err)
		return
	}

	var mockMountData config.MountPointData

	for _, d := range volumes.Partitions {
		if strings.HasPrefix(d.Name, "loop") && d.Label == "_EXT4" {
			mockMountData.Name = d.Name
			mockMountData.Path = filepath.Join("/mnt", d.Label)
			mockMountData.FSType = d.Type
			mockMountData.Flags = []config.MounDataFlag{config.MS_NOATIME}
			previus_device = d.Name
			t.Logf("Selected loop device: %v", mockMountData)
		}
	}
	if mockMountData.Name == "" {
		t.Skip("Test failed: loop device not found for mounting")
		return
	}

	body, _ := json.Marshal(mockMountData)
	requestPath := "/volume/" + previus_device + "/mount"
	t.Logf("Request path: %s", requestPath)
	req, err := http.NewRequest("POST", requestPath, bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	// Set up gorilla/mux router
	router := mux.NewRouter()
	router.HandleFunc("/volume/{volume_name}/mount", mountVolume).Methods("POST")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v\n %v",
			status, http.StatusCreated, rr)
	}

	// Check the response body is what we expect.
	var responseData config.MountPointData
	err = json.Unmarshal(rr.Body.Bytes(), &responseData)
	if err != nil {
		t.Errorf("Unable to parse response body: %v", err)
	}

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
	req, err := http.NewRequest("DELETE", "/volume/nonexistent/mount", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/volume/{volume_name}/mount", umountVolume).Methods("DELETE")

	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}

	expected := `{"code":0,"error":"No mount on nonexistent found!","body":""}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
func TestUmountVolumeSuccess(t *testing.T) {

	if previus_device == "" {
		t.Skip("Test skip: not prevision mounted volume found")
		return
	}

	// Create a request
	req, err := http.NewRequest("DELETE", "/volume/"+previus_device+"/mount", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set up gorilla/mux router
	router := mux.NewRouter()
	router.HandleFunc("/volume/{volume_name}/mount", umountVolume).Methods("DELETE")

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}

	// Check that the response body is empty
	if rr.Body.String() != "" {
		t.Errorf("handler returned unexpected body: got %v want empty", rr.Body.String())
	}
}
