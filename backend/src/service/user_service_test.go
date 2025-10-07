package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

type UserServiceSuite struct {
	suite.Suite
	app              *fxtest.App
	userService      service.UserServiceInterface
	mockUserRepo     repository.SambaUserRepositoryInterface
	mockDirtyService service.DirtyDataServiceInterface
	mockShareService service.ShareServiceInterface
	ctx              context.Context
	cancel           context.CancelFunc
}

func TestUserServiceSuite(t *testing.T) {
	suite.Run(t, new(UserServiceSuite))
}

func (suite *UserServiceSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			service.NewUserService,
			mock.Mock[repository.SambaUserRepositoryInterface],
			mock.Mock[service.DirtyDataServiceInterface],
			mock.Mock[service.ShareServiceInterface],
		),
		fx.Populate(&suite.userService),
		fx.Populate(&suite.mockUserRepo),
		fx.Populate(&suite.mockDirtyService),
		fx.Populate(&suite.mockShareService),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *UserServiceSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
		if wg, ok := suite.ctx.Value("wg").(*sync.WaitGroup); ok {
			wg.Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

// Test ListUsers success
func (suite *UserServiceSuite) TestListUsersSuccess() {
	dbUsers := dbom.SambaUsers{
		{Username: "user1", Password: "pass1", IsAdmin: false},
		{Username: "user2", Password: "pass2", IsAdmin: false},
	}
	mock.When(suite.mockUserRepo.All()).ThenReturn(dbUsers, nil)

	users, err := suite.userService.ListUsers()

	suite.NoError(err)
	suite.NotNil(users)
	suite.Len(users, 2)
	suite.Equal("user1", users[0].Username)
	suite.Equal("user2", users[1].Username)
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).All()
}

// Test ListUsers with repository error
func (suite *UserServiceSuite) TestListUsersRepositoryError() {
	mockErr := errors.New("repository error")
	mock.When(suite.mockUserRepo.All()).ThenReturn(nil, mockErr)

	users, err := suite.userService.ListUsers()

	suite.Error(err)
	suite.Nil(users)
	suite.Contains(err.Error(), "failed to list users from repository")
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).All()
}

// Test CreateUser success
func (suite *UserServiceSuite) TestCreateUserSuccess() {
	userDto := dto.User{
		Username: "newuser",
		Password: "password123",
	}
	mock.When(suite.mockUserRepo.Create(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	createdUser, err := suite.userService.CreateUser(userDto)

	suite.NoError(err)
	suite.NotNil(createdUser)
	suite.Equal("newuser", createdUser.Username)
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).Create(mock.Any[*dbom.SambaUser]())
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyUsers()
}

// Test CreateUser with duplicate user
func (suite *UserServiceSuite) TestCreateUserDuplicate() {
	userDto := dto.User{
		Username: "existinguser",
		Password: "password123",
	}
	mock.When(suite.mockUserRepo.Create(mock.Any[*dbom.SambaUser]())).ThenReturn(errors.WithStack(gorm.ErrDuplicatedKey))

	createdUser, err := suite.userService.CreateUser(userDto)

	suite.Error(err)
	suite.Nil(createdUser)
	suite.True(errors.Is(err, dto.ErrorUserAlreadyExists))
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).Create(mock.Any[*dbom.SambaUser]())
}

// Test UpdateUser success
func (suite *UserServiceSuite) TestUpdateUserSuccess() {
	existingUser := &dbom.SambaUser{Username: "oldname", Password: "oldpass"}
	updatedDto := dto.User{
		Username: "oldname",
		Password: "newpass",
	}
	mock.When(suite.mockUserRepo.GetUserByName("oldname")).ThenReturn(existingUser, nil)
	mock.When(suite.mockUserRepo.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	result, err := suite.userService.UpdateUser("oldname", updatedDto)

	suite.NoError(err)
	suite.NotNil(result)
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).GetUserByName("oldname")
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).Save(mock.Any[*dbom.SambaUser]())
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyUsers()
}

// Test UpdateUser not found
func (suite *UserServiceSuite) TestUpdateUserNotFound() {
	updatedDto := dto.User{
		Username: "newname",
		Password: "newpass",
	}
	mock.When(suite.mockUserRepo.GetUserByName("nonexistent")).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))

	result, err := suite.userService.UpdateUser("nonexistent", updatedDto)

	suite.Error(err)
	suite.Nil(result)
	suite.True(errors.Is(err, dto.ErrorUserNotFound))
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).GetUserByName("nonexistent")
}

// Test UpdateUser with rename
func (suite *UserServiceSuite) TestUpdateUserWithRename() {
	existingUser := &dbom.SambaUser{Username: "oldname", Password: "oldpass"}
	updatedDto := dto.User{
		Username: "newname",
		Password: "newpass",
	}
	mock.When(suite.mockUserRepo.GetUserByName("oldname")).ThenReturn(existingUser, nil)
	mock.When(suite.mockUserRepo.GetUserByName("newname")).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))
	mock.When(suite.mockUserRepo.Rename("oldname", "newname")).ThenReturn(nil)
	mock.When(suite.mockUserRepo.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	result, err := suite.userService.UpdateUser("oldname", updatedDto)

	suite.NoError(err)
	suite.NotNil(result)
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).Rename("oldname", "newname")
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyUsers()
}

// Test DeleteUser success
func (suite *UserServiceSuite) TestDeleteUserSuccess() {
	mock.When(suite.mockUserRepo.Delete("testuser")).ThenReturn(nil)

	err := suite.userService.DeleteUser("testuser")

	suite.NoError(err)
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).Delete("testuser")
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyUsers()
}

// Test DeleteUser not found
func (suite *UserServiceSuite) TestDeleteUserNotFound() {
	mock.When(suite.mockUserRepo.Delete("nonexistent")).ThenReturn(errors.WithStack(gorm.ErrRecordNotFound))

	err := suite.userService.DeleteUser("nonexistent")

	suite.Error(err)
	suite.True(errors.Is(err, dto.ErrorUserNotFound))
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).Delete("nonexistent")
}

// Test UpdateAdminUser success
func (suite *UserServiceSuite) TestUpdateAdminUserSuccess() {
	adminUser := dbom.SambaUser{Username: "admin", Password: "oldpass", IsAdmin: true}
	updatedDto := dto.User{
		Username: "admin",
		Password: "newpass",
	}
	mock.When(suite.mockUserRepo.GetAdmin()).ThenReturn(adminUser, nil)
	mock.When(suite.mockUserRepo.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	result, err := suite.userService.UpdateAdminUser(updatedDto)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsAdmin)
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).GetAdmin()
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).Save(mock.Any[*dbom.SambaUser]())
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyUsers()
}

// Test UpdateAdminUser with rename
func (suite *UserServiceSuite) TestUpdateAdminUserWithRename() {
	adminUser := dbom.SambaUser{Username: "admin", Password: "oldpass", IsAdmin: true}
	updatedDto := dto.User{
		Username: "newadmin",
		Password: "newpass",
	}
	mock.When(suite.mockUserRepo.GetAdmin()).ThenReturn(adminUser, nil)
	mock.When(suite.mockUserRepo.GetUserByName("newadmin")).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))
	mock.When(suite.mockUserRepo.Rename("admin", "newadmin")).ThenReturn(nil)
	mock.When(suite.mockUserRepo.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	result, err := suite.userService.UpdateAdminUser(updatedDto)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsAdmin)
	mock.Verify(suite.mockUserRepo, matchers.Times(1)).Rename("admin", "newadmin")
	mock.Verify(suite.mockDirtyService, matchers.Times(1)).SetDirtyUsers()
}
