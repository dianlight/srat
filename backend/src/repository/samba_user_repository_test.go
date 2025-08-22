package repository_test

import (
	"context"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/unixsamba"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

type SambaUserRepositorySuite struct {
	suite.Suite
	repo   repository.SambaUserRepositoryInterface
	ctx    context.Context
	cancel context.CancelFunc
	db     *gorm.DB
	app    *fxtest.App
	ctrl   *matchers.MockController
}

func TestSambaUserRepositorySuite(t *testing.T) {
	suite.Run(t, new(SambaUserRepositorySuite))
}

func (suite *SambaUserRepositorySuite) SetupTest() {
	suite.ctrl = mock.NewMockController(suite.T())

	// Mock unixsamba dependencies
	unixsamba.SetCommandExecutor(mock.Mock[unixsamba.CommandExecutor](suite.ctrl))
	unixsamba.SetOSUserLookuper(mock.Mock[unixsamba.OSUserLookuper](suite.ctrl))

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return suite.ctrl },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			func() *dto.ContextState {
				sharedResources := dto.ContextState{}
				sharedResources.ReadOnlyMode = false
				sharedResources.Heartbeat = 1
				sharedResources.DockerInterface = "hassio"
				sharedResources.DockerNet = "172.30.32.0/23"
				sharedResources.DatabasePath = "file::memory:?cache=shared&_pragma=foreign_keys(1)"
				return &sharedResources
			},
			dbom.NewDB,
			repository.NewSambaUserRepository,
		),
		fx.Populate(&suite.repo),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
		fx.Populate(&suite.db),
	)
	suite.app.RequireStart()
}

func (suite *SambaUserRepositorySuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
		suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
	}
	suite.app.RequireStop()
}

func (suite *SambaUserRepositorySuite) TestGetAdminSuccess() {
	// Arrange
	adminUser := &dbom.SambaUser{
		Username: "admin",
		IsAdmin:  true,
		Password: "password123",
	}
	err := suite.db.Create(adminUser).Error
	suite.Require().NoError(err)

	// Create non-admin user
	regularUser := &dbom.SambaUser{
		Username: "user1",
		IsAdmin:  false,
		Password: "password456",
	}
	err = suite.db.Create(regularUser).Error
	suite.Require().NoError(err)

	// Act
	result, err := suite.repo.GetAdmin()

	// Assert
	suite.NoError(err)
	suite.Equal("admin", result.Username)
	suite.True(result.IsAdmin)
	suite.Equal("password123", result.Password)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.SambaUser{})
}

func (suite *SambaUserRepositorySuite) TestGetAdminNotFound() {
	// Arrange - Create only non-admin users
	regularUser := &dbom.SambaUser{
		Username: "user1",
		IsAdmin:  false,
		Password: "password456",
	}
	err := suite.db.Create(regularUser).Error
	suite.Require().NoError(err)

	// Act
	result, err := suite.repo.GetAdmin()

	// Assert
	suite.Error(err)
	suite.ErrorIs(err, gorm.ErrRecordNotFound)
	suite.Equal(dbom.SambaUser{}, result)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.SambaUser{})
}

func (suite *SambaUserRepositorySuite) TestGetAdminEmptyDatabase() {
	// Act
	result, err := suite.repo.GetAdmin()

	// Assert
	suite.Error(err)
	suite.ErrorIs(err, gorm.ErrRecordNotFound)
	suite.Equal(dbom.SambaUser{}, result)
}

func (suite *SambaUserRepositorySuite) TestGetAdminMultipleAdmins() {
	// Arrange - Create multiple admin users
	admin1 := &dbom.SambaUser{
		Username: "admin1",
		IsAdmin:  true,
		Password: "password1",
	}
	err := suite.db.Create(admin1).Error
	suite.Require().NoError(err)

	admin2 := &dbom.SambaUser{
		Username: "admin2",
		IsAdmin:  true,
		Password: "password2",
	}
	err = suite.db.Create(admin2).Error
	suite.Require().NoError(err)

	// Act
	result, err := suite.repo.GetAdmin()

	// Assert
	suite.NoError(err)
	suite.True(result.IsAdmin)
	// Should return one of the admin users (first one found)
	suite.Contains([]string{"admin1", "admin2"}, result.Username)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.SambaUser{})
}

func (suite *SambaUserRepositorySuite) TestAllUsers() {
	// Arrange
	users := []dbom.SambaUser{
		{Username: "admin", IsAdmin: true, Password: "adminpass"},
		{Username: "user1", IsAdmin: false, Password: "userpass1"},
		{Username: "user2", IsAdmin: false, Password: "userpass2"},
	}

	for _, user := range users {
		err := suite.db.Create(&user).Error
		suite.Require().NoError(err)
	}

	// Act
	result, err := suite.repo.All()

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Len(result, 3)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.SambaUser{})
}

