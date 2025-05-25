package api

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/shomali11/util/xhashes"
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

// RegisterVolumeHandlers registers the HTTP handlers for volume-related operations.
// It sets up the following routes:
// - GET /volumes: Lists all volumes.
// - POST /volume/{id}/mount: Mounts a volume with the specified ID.
// - DELETE /volume/{id}/mount: Unmounts a volume with the specified ID.
//
// Parameters:
// - api: The huma.API instance to register the handlers with.
func (self *VolumeHandler) RegisterVolumeHandlers(api huma.API) {
	huma.Get(api, "/volumes", self.ListVolumes, huma.OperationTags("volume"))
	huma.Post(api, "/volume/{mount_path_hash}/mount", self.MountVolume, huma.OperationTags("volume"))
	huma.Delete(api, "/volume/{mount_path_hash}/mount", self.UmountVolume, huma.OperationTags("volume"))
}

func (self *VolumeHandler) ListVolumes(ctx context.Context, input *struct{}) (*struct{ Body *[]dto.Disk }, error) {
	volumes, err := self.vservice.GetVolumesData()
	if err != nil {
		return nil, err
	}
	return &struct{ Body *[]dto.Disk }{Body: volumes}, nil
}

func (self *VolumeHandler) MountVolume(ctx context.Context, input *struct {
	MountPathHash string             `path:"mount_path_hash"`
	Body          dto.MountPointData `required:"true"`
}) (*struct{ Body dto.MountPointData }, error) {

	mount_data := input.Body

	if mount_data.Path == "" || mount_data.PathHash != xhashes.MD5(mount_data.Path) {
		return nil, huma.Error409Conflict("Inconsistent MountPath provided in the request")
	}

	errE := self.vservice.MountVolume(mount_data)
	if errE != nil {
		if errors.Is(errE, dto.ErrorMountFail) {
			return nil, huma.Error422UnprocessableEntity(errE.Details()["Message"].(string), errE)
		} else if errors.Is(errE, dto.ErrorDeviceNotFound) {
			return nil, huma.Error404NotFound("Device Not Found", errE)
		} else if errors.Is(errE, dto.ErrorInvalidParameter) {
			return nil, huma.Error406NotAcceptable("Invalid Parameter", errE)
		} else {
			return nil, huma.Error500InternalServerError("Unknown Error", errE)
		}
	}

	dbom_mount_data, err := self.mount_repo.FindByPath(mount_data.Path)
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
	MountPathHash string `path:"mount_path_hash"`
	Force         bool   `query:"force" default:"false" doc:"Force umount operation"`
	Lazy          bool   `query:"lazy" default:"false" doc:"Lazy umount operation"`
}) (*struct{}, error) {

	mountPath, err := self.vservice.PathHashToPath(input.MountPathHash)
	if err != nil {
		return nil, huma.Error404NotFound("No mount point found for the provided mount pathhash", nil)
	}

	err = self.vservice.UnmountVolume(mountPath, input.Force, input.Lazy)
	if err != nil {
		return nil, huma.Error406NotAcceptable(fmt.Sprintf("%v", err.Details()["Detail"]), err)
	}

	self.dirtyservice.SetDirtyVolumes()
	return nil, nil
}
