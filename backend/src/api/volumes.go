package api

import (
	"context"
	"errors"
	"regexp"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"gorm.io/gorm"
)

var invalidCharactere = regexp.MustCompile(`[^a-zA-Z0-9-]`)

//var extractDeviceName = regexp.MustCompile(`/dev/(\w+)\d+`)
//var extractBlockName = regexp.MustCompile(`/dev/(\w+\d+)`)

type VolumeHandler struct {
	apiContext   *dto.ContextState
	vservice     service.VolumeServiceInterface
	mount_repo   repository.MountPointPathRepositoryInterface
	dirtyservice service.DirtyDataServiceInterface
}

func NewVolumeHandler(vservice service.VolumeServiceInterface, mount_repo repository.MountPointPathRepositoryInterface, apiContext *dto.ContextState, dirtyservice service.DirtyDataServiceInterface) *VolumeHandler {
	p := new(VolumeHandler)
	p.vservice = vservice
	p.mount_repo = mount_repo
	p.apiContext = apiContext
	p.dirtyservice = dirtyservice
	return p
}

func (self *VolumeHandler) RegisterVolumeHandlers(api huma.API) {
	huma.Get(api, "/volumes", self.ListVolumes, huma.OperationTags("volume"))
	huma.Post(api, "/volume/{id}/mount", self.MountVolume, huma.OperationTags("volume"))
	huma.Delete(api, "/volume/{id}/mount", self.UmountVolume, huma.OperationTags("volume"))
}

func (self *VolumeHandler) ListVolumes(ctx context.Context, input *struct{}) (*struct{ Body *dto.BlockInfo }, error) {
	volumes, err := self.vservice.GetVolumesData()
	if err != nil {
		return nil, err
	}
	return &struct{ Body *dto.BlockInfo }{Body: volumes}, nil
}

func (self *VolumeHandler) MountVolume(ctx context.Context, input *struct {
	ID   uint `path:"id" exclusiveMinimum:"0" example:"1234" doc:"ID of the driver to mount"`
	Body dto.MountPointData
}) (*struct{ Body dto.MountPointData }, error) {

	mount_data := input.Body

	if mount_data.ID != 0 && mount_data.ID != input.ID {
		return nil, huma.Error409Conflict("Inconsistent ID provided in the request")
	}

	mount_data.ID = input.ID

	err := self.vservice.MountVolume(mount_data)
	if err != nil {
		if einfo, ok := err.(*dto.ErrorInfo); ok {
			switch einfo.Code {
			case dto.ErrorCodes.INVALID_PARAMETER:
				return nil, huma.Error406NotAcceptable("Invalid Parameter", einfo)
			case dto.ErrorCodes.DEVICE_NOT_FOUND:
				return nil, huma.Error404NotFound("Device Not Found", einfo)
			case dto.ErrorCodes.MOUNT_FAIL:
				return nil, huma.Error422UnprocessableEntity(einfo.Message, einfo)
			default:
				return nil, huma.Error500InternalServerError(einfo.Message, einfo)
			}
		}
		return nil, huma.Error500InternalServerError("Unkown Error", err)
	}

	dbom_mount_data, err := self.mount_repo.FindByID(input.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, huma.Error404NotFound("Internal:Device Not Found", err)
		}
		return nil, err
	}
	var conv converter.DtoToDbomConverterImpl
	mounted_data := dto.MountPointData{}
	err = conv.MountPointPathToMountPointData(*dbom_mount_data, &mounted_data)
	if err != nil {
		return nil, err
	}
	self.dirtyservice.SetDirtyVolumes()

	return &struct{ Body dto.MountPointData }{Body: mounted_data}, nil
}

func (self *VolumeHandler) UmountVolume(ctx context.Context, input *struct {
	ID    uint `path:"id" exclusiveMinimum:"0" example:"1234" doc:"ID of the driver to mount"`
	Force bool `query:"force" default:"false" doc:"Force umount operation"`
	Lazy  bool `query:"lazy" default:"false" doc:"Lazy umount operation"`
}) (*struct{}, error) {

	if ok, _ := self.mount_repo.Exists(input.ID); !ok {
		return nil, huma.Error404NotFound("No mount point found for the provided ID", nil)
	}

	err := self.vservice.UnmountVolume(input.ID, input.Force, input.Lazy)
	if err != nil {
		return nil, err
	}

	self.dirtyservice.SetDirtyVolumes()
	return nil, nil
}
