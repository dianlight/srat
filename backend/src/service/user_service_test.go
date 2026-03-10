package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/unixsamba"
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
	app    *fxtest.App
	db     *gorm.DB
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
	appFS  *unixsamba.MockSystem
	//	userRepoMock repository.SambaUserRepositoryInterface
	dirtyService DirtyDataServiceInterface
	shareMock    ShareServiceInterface
	userService  UserServiceInterface
}

// Test runner
func TestUserServiceSuite(t *testing.T) {
	suite.Run(t, new(UserServiceSuite))
}

func (suite *UserServiceSuite) SetupTest() {
	os.Setenv("SRAT_MOCK", "true")
	suite.wg = &sync.WaitGroup{}

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), ctxkeys.WaitGroup, suite.wg)
				return context.WithCancel(ctx)
			},
			func() *dto.ContextState {
				return &dto.ContextState{
					DatabasePath: "file::memory:?cache=shared&_pragma=foreign_keys(1)",
				}
			},
			dbom.NewDB,
			NewUserService,
			NewDirtyDataService,
			NewSettingService,
			events.NewEventBus,
			mock.Mock[TelemetryServiceInterface],
			mock.Mock[ShareServiceInterface],
		),
		fx.Populate(&suite.ctx, &suite.cancel),
		fx.Populate(&suite.dirtyService),
		fx.Populate(&suite.shareMock),
		fx.Populate(&suite.userService),
		fx.Populate(&suite.db),
	)

	suite.appFS = unixsamba.NewMockSystem()
	unixsamba.SetCommandExecutor(suite.appFS)
	unixsamba.SetOSUserLookuper(suite.appFS)

	suite.app.RequireStart()

	suite.Require().NoError(suite.db.Exec("DELETE FROM user_rw_share").Error)
	suite.Require().NoError(suite.db.Exec("DELETE FROM user_ro_share").Error)
	suite.Require().NoError(suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&dbom.ExportedShare{}).Error)
	suite.Require().NoError(suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&dbom.SambaUser{}).Error)

}

