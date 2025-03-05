package api_test

import (
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/go-fuego/fuego"
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
	userhandler := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	ctx := fuego.NewMockContextNoBody()
	users, err := userhandler.ListUsers(ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(users)

	// Check the response body is what we expect.
	var configs config.Config
	err = configs.FromContext(testContext)
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

func (suite *UserHandlerSuite) TestGetUserAdminHandler() {
	userhandler := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	ctx := fuego.NewMockContextNoBody()

	ret, err := userhandler.GetAdminUser(ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(ret)
	suite.Equal("dianlight", *ret.Username)
	suite.True(*ret.IsAdmin)
}

func (suite *UserHandlerSuite) TestCreateUserHandler() {
	userhandler := api.NewUserHandler(&apiContextState, suite.dirtyservice)

	user := dto.User{
		Username: pointer.String("PIPPO"),
		Password: pointer.String("PLUTO"),
	}
	ctx := fuego.NewMockContext(user)

	result, err := userhandler.CreateUser(ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(result)

	user.IsAdmin = pointer.Bool(false)

	dbuser := dbom.SambaUser{
		Username: "PIPPO",
	}
	err = dbuser.Get()
	suite.Require().NoError(err)
	suite.Equal(dbuser.Password, *user.Password)
	suite.False(*user.IsAdmin)
}

func (suite *UserHandlerSuite) TestCreateUserDuplicateHandler() {
	userhandler := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	user := dto.User{
		Username: pointer.String("backupuser"),
		Password: pointer.String("\u003cbackupuser secret password\u003e"),
	}
	ctx := fuego.NewMockContext(user)

	result, err := userhandler.CreateUser(ctx)
	suite.Require().Error(err)
	suite.Nil(result)
	suite.ErrorAs(err, &fuego.ConflictError{})
}

func (suite *UserHandlerSuite) TestUpdateUserHandler() {
	userhandler := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	user := dto.User{
		Password: pointer.String("/pippo"),
	}
	ctx := fuego.NewMockContext(user)
	username := "utente2"
	ctx.PathParams = map[string]string{
		"username": username,
	}

	updated, err := userhandler.UpdateUser(ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(updated)

	suite.Equal(username, *updated.Username)
	suite.Equal(*user.Password, *updated.Password)
}

func (suite *UserHandlerSuite) TestUpdateAdminUserHandler() {
	userhandler := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	user := dto.User{
		Password: pointer.String("/pluto||admin"),
	}
	ctx := fuego.NewMockContext(user)
	ruser, err := userhandler.UpdateAdminUser(ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(ruser)

	suite.Equal("dianlight", *ruser.Username)
	suite.Equal(*user.Password, *ruser.Password)
	suite.True(*ruser.IsAdmin)
}

func (suite *UserHandlerSuite) TestDeleteuserHandler() {
	userhandler := api.NewUserHandler(&apiContextState, suite.dirtyservice)
	ctx := fuego.NewMockContextNoBody()
	ctx.PathParams = map[string]string{
		"username": "utente1",
	}
	result, err := userhandler.DeleteUser(ctx)
	suite.Require().NoError(err)
	suite.True(result)
}
