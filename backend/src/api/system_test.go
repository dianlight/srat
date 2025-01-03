package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/dianlight/srat/dto"
)

func TestHealthCheckHandler(t *testing.T) {
	req, err := http.NewRequestWithContext(testContext, "GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheckHandler)

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

	var response dto.HealthPing
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response body: %v", err)
	}

	if !response.Alive {
		t.Errorf("Expected Alive to be true, got false")
	}
}
func TestHealthCheckHandlerDoesNotModifyGlobalHealthData(t *testing.T) {
	originalHealthData := *healthData
	req, err := http.NewRequestWithContext(testContext, "GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheckHandler)

	handler.ServeHTTP(rr, req)

	if !reflect.DeepEqual(*healthData, originalHealthData) {
		t.Errorf("HealthCheckHandler modified global healthData. Original: %+v, Modified: %+v", originalHealthData, *healthData)
	}
}

func TestGetNICsHandler(t *testing.T) {
	req, err := http.NewRequestWithContext(testContext, "GET", "/nics", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetNICsHandler)

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
	handler := http.HandlerFunc(GetFSHandler)

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

func TestMainHealthCheckHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheckHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `{"alive":true,"read_only":true`
	if rr.Body.String()[:len(expected)] != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
