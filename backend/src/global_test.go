package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestGetGlobalConfigHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/global", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getGlobalConfig)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `{"workgroup":"WORKGROUP","mountoptions":["nosuid","relatime","noexec"],"allow_hosts":["10.0.0.0/8","100.0.0.0/8","172.16.0.0/12","192.168.0.0/16","169.254.0.0/16","fe80::/10","fc00::/7"],"veto_files":["._*",".DS_Store","Thumbs.db","icon?",".Trashes"],"compatibility_mode":false,"recyle_bin_enabled":false,"interfaces":["wlan0","end0"],"bind_all_interfaces":true,"log_level":"","multi_channel":false}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestUpdateGlobalConfigHandler(t *testing.T) {
	glc := GlobalConfig{
		Workgroup: "pluto&admin",
	}

	jsonBody, jsonError := json.Marshal(glc)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	req, err := http.NewRequest("PATCH", "/global", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/global", updateGlobalConfig).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var res GlobalConfig
	err = json.Unmarshal(rr.Body.Bytes(), &res)
	if err != nil {
		t.Errorf("Unable to decode JSON %s", err.Error())
	}

	//pretty.Logf("res: %v", res)

	if res.Workgroup != glc.Workgroup {
		t.Errorf("handler returned unexpected body: got %v want %v",
			res.Workgroup, glc.Workgroup)
	}

}