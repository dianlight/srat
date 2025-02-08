package repository_test

import (
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type MountPointPathRepositorySuite struct {
	suite.Suite
	mount_repo repository.MountPointPathRepositoryInterface
	// VariableThatShouldStartAtFive int
}

func TestMountPointPathRepositorySuite(t *testing.T) {
	csuite := new(MountPointPathRepositorySuite)
	csuite.mount_repo = repository.NewMountPointPathRepository(dbom.GetDB())
	/*
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		csuite.mockBoradcaster = NewMockBroadcasterServiceInterface(ctrl)
		csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()
		csuite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()
	*/
	suite.Run(t, csuite)
}

func (suite *MountPointPathRepositorySuite) TestMountPointDataSaveWithoutData() {

	testMountPoint := dbom.MountPointPath{
		Path: "/addons",
	}

	err := suite.mount_repo.Save(&testMountPoint)

	suite.NoError(err)
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *MountPointPathRepositorySuite) TestMountPointDataSave() {

	testMountPoint := dbom.MountPointPath{
		Path: "/mnt/test",
		//Label:  "Test Drive",
		Source: "test_drive",
		FSType: "ext4",
		Flags:  []dto.MounDataFlag{dto.MS_RDONLY, dto.MS_NOATIME},
		//Data:   "rw,noatime",
		//DeviceId: 12344,
	}

	err := suite.mount_repo.Save(&testMountPoint)

	suite.NoError(err)
	dbom.GetDB().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbom.MountPointPath{})
}

func (suite *MountPointPathRepositorySuite) TestMountPointDataAll() {

	expectedMountPoints := []dbom.MountPointPath{
		{
			Path: "/mnt/test1",
			//Label:  "Test 1",
			Source: "test1",
			FSType: "ext4",
			Flags:  []dto.MounDataFlag{dto.MS_RDONLY, dto.MS_NOATIME},
			//Data:     "rw,noatime",
			DeviceId: 12345,
		},
		{
			Path: "/mnt/test2",
			//Label:  "Test 2",
			Source: "test2",
			FSType: "ntfs",
			Flags:  []dto.MounDataFlag{dto.MS_BIND},
			//Data:     "bind",
			DeviceId: 12346,
		},
	}

	err := suite.mount_repo.Save(&expectedMountPoints[0])
	suite.NoError(err)
	err = suite.mount_repo.Save(&expectedMountPoints[1])
	suite.NoError(err)

	mountPoints, err := suite.mount_repo.All()

	suite.NoError(err)
	if !cmp.Equal(expectedMountPoints, mountPoints, cmpopts.IgnoreFields(dbom.MountPointPath{}, "CreatedAt", "UpdatedAt")) {
		suite.Equal(expectedMountPoints, mountPoints)
		//		t.Errorf("FuncUnderTest() mismatch")
	}
	//assert.Equal(t, expectedMountPoints, mountPoints)
	suite.Len(mountPoints, 2)

	for i, mp := range mountPoints {
		suite.Equal(expectedMountPoints[i].Path, mp.Path)
		//assert.Equal(t, expectedMountPoints[i].Label, mp.Label)
		suite.Equal(expectedMountPoints[i].Source, mp.Source)
		suite.Equal(expectedMountPoints[i].FSType, mp.FSType)
		suite.Equal(expectedMountPoints[i].Flags, mp.Flags)
		//assert.Equal(t, expectedMountPoints[i].Data, mp.Data)
	}
}
