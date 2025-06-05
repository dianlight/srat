package repository_test

import (
	"context"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/repository"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

type MountPointPathRepositorySuite struct {
	suite.Suite
	mount_repo repository.MountPointPathRepositoryInterface
	ctx        context.Context
	cancel     context.CancelFunc
	db         *gorm.DB
	app        *fxtest.App
}

func TestMountPointPathRepositorySuite(t *testing.T) {
	suite.Run(t, new(MountPointPathRepositorySuite))
}

func (suite *MountPointPathRepositorySuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			fx.Annotate(
				func() string { return "file::memory:?cache=shared&_pragma=foreign_keys(1)" },
				fx.ResultTags(`name:"db_path"`),
			),
			dbom.NewDB,
			repository.NewMountPointPathRepository,
		),
		fx.Populate(&suite.mount_repo),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
		fx.Populate(&suite.db),
	)
	defer suite.app.RequireStart()
}

func (suite *MountPointPathRepositorySuite) TearDownTest() {
	suite.cancel()
	suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
	suite.app.RequireStop()
}

func (suite *MountPointPathRepositorySuite) TestMountPointDataSaveWithoutData() {

	testMountPoint := dbom.MountPointPath{
		Path: "/addons",
		Type: "ADDON",
	}

	err := suite.mount_repo.Save(&testMountPoint)

	suite.Require().NoError(err)
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *MountPointPathRepositorySuite) TestMountPointDataSave() {

	testMountPoint := dbom.MountPointPath{
		Path:   "/mnt/test",
		Device: "test_drive",
		FSType: "ext4",
		Flags: &dbom.MounDataFlags{
			dbom.MounDataFlag{Name: "noatime", NeedsValue: false},
			dbom.MounDataFlag{Name: "ro", NeedsValue: false},
		},
		Type: "ADDON",
		Data: &dbom.MounDataFlags{
			dbom.MounDataFlag{Name: "umask", NeedsValue: true, FlagValue: "1000"},
			dbom.MounDataFlag{Name: "force", NeedsValue: false},
		},
		//DeviceId: 12344,
	}

	err := suite.mount_repo.Save(&testMountPoint)

	suite.Require().NoError(err)

	suite.Contains(*testMountPoint.Flags, dbom.MounDataFlag{Name: "noatime", NeedsValue: false})
	suite.Contains(*testMountPoint.Flags, dbom.MounDataFlag{Name: "ro", NeedsValue: false})

	suite.Contains(*testMountPoint.Data, dbom.MounDataFlag{Name: "umask", NeedsValue: true, FlagValue: "1000"})
	suite.Contains(*testMountPoint.Data, dbom.MounDataFlag{Name: "force", NeedsValue: false})

	// double check from DB
	foundMountPoint, err := suite.mount_repo.FindByPath("/mnt/test")

	suite.Require().NoError(err)
	suite.Require().NotNil(foundMountPoint)
	suite.Equal(testMountPoint.Path, foundMountPoint.Path)
	suite.Equal(testMountPoint.Device, foundMountPoint.Device)
	suite.Equal(testMountPoint.FSType, foundMountPoint.FSType)

	suite.Len(*foundMountPoint.Flags, 2)
	suite.Len(*foundMountPoint.Data, 2)

	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *MountPointPathRepositorySuite) TestMountPointDataSaveNilFlags() {

	testMountPoint := dbom.MountPointPath{
		Path:   "/mnt/test",
		Device: "test_drive",
		FSType: "ext4",
		Flags:  &dbom.MounDataFlags{},
		Type:   "ADDON",
		Data:   &dbom.MounDataFlags{},
	}

	err := suite.mount_repo.Save(&testMountPoint)

	suite.Require().NoError(err)

	suite.Empty(testMountPoint.Flags)
	suite.Empty(testMountPoint.Data)

	// double check from DB
	foundMountPoint, err := suite.mount_repo.FindByPath("/mnt/test")

	suite.Require().NoError(err)
	suite.Require().NotNil(foundMountPoint)
	suite.Equal(testMountPoint.Path, foundMountPoint.Path)
	suite.Equal(testMountPoint.Device, foundMountPoint.Device)
	suite.Equal(testMountPoint.FSType, foundMountPoint.FSType)

	suite.Empty(testMountPoint.Flags)
	suite.Empty(testMountPoint.Data)

	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *MountPointPathRepositorySuite) TestMountPointDataSaveAndUpdateFlagsToNil() {

	testMountPoint := dbom.MountPointPath{
		Path:   "/mnt/test",
		Device: "test_drive",
		FSType: "ext4",
		Flags: &dbom.MounDataFlags{
			dbom.MounDataFlag{Name: "noatime", NeedsValue: false},
			dbom.MounDataFlag{Name: "ro", NeedsValue: false},
		},
		Type: "ADDON",
		Data: &dbom.MounDataFlags{
			dbom.MounDataFlag{Name: "umask", NeedsValue: true, FlagValue: "1000"},
			dbom.MounDataFlag{Name: "force", NeedsValue: false},
		},
		//DeviceId: 12344,
	}

	err := suite.mount_repo.Save(&testMountPoint)

	suite.Require().NoError(err)

	suite.Contains(*testMountPoint.Flags, dbom.MounDataFlag{Name: "noatime", NeedsValue: false})
	suite.Contains(*testMountPoint.Flags, dbom.MounDataFlag{Name: "ro", NeedsValue: false})

	suite.Contains(*testMountPoint.Data, dbom.MounDataFlag{Name: "umask", NeedsValue: true, FlagValue: "1000"})
	suite.Contains(*testMountPoint.Data, dbom.MounDataFlag{Name: "force", NeedsValue: false})

	// set flags and data to nil
	testMountPoint.Flags = nil
	testMountPoint.Data = nil

	err = suite.mount_repo.Save(&testMountPoint)

	suite.Require().NoError(err)

	suite.Require().NotNil(testMountPoint.Flags)
	suite.Require().NotNil(testMountPoint.Data)

	suite.Contains(*testMountPoint.Flags, dbom.MounDataFlag{Name: "noatime", NeedsValue: false})
	suite.Contains(*testMountPoint.Flags, dbom.MounDataFlag{Name: "ro", NeedsValue: false})

	suite.Contains(*testMountPoint.Data, dbom.MounDataFlag{Name: "umask", NeedsValue: true, FlagValue: "1000"})
	suite.Contains(*testMountPoint.Data, dbom.MounDataFlag{Name: "force", NeedsValue: false})

	// double check from DB
	foundMountPoint, err := suite.mount_repo.FindByPath("/mnt/test")

	suite.Require().NoError(err)
	suite.Require().NotNil(foundMountPoint)
	suite.Equal(testMountPoint.Path, foundMountPoint.Path)
	suite.Equal(testMountPoint.Device, foundMountPoint.Device)
	suite.Equal(testMountPoint.FSType, foundMountPoint.FSType)

	suite.Len(*foundMountPoint.Flags, 2)
	suite.Len(*foundMountPoint.Data, 2)

	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *MountPointPathRepositorySuite) TestMountPointDataAll() {

	expectedMountPoints := []dbom.MountPointPath{
		{
			Path: "/mnt/test1",
			//Label:  "Test 1",
			Device: "test1",
			FSType: "ext4",
			Flags: &dbom.MounDataFlags{
				dbom.MounDataFlag{Name: "noatime", NeedsValue: false},
				dbom.MounDataFlag{Name: "rw", NeedsValue: false},
			},
			//Data:     "rw,noatime",
			DeviceId: 12345,
			Type:     "ADDON",
		},
		{
			Path: "/mnt/test2",
			//Label:  "Test 2",
			Device: "test2",
			FSType: "ntfs",
			Flags: &dbom.MounDataFlags{
				dbom.MounDataFlag{Name: "noexec", NeedsValue: false},
				dbom.MounDataFlag{Name: "ro", NeedsValue: false},
			},
			//Data:     "bind",
			DeviceId: 12346,
			Type:     "ADDON",
		},
	}

	err := suite.mount_repo.Save(&expectedMountPoints[0])
	suite.Require().NoError(err)
	err = suite.mount_repo.Save(&expectedMountPoints[1])
	suite.Require().NoError(err)

	mountPoints, err := suite.mount_repo.All()

	suite.Require().NoError(err)
	if !cmp.Equal(expectedMountPoints, mountPoints, cmpopts.IgnoreFields(dbom.MountPointPath{}, "CreatedAt", "UpdatedAt")) {
		suite.Equal(expectedMountPoints, mountPoints)
		//		t.Errorf("FuncUnderTest() mismatch")
	}
	//assert.Equal(t, expectedMountPoints, mountPoints)
	suite.Len(mountPoints, 2)

	for i, mp := range mountPoints {
		suite.Equal(expectedMountPoints[i].Path, mp.Path)
		//assert.Equal(t, expectedMountPoints[i].Label, mp.Label)
		suite.Equal(expectedMountPoints[i].Device, mp.Device)
		suite.Equal(expectedMountPoints[i].FSType, mp.FSType)
		suite.Equal(expectedMountPoints[i].Flags, mp.Flags)
		//assert.Equal(t, expectedMountPoints[i].Data, mp.Data)
	}
}
