package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/shomali11/util/xhashes"
	"github.com/stretchr/testify/suite"
	"github.com/xorcare/pointer"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type VolumeHandlerSuite struct {
	suite.Suite
	app           *fxtest.App
	handler       *api.VolumeHandler
	mockVolumeSvc service.VolumeServiceInterface
	mockShareSvc  service.ShareServiceInterface
	ctx           context.Context
	cancel        context.CancelFunc
}

func TestVolumeHandlerSuite(t *testing.T) { suite.Run(t, new(VolumeHandlerSuite)) }

func (suite *VolumeHandlerSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			func() *dto.ContextState { return &dto.ContextState{} },
			api.NewVolumeHandler,
			mock.Mock[service.VolumeServiceInterface],
			mock.Mock[service.ShareServiceInterface],
		),
		fx.Populate(&suite.handler),
		fx.Populate(&suite.mockVolumeSvc),
		fx.Populate(&suite.mockShareSvc),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *VolumeHandlerSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
		if wg, ok := suite.ctx.Value("wg").(*sync.WaitGroup); ok {
			wg.Wait()
		}
	}
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

// TestListVolumes_ReturnsDiskPartitionMountPointData verifies that the /volumes endpoint
// returns nested structures including disks, their partitions, and mount point data.
func (suite *VolumeHandlerSuite) TestListVolumes_ReturnsDiskPartitionMountPointData() {
	diskID := "disk1"
	partID := "part1"
	devicePath := "/dev/sda1"
	mountPath := "/mnt/data"
	mountPoint := dto.MountPointData{
		Path:             mountPath,
		PathHash:         xhashes.SHA1(mountPath),
		DeviceId:         partID,
		IsMounted:        true,
		IsWriteSupported: pointer.Bool(true),
		Type:             "ADDON",
	}
	partition := dto.Partition{
		Id:             &partID,
		DevicePath:     &devicePath,
		MountPointData: &map[string]dto.MountPointData{mountPath: mountPoint},
	}
	disk := dto.Disk{
		Id:         &diskID,
		Partitions: &map[string]dto.Partition{partID: partition},
	}
	disks := &[]dto.Disk{disk}

	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(disks)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterVolumeHandlers(apiInst)

	resp := apiInst.Get("/volumes")
	suite.Require().Equal(http.StatusOK, resp.Code)

	var out []dto.Disk
	suite.NoError(json.Unmarshal(resp.Body.Bytes(), &out))
	suite.Require().Len(out, 1, "Expected one disk returned")

	// Validate Disk
	gotDisk := out[0]
	suite.Require().NotNil(gotDisk.Partitions, "Partitions map must be present")
	suite.Require().Len(*gotDisk.Partitions, 1, "Expected one partition")

	// Validate Partition
	gotPart, ok := (*gotDisk.Partitions)[partID]
	suite.Require().True(ok, "Partition id missing in map")
	suite.Require().NotNil(gotPart.MountPointData, "MountPointData map must be present")
	suite.Require().Len(*gotPart.MountPointData, 1, "Expected one mount point")

	// Validate MountPointData
	gotMP, ok := (*gotPart.MountPointData)[mountPath]
	suite.Require().True(ok, "Mount point path missing in map")
	suite.Equal(mountPath, gotMP.Path)
	suite.Equal(gotMP.PathHash, xhashes.SHA1(mountPath))
	suite.Equal(partID, gotMP.DeviceId)
	suite.True(gotMP.IsMounted, "Mount point should be marked mounted")
	suite.True(*gotMP.IsWriteSupported, "Mount point should support write")
	suite.Equal("ADDON", gotMP.Type)

	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).GetVolumesData()
}

