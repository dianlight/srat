package service

import (
	"context"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/repository"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

// UserServiceSuite contains unit tests for user_service.go
type UserServiceSuite struct {
	suite.Suite
	app          *fxtest.App
	ctx          context.Context
	cancel       context.CancelFunc
	wg           *sync.WaitGroup
	ctrl         *matchers.MockController
	userRepoMock repository.SambaUserRepositoryInterface
	dirtyService DirtyDataServiceInterface
	shareMock    ShareServiceInterface
	userService  UserServiceInterface
}

// Test runner
func TestUserServiceSuite(t *testing.T) {
	suite.Run(t, new(UserServiceSuite))
}

func (suite *UserServiceSuite) SetupTest() {
	suite.wg = &sync.WaitGroup{}

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), "wg", suite.wg)
				return context.WithCancel(ctx)
			},
			NewUserService,
			NewDirtyDataService,
			NewSettingService,
			events.NewEventBus,
			mock.Mock[repository.SambaUserRepositoryInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[TelemetryServiceInterface],
			mock.Mock[ShareServiceInterface],
		),
		fx.Populate(&suite.ctx, &suite.cancel),
		fx.Populate(&suite.userRepoMock),
		fx.Populate(&suite.dirtyService),
		fx.Populate(&suite.shareMock),
		fx.Populate(&suite.userService),
	)

	suite.app.RequireStart()

}

func (suite *UserServiceSuite) TestListUsers_Success() {
	// Arrange
	dbUsers := []dbom.SambaUser{
		{
			Username: "testuser1",
			Password: "password1",
			IsAdmin:  false,
		},
		{
			Username: "testuser2",
			Password: "password2",
			IsAdmin:  false,
		},
	}

	mock.When(suite.userRepoMock.All()).ThenReturn(dbUsers, nil)

	// Act
	users, err := suite.userService.ListUsers()

	// Assert
	suite.NoError(err)
	suite.Len(users, 2)
	suite.Equal("testuser1", users[0].Username)
	suite.Equal("testuser2", users[1].Username)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).All()
}

func (suite *UserServiceSuite) TestListUsers_RepositoryError() {
	// Arrange
	mock.When(suite.userRepoMock.All()).ThenReturn(nil, errors.New("database error"))

	// Act
	users, err := suite.userService.ListUsers()

	// Assert
	suite.Error(err)
	suite.Nil(users)
	suite.Contains(err.Error(), "failed to list users from repository")
	mock.Verify(suite.userRepoMock, matchers.Times(1)).All()
}

func (suite *UserServiceSuite) TestListUsers_EmptyList() {
	// Arrange
	dbUsers := []dbom.SambaUser{}
	mock.When(suite.userRepoMock.All()).ThenReturn(dbUsers, nil)

	// Act
	users, err := suite.userService.ListUsers()

	// Assert
	suite.NoError(err)
	suite.Empty(users)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).All()
}

