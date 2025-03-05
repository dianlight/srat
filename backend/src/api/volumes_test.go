package api_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/go-fuego/fuego"
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
	gomock "go.uber.org/mock/gomock"
)

type VolumeHandlerSuite struct {
	suite.Suite
	dirtyservice      service.DirtyDataServiceInterface
	mockVolumeService *MockVolumeServiceInterface
	mount_repo        repository.MountPointPathRepositoryInterface
}

func TestVolumeHandlerSuite(t *testing.T) {
	csuite := new(VolumeHandlerSuite)
	csuite.dirtyservice = service.NewDirtyDataService(testContext)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	//	csuite.mockBoradcaster = NewMockBroadcasterServiceInterface(ctrl)
	//	csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()
	//	csuite.mockBoradcaster.EXPECT().BroadcastMessage(gomock.Any()).AnyTimes()
	bogusBlockInfo := dto.BlockInfo{
		TotalSizeBytes: 99999,
		Partitions: []*dto.BlockPartition{
			{
				Name:       "bogus1",
				MountPoint: "/mnt/bogus1",
				Label:      "_EXT4",
				SizeBytes:  10000,
				Type:       "ext4",
			},
			{
				Name:       "bogus2",
				MountPoint: "/mnt/bogus2",
				Label:      "testvolume",
				SizeBytes:  10000,
				Type:       "vfat",
			},
		},
	}
	csuite.mockVolumeService = NewMockVolumeServiceInterface(ctrl)
	csuite.mockVolumeService.EXPECT().GetVolumesData().AnyTimes().Return(&bogusBlockInfo, nil)
	csuite.mockVolumeService.EXPECT().MountVolume(gomock.Any()).AnyTimes().Return(nil)
	csuite.mockVolumeService.EXPECT().UnmountVolume(gomock.Any(), false, false).AnyTimes().Return(nil)

	csuite.mount_repo = repository.NewMountPointPathRepository(dbom.GetDB())

	suite.Run(t, csuite)
}

func (suite *VolumeHandlerSuite) TestListVolumessHandler() {
	volume := api.NewVolumeHandler(suite.mockVolumeService, suite.mount_repo, &apiContextState, suite.dirtyservice)
	ctx := fuego.NewMockContextNoBody()
	volumes, err := volume.ListVolumes(ctx)
	suite.Require().NoError(err)

	suite.NotNil(funk.Find(volumes.Partitions, func(d *dto.BlockPartition) bool {
		return d.Label == "testvolume"
	}))
	suite.NotNil(funk.Find(volumes.Partitions, func(d *dto.BlockPartition) bool {
		return d.Label == "_EXT4"
	}), "Expected _EXT4 volume not found %+v", funk.Map(volumes.Partitions, func(d *dto.BlockPartition) string {
		return d.Label + "[" + d.Name + "]"
	}))

}

func (suite *VolumeHandlerSuite) TestMountVolumeHandler() {
	volume := api.NewVolumeHandler(suite.mockVolumeService, suite.mount_repo, &apiContextState, suite.dirtyservice)
	// Check if loop device is available for mounting
	volumes, err := suite.mockVolumeService.GetVolumesData()
	suite.Require().NoError(err)

	var mockMountData dbom.MountPointPath

	for _, d := range volumes.Partitions {
		if strings.HasPrefix(d.Name, "bogus") && d.Label == "_EXT4" {
			mockMountData.Source = d.Name
			mockMountData.Path = filepath.Join("/mnt", d.Label)
			mockMountData.FSType = d.Type
			mockMountData.Flags = []dto.MounDataFlag{dto.MS_NOATIME}
			suite.T().Logf("Selected loop device: %v", mockMountData)
			break
		}
	}
	err = suite.mount_repo.Save(&mockMountData)
	suite.Require().NoError(err)

	conv := converter.DtoToDbomConverterImpl{}
	var mockMountData1 dto.MountPointData
	err = conv.MountPointPathToMountPointData(mockMountData, &mockMountData1)
	suite.Require().NoError(err)

	ctx := fuego.NewMockContext(mockMountData1)
	ctx.PathParams = map[string]string{
		"id": fmt.Sprintf("%d", mockMountData.ID),
	}

	responseData, err := volume.MountVolume(ctx)

	suite.Require().NoError(err)

	// Verify the response data
	if !strings.HasPrefix(responseData.Path, mockMountData.Path) {
		suite.T().Errorf("Unexpected path in response: got %v want %v", responseData.Path, mockMountData.Path)
	}
	if responseData.FSType != mockMountData.FSType {
		suite.T().Errorf("Unexpected FSType in response: got %v want %v", responseData.FSType, mockMountData.FSType)
	}
	if !reflect.DeepEqual(responseData.Flags, mockMountData.Flags) {
		suite.T().Errorf("Unexpected Flags in response: got %v want %v", responseData.Flags, mockMountData.Flags)
	}
}

func (suite *VolumeHandlerSuite) TestUmountVolumeNonExistent() {
	volume := api.NewVolumeHandler(suite.mockVolumeService, suite.mount_repo, &apiContextState, suite.dirtyservice)
	ctx := fuego.NewMockContextNoBody()
	ctx.PathParams = map[string]string{
		"id": "999999",
	}

	_, err := volume.UmountVolume(ctx)
	suite.Require().Error(err)
	//suite.ErrorIs(err, fuego.NotFoundError{})
}

func (suite *VolumeHandlerSuite) TestUmountVolumeSuccess() {

	volume := api.NewVolumeHandler(suite.mockVolumeService, suite.mount_repo, &apiContextState, suite.dirtyservice)
	ctx := fuego.NewMockContextNoBody()
	ctx.PathParams = map[string]string{
		"id": "1",
	}

	res, err := volume.UmountVolume(ctx)
	suite.Require().NoError(err)
	suite.True(res)
}
