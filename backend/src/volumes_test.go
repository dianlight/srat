// endpoints_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

	t.Log(pretty.Sprint(rr.Body))
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
