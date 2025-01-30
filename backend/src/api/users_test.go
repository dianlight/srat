// endpoints_test.go
package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
	"github.com/xorcare/pointer"
)

func TestListUsersHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/users", nil)
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ListUsers)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	var users []dto.User
	jsonError := json.Unmarshal(rr.Body.Bytes(), &users)
	require.NoError(t, jsonError)

	// Check the response body is what we expect.
	var configs config.Config
	err = configs.FromContext(testContext)
	require.NoError(t, err)

	assert.Len(t, users, len(configs.OtherUsers)+2, users)

	for _, user := range users {
		if *user.Username == "utente1" {
			continue
		}
		if *user.Username != "dianlight" {
			ou := funk.Find(configs.OtherUsers, func(u config.User) bool { return u.Username == *user.Username }).(config.User)
			assert.Equal(t, &ou.Password, user.Password)
			assert.False(t, *user.IsAdmin)
		} else {
			assert.True(t, *user.IsAdmin)
		}
	}
}

/*
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
	context_state := (&dto.ContextState{}).FromContext(testContext)
	bu, index := context_state.Users.Get("backupuser")
	assert.NotEqual(t, index, -1)
	expected, jsonError := json.Marshal(bu)
	require.NoError(t, jsonError)
	assert.NotEmpty(t, expected)
	assert.Equal(t, string(expected), rr.Body.String())
}
*/

func TestCreateUserHandler(t *testing.T) {

	user := dto.User{
		Username: pointer.String("PIPPO"),
		Password: pointer.String("PLUTO"),
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
	router.HandleFunc("/user/{username}", api.CreateUser).Methods(http.MethodPut)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Check the response body is what we expect.
	user.IsAdmin = pointer.Bool(false)
	expected, jsonError := json.Marshal(user)
	require.NoError(t, jsonError)
	assert.Equal(t, string(expected), rr.Body.String())

	//context_state := (&dto.ContextState{}).FromContext(testContext)
	dbuser := dbom.SambaUser{
		Username: "PIPPO",
	}
	err = dbuser.Get()
	require.NoError(t, err)
	assert.Equal(t, dbuser.Password, *user.Password)
}

func TestCreateUserDuplicateHandler(t *testing.T) {

	user := config.User{
		Username: "backupuser",
		Password: "\u003cbackupuser secret password\u003e",
	}

	jsonBody, jsonError := json.Marshal(user)
	require.NoError(t, jsonError)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PUT", "/user/backupuser", strings.NewReader(string(jsonBody)))
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", api.CreateUser).Methods(http.MethodPut)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusConflict, rr.Code)

	// Check the response body is what we expect.
	assert.Contains(t, rr.Body.String(), "User already exists")
}

func TestUpdateUserHandler(t *testing.T) {

	user := dto.User{
		Password: pointer.String("/pippo"),
	}

	//context_state := (&dto.ContextState{}).FromContext(testContext)
	username := "utente2"

	jsonBody, jsonError := json.Marshal(user)
	require.NoError(t, jsonError)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/user/"+username, strings.NewReader(string(jsonBody)))
	require.NoError(t, err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", api.UpdateUser).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	updated := dto.User{}
	jsonError = json.Unmarshal(rr.Body.Bytes(), &updated)
	require.NoError(t, jsonError)
	assert.Equal(t, username, *updated.Username)
	assert.Equal(t, *user.Password, *updated.Password)
}

func TestUpdateAdminUserHandler(t *testing.T) {

	user := dto.User{
		Password: pointer.String("/pluto||admin"),
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
	router.HandleFunc("/adminUser", api.UpdateAdminUser).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	user.Username = pointer.String("dianlight")
	user.IsAdmin = pointer.Bool(true)
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
	req, err := http.NewRequestWithContext(testContext, "DELETE", "/user/utente1", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", api.DeleteUser).Methods(http.MethodDelete)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