func (suite *UserServiceSuite) TestCreateUser_Success() {
	// Arrange
	userDto := dto.User{
		Username: "newuser",
		Password: "newpassword",
		IsAdmin:  false,
	}

	mock.When(suite.userRepoMock.Create(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Act
	createdUser, err := suite.userService.CreateUser(userDto)

	// Assert
	suite.NoError(err)
	suite.NotNil(createdUser)
	suite.Equal("newuser", createdUser.Username)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Create(mock.Any[*dbom.SambaUser]())

	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
}

func (suite *UserServiceSuite) TestCreateUser_DuplicateUsername() {
	// Arrange
	userDto := dto.User{
		Username: "existinguser",
		Password: "password",
		IsAdmin:  false,
	}

	mock.When(suite.userRepoMock.Create(mock.Any[*dbom.SambaUser]())).ThenReturn(errors.WithStack(gorm.ErrDuplicatedKey))

	// Act
	createdUser, err := suite.userService.CreateUser(userDto)

	// Assert
	suite.Error(err)
	suite.Nil(createdUser)
	suite.True(errors.Is(err, dto.ErrorUserAlreadyExists))
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Create(mock.Any[*dbom.SambaUser]())
}

func (suite *UserServiceSuite) TestCreateUser_RepositoryError() {
	// Arrange
	userDto := dto.User{
		Username: "newuser",
		Password: "password",
		IsAdmin:  false,
	}

	mock.When(suite.userRepoMock.Create(mock.Any[*dbom.SambaUser]())).ThenReturn(errors.New("database error"))

	// Act
	createdUser, err := suite.userService.CreateUser(userDto)

	// Assert
	suite.Error(err)
	suite.Nil(createdUser)
	suite.Contains(err.Error(), "failed to create user in repository")
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Create(mock.Any[*dbom.SambaUser]())
}

func (suite *UserServiceSuite) TestUpdateUser_Success() {
	// Arrange
	currentUsername := "oldusername"
	userDto := dto.User{
		Username: "oldusername",
		Password: "newpassword",
		IsAdmin:  false,
	}

	existingDbUser := dbom.SambaUser{
		Username: currentUsername,
		Password: "oldpassword",
		IsAdmin:  false,
	}

	mock.When(suite.userRepoMock.GetUserByName(currentUsername)).ThenReturn(&existingDbUser, nil)
	mock.When(suite.userRepoMock.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Act
	updatedUser, err := suite.userService.UpdateUser(currentUsername, userDto)

	// Assert
	suite.NoError(err)
	suite.NotNil(updatedUser)
	suite.Equal(currentUsername, updatedUser.Username)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(currentUsername)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Save(mock.Any[*dbom.SambaUser]())
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
}

func (suite *UserServiceSuite) TestUpdateUser_UserNotFound() {
	// Arrange
	currentUsername := "nonexistent"
	userDto := dto.User{
		Username: "nonexistent",
		Password: "password",
		IsAdmin:  false,
	}

	mock.When(suite.userRepoMock.GetUserByName(currentUsername)).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))

	// Act
	updatedUser, err := suite.userService.UpdateUser(currentUsername, userDto)

	// Assert
	suite.Error(err)
	suite.Nil(updatedUser)
	suite.True(errors.Is(err, dto.ErrorUserNotFound))
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(currentUsername)
}

func (suite *UserServiceSuite) TestUpdateUser_RenameSuccess() {
	// Arrange
	currentUsername := "oldname"
	newUsername := "newname"
	userDto := dto.User{
		Username: newUsername,
		Password: "password",
		IsAdmin:  false,
	}

	existingDbUser := dbom.SambaUser{
		Username: currentUsername,
		Password: "oldpassword",
		IsAdmin:  false,
	}

	mock.When(suite.userRepoMock.GetUserByName(currentUsername)).ThenReturn(&existingDbUser, nil)
	mock.When(suite.userRepoMock.GetUserByName(newUsername)).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))
	mock.When(suite.userRepoMock.Rename(currentUsername, newUsername)).ThenReturn(nil)
	mock.When(suite.userRepoMock.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Act
	updatedUser, err := suite.userService.UpdateUser(currentUsername, userDto)

	// Assert
	suite.NoError(err)
	suite.NotNil(updatedUser)
	suite.Equal(newUsername, updatedUser.Username)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(currentUsername)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(newUsername)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Rename(currentUsername, newUsername)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Save(mock.Any[*dbom.SambaUser]())
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
}

func (suite *UserServiceSuite) TestUpdateUser_RenameToExistingUser() {
	// Arrange
	currentUsername := "oldname"
	newUsername := "existinguser"
	userDto := dto.User{
		Username: newUsername,
		Password: "password",
		IsAdmin:  false,
	}

	existingDbUser := dbom.SambaUser{
		Username: currentUsername,
		Password: "oldpassword",
		IsAdmin:  false,
	}

	conflictingUser := dbom.SambaUser{
		Username: newUsername,
		Password: "existingpassword",
		IsAdmin:  false,
	}

	mock.When(suite.userRepoMock.GetUserByName(currentUsername)).ThenReturn(&existingDbUser, nil)
	mock.When(suite.userRepoMock.GetUserByName(newUsername)).ThenReturn(&conflictingUser, nil)

	// Act
	updatedUser, err := suite.userService.UpdateUser(currentUsername, userDto)

	// Assert
	suite.Error(err)
	suite.Nil(updatedUser)
	suite.True(errors.Is(err, dto.ErrorUserAlreadyExists))
	suite.Contains(err.Error(), "cannot rename to")
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(currentUsername)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(newUsername)
}

