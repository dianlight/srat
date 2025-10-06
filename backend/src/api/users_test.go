package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type UserHandlerSuite struct {
	suite.Suite
	app             *fxtest.App
	handler         *api.UserHandler
	mockUserService service.UserServiceInterface
	ctx             context.Context
	cancel          context.CancelFunc
}

func (suite *UserHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			api.NewUserHandler,
			mock.Mock[service.UserServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockUserService),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *UserHandlerSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
		suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
	}
	suite.app.RequireStop()
}

func (suite *UserHandlerSuite) TestListUsersSuccess() {
	expectedUsers := []dto.User{
		{Username: "user1", IsAdmin: true},
		{Username: "user2", IsAdmin: false},
	}

	// Configure mock expectations
	mock.When(suite.mockUserService.ListUsers()).ThenReturn(expectedUsers, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterUserHandler(api)

	// Make HTTP request
	resp := api.Get("/users")
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Parse response
	var result []dto.User
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(expectedUsers, result)
}

func (suite *UserHandlerSuite) TestListUsersError() {
	expectedErr := errors.New("database error")

	// Configure mock expectations
	mock.When(suite.mockUserService.ListUsers()).ThenReturn(nil, expectedErr)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterUserHandler(api)

	// Make HTTP request
	resp := api.Get("/users")
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)
}

func (suite *UserHandlerSuite) TestCreateUserSuccess() {
	input := dto.User{Username: "newuser", Password: "password123", IsAdmin: false}
	expectedUser := &dto.User{Username: "newuser", IsAdmin: false}

	// Configure mock expectations
	mock.When(suite.mockUserService.CreateUser(mock.Any[dto.User]())).ThenReturn(expectedUser, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterUserHandler(api)

	// Make HTTP request
	resp := api.Post("/user", input)
	suite.Require().Equal(http.StatusCreated, resp.Code)

	// Parse response
	var result dto.User
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(expectedUser.Username, result.Username)
	suite.Equal(expectedUser.IsAdmin, result.IsAdmin)
}

func (suite *UserHandlerSuite) TestCreateUserAlreadyExists() {
	input := dto.User{Username: "existinguser", Password: "password123"}

	// Configure mock expectations
	mock.When(suite.mockUserService.CreateUser(mock.Any[dto.User]())).ThenReturn(nil, errors.WithStack(dto.ErrorUserAlreadyExists))

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterUserHandler(api)

	// Make HTTP request
	resp := api.Post("/user", input)
	suite.Require().Equal(http.StatusConflict, resp.Code)
}

func (suite *UserHandlerSuite) TestCreateUserError() {
	input := dto.User{Username: "erroruser", Password: "password123"}
	expectedErr := errors.New("database connection failed")

	// Configure mock expectations
	mock.When(suite.mockUserService.CreateUser(mock.Any[dto.User]())).ThenReturn(nil, expectedErr)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterUserHandler(api)

	// Make HTTP request
	resp := api.Post("/user", input)
	suite.Require().Equal(http.StatusInternalServerError, resp.Code)
}

func (suite *UserHandlerSuite) TestUpdateUserSuccess() {
	username := "testuser"
	input := dto.User{Username: username, Password: "newpassword", IsAdmin: true}
	expectedUser := &dto.User{Username: username, IsAdmin: true}

	// Configure mock expectations
	mock.When(suite.mockUserService.UpdateUser(mock.Equal(username), mock.Any[dto.User]())).ThenReturn(expectedUser, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterUserHandler(api)

	// Make HTTP request
	resp := api.Put("/user/"+username, input)
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Parse response
	var result dto.User
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(expectedUser.Username, result.Username)
	suite.Equal(expectedUser.IsAdmin, result.IsAdmin)
}

func (suite *UserHandlerSuite) TestUpdateUserNotFound() {
	username := "nonexistentuser"
	input := dto.User{Username: username, Password: "password"}

	// Configure mock expectations
	mock.When(suite.mockUserService.UpdateUser(mock.Equal(username), mock.Any[dto.User]())).ThenReturn(nil, errors.WithStack(dto.ErrorUserNotFound))

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterUserHandler(api)

	// Make HTTP request
	resp := api.Put("/user/"+username, input)
	suite.Require().Equal(http.StatusNotFound, resp.Code)
}

func (suite *UserHandlerSuite) TestUpdateUserAlreadyExists() {
	username := "testuser"
	input := dto.User{Username: "existingname", Password: "password"}

	// Configure mock expectations
	mock.When(suite.mockUserService.UpdateUser(mock.Equal(username), mock.Any[dto.User]())).ThenReturn(nil, errors.WithStack(dto.ErrorUserAlreadyExists))

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterUserHandler(api)

	// Make HTTP request
	resp := api.Put("/user/"+username, input)
	suite.Require().Equal(http.StatusConflict, resp.Code)
}

func (suite *UserHandlerSuite) TestUpdateAdminUserSuccess() {
	input := dto.User{Username: "admin", Password: "newadminpass", IsAdmin: true}
	expectedUser := &dto.User{Username: "admin", IsAdmin: true}

	// Configure mock expectations
	mock.When(suite.mockUserService.UpdateAdminUser(mock.Any[dto.User]())).ThenReturn(expectedUser, nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterUserHandler(api)

	// Make HTTP request
	resp := api.Put("/useradmin", input)
	suite.Require().Equal(http.StatusOK, resp.Code)

	// Parse response
	var result dto.User
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	suite.Require().NoError(err)

	// Assert
	suite.Equal(expectedUser.Username, result.Username)
	suite.Equal(expectedUser.IsAdmin, result.IsAdmin)
}

func (suite *UserHandlerSuite) TestUpdateAdminUserNotFound() {
	input := dto.User{Username: "admin", Password: "password"}

	// Configure mock expectations
	mock.When(suite.mockUserService.UpdateAdminUser(mock.Any[dto.User]())).ThenReturn(nil, errors.WithStack(dto.ErrorUserNotFound))

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterUserHandler(api)

	// Make HTTP request
	resp := api.Put("/useradmin", input)
	suite.Require().Equal(http.StatusNotFound, resp.Code)
}

func (suite *UserHandlerSuite) TestDeleteUserSuccess() {
	username := "testuser"

	// Configure mock expectations
	mock.When(suite.mockUserService.DeleteUser(username)).ThenReturn(nil)

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterUserHandler(api)

	// Make HTTP request
	resp := api.Delete("/user/" + username)
	suite.Require().Equal(http.StatusNoContent, resp.Code)
}

func (suite *UserHandlerSuite) TestDeleteUserNotFound() {
	username := "nonexistentuser"

	// Configure mock expectations
	mock.When(suite.mockUserService.DeleteUser(username)).ThenReturn(errors.WithStack(dto.ErrorUserNotFound))

	// Setup humatest
	_, api := humatest.New(suite.T())
	suite.handler.RegisterUserHandler(api)

	// Make HTTP request
	resp := api.Delete("/user/" + username)
	suite.Require().Equal(http.StatusNotFound, resp.Code)
}

func TestUserHandlerSuite(t *testing.T) {
	suite.Run(t, new(UserHandlerSuite))
}