func (suite *SambaUserRepositorySuite) TestAllUsersEmpty() {
	// Act
	result, err := suite.repo.All()

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Empty(result)
}

func (suite *SambaUserRepositorySuite) TestSaveUser() {
	// Arrange
	user := &dbom.SambaUser{
		Username: "testuser",
		IsAdmin:  false,
		Password: "testpass",
	}

	// Act
	errE := suite.repo.Save(user)

	// Assert
	suite.NoError(errE)

	// Verify user was saved
	var savedUser dbom.SambaUser
	err := suite.db.Where("username = ?", "testuser").First(&savedUser).Error
	suite.NoError(err)
	suite.Equal("testuser", savedUser.Username)
	suite.Equal("testpass", savedUser.Password)
	suite.False(savedUser.IsAdmin)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.SambaUser{})
}

func (suite *SambaUserRepositorySuite) TestCreateUser() {
	// Arrange
	user := &dbom.SambaUser{
		Username: "newuser",
		IsAdmin:  false,
		Password: "newpass",
	}

	// Act
	errE := suite.repo.Create(user)

	// Assert
	suite.NoError(errE)

	// Verify user was created
	var createdUser dbom.SambaUser
	err := suite.db.Where("username = ?", "newuser").First(&createdUser).Error
	suite.NoError(err)
	suite.Equal("newuser", createdUser.Username)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.SambaUser{})
}

func (suite *SambaUserRepositorySuite) TestGetUserByNameSuccess() {
	// Arrange
	user := &dbom.SambaUser{
		Username: "findme",
		IsAdmin:  false,
		Password: "findpass",
	}
	err := suite.db.Create(user).Error
	suite.Require().NoError(err)

	// Act
	result, err := suite.repo.GetUserByName("findme")

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("findme", result.Username)
	suite.False(result.IsAdmin)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.SambaUser{})
}

func (suite *SambaUserRepositorySuite) TestGetUserByNameNotFound() {
	// Act
	result, err := suite.repo.GetUserByName("nonexistent")

	// Assert
	suite.Error(err)
	suite.Nil(result)
}

func (suite *SambaUserRepositorySuite) TestGetUserByNameAdminUser() {
	// Arrange - Create admin user
	admin := &dbom.SambaUser{
		Username: "admin",
		IsAdmin:  true,
		Password: "adminpass",
	}
	err := suite.db.Create(admin).Error
	suite.Require().NoError(err)

	// Act - Try to get admin user with GetUserByName (should fail as it excludes admins)
	result, err := suite.repo.GetUserByName("admin")

	// Assert
	suite.Error(err)
	suite.Nil(result)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.SambaUser{})
}

func (suite *SambaUserRepositorySuite) TestDeleteUser() {
	// Arrange
	user := &dbom.SambaUser{
		Username: "deleteme",
		IsAdmin:  false,
		Password: "deletepass",
	}
	err := suite.db.Create(user).Error
	suite.Require().NoError(err)

	// Act
	err = suite.repo.Delete("deleteme")

	// Assert
	suite.NoError(err)

	// Verify user was deleted
	var deletedUser dbom.SambaUser
	err = suite.db.Where("username = ?", "deleteme").First(&deletedUser).Error
	suite.Error(err)
	suite.Equal(gorm.ErrRecordNotFound, err)
}

func (suite *SambaUserRepositorySuite) TestDeleteAdminUser() {
	// Arrange - Create admin user
	admin := &dbom.SambaUser{
		Username: "admin",
		IsAdmin:  true,
		Password: "adminpass",
	}
	err := suite.db.Create(admin).Error
	suite.Require().NoError(err)

	// Act - Try to delete admin user (should not work due to is_admin = false constraint)
	err = suite.repo.Delete("admin")

	// Assert
	suite.NoError(err) // The delete operation succeeds but doesn't delete anything

	// Verify admin user still exists
	var adminUser dbom.SambaUser
	err = suite.db.Where("username = ?", "admin").First(&adminUser).Error
	suite.NoError(err)
	suite.Equal("admin", adminUser.Username)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.SambaUser{})
}

func (suite *SambaUserRepositorySuite) TestSaveAllUsers() {
	// Arrange
	users := &dbom.SambaUsers{
		{Username: "user1", IsAdmin: false, Password: "pass1"},
		{Username: "user2", IsAdmin: false, Password: "pass2"},
		{Username: "user3", IsAdmin: false, Password: "pass3"},
	}

	// Act
	err := suite.repo.SaveAll(users)

	// Assert
	suite.NoError(err)

	// Verify all users were saved
	var count int64
	suite.db.Model(&dbom.SambaUser{}).Count(&count)
	suite.Equal(int64(3), count)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.SambaUser{})
}
