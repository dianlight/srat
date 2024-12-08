// endpoints_test.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
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

	log.Println(rr.Body.String())
	if len(rr.Body.String()) == 0 {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), "[]")
	}
	/*

		// Check the response body is what we expect.
		expected, jsonError := json.Marshal([]Volume{})
		if jsonError != nil {
			t.Errorf("Unable to encode JSON %s", jsonError.Error())
		}
		if rr.Body.String() != string(expected) {
			t.Errorf("handler returned unexpected body: got %v want %v",
				rr.Body.String(), string(expected))
		}
	*/
}

func TestGetVolumeHandler(t *testing.T) {

	volumes, errs := _getVolumesData()
	if len(errs) != 0 {
		t.Logf("Warning on _getVolumesData %v", errs)
	}

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/volume/"+volumes[0].Label, nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/volume/{volume_name}", getVolume).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(volumes[0])
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	if rr.Body.String() != string(expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(expected))
	}
}

/*
func TestCreateShareHandler(t *testing.T) {

	share := Share{
		Path: "/pippo",
		FS:   "tmpfs",
	}

	jsonBody, jsonError := json.Marshal(share)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("PUT", "/share/PIPPO", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", createShare).Methods(http.MethodPut)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(share)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	if rr.Body.String() != string(expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(expected))
	}
}

func TestCreateShareDuplicateHandler(t *testing.T) {

	share := Share{
		Path: "/mnt/LIBRARY",
		FS:   "ext4",
	}

	jsonBody, jsonError := json.Marshal(share)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("PUT", "/share/LIBRARY", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", createShare).Methods(http.MethodPut)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusConflict {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(ResponseError{Error: "Share already exists", Body: share})
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	if rr.Body.String() != string(expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(expected))
	}
}

func TestUpdateShareHandler(t *testing.T) {

	share := Share{
		Path: "/pippo",
	}

	jsonBody, jsonError := json.Marshal(share)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("PATCH", "/share/LIBRARY", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", updateShare).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	share.FS = "ext4"
	expected, jsonError := json.Marshal(share)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	if rr.Body.String() != string(expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(expected))
	}
}

func TestDeleteShareHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("DELETE", "/share/LIBRARY", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/share/{share_name}", deleteShare).Methods(http.MethodDelete)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
*/
