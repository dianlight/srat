package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	"github.com/xorcare/pointer"
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

	bogusMountData := &dto.MountPointData{
		Device: "bogus1",
		Path:   "/mnt/bogus1",
		FSType: "ext4",
		Flags:  []string{"MS_NOATIME", "MS_RDONLY"},
		Type:   "ADDON",
	}

	bogusDisks := []dto.Disk{
		{
			Vendor: pointer.String("bogus"),
			Partitions: &[]dto.Partition{
				{
					Device: pointer.String("/dev/bogus1"),
					Name:   pointer.String("_EXT4"),
					Size:   pointer.Int(10000),
					MountPointData: &[]dto.MountPointData{
						*bogusMountData,
					},
				},
			},
		},
	}

	csuite.mockVolumeService = NewMockVolumeServiceInterface(ctrl)
	csuite.mockVolumeService.EXPECT().GetVolumesData().AnyTimes().Return(&bogusDisks, nil)
	csuite.mockVolumeService.EXPECT().MountVolume(gomock.Any()).AnyTimes().Return(nil)
	csuite.mockVolumeService.EXPECT().UnmountVolume(gomock.Any(), false, false).AnyTimes().Return(nil)

	csuite.mount_repo = repository.NewMountPointPathRepository(dbom.GetDB())

	conv := converter.DtoToDbomConverterImpl{}
	mountPath := &dbom.MountPointPath{}
	err := conv.MountPointDataToMountPointPath(*bogusMountData, mountPath)
	if err != nil {
		csuite.T().Errorf("Error converting mount point data: %v", err)
	}

	err = csuite.mount_repo.Save(mountPath)
	if err != nil {
		csuite.T().Errorf("Error saving mount point data: %v", err)
	}

	suite.Run(t, csuite)
}

func (suite *VolumeHandlerSuite) TestListVolumessHandler() {
	volume := api.NewVolumeHandler(suite.mockVolumeService, suite.mount_repo, &apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	volume.RegisterVolumeHandlers(api)

	rr := api.Get("/volumes")
	suite.Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	var volumes []dto.Disk
	err2 := json.NewDecoder(rr.Body).Decode(&volumes)
	if err2 != nil {
		suite.T().Errorf("handler error in decode body %v", err2)
	}

	suite.NotNil(funk.Find(volumes, func(d dto.Disk) bool {
		return funk.Find(*d.Partitions, func(p dto.Partition) bool {
			return *p.Device == "/dev/bogus1"
		}) != nil
	}))
	suite.NotNil(funk.Find(volumes, func(d dto.Disk) bool {
		return funk.Find(*d.Partitions, func(p dto.Partition) bool {
			return *p.Name == "_EXT4"
		}) != nil
	}), "Expected _EXT4 volume not found %+v", funk.Map(volumes, func(d dto.Disk) []string {
		return funk.Map(*d.Partitions, func(p dto.Partition) string {
			return *p.Name + "[" + *p.Device + "]"
		}).([]string)
	}))

}

//var

func (suite *VolumeHandlerSuite) TestMountVolumeHandler() {
	volume := api.NewVolumeHandler(suite.mockVolumeService, suite.mount_repo, &apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	volume.RegisterVolumeHandlers(api)

	volumes, err := suite.mockVolumeService.GetVolumesData()
	suite.Require().NoError(err)
	suite.Require().NotEmpty(volumes)
	suite.Require().NotEmpty((*volumes)[0].Partitions)
	suite.Require().NotEmpty((*(*volumes)[0].Partitions)[0].MountPointData)

	rr := api.Post(fmt.Sprintf("/volume/%s/mount", url.PathEscape((*(*(*volumes)[0].Partitions)[0].MountPointData)[0].Path)), (*(*(*volumes)[0].Partitions)[0].MountPointData)[0])
	suite.Equal(http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())

	// Check the response body is what we expect.
	var responseData dto.MountPointData
	err = json.Unmarshal(rr.Body.Bytes(), &responseData)
	suite.Require().NoError(err)

	// Verify the response data
	if !strings.HasPrefix(responseData.Device, (*(*(*volumes)[0].Partitions)[0].MountPointData)[0].Device) {
		suite.T().Errorf("Unexpected path in response: got %v want %v", responseData.Device, (*(*(*volumes)[0].Partitions)[0].MountPointData)[0].Device)
	}

	suite.NotEmpty(responseData.Device, "Device should not be empty")
}

func (suite *VolumeHandlerSuite) TestUmountVolumeNonExistent() {
	volume := api.NewVolumeHandler(suite.mockVolumeService, suite.mount_repo, &apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	volume.RegisterVolumeHandlers(api)

	rr := api.Delete("/volume/999999/mount")
	suite.Require().Equal(http.StatusNotFound, rr.Code)

	suite.JSONEq(`{"title":"Not Found","status":404,"detail":"No mount point found for the provided mount path","errors":[null]}`, rr.Body.String())
}
func (suite *VolumeHandlerSuite) TestUmountVolumeSuccess() {
	volume := api.NewVolumeHandler(suite.mockVolumeService, suite.mount_repo, &apiContextState, suite.dirtyservice)
	_, api := humatest.New(suite.T())
	volume.RegisterVolumeHandlers(api)

	rr := api.Delete(fmt.Sprintf("/volume/%s/mount", url.PathEscape("/mnt/bogus1")))
	suite.Equal(http.StatusNoContent, rr.Code, "Body %#v", rr.Body.String())
	suite.Empty(rr.Body.String(), "Body should be empty")
}