func (suite *UserServiceSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
	}
	if suite.ctx != nil {
		if wg, ok := suite.ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok && wg != nil {
			wg.Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
	unixsamba.ResetExecutorsToDefaults()
}

func (suite *UserServiceSuite) TestListUsers_Success() {
	// Arrange
	dbUsers := []dbom.SambaUser{
		{
			Username: "testuser1",
			Password: "password1",
			IsAdmin:  false,
			RwShares: []dbom.ExportedShare{
				{Name: "rwshare1"},
				{Name: "rwshare2"},
			},
		},
		{
			Username: "testuser2",
			Password: "password1",
			IsAdmin:  false,
			RoShares: []dbom.ExportedShare{
				{Name: "roshare1"},
			},
		},
	}

	suite.appFS.AddUser("testuser1", "password1")
	suite.appFS.AddUser("testuser2", "password1")

	suite.Require().NoError(suite.db.Create(&dbUsers[1].RoShares).Error)
	suite.Require().NoError(suite.db.Create(&dbUsers[0].RwShares).Error)
	suite.Require().NoError(suite.db.Create(&dbUsers).Error)
	//mock.When(suite.userRepoMock.All()).ThenReturn(dbUsers, nil)

	// Act
	users, err := suite.userService.ListUsers()

	// Assert
	suite.NoError(err)
	suite.GreaterOrEqual(len(users), 2)
	var usernames []string
	var rwShares []string
	var roShares []string
	for _, u := range users {
		suite.True(u.IsValid)
		usernames = append(usernames, u.Username)
		for _, share := range u.RwShares {
			rwShares = append(rwShares, share)
		}
		for _, share := range u.RoShares {
			roShares = append(roShares, share)
		}
	}
	suite.Contains(usernames, "testuser1")
	suite.Contains(usernames, "testuser2")
	suite.Contains(rwShares, "rwshare1")
	suite.Contains(rwShares, "rwshare2")
	suite.Contains(roShares, "roshare1")

}

/*
func (suite *UserServiceSuite) TestListUsers_RepositoryError() {
	// Arrange
	mock.When(suite.userRepoMock.All()).ThenReturn(nil, errors.New("database error"))

	// Act
	users, err := suite.userService.ListUsers()

	// Assert
	suite.Error(err)
	suite.Nil(users)
	suite.Contains(err.Error(), "failed to list users from repository")
	mock.Verify(suite.userRepoMock, matchers.Times(2)).All()
}
*/

/*
func (suite *UserServiceSuite) TestListUsers_EmptyList() {
	// Arrange
	//dbUsers := []dbom.SambaUser{}
	suite.Require().NoError(suite.db.Delete(&dbom.SambaUser{}).Error)
	//mock.When(suite.userRepoMock.All()).ThenReturn(dbUsers, nil)

	// Act
	users, err := suite.userService.ListUsers()

	// Assert
	suite.NoError(err)
	suite.Empty(users)
	//mock.Verify(suite.userRepoMock, matchers.Times(2)).All()
}
*/

func (suite *UserServiceSuite) TestCreateUser_Success() {
	// Arrange
	userDto := dto.User{
		Username: "newuser" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Password: dto.NewSecret("newpassword"),
		IsAdmin:  false,
	}

	//mock.When(suite.userRepoMock.Create(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Act
	createdUser, err := suite.userService.CreateUser(userDto)

	// Assert
	suite.Require().NoError(err, "expected no error but got '%v' '%v'", err, errors.Unwrap(err))
	suite.Require().NotNil(createdUser)
	suite.Equal(userDto.Username, createdUser.Username)
	//mock.Verify(suite.userRepoMock, matchers.Times(2)).Create(mock.Any[*dbom.SambaUser]())
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
	suite.Require().NoError(suite.db.Where("username = ?", userDto.Username).First(&dbom.SambaUser{}).Error)
}

/*

func (suite *UserServiceSuite) TestCreateUser_NoSuccess() {
	// Arrange
	userDto := dto.User{
		Username: "newuser" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Password: dto.NewSecret("newpassword"),
		IsAdmin:  false,
	}

	suite.appFS.AddUser(userDto.Username, "different_password")

	//mock.When(suite.userRepoMock.Create(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Act
	createdUser, err := suite.userService.CreateUser(userDto)

	// Assert
	suite.Require().Error(err)
	suite.Require().Nil(createdUser)
}
*/

func (suite *UserServiceSuite) TestCreateUser_DeletedSuccess() {

	username := fmt.Sprintf("newuser%d", time.Now().Unix())
	// Arrange
	userDto := dto.User{
		Username: username,
		Password: dto.NewSecret("newpassword"),
		IsAdmin:  false,
	}

	suite.Require().NoError(suite.db.Create(&dbom.SambaUser{Username: username, Password: "oldpassword", IsAdmin: false}).Error)
	suite.Require().NoError(suite.db.Delete(&dbom.SambaUser{}, "username = ?", username).Error)
	//mock.When(suite.userRepoMock.Create(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Act
	createdUser, err := suite.userService.CreateUser(userDto)

	// Assert
	suite.Require().NoError(err, " expected no error but got '%v' '%v'", err, errors.Unwrap(err))
	suite.Require().NotNil(createdUser)
	suite.Equal(username, createdUser.Username)
	//mock.Verify(suite.userRepoMock, matchers.Times(2)).Create(mock.Any[*dbom.SambaUser]())
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
	suite.Require().NoError(suite.db.Where("username = ?", username).First(&dbom.SambaUser{}).Error)
}

func (suite *UserServiceSuite) TestCreateUser_DuplicateUsername() {
	// Arrange
	username := fmt.Sprintf("existinguser%d", time.Now().Unix())
	userDto := dto.User{
		Username: username,
		Password: dto.NewSecret("password"),
		IsAdmin:  false,
	}

	suite.Require().NoError(suite.db.Create(&dbom.SambaUser{Username: username, Password: "password"}).Error)
	//mock.When(suite.userRepoMock.Create(mock.Any[*dbom.SambaUser]())).ThenReturn(errors.WithStack(gorm.ErrDuplicatedKey))

	// Act
	createdUser, err := suite.userService.CreateUser(userDto)

	// Assert
	suite.Require().Error(err)
	suite.Require().Nil(createdUser)
	suite.True(errors.Is(err, dto.ErrorUserAlreadyExists))
	//mock.Verify(suite.userRepoMock, matchers.Times(2)).Create(mock.Any[*dbom.SambaUser]())
}

/*
func (suite *UserServiceSuite) TestCreateUser_RepositoryError() {
	// Arrange
	userDto := dto.User{
		Username: "newuser",
		Password: dto.NewSecret("password"),
		IsAdmin:  false,
	}

	mock.When(suite.userRepoMock.Create(mock.Any[*dbom.SambaUser]())).ThenReturn(errors.New("database error"))

	// Act
	createdUser, err := suite.userService.CreateUser(userDto)

	// Assert
	suite.Error(err)
	suite.Nil(createdUser)
	suite.Contains(err.Error(), "failed to create user in repository")
	mock.Verify(suite.userRepoMock, matchers.Times(2)).Create(mock.Any[*dbom.SambaUser]())
}
*/

func (suite *UserServiceSuite) TestUpdateUser_Success() {
	// Arrange
	currentUsername := "oldusername" + fmt.Sprintf("%d", time.Now().UnixNano())
	userDto := dto.User{
		Username: currentUsername,
		Password: dto.NewSecret("newpassword"),
		IsAdmin:  false,
	}

	existingDbUser := dto.User{
		Username: currentUsername,
		Password: dto.NewSecret("oldpassword"),
		IsAdmin:  false,
	}

	_, err := suite.userService.CreateUser(existingDbUser)
	suite.Require().NoError(err)

	//	mock.When(suite.userRepoMock.GetUserByName(currentUsername)).ThenReturn(&existingDbUser, nil)
	//	mock.When(suite.userRepoMock.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Act
	updatedUser, err := suite.userService.UpdateUser(currentUsername, userDto)

	// Assert
	suite.Require().NoError(err, "expected no error but got '%v' '%v'", err, errors.Unwrap(err))
	suite.Require().NotNil(updatedUser)
	suite.Equal(currentUsername, updatedUser.Username)
	suite.False(updatedUser.IsAdmin)
	suite.Equal("newpassword", updatedUser.Password.Expose())
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(currentUsername)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).Save(mock.Any[*dbom.SambaUser]())
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
}

