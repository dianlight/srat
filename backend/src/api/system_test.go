package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dianlight/srat/api"
)

func TestGetNICsHandler(t *testing.T) {
	req, err := http.NewRequestWithContext(testContext, "GET", "/nics", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.GetNICsHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong content type: got %v want %v",
			contentType, expectedContentType)
	}

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response body: %v", err)
	}
	t.Logf("%v", response)

	if nics, ok := response["nics"].([]interface{}); !ok || len(nics) == 0 {
		t.Errorf("Response does not contain any network interfaces")
	}
}

func TestGetFSHandler(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequestWithContext(testContext, "GET", "/filesystems", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.GetFSHandler)

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the content type
	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong content type: got %v want %v",
			contentType, expectedContentType)
	}

	// Check the response body
	var filesystems []string
	err = json.Unmarshal(rr.Body.Bytes(), &filesystems)
	if err != nil {
		t.Errorf("Failed to unmarshal response body: %v", err)
	}

	// Verify that the response contains file systems data
	if len(filesystems) == 0 {
		t.Errorf("Expected file systems data in response, got empty array")
	}

	//t.Error(filesystems)

	// You might want to add more specific checks here, such as verifying
	// the presence of expected file systems or the format of the data
}
