package repository_test

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/unixsamba"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/snapcore/snapd/osutil"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

type ExportedSharesRepositorySuite struct {
	suite.Suite
	export_share_repo repository.ExportedShareRepositoryInterface
	ctx               context.Context
	cancel            context.CancelFunc
	db                *gorm.DB
	app               *fxtest.App
}

func TestExportedSharesSuite(t *testing.T) {
	suite.Run(t, new(ExportedSharesRepositorySuite))
}

func (suite *ExportedSharesRepositorySuite) SetupTest() {
	data, err := os.ReadFile("../../test/data/mount_info.txt")
	if err != nil {
		log.Fatal(err)
	}

	ctrl := mock.NewMockController(suite.T())
	unixsamba.SetCommandExecutor(mock.Mock[unixsamba.CommandExecutor](ctrl))
	unixsamba.SetOSUserLookuper(mock.Mock[unixsamba.OSUserLookuper](ctrl))

	osutil.MockMountInfo(string(data))
	//suite.ctx = context.Background()
	//suite.dirtyDataService = NewDirtyDataService(suite.ctx)
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
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

			//			fx.Annotate(
			//				func() string { return "file::memory:?cache=shared&_pragma=foreign_keys(1)" },
			//				fx.ResultTags(`name:"db_path"`),
			//			),
			dbom.NewDB,
			repository.NewExportedShareRepository,
			service.NewFilesystemService,
		),
		fx.Populate(&suite.export_share_repo),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
		fx.Populate(&suite.db),
	)
	suite.app.RequireStart()
}

func (suite *ExportedSharesRepositorySuite) TearDownTest() {
	suite.cancel()
	suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
	suite.app.RequireStop()
}

func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_Save() {
	// Arrange
	share := &dbom.ExportedShare{
		Name: "test_share",
		MountPointData: dbom.MountPointPath{
			Path:   "/mnt/test_share",
			Device: "test_source",
			FSType: "ext4",
			Type:   "ADDON",
			Data: &dbom.MounDataFlags{
				{Name: "noatime", NeedsValue: false},
			},
			Flags: &dbom.MounDataFlags{
				{Name: "opt", NeedsValue: true, FlagValue: "test1"},
			},
		},
	}

	// Act
	err := suite.export_share_repo.Save(share)

	// Assert
	suite.Require().NoError(err)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.ExportedShare{})
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_SaveAll() {
	// Arrange
	shares := &[]dbom.ExportedShare{
		{
			Name: "test_share1",
			MountPointData: dbom.MountPointPath{
				Path:   "/mnt/test_share1",
				Device: "test_source1",
				FSType: "ext4",
				Type:   "ADDON",
			},
		},
		{
			Name: "test_share2",
			MountPointData: dbom.MountPointPath{
				Path:   "/mnt/test_share2",
				Device: "test_source2",
				FSType: "ntfs",
				Type:   "ADDON",
			},
		},
	}

	// Act
	err := suite.export_share_repo.SaveAll(shares)

	// Assert
	suite.Require().NoError(err)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.ExportedShare{})
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_FindByName() {
	// Arrange
	share := &dbom.ExportedShare{
		Name: "find_me",
		MountPointData: dbom.MountPointPath{
			Path:   "/mnt/find_me",
			Device: "find_source",
			FSType: "ext4",
			Type:   "ADDON",
		},
		Users: []dbom.SambaUser{
			{Username: "user1a", Password: "pass1"},
			{Username: "user2a", Password: "pass2"},
		},
		RoUsers: []dbom.SambaUser{
			{Username: "rouser1a", Password: "ropass1"},
			{Username: "rouser2a", Password: "ropass2"},
		},
	}
	err := suite.export_share_repo.Save(share)
	suite.Require().NoError(err)

	// Act
	foundShare, err := suite.export_share_repo.FindByName("find_me")

	// Assert
	suite.Require().NoError(err)
	suite.Require().NotNil(foundShare)
	suite.Equal("find_me", foundShare.Name)
	suite.Len(foundShare.Users, 2)
	suite.Len(foundShare.RoUsers, 2)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.ExportedShare{})
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.SambaUser{})
}

func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_FindByName_NotFound() {
	// Act
	foundShare, err := suite.export_share_repo.FindByName("not_found")

	// Assert
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, gorm.ErrRecordNotFound)
	suite.Nil(foundShare)
}

func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_All() {
	// Arrange
	shares := []dbom.ExportedShare{
		{
			Name: "all_share1",
			MountPointData: dbom.MountPointPath{
				Path:   "/mnt/all_share1",
				Device: "all_source1",
				FSType: "ext4",
				Type:   "ADDON",
			},
			Users:   []dbom.SambaUser{},
			RoUsers: []dbom.SambaUser{},
		},
		{
			Name: "all_share2",
			MountPointData: dbom.MountPointPath{
				Path:   "/mnt/all_share2",
				Device: "all_source2",
				FSType: "ntfs",
				Type:   "ADDON",
			},
			Users:   []dbom.SambaUser{},
			RoUsers: []dbom.SambaUser{},
		},
	}
	for _, share := range shares {
		err := suite.export_share_repo.Save(&share)
		suite.Require().NoError(err)
	}

	allShares, err := suite.export_share_repo.All()
	suite.Require().NoError(err)
	suite.Require().Len(*allShares, 2)
	if !cmp.Equal(shares, *allShares, cmpopts.IgnoreFields(dbom.ExportedShare{}, "CreatedAt", "UpdatedAt", "DeletedAt", "MountPointDataPath", "MountPointData")) {
		suite.Equal(shares, *allShares)
	}

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.ExportedShare{})
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}
func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_Delete() {
	// Arrange
	share := &dbom.ExportedShare{
		Name: "delete_me",
		MountPointData: dbom.MountPointPath{
			Path:   "/mnt/delete_me",
			Device: "delete_source",
			FSType: "ext4",
			Type:   "ADDON",
		},
	}
	err := suite.export_share_repo.Save(share)
	suite.Require().NoError(err)

	// Act
	err = suite.export_share_repo.Delete("delete_me")

	// Assert
	suite.Require().NoError(err)

	// Check that it is really deleted
	_, err = suite.export_share_repo.FindByName("delete_me")
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, gorm.ErrRecordNotFound)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *ExportedSharesRepositorySuite) TestExportedShareRepository_UpdateName() {
	// Arrange
	share := &dbom.ExportedShare{
		Name: "old_name",
		MountPointData: dbom.MountPointPath{
			Path:   "/mnt/old_name",
			Device: "old_source",
			FSType: "ext4",
			Type:   "ADDON",
		},
	}
	err := suite.export_share_repo.Save(share)
	suite.Require().NoError(err)

	// Act
	err = suite.export_share_repo.UpdateName("old_name", "new_name")

	// Assert
	suite.Require().NoError(err)

	// Check that it is really updated
	_, err = suite.export_share_repo.FindByName("old_name")
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, gorm.ErrRecordNotFound)

	newShare, err := suite.export_share_repo.FindByName("new_name")
	suite.Require().NoError(err)
	suite.Require().NotNil(newShare)
	suite.Equal("new_name", newShare.Name)

	// Cleanup
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.ExportedShare{})
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}
