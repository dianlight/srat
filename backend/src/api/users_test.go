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
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
	"github.com/xorcare/pointer"
)

type UserHandlerSuite struct {
	suite.Suite
	//mockBoradcaster *MockBroadcasterServiceInterface
	// VariableThatShouldStartAtFive int
}

func TestSUserHandlerSuite(t *testing.T) {
	csuite := new(UserHandlerSuite)
	//ctrl := gomock.NewController(t)
	//defer ctrl.Finish()
	//csuite.mockBoradcaster = NewMockBroadcasterServiceInterface(ctrl)
	//csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()
	//csuite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()
	suite.Run(t, csuite)
}

func (suite *UserHandlerSuite) TestListUsersHandler() {
	api := api.NewUserHandler(testContext)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "GET", "/users", nil)
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ListUsers)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	var users []dto.User
	jsonError := json.Unmarshal(rr.Body.Bytes(), &users)
	require.NoError(suite.T(), jsonError)

	// Check the response body is what we expect.
	var configs config.Config
	err = configs.FromContext(testContext)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), users, len(configs.OtherUsers)+2, users)

	for _, user := range users {
		assert.NotEmpty(suite.T(), user.Username)
		if *user.Username != "dianlight" {
			ou := funk.Find(configs.OtherUsers, func(u config.User) bool { return u.Username == *user.Username })
			if ou != nil {
				assert.Equal(suite.T(), (ou.(config.User)).Password, *user.Password)
				assert.False(suite.T(), *user.IsAdmin)
			}
		} else {
			assert.True(suite.T(), *user.IsAdmin)
		}
	}
}

func (suite *UserHandlerSuite) TestGetUserHandler() {
	api := api.NewUserHandler(testContext)
	req, err := http.NewRequestWithContext(testContext, "GET", "/useradmin", nil)
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/useradmin", api.GetAdminUser).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	ret := &dto.User{}
	jsonError := json.Unmarshal(rr.Body.Bytes(), &ret)
	require.NoError(suite.T(), jsonError)
	assert.NotEmpty(suite.T(), ret)
	assert.Equal(suite.T(), "dianlight", *ret.Username)
	assert.True(suite.T(), *ret.IsAdmin)
}

func (suite *UserHandlerSuite) TestCreateUserHandler() {
	api := api.NewUserHandler(testContext)

	user := dto.User{
		Username: pointer.String("PIPPO"),
		Password: pointer.String("PLUTO"),
	}

	jsonBody, jsonError := json.Marshal(user)
	require.NoError(suite.T(), jsonError)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PUT", "/user/PIPPO", strings.NewReader(string(jsonBody)))
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", api.CreateUser).Methods(http.MethodPut)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(suite.T(), http.StatusCreated, rr.Code)

	// Check the response body is what we expect.
	user.IsAdmin = pointer.Bool(false)
	expected, jsonError := json.Marshal(user)
	require.NoError(suite.T(), jsonError)
	assert.Equal(suite.T(), string(expected), rr.Body.String())

	//context_state := (&dto.ContextState{}).FromContext(testContext)
	dbuser := dbom.SambaUser{
		Username: "PIPPO",
	}
	err = dbuser.Get()
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), dbuser.Password, *user.Password)
	assert.False(suite.T(), *user.IsAdmin)
}

func (suite *UserHandlerSuite) TestCreateUserDuplicateHandler() {
	api := api.NewUserHandler(testContext)
	user := config.User{
		Username: "backupuser",
		Password: "\u003cbackupuser secret password\u003e",
	}

	jsonBody, jsonError := json.Marshal(user)
	require.NoError(suite.T(), jsonError)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PUT", "/user/backupuser", strings.NewReader(string(jsonBody)))
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", api.CreateUser).Methods(http.MethodPut)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(suite.T(), http.StatusConflict, rr.Code)

	// Check the response body is what we expect.
	assert.Contains(suite.T(), rr.Body.String(), "User already exists")
}

func (suite *UserHandlerSuite) TestUpdateUserHandler() {
	api := api.NewUserHandler(testContext)
	user := dto.User{
		Password: pointer.String("/pippo"),
	}

	//context_state := (&dto.ContextState{}).FromContext(testContext)
	username := "utente2"

	jsonBody, jsonError := json.Marshal(user)
	require.NoError(suite.T(), jsonError)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/user/"+username, strings.NewReader(string(jsonBody)))
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", api.UpdateUser).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	updated := dto.User{}
	jsonError = json.Unmarshal(rr.Body.Bytes(), &updated)
	require.NoError(suite.T(), jsonError)
	assert.Equal(suite.T(), username, *updated.Username)
	assert.Equal(suite.T(), *user.Password, *updated.Password)
}

func (suite *UserHandlerSuite) TestUpdateAdminUserHandler() {
	api := api.NewUserHandler(testContext)
	user := dto.User{
		Password: pointer.String("/pluto||admin"),
	}

	jsonBody, jsonError := json.Marshal(user)
	require.NoError(suite.T(), jsonError)
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/adminUser", strings.NewReader(string(jsonBody)))
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/adminUser", api.UpdateAdminUser).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	user.Username = pointer.String("dianlight")
	user.IsAdmin = pointer.Bool(true)
	expected, jsonError := json.Marshal(user)
	require.NoError(suite.T(), jsonError)
	assert.Equal(suite.T(), string(expected), rr.Body.String())
}

func (suite *UserHandlerSuite) TestDeleteuserHandler() {
	api := api.NewUserHandler(testContext)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "DELETE", "/user/utente1", nil)
	require.NoError(suite.T(), err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", api.DeleteUser).Methods(http.MethodDelete)
	router.ServeHTTP(rr, req)

	assert.Equal(suite.T(), http.StatusNoContent, rr.Code)
}
