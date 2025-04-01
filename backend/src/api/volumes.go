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
	huma.Post(api, "/volume/{id}/mount", self.MountVolume, huma.OperationTags("volume"))
	huma.Delete(api, "/volume/{id}/mount", self.UmountVolume, huma.OperationTags("volume"))
}

// ListVolumes retrieves a list of volumes by calling the vservice's GetVolumesData method.
// It returns a struct containing the volumes data wrapped in a dto.BlockInfo, or an error if the retrieval fails.
//
// Parameters:
// - ctx: The context for the request, which can be used for cancellation and deadlines.
// - input: An empty struct as input, which is not used in this function.
//
// Returns:
// - A struct containing the volumes data wrapped in a dto.BlockInfo.
// - An error if there is an issue retrieving the volumes data.
func (self *VolumeHandler) ListVolumes(ctx context.Context, input *struct{}) (*struct{ Body *dto.BlockInfo }, error) {
	volumes, err := self.vservice.GetVolumesData()
	if err != nil {
		return nil, err
	}
	return &struct{ Body *dto.BlockInfo }{Body: volumes}, nil
}

// MountVolume handles the mounting of a volume based on the provided input.
// It validates the input data, checks for consistency, and attempts to mount the volume using the volume service.
// If the mounting process encounters an error, it returns an appropriate HTTP error response.
// Upon successful mounting, it retrieves the mounted volume data from the repository, converts it, and marks the volumes as dirty.
//
// Parameters:
//   - ctx: The context for the request.
//   - input: A struct containing the ID of the driver to mount and the body with mount point data.
//
// Returns:
//   - A struct containing the mounted volume data in the body, or an error if the mounting process fails.
//
// Possible Errors:
//   - 409 Conflict: If the provided ID in the request is inconsistent.
//   - 406 Not Acceptable: If an invalid parameter is provided.
//   - 404 Not Found: If the device is not found.
//   - 422 Unprocessable Entity: If the mounting process fails.
//   - 500 Internal Server Error: For unknown errors or other internal server errors.
func (self *VolumeHandler) MountVolume(ctx context.Context, input *struct {
	ID   uint `path:"id" exclusiveMinimum:"0" example:"1234" doc:"ID of the driver to mount"`
	Body dto.MountPointData
}) (*struct{ Body dto.MountPointData }, error) {

	mount_data := input.Body

	if mount_data.ID != 0 && mount_data.ID != input.ID {
		return nil, huma.Error409Conflict("Inconsistent ID provided in the request")
	}

	mount_data.ID = input.ID

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

// UmountVolume handles the unmounting of a volume based on the provided input parameters.
// It checks if the volume exists, and if so, proceeds to unmount it using the specified options.
// If the volume does not exist, it returns a 404 error.
//
// Parameters:
//
//	ctx - The context for the request.
//	input - A struct containing the following fields:
//	  ID - The ID of the volume to unmount. Must be a positive integer.
//	  Force - A boolean indicating whether to force the unmount operation. Default is false.
//	  Lazy - A boolean indicating whether to perform a lazy unmount operation. Default is false.
//
// Returns:
//
//	An empty struct and nil error if the operation is successful.
//	An error if the volume does not exist or if the unmount operation fails.
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
