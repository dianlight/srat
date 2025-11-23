package repository_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
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
		Path:     "/addons",
		Type:     "ADDON",
		DeviceId: "sda1",
	}

	err := suite.mount_repo.Save(&testMountPoint)

	suite.Require().NoError(err)
	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *MountPointPathRepositorySuite) TestMountPointDataSave() {

	testMountPoint := dbom.MountPointPath{
		Path:     "/mnt/test",
		DeviceId: "test_drive",
		FSType:   "ext4",
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
	suite.Equal(testMountPoint.DeviceId, foundMountPoint.DeviceId)
	suite.Equal(testMountPoint.FSType, foundMountPoint.FSType)

	suite.Len(*foundMountPoint.Flags, 2)
	suite.Len(*foundMountPoint.Data, 2)

	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *MountPointPathRepositorySuite) TestMountPointDataSaveNilFlags() {

	testMountPoint := dbom.MountPointPath{
		Path:     "/mnt/test",
		DeviceId: "test_drive",
		FSType:   "ext4",
		Flags:    &dbom.MounDataFlags{},
		Type:     "ADDON",
		Data:     &dbom.MounDataFlags{},
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
	suite.Equal(testMountPoint.DeviceId, foundMountPoint.DeviceId)
	suite.Equal(testMountPoint.FSType, foundMountPoint.FSType)

	suite.Empty(testMountPoint.Flags)
	suite.Empty(testMountPoint.Data)

	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *MountPointPathRepositorySuite) TestMountPointDataSaveAndUpdateFlagsToNil() {

	testMountPoint := dbom.MountPointPath{
		Path:     "/mnt/test",
		DeviceId: "test_drive",
		FSType:   "ext4",
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
	suite.Equal(testMountPoint.DeviceId, foundMountPoint.DeviceId)
	suite.Equal(testMountPoint.FSType, foundMountPoint.FSType)

	suite.Len(*foundMountPoint.Flags, 2)
	suite.Len(*foundMountPoint.Data, 2)

	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *MountPointPathRepositorySuite) TestMountPointDataAll() {

	expectedMountPoints := []*dbom.MountPointPath{
		{
			Path: "/mnt/test1",
			//Label:  "Test 1",
			DeviceId: "test1",
			FSType:   "ext4",
			Flags: &dbom.MounDataFlags{
				dbom.MounDataFlag{Name: "noatime", NeedsValue: false},
				dbom.MounDataFlag{Name: "rw", NeedsValue: false},
			},
			//Data:     "rw,noatime",
			Type: "ADDON",
		},
		{
			Path: "/mnt/test2",
			//Label:  "Test 2",
			DeviceId: "test2",
			FSType:   "ntfs",
			Flags: &dbom.MounDataFlags{
				dbom.MounDataFlag{Name: "noexec", NeedsValue: false},
				dbom.MounDataFlag{Name: "ro", NeedsValue: false},
			},
			//Data:     "bind",
			Type: "ADDON",
		},
	}

	err := suite.mount_repo.Save(expectedMountPoints[0])
	suite.Require().NoError(err)
	err = suite.mount_repo.Save(expectedMountPoints[1])
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
		suite.Equal(expectedMountPoints[i].DeviceId, mp.DeviceId)
		suite.Equal(expectedMountPoints[i].FSType, mp.FSType)
		suite.Equal(expectedMountPoints[i].Flags, mp.Flags)
		//assert.Equal(t, expectedMountPoints[i].Data, mp.Data)
	}
}

func (suite *MountPointPathRepositorySuite) TestConcurrentSaveAndAllNoBusy() {
	// Seed with an initial record
	mp := dbom.MountPointPath{
		Path:     "/mnt/concurrent",
		DeviceId: "concurrent0",
		FSType:   "ext4",
		Type:     "ADDON",
	}
	err := suite.mount_repo.Save(&mp)
	suite.Require().NoError(err)

	// Run concurrent readers and writers
	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writer goroutine: updates device name in a loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
			}
			i++
			mp.DeviceId = fmt.Sprintf("concurrent%d", i)
			_ = suite.mount_repo.Save(&mp) // ignore error to keep pressure; any error would fail later on read/assert
		}
	}()

	// Reader goroutine: calls All repeatedly
	readErrCh := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 1000; j++ {
			if _, err := suite.mount_repo.All(); err != nil {
				readErrCh <- err
				return
			}
		}
		readErrCh <- nil
	}()

	// Wait for reader to finish or error
	readErr := <-readErrCh
	close(stop)
	wg.Wait()

	suite.Require().NoError(readErr)
}

