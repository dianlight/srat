// endpoints_test.go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUsersHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/users", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(ListUsers)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	addon_config := testContext.Value("addon_config").(*config.Config)
	expected, jsonError := json.Marshal(addon_config.OtherUsers)
	assert.NotEmpty(t, expected)
	require.NoError(t, jsonError)
	assert.Equal(t, string(expected), rr.Body.String())
}

func TestGetUserHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/user/backupuser", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", GetUser).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	addon_config := testContext.Value("addon_config").(*config.Config)
	index := slices.IndexFunc(addon_config.OtherUsers, func(u config.User) bool { return u.Username == "backupuser" })
	assert.NotEqual(t, index, -1)
	expected, jsonError := json.Marshal(addon_config.OtherUsers[index])
	require.NoError(t, jsonError)
	assert.NotEmpty(t, expected)
	assert.Equal(t, string(expected), rr.Body.String())
}

func TestCreateUserHandler(t *testing.T) {

	user := dto.User{
		Username: "PIPPO",
		Password: "PLUTO",
	}

	jsonBody, jsonError := json.Marshal(user)
	require.NoError(t, jsonError)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PUT", "/user/PIPPO", strings.NewReader(string(jsonBody)))
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", CreateUser).Methods(http.MethodPut)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(user)
	require.NoError(t, jsonError)
	assert.Equal(t, string(expected), rr.Body.String())

	addon_config := testContext.Value("addon_config").(*config.Config)
	index := slices.IndexFunc(addon_config.OtherUsers, func(u config.User) bool { return u.Username == "PIPPO" })
	assert.NotEqual(t, index, -1)
	assert.Equal(t, addon_config.OtherUsers[index].Password, user.Password)
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
	req, err := http.NewRequestWithContext(testContext, "PUT", "/user/backupuser", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", CreateUser).Methods(http.MethodPut)
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
	require.NoError(t, jsonError)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/user/PIPPO", strings.NewReader(string(jsonBody)))
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", UpdateUser).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	expected, jsonError := json.Marshal(user)
	require.NoError(t, jsonError)
	assert.Equal(t, string(expected), rr.Body.String())
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
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/adminUser", strings.NewReader(string(jsonBody)))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/adminUser", UpdateAdminUser).Methods(http.MethodPatch, http.MethodPost)
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
	req, err := http.NewRequestWithContext(testContext, "DELETE", "/user/PIPPO", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", DeleteUser).Methods(http.MethodDelete)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