func (suite *UserServiceSuite) TestUpdateUser_ChangePassword() {
	// Arrange
	username := fmt.Sprintf("pwduser%d", time.Now().UnixNano())
	oldPassword := "oldpassword"
	newPassword := "newpassword"

	existingDbUser := dto.User{
		Username: username,
		Password: dto.NewSecret(oldPassword),
		IsAdmin:  false,
	}
	_, err := suite.userService.CreateUser(existingDbUser)
	suite.Require().NoError(err)

	userDto := dto.User{
		Username: username,
		Password: dto.NewSecret(newPassword),
		IsAdmin:  false,
	}

	// Act
	updatedUser, err := suite.userService.UpdateUser(username, userDto)

	// Assert
	suite.Require().NoError(err, "expected no error but got '%v' '%v'", err, errors.Unwrap(err))
	suite.Require().NotNil(updatedUser)
	suite.Equal(username, updatedUser.Username)
	suite.Equal(newPassword, updatedUser.Password.Expose())

	var persistedUser dbom.SambaUser
	suite.Require().NoError(suite.db.Where("username = ?", username).First(&persistedUser).Error)
	suite.Equal(newPassword, persistedUser.Password)
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
}

func (suite *UserServiceSuite) TestUpdateUser_UserNotFound() {
	// Arrange
	currentUsername := "nonexistent"
	userDto := dto.User{
		Username: "nonexistent",
		Password: dto.NewSecret("password"),
		IsAdmin:  false,
	}

	//mock.When(suite.userRepoMock.GetUserByName(currentUsername)).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))

	// Act
	updatedUser, err := suite.userService.UpdateUser(currentUsername, userDto)

	// Assert
	suite.Error(err)
	suite.Nil(updatedUser)
	suite.True(errors.Is(err, dto.ErrorUserNotFound), "expected ErrorUserNotFound but got %v", err)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(currentUsername)
}

func (suite *UserServiceSuite) TestUpdateUser_RenameSuccess() {
	// Arrange
	currentUsername := fmt.Sprintf("oldname%s", rand.Text()[0:3])
	newUsername := fmt.Sprintf("newname%s", rand.Text()[0:3])
	userDto := dto.User{
		Username: newUsername,
		Password: dto.NewSecret("password"),
		IsAdmin:  false,
	}

	existingDbUser := dto.User{
		Username: currentUsername,
		Password: dto.NewSecret("oldpassword"),
		IsAdmin:  false,
	}

	_, err := suite.userService.CreateUser(existingDbUser)
	suite.Require().NoError(err)

	//mock.When(suite.userRepoMock.GetUserByName(currentUsername)).ThenReturn(&existingDbUser, nil)
	//mock.When(suite.userRepoMock.GetUserByName(newUsername)).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))
	//mock.When(suite.userRepoMock.Rename(currentUsername, newUsername)).ThenReturn(nil)
	//mock.When(suite.userRepoMock.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Act
	updatedUser, err := suite.userService.UpdateUser(currentUsername, userDto)

	// Assert
	suite.Require().NoError(err, "expected no error but got '%v' '%v'", err, errors.Unwrap(err))
	suite.Require().NotNil(updatedUser)
	suite.Equal(newUsername, updatedUser.Username)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(currentUsername)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(newUsername)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).Rename(currentUsername, newUsername)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).Save(mock.Any[*dbom.SambaUser]())
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
}