func (suite *MountPointPathRepositorySuite) TestFindByDevice() {
	// Setup test data
	mp := dbom.MountPointPath{
		Path:     "/mnt/device-test",
		DeviceId: "sdb1",
		FSType:   "ext4",
		Type:     "ADDON",
	}
	err := suite.mount_repo.Save(&mp)
	suite.Require().NoError(err)

	// Test finding by device ID (exact match)
	found, err := suite.mount_repo.FindByDevice("sdb1")
	suite.Require().NoError(err)
	suite.Equal("/mnt/device-test", found[0].Path)
	suite.Equal("sdb1", found[0].DeviceId)

	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

/*
func (suite *MountPointPathRepositorySuite) TestAllByDeviceId() {
	// Setup test data
	mp1 := dbom.MountPointPath{
		Path:     "/mnt/test1",
		DeviceId: "sda1",
		FSType:   "ext4",
		Type:     "ADDON",
	}
	mp2 := dbom.MountPointPath{
		Path:     "/mnt/test2",
		DeviceId: "sdb1",
		FSType:   "ntfs",
		Type:     "ADDON",
	}

	err := suite.mount_repo.Save(&mp1)
	suite.Require().NoError(err)
	err = suite.mount_repo.Save(&mp2)
	suite.Require().NoError(err)

	// Get all mount points mapped by device ID
	mpMap, err := suite.mount_repo.AllByDeviceId()
	suite.Require().NoError(err)
	suite.Len(mpMap, 2)
	suite.Contains(mpMap, "sda1")
	suite.Contains(mpMap, "sdb1")
	suite.Equal("/mnt/test1", mpMap["sda1"].Path)
	suite.Equal("/mnt/test2", mpMap["sdb1"].Path)

	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}
*/

/*
func (suite *MountPointPathRepositorySuite) TestExists() {
	// Setup test data
	mp := dbom.MountPointPath{
		Path:     "/mnt/exists-test",
		DeviceId: "sdc1",
		FSType:   "ext4",
		Type:     "ADDON",
	}
	err := suite.mount_repo.Save(&mp)
	suite.Require().NoError(err)

	// Test path that exists
	exists, err := suite.mount_repo.Exists("/mnt/exists-test")
	suite.Require().NoError(err)
	suite.True(exists)

	// Test path that doesn't exist
	exists, err = suite.mount_repo.Exists("/mnt/nonexistent")
	suite.Require().NoError(err)
	suite.False(exists)

	suite.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}
*/

func (suite *MountPointPathRepositorySuite) TestDelete() {
	// Setup test data
	mp := dbom.MountPointPath{
		Path:     "/mnt/delete-test",
		DeviceId: "sdd1",
		FSType:   "ext4",
		Type:     "ADDON",
	}
	err := suite.mount_repo.Save(&mp)
	suite.Require().NoError(err)

	// Verify it exists
	found, err := suite.mount_repo.FindByPath(mp.Path)
	suite.Require().NoError(err)
	suite.NotNil(found)

	// Delete it
	err = suite.mount_repo.Delete(mp.Path)
	suite.Require().NoError(err)

	// Verify it no longer exists
	found, err = suite.mount_repo.FindByPath(mp.Path)
	suite.Require().ErrorAs(err, &gorm.ErrRecordNotFound)
	suite.Nil(found)
}
