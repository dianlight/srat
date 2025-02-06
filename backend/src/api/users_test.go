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
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
	"github.com/xorcare/pointer"
)

type UserHandlerSuite struct {
	suite.Suite
	//mockBoradcaster *MockBroadcasterServiceInterface
	// VariableThatShouldStartAtFive int
}

func TestUserHandlerSuite(t *testing.T) {
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
	suite.Require().NoError(err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ListUsers)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	suite.Equal(http.StatusOK, rr.Code)

	var users []dto.User
	jsonError := json.Unmarshal(rr.Body.Bytes(), &users)
	suite.Require().NoError(jsonError)

	// Check the response body is what we expect.
	var configs config.Config
	err = configs.FromContext(testContext)
	suite.Require().NoError(err)

	suite.Len(users, len(configs.OtherUsers)+2, users)

	for _, user := range users {
		suite.NotEmpty(user.Username)
		if *user.Username != "dianlight" {
			ou := funk.Find(configs.OtherUsers, func(u config.User) bool { return u.Username == *user.Username })
			if ou != nil {
				suite.Equal((ou.(config.User)).Password, *user.Password)
				suite.False(*user.IsAdmin)
			}
		} else {
			suite.True(*user.IsAdmin)
		}
	}
}

func (suite *UserHandlerSuite) TestGetUserHandler() {
	api := api.NewUserHandler(testContext)
	req, err := http.NewRequestWithContext(testContext, "GET", "/useradmin", nil)
	suite.Require().NoError(err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/useradmin", api.GetAdminUser).Methods(http.MethodGet)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	suite.Equal(http.StatusOK, rr.Code)

	ret := &dto.User{}
	jsonError := json.Unmarshal(rr.Body.Bytes(), &ret)
	suite.Require().NoError(jsonError)
	suite.NotEmpty(ret)
	suite.Equal("dianlight", *ret.Username)
	suite.True(*ret.IsAdmin)
}

func (suite *UserHandlerSuite) TestCreateUserHandler() {
	api := api.NewUserHandler(testContext)

	user := dto.User{
		Username: pointer.String("PIPPO"),
		Password: pointer.String("PLUTO"),
	}

	jsonBody, jsonError := json.Marshal(user)
	suite.Require().NoError(jsonError)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PUT", "/user/PIPPO", strings.NewReader(string(jsonBody)))
	suite.Require().NoError(err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", api.CreateUser).Methods(http.MethodPut)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	suite.Equal(http.StatusCreated, rr.Code)

	// Check the response body is what we expect.
	user.IsAdmin = pointer.Bool(false)
	expected, jsonError := json.Marshal(user)
	suite.Require().NoError(jsonError)
	suite.Equal(string(expected), rr.Body.String())

	//context_state := (&dto.ContextState{}).FromContext(testContext)
	dbuser := dbom.SambaUser{
		Username: "PIPPO",
	}
	err = dbuser.Get()
	suite.Require().NoError(err)
	suite.Equal(dbuser.Password, *user.Password)
	suite.False(*user.IsAdmin)
}

func (suite *UserHandlerSuite) TestCreateUserDuplicateHandler() {
	api := api.NewUserHandler(testContext)
	user := config.User{
		Username: "backupuser",
		Password: "\u003cbackupuser secret password\u003e",
	}

	jsonBody, jsonError := json.Marshal(user)
	suite.Require().NoError(jsonError)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PUT", "/user/backupuser", strings.NewReader(string(jsonBody)))
	suite.Require().NoError(err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", api.CreateUser).Methods(http.MethodPut)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	suite.Equal(http.StatusConflict, rr.Code)

	// Check the response body is what we expect.
	suite.Contains(rr.Body.String(), "User already exists")
}

func (suite *UserHandlerSuite) TestUpdateUserHandler() {
	api := api.NewUserHandler(testContext)
	user := dto.User{
		Password: pointer.String("/pippo"),
	}

	//context_state := (&dto.ContextState{}).FromContext(testContext)
	username := "utente2"

	jsonBody, jsonError := json.Marshal(user)
	suite.Require().NoError(jsonError)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/user/"+username, strings.NewReader(string(jsonBody)))
	suite.Require().NoError(err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", api.UpdateUser).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	suite.Equal(http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	updated := dto.User{}
	jsonError = json.Unmarshal(rr.Body.Bytes(), &updated)
	suite.Require().NoError(jsonError)
	suite.Equal(username, *updated.Username)
	suite.Equal(*user.Password, *updated.Password)
}

func (suite *UserHandlerSuite) TestUpdateAdminUserHandler() {
	api := api.NewUserHandler(testContext)
	user := dto.User{
		Password: pointer.String("/pluto||admin"),
	}

	jsonBody, jsonError := json.Marshal(user)
	suite.Require().NoError(jsonError)
	req, err := http.NewRequestWithContext(testContext, "PATCH", "/adminUser", strings.NewReader(string(jsonBody)))
	suite.Require().NoError(err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/adminUser", api.UpdateAdminUser).Methods(http.MethodPatch, http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	suite.Equal(http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	user.Username = pointer.String("dianlight")
	user.IsAdmin = pointer.Bool(true)
	expected, jsonError := json.Marshal(user)
	suite.Require().NoError(jsonError)
	suite.Equal(string(expected), rr.Body.String())
}

func (suite *UserHandlerSuite) TestDeleteuserHandler() {
	api := api.NewUserHandler(testContext)
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "DELETE", "/user/utente1", nil)
	suite.Require().NoError(err)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/user/{username}", api.DeleteUser).Methods(http.MethodDelete)
	router.ServeHTTP(rr, req)

	suite.Equal(http.StatusNoContent, rr.Code)
}
