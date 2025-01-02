// endpoints_test.go
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/data"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
)

func TestListUsersHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/users", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(listUsers)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(data.Config.OtherUsers)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	if rr.Body.String() != string(expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(expected))
	}
}

func TestGetUserHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/user/backupuser", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", getUser).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	index := slices.IndexFunc(data.Config.OtherUsers, func(u config.User) bool { return u.Username == "backupuser" })
	if index == -1 {
		t.Error("User not found")
	} else {
		expected, jsonError := json.Marshal(data.Config.OtherUsers[index])
		if jsonError != nil {
			t.Errorf("Unable to encode JSON %s", jsonError.Error())
		}
		if rr.Body.String() != string(expected) {
			t.Errorf("handler returned unexpected body: got %v want %v",
				rr.Body.String(), string(expected))
		}
	}
}

func TestCreateUserHandler(t *testing.T) {

	user := config.User{
		Username: "PIPPO",
		Password: "PLUTO",
	}

	jsonBody, jsonError := json.Marshal(user)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("PUT", "/user/PIPPO", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", createUser).Methods(http.MethodPut)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(user)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	if rr.Body.String() != string(expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(expected))
	}
}

func TestCreateUserDuplicateHandler(t *testing.T) {

	user := config.User{
		Username: "backupuser",
		Password: "\u003cbackupuser secret password\u003e",
	}

	jsonBody, jsonError := json.Marshal(user)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("PUT", "/user/backupuser", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", createUser).Methods(http.MethodPut)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusConflict {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(dto.ResponseError{Error: "User already exists", Body: user})
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	if rr.Body.String() != string(expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(expected))
	}
}

func TestUpdateUserHandler(t *testing.T) {

	user := config.User{
		Password: "/pippo",
	}

	jsonBody, jsonError := json.Marshal(user)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("PATCH", "/user/PIPPO", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", updateUser).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(user)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	if rr.Body.String() != string(expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(expected))
	}
}

func TestUpdateAdminUserHandler(t *testing.T) {

	user := config.User{
		Password: "/pluto||admin",
	}

	jsonBody, jsonError := json.Marshal(user)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("PATCH", "/adminUser", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/adminUser", updateAdminUser).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	user.Password = "/pluto||admin"
	expected, jsonError := json.Marshal(user)
	if jsonError != nil {
		t.Errorf("Unable to encode JSON %s", jsonError.Error())
	}
	if rr.Body.String() != string(expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), string(expected))
	}
}

func TestDeleteuserHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("DELETE", "/user/PIPPO", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", deleteUser).Methods(http.MethodDelete)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