// TestPatchMountPointSettings_UpdatesIsToMountAtStartup verifies that PatchMountPointSettings
// correctly updates the mount point configuration and the change is reflected in GetVolumesData.
func (suite *VolumeHandlerSuite) TestPatchMountPointSettings_UpdatesIsToMountAtStartup() {
	diskID := "disk1"
	partID := "part1"
	devicePath := "/dev/sda1"
	mountPath := "/mnt/data"
	mountPathHash := xhashes.SHA1(mountPath)

	// Initial state: mount point with IsToMountAtStartup = false
	mountPointInitial := dto.MountPointData{
		Path:               mountPath,
		PathHash:           mountPathHash,
		DeviceId:           partID,
		IsMounted:          false,
		IsToMountAtStartup: pointer.Bool(false),
		Type:               "ADDON",
	}
	partition := dto.Partition{
		Id:             &partID,
		DevicePath:     &devicePath,
		MountPointData: &map[string]dto.MountPointData{mountPath: mountPointInitial},
	}
	disk := dto.Disk{
		Id:         &diskID,
		Partitions: &map[string]dto.Partition{partID: partition},
	}
	disksInitial := &[]dto.Disk{disk}

	// Updated state: mount point with IsToMountAtStartup = true
	mountPointUpdated := mountPointInitial
	mountPointUpdated.IsToMountAtStartup = pointer.Bool(true)
	partitionUpdated := partition
	partitionUpdated.MountPointData = &map[string]dto.MountPointData{mountPath: mountPointUpdated}
	diskUpdated := disk
	diskUpdated.Partitions = &map[string]dto.Partition{partID: partitionUpdated}
	disksUpdated := &[]dto.Disk{diskUpdated}

	// Mock GetVolumesData to return initial state, then updated state
	mock.When(suite.mockVolumeSvc.GetVolumesData()).ThenReturn(disksInitial).ThenReturn(disksUpdated)
	mock.When(suite.mockVolumeSvc.PathHashToPath(mountPathHash)).ThenReturn(mountPath, nil)
	mock.When(suite.mockVolumeSvc.PatchMountPointSettings(mock.Any[string](), mock.Any[dto.MountPointData]())).
		ThenReturn(&mountPointUpdated, nil)

	_, apiInst := humatest.New(suite.T())
	suite.handler.RegisterVolumeHandlers(apiInst)

	// Step 1: Get initial volumes data
	resp1 := apiInst.Get("/volumes")
	suite.Require().Equal(http.StatusOK, resp1.Code)

	var disks1 []dto.Disk
	suite.NoError(json.Unmarshal(resp1.Body.Bytes(), &disks1))
	suite.Require().Len(disks1, 1)
	mp1 := (*(*disks1[0].Partitions)[partID].MountPointData)[mountPath]
	suite.Equal(false, *mp1.IsToMountAtStartup, "Initial IsToMountAtStartup should be false")

	// Step 2: Patch IsToMountAtStartup to true
	patchBody := dto.MountPointData{
		Path:               mountPath,
		Type:               "ADDON",
		IsToMountAtStartup: pointer.Bool(true),
	}
	resp2 := apiInst.Patch("/volume/"+mountPathHash+"/settings", patchBody)
	suite.Require().Equal(http.StatusOK, resp2.Code)

	var patchedMP dto.MountPointData
	suite.NoError(json.Unmarshal(resp2.Body.Bytes(), &patchedMP))
	suite.Equal(true, *patchedMP.IsToMountAtStartup, "Patched MountPointData should have IsToMountAtStartup=true")

	// Step 3: Get volumes data again and verify the change
	resp3 := apiInst.Get("/volumes")
	suite.Require().Equal(http.StatusOK, resp3.Code)

	var disks2 []dto.Disk
	suite.NoError(json.Unmarshal(resp3.Body.Bytes(), &disks2))
	suite.Require().Len(disks2, 1)
	mp2 := (*(*disks2[0].Partitions)[partID].MountPointData)[mountPath]
	suite.Equal(true, *mp2.IsToMountAtStartup, "GetVolumesData should reflect patched IsToMountAtStartup=true")

	mock.Verify(suite.mockVolumeSvc, matchers.Times(2)).GetVolumesData()
	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).PathHashToPath(mountPathHash)
	mock.Verify(suite.mockVolumeSvc, matchers.Times(1)).PatchMountPointSettings(mock.Any[string](), mock.Any[dto.MountPointData]())
}
