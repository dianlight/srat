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
