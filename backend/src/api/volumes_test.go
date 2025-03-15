package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
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
	_, api := humatest.New(suite.T())
	volume.RegisterVolumeHandlers(api)

	rr := api.Get("/volumes")
	suite.Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var volumes dto.BlockInfo
	err2 := json.NewDecoder(rr.Body).Decode(&volumes)
	if err2 != nil {
		suite.T().Errorf("handler error in decode body %v", err2)
	}

	suite.NotNil(funk.Find(volumes.Partitions, func(d *dto.BlockPartition) bool {
		return d.Label == "testvolume"
	}))
	suite.NotNil(funk.Find(volumes.Partitions, func(d *dto.BlockPartition) bool {
		return d.Label == "_EXT4"
	}), "Expected _EXT4 volume not found %+v", funk.Map(volumes.Partitions, func(d *dto.BlockPartition) string {
		return d.Label + "[" + d.Name + "]"
	}))

}

//var

func (suite *VolumeHandlerSuite) TestMountVolumeHandler() {
	volume := api.NewVolumeHandler(suite.mockVolumeService, suite.mount_repo, &apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	volume.RegisterVolumeHandlers(api)

	volumes, err := suite.mockVolumeService.GetVolumesData()
	suite.Require().NoError(err)

	var mockMountPath dbom.MountPointPath

	for _, d := range volumes.Partitions {
		if strings.HasPrefix(d.Name, "bogus") && d.Label == "_EXT4" {
			mockMountPath.Source = d.Name
			mockMountPath.Path = filepath.Join("/mnt", d.Label)
			mockMountPath.FSType = d.Type
			mockMountPath.Flags = []dbom.MounDataFlag{dbom.MS_NOATIME}
			suite.T().Logf("Selected loop device: %v", mockMountPath)
			break
		}
	}
	err = suite.mount_repo.Save(&mockMountPath)
	suite.Require().NoError(err)

	conv := converter.DtoToDbomConverterImpl{}
	var mockMountData dto.MountPointData
	conv.MountPointPathToMountPointData(mockMountPath, &mockMountData)
	rr := api.Post(fmt.Sprintf("/volume/%d/mount", mockMountData.ID), mockMountData)
	suite.Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	// Check the response body is what we expecsuite.T().
	var responseData dto.MountPointData
	err = json.Unmarshal(rr.Body.Bytes(), &responseData)
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
	_, api := humatest.New(suite.T())
	volume.RegisterVolumeHandlers(api)

	rr := api.Delete("/volume/999999/mount")
	suite.Require().Equal(http.StatusNotFound, rr.Code)

	suite.JSONEq(`{"title":"Not Found","status":404,"detail":"No mount point found for the provided ID","errors":[null]}`, rr.Body.String())
}
func (suite *VolumeHandlerSuite) TestUmountVolumeSuccess() {
	volume := api.NewVolumeHandler(suite.mockVolumeService, suite.mount_repo, &apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	volume.RegisterVolumeHandlers(api)

	rr := api.Delete(fmt.Sprintf("/volume/%d/mount", 1))
	suite.Equal(http.StatusNoContent, rr.Code, "Body %#v", rr.Body.String())
	suite.Empty(rr.Body.String(), "Body should be empty")
}