func (suite *UserServiceSuite) TestUpdateUser_RenameWithShares_Success() {
	// Arrange
	currentUsername := fmt.Sprintf("rnshares_old%d", time.Now().UnixNano())
	newUsername := fmt.Sprintf("rnshares_new%d", time.Now().UnixNano())

	// Create user via service so the mock system registers it for samba operations
	_, err := suite.userService.CreateUser(dto.User{
		Username: currentUsername,
		Password: dto.NewSecret("password"),
		IsAdmin:  false,
	})
	suite.Require().NoError(err)

	// Create shares and associate them with the user directly in the DB
	rwShare := dbom.ExportedShare{Name: fmt.Sprintf("rnrw_%d", time.Now().UnixNano())}
	roShare := dbom.ExportedShare{Name: fmt.Sprintf("rnro_%d", time.Now().UnixNano())}
	suite.Require().NoError(suite.db.Create(&rwShare).Error)
	suite.Require().NoError(suite.db.Create(&roShare).Error)
	suite.Require().NoError(suite.db.Model(&dbom.SambaUser{Username: currentUsername}).Association("RwShares").Append(&rwShare))
	suite.Require().NoError(suite.db.Model(&dbom.SambaUser{Username: currentUsername}).Association("RoShares").Append(&roShare))

	userDto := dto.User{
		Username: newUsername,
		Password: dto.NewSecret("password"),
		IsAdmin:  false,
	}

	// Act
	updatedUser, err := suite.userService.UpdateUser(currentUsername, userDto)

	// Assert
	suite.Require().NoError(err, "expected no error but got '%v'", err)
	suite.Require().NotNil(updatedUser)
	suite.Equal(newUsername, updatedUser.Username)
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)

	// Verify shares are still associated with the renamed user (CASCADE on primary key update)
	var renamedUser dbom.SambaUser
	suite.Require().NoError(
		suite.db.Preload("RwShares").Preload("RoShares").
			Where("username = ?", newUsername).
			First(&renamedUser).Error,
	)
	suite.Require().Len(renamedUser.RwShares, 1)
	suite.Require().Len(renamedUser.RoShares, 1)
	suite.Equal(rwShare.Name, renamedUser.RwShares[0].Name)
	suite.Equal(roShare.Name, renamedUser.RoShares[0].Name)
}