func (suite *UserServiceSuite) TestUpdateUser_SaveError() {
	// Arrange
	currentUsername := "user1"
	userDto := dto.User{
		Username: "user1",
		Password: "newpassword",
		IsAdmin:  false,
	}

	existingDbUser := dbom.SambaUser{
		Username: currentUsername,
		Password: "oldpassword",
		IsAdmin:  false,
	}

	mock.When(suite.userRepoMock.GetUserByName(currentUsername)).ThenReturn(&existingDbUser, nil)
	mock.When(suite.userRepoMock.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(errors.New("save error"))

	// Act
	updatedUser, err := suite.userService.UpdateUser(currentUsername, userDto)

	// Assert
	suite.Error(err)
	suite.Nil(updatedUser)
	suite.Contains(err.Error(), "failed to save updated user")
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(currentUsername)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Save(mock.Any[*dbom.SambaUser]())
}

func (suite *UserServiceSuite) TestUpdateAdminUser_Success() {
	// Arrange
	adminDto := dto.User{
		Username: "admin",
		Password: "newadminpass",
		IsAdmin:  true,
	}

	existingAdmin := dbom.SambaUser{
		Username: "admin",
		Password: "oldadminpass",
		IsAdmin:  true,
	}

	mock.When(suite.userRepoMock.GetAdmin()).ThenReturn(existingAdmin, nil)
	mock.When(suite.userRepoMock.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Act
	updatedAdmin, err := suite.userService.UpdateAdminUser(adminDto)

	// Assert
	suite.NoError(err)
	suite.NotNil(updatedAdmin)
	suite.True(updatedAdmin.IsAdmin)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetAdmin()
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Save(mock.Any[*dbom.SambaUser]())
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
}

func (suite *UserServiceSuite) TestUpdateAdminUser_GetAdminError() {
	// Arrange
	adminDto := dto.User{
		Username: "admin",
		Password: "password",
		IsAdmin:  true,
	}

	mock.When(suite.userRepoMock.GetAdmin()).ThenReturn(dbom.SambaUser{}, errors.New("admin not found"))

	// Act
	updatedAdmin, err := suite.userService.UpdateAdminUser(adminDto)

	// Assert
	suite.Error(err)
	suite.Nil(updatedAdmin)
	suite.Contains(err.Error(), "failed to get admin user")
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetAdmin()
}

func (suite *UserServiceSuite) TestUpdateAdminUser_RenameSuccess() {
	// Arrange
	newAdminName := "newadmin"
	adminDto := dto.User{
		Username: newAdminName,
		Password: "password",
		IsAdmin:  true,
	}

	existingAdmin := dbom.SambaUser{
		Username: "oldadmin",
		Password: "oldpassword",
		IsAdmin:  true,
	}

	mock.When(suite.userRepoMock.GetAdmin()).ThenReturn(existingAdmin, nil)
	mock.When(suite.userRepoMock.GetUserByName(newAdminName)).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))
	mock.When(suite.userRepoMock.Rename("oldadmin", newAdminName)).ThenReturn(nil)
	mock.When(suite.userRepoMock.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Act
	updatedAdmin, err := suite.userService.UpdateAdminUser(adminDto)

	// Assert
	suite.NoError(err)
	suite.NotNil(updatedAdmin)
	suite.Equal(newAdminName, updatedAdmin.Username)
	suite.True(updatedAdmin.IsAdmin)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetAdmin()
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(newAdminName)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Rename("oldadmin", newAdminName)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Save(mock.Any[*dbom.SambaUser]())
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
}

func (suite *UserServiceSuite) TestUpdateAdminUser_RenameToExistingUser() {
	// Arrange
	newAdminName := "existinguser"
	adminDto := dto.User{
		Username: newAdminName,
		Password: "password",
		IsAdmin:  true,
	}

	existingAdmin := dbom.SambaUser{
		Username: "admin",
		Password: "password",
		IsAdmin:  true,
	}

	conflictingUser := dbom.SambaUser{
		Username: newAdminName,
		Password: "password",
		IsAdmin:  false,
	}

	mock.When(suite.userRepoMock.GetAdmin()).ThenReturn(existingAdmin, nil)
	mock.When(suite.userRepoMock.GetUserByName(newAdminName)).ThenReturn(&conflictingUser, nil)

	// Act
	updatedAdmin, err := suite.userService.UpdateAdminUser(adminDto)

	// Assert
	suite.Error(err)
	suite.Nil(updatedAdmin)
	suite.True(errors.Is(err, dto.ErrorUserAlreadyExists))
	suite.Contains(err.Error(), "cannot rename admin to")
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetAdmin()
	mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(newAdminName)
}

func (suite *UserServiceSuite) TestDeleteUser_Success() {
	// Arrange
	username := "userToDelete"

	mock.When(suite.userRepoMock.Delete(username)).ThenReturn(nil)

	// Act
	err := suite.userService.DeleteUser(username)

	// Assert
	suite.NoError(err)
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Delete(username)
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
}

func (suite *UserServiceSuite) TestDeleteUser_UserNotFound() {
	// Arrange
	username := "nonexistent"

	mock.When(suite.userRepoMock.Delete(username)).ThenReturn(errors.WithStack(gorm.ErrRecordNotFound))

	// Act
	err := suite.userService.DeleteUser(username)

	// Assert
	suite.Error(err)
	suite.True(errors.Is(err, dto.ErrorUserNotFound))
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Delete(username)
}

func (suite *UserServiceSuite) TestDeleteUser_RepositoryError() {
	// Arrange
	username := "user1"

	mock.When(suite.userRepoMock.Delete(username)).ThenReturn(errors.New("database error"))

	// Act
	err := suite.userService.DeleteUser(username)

	// Assert
	suite.Error(err)
	suite.Contains(err.Error(), "failed to delete user")
	mock.Verify(suite.userRepoMock, matchers.Times(1)).Delete(username)
}
