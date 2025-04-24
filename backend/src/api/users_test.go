package api_test

/*
import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
	"github.com/xorcare/pointer"
)

type UserHandlerSuite struct {
	suite.Suite
	dirtyservice service.DirtyDataServiceInterface
}

func TestUserHandlerSuite(t *testing.T) {
	csuite := new(UserHandlerSuite)
	csuite.dirtyservice = service.NewDirtyDataService(testContext)
	//ctrl := gomock.NewController(t)
	//defer ctrl.Finish()
	//csuite.mockBoradcaster = NewMockBroadcasterServiceInterface(ctrl)
	//csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()
	//csuite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()
	suite.Run(t, csuite)
}

func (suite *UserHandlerSuite) TestListUsersHandler() {
	userHanlder := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	userHanlder.RegisterUserHandler(api)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := api.Get("/users")
	suite.Equal(http.StatusOK, rr.Code)

	var users []dto.User
	jsonError := json.Unmarshal(rr.Body.Bytes(), &users)
	suite.Require().NoError(jsonError)

	// Check the response body is what we expect.
	var configs config.Config
	err := configs.FromContext(testContext)
	suite.Require().NoError(err)

	//suite.Len(users, len(configs.OtherUsers)+2, users)

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
	userHanlder := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	userHanlder.RegisterUserHandler(api)

	rr := api.Get("/useradmin")

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
	userHanlder := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	userHanlder.RegisterUserHandler(api)

	user := dto.User{
		Username: pointer.String("PIPPO"),
		Password: pointer.String("PLUTO"),
	}

	rr := api.Post("/user", user)

	// Check the status code is what we expect.
	suite.Equal(http.StatusCreated, rr.Code)

	// Check the response body is what we expect.
	user.IsAdmin = pointer.Bool(false)
	expected, jsonError := json.Marshal(user)
	suite.Require().NoError(jsonError)
	suite.Equal(string(expected), strings.TrimSpace(rr.Body.String()))

	//context_state := (&dto.ContextState{}).FromContext(testContext)
	dbuser := dbom.SambaUser{
		Username: "PIPPO",
	}
	err := dbuser.Get()
	suite.Require().NoError(err)
	suite.Equal(dbuser.Password, *user.Password)
	suite.False(*user.IsAdmin)
}

func (suite *UserHandlerSuite) TestCreateUserDuplicateHandler() {
	userHanlder := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	userHanlder.RegisterUserHandler(api)

	user := config.User{
		Username: "backupuser",
		Password: "\u003cbackupuser secret password\u003e",
	}

	rr := api.Post("/user", user)
	// Check the status code is what we expect.
	suite.Equal(http.StatusConflict, rr.Code)

	// Check the response body is what we expect.
	suite.Contains(rr.Body.String(), "User already exists")
}

func (suite *UserHandlerSuite) TestUpdateUserHandler() {
	userHanlder := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	userHanlder.RegisterUserHandler(api)

	user := dto.User{
		Password: pointer.String("/pippo"),
	}

	username := "utente2"
	rr := api.Put("/user/"+username, user)

	// Check the status code is what we expect.
	suite.Equal(http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	updated := dto.User{}
	jsonError := json.Unmarshal(rr.Body.Bytes(), &updated)
	suite.Require().NoError(jsonError)
	suite.Equal(username, *updated.Username)
	suite.Equal(*user.Password, *updated.Password)
}

func (suite *UserHandlerSuite) TestUpdateAdminUserHandler() {
	userHanlder := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	userHanlder.RegisterUserHandler(api)

	user := dto.User{
		Password: pointer.String("/pluto||admin"),
	}

	rr := api.Put("/useradmin", user)

	// Check the status code is what we expect.
	suite.Equal(http.StatusOK, rr.Code)

	// Check the response body is what we expect.
	user.Username = pointer.String("dianlight")
	user.IsAdmin = pointer.Bool(true)
	expected, jsonError := json.Marshal(user)
	suite.Require().NoError(jsonError)
	suite.Equal(string(expected), strings.TrimSpace(rr.Body.String()))
}

func (suite *UserHandlerSuite) TestDeleteuserHandler() {
	userHanlder := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	userHanlder.RegisterUserHandler(api)

	rr := api.Delete("/user/utente1")

	suite.Equal(http.StatusNoContent, rr.Code)
}
*/