func (suite *UserServiceSuite) TestUpdateUser_RenameToExistingUser() {
	// Arrange
	currentUsername := fmt.Sprintf("roldname%d", time.Now().Unix())
	newUsername := fmt.Sprintf("rexistinguser%d", time.Now().Unix())
	userDto := dto.User{
		Username: newUsername,
		Password: dto.NewSecret("password"),
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

	suite.Require().NoError(suite.db.Create(&existingDbUser).Error)
	suite.Require().NoError(suite.db.Create(&conflictingUser).Error)

	//mock.When(suite.userRepoMock.GetUserByName(currentUsername)).ThenReturn(&existingDbUser, nil)
	//mock.When(suite.userRepoMock.GetUserByName(newUsername)).ThenReturn(&conflictingUser, nil)

	// Act
	updatedUser, err := suite.userService.UpdateUser(currentUsername, userDto)

	// Assert
	suite.Error(err)
	suite.Nil(updatedUser)
	suite.True(errors.Is(err, dto.ErrorUserAlreadyExists))
	suite.Contains(err.Error(), "cannot rename to")
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(currentUsername)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(newUsername)
}

func (suite *UserServiceSuite) TestUpdateUser_SaveError() {
	// Arrange
	currentUsername := "user1"
	userDto := dto.User{
		Username: "user1",
		Password: dto.NewSecret("newpassword"),
		IsAdmin:  false,
	}
	/*
		existingDbUser := dbom.SambaUser{
			Username: currentUsername,
			Password: "oldpassword",
			IsAdmin:  false,
		}

		suite.Require().NoError(suite.db.Create(&existingDbUser).Error)
	*/
	//	mock.When(suite.userRepoMock.GetUserByName(currentUsername)).ThenReturn(&existingDbUser, nil)
	//	mock.When(suite.userRepoMock.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(errors.New("save error"))

	// Act
	updatedUser, err := suite.userService.UpdateUser(currentUsername, userDto)

	// Assert
	suite.Require().Error(err)
	suite.Nil(updatedUser)
	suite.Contains(err.Error(), "User not found")
	// mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(currentUsername)
	// mock.Verify(suite.userRepoMock, matchers.Times(1)).Save(mock.Any[*dbom.SambaUser]())
}

func (suite *UserServiceSuite) TestUpdateAdminUser_Success() {

	username := fmt.Sprintf("hadmin%d", time.Now().Unix())
	// Arrange
	adminDto := dto.User{
		Username: username,
		Password: dto.NewSecret("newadminpass"),
		IsAdmin:  true,
	}

	existingAdmin := dto.User{
		Username: username,
		Password: dto.NewSecret("oldadminpass"),
		IsAdmin:  true,
	}

	suite.Require().NoError(suite.db.Delete(&dbom.SambaUser{}, "is_admin = ?", true).Error)
	_, err := suite.userService.CreateUser(existingAdmin)
	suite.Require().NoError(err)

	//mock.When(suite.userRepoMock.GetAdmin()).ThenReturn(existingAdmin, nil)
	//mock.When(suite.userRepoMock.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Act
	updatedAdmin, err := suite.userService.UpdateAdminUser(adminDto)

	// Assert
	suite.Require().NoError(err, "expected no error but got '%v' '%v'", err, errors.Unwrap(err))
	suite.Require().NotNil(updatedAdmin)
	suite.Equal(username, updatedAdmin.Username)
	suite.Equal("newadminpass", updatedAdmin.Password.Expose())
	suite.Require().NotNil(updatedAdmin)
	suite.True(updatedAdmin.IsAdmin)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).GetAdmin()
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).Save(mock.Any[*dbom.SambaUser]())
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
}

func (suite *UserServiceSuite) TestUpdateAdminUser_RenameSuccess() {
	// Arrange
	newAdminName := fmt.Sprintf("newadmin%d", time.Now().Unix())
	oldAdminName := fmt.Sprintf("oldadmin%d", time.Now().Unix())
	adminDto := dto.User{
		Username: newAdminName,
		Password: dto.NewSecret("password"),
		IsAdmin:  true,
	}

	existingAdmin := dto.User{
		Username: oldAdminName,
		Password: dto.NewSecret("oldpassword"),
		IsAdmin:  true,
	}

	suite.Require().NoError(suite.db.Delete(&dbom.SambaUser{}, "is_admin = ?", true).Error)
	_, err := suite.userService.CreateUser(existingAdmin)
	suite.Require().NoError(err)

	//mock.When(suite.userRepoMock.GetAdmin()).ThenReturn(existingAdmin, nil)
	//mock.When(suite.userRepoMock.GetUserByName(newAdminName)).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))
	//mock.When(suite.userRepoMock.Rename("oldadmin", newAdminName)).ThenReturn(nil)
	//mock.When(suite.userRepoMock.Save(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Act
	updatedAdmin, err := suite.userService.UpdateAdminUser(adminDto)

	// Assert
	suite.Require().NoError(err, "expected no error but got '%v' '%v'", err, errors.Unwrap(err))
	suite.Require().NotNil(updatedAdmin)
	suite.Equal(newAdminName, updatedAdmin.Username)
	suite.True(updatedAdmin.IsAdmin)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).GetAdmin()
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(newAdminName)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).Rename("oldadmin", newAdminName)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).Save(mock.Any[*dbom.SambaUser]())
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
}

func (suite *UserServiceSuite) TestUpdateAdminUser_RenameToExistingUser() {
	// Arrange
	newAdminName := "existinguser"
	adminDto := dto.User{
		Username: newAdminName,
		Password: dto.NewSecret("password"),
		IsAdmin:  true,
	}

	existingAdmin := dto.User{
		Username: "admin",
		Password: dto.NewSecret("password"),
		IsAdmin:  true,
	}

	conflictingUser := dto.User{
		Username: newAdminName,
		Password: dto.NewSecret("password"),
		IsAdmin:  false,
	}

	suite.Require().NoError(suite.db.Delete(&dbom.SambaUser{}, "is_admin = ?", true).Error)

	_, err := suite.userService.CreateUser(existingAdmin)
	suite.Require().NoError(err)
	_, err = suite.userService.CreateUser(conflictingUser)
	suite.Require().NoError(err)

	//mock.When(suite.userRepoMock.GetAdmin()).ThenReturn(existingAdmin, nil)
	//mock.When(suite.userRepoMock.GetUserByName(newAdminName)).ThenReturn(&conflictingUser, nil)

	// Act
	updatedAdmin, err := suite.userService.UpdateAdminUser(adminDto)

	// Assert
	suite.Error(err)
	suite.Nil(updatedAdmin)
	suite.True(errors.Is(err, dto.ErrorUserAlreadyExists))
	suite.Contains(err.Error(), "User already exists")
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).GetAdmin()
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).GetUserByName(newAdminName)
}

func (suite *UserServiceSuite) TestDeleteUser_Success() {
	// Arrange
	username := "userToDelete"

	_, err := suite.userService.CreateUser(dto.User{Username: username, Password: dto.NewSecret("password")})
	suite.Require().NoError(err)

	// Act
	err = suite.userService.DeleteUser(username)

	// Assert
	suite.NoError(err)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).Delete(username)
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)
}

func (suite *UserServiceSuite) TestDeleteUser_Success_Reget() {
	// Arrange
	username := "userToDeleteRg"

	_, err := suite.userService.CreateUser(dto.User{Username: username, Password: dto.NewSecret("password")})
	suite.Require().NoError(err)

	// Act
	err = suite.userService.DeleteUser(username)

	// Assert
	suite.Require().NoError(err)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).Delete(username)
	suite.Require().True(suite.dirtyService.GetDirtyDataTracker().Users)

	// Try to get the deleted user
	var user dbom.SambaUser
	result := suite.db.Where("username = ?", username).First(&user)
	suite.Require().Error(result.Error)
	suite.True(errors.Is(result.Error, gorm.ErrRecordNotFound))
}

func (suite *UserServiceSuite) TestDeleteUser_WithShares_Success() {
	// Arrange
	username := fmt.Sprintf("delshares_%d", time.Now().UnixNano())

	// Create user via service so the mock system registers it for samba operations
	_, err := suite.userService.CreateUser(dto.User{
		Username: username,
		Password: dto.NewSecret("password"),
		IsAdmin:  false,
	})
	suite.Require().NoError(err)

	// Create shares and associate them with the user directly in the DB
	rwShare := dbom.ExportedShare{Name: fmt.Sprintf("delrw_%d", time.Now().UnixNano())}
	roShare := dbom.ExportedShare{Name: fmt.Sprintf("delro_%d", time.Now().UnixNano())}
	suite.Require().NoError(suite.db.Create(&rwShare).Error)
	suite.Require().NoError(suite.db.Create(&roShare).Error)
	suite.Require().NoError(suite.db.Model(&dbom.SambaUser{Username: username}).Association("RwShares").Append(&rwShare))
	suite.Require().NoError(suite.db.Model(&dbom.SambaUser{Username: username}).Association("RoShares").Append(&roShare))

	// Act
	err = suite.userService.DeleteUser(username)

	// Assert
	suite.Require().NoError(err)
	suite.True(suite.dirtyService.GetDirtyDataTracker().Users)

	// Verify the user is soft-deleted (not found by normal query)
	var deletedUser dbom.SambaUser
	result := suite.db.Where("username = ?", username).First(&deletedUser)
	suite.True(errors.Is(result.Error, gorm.ErrRecordNotFound))

	// Verify the shares themselves are NOT deleted (user deletion must not cascade to shares)
	var existingRwShare dbom.ExportedShare
	suite.Require().NoError(suite.db.Where("name = ?", rwShare.Name).First(&existingRwShare).Error)
	suite.Equal(rwShare.Name, existingRwShare.Name)

	var existingRoShare dbom.ExportedShare
	suite.Require().NoError(suite.db.Where("name = ?", roShare.Name).First(&existingRoShare).Error)
	suite.Equal(roShare.Name, existingRoShare.Name)
}

func (suite *UserServiceSuite) TestDeleteUser_UserNotFound() {
	// Arrange
	username := "nonexistent"

	//suite.Require().NoError(suite.db.Create(&dbom.SambaUser{Username: username}).Error)

	// Act
	err := suite.userService.DeleteUser(username)

	// Assert
	suite.Error(err)
	suite.True(errors.Is(err, dto.ErrorUserNotFound), "expected ErrorUserNotFound but got %v", err)
	//mock.Verify(suite.userRepoMock, matchers.Times(1)).Delete(username)
}

/*
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
*/
