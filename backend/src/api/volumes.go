package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/dianlight/srat/tlog"
	"github.com/shomali11/util/xhashes"
)

//var invalidCharactere = regexp.MustCompile(`[^a-zA-Z0-9-]`)

//var extractDeviceName = regexp.MustCompile(`/dev/(\w+)\d+`)
//var extractBlockName = regexp.MustCompile(`/dev/(\w+\d+)`)

type VolumeHandler struct {
	apiContext   *dto.ContextState
	vservice     service.VolumeServiceInterface
	shareService service.ShareServiceInterface
	//dirtyservice service.DirtyDataServiceInterface
}

func NewVolumeHandler(
	vservice service.VolumeServiceInterface,
	shareService service.ShareServiceInterface,
	apiContext *dto.ContextState,
) *VolumeHandler {
	p := new(VolumeHandler)
	p.vservice = vservice
	p.shareService = shareService
	p.apiContext = apiContext
	//p.dirtyservice = dirtyservice
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
	huma.Patch(api, "/volume/{mount_path_hash}/settings", self.PatchMountPointSettings, huma.OperationTags("volume"))
	// huma.Post(api, "/volume/disk/{disk_id}/eject", self.EjectDiskHandler, huma.OperationTags("volume"))
}

func (self *VolumeHandler) ListVolumes(ctx context.Context, input *struct{}) (*struct{ Body *[]dto.Disk }, error) {
	volumes := self.vservice.GetVolumesData()
	return &struct{ Body *[]dto.Disk }{Body: volumes}, nil
}

func (self *VolumeHandler) MountVolume(ctx context.Context, input *struct {
	MountPathHash string             `path:"mount_path_hash"`
	Body          dto.MountPointData `required:"true"`
}) (*struct{ Body dto.MountPointData }, error) {

	mount_data := input.Body

	if mount_data.Path == "" || mount_data.PathHash != xhashes.SHA1(mount_data.Path) {
		return nil, huma.Error409Conflict("Inconsistent MountPath provided in the request")
	}

	errE := self.vservice.MountVolume(&mount_data)
	if errE != nil {
		if errors.Is(errE, dto.ErrorMountFail) {
			tlog.ErrorContext(ctx, "Failed to mount volume", "mount_path", mount_data.Path, "error", errE)
			if errE.Details() != nil {
				var errMessage string
				for key, value := range errE.Details() {
					errMessage += fmt.Sprintf("%s: %v\n", key, value)
				}
				return nil, huma.Error422UnprocessableEntity(errMessage, errE)
			} else {
				return nil, huma.Error422UnprocessableEntity("Failed to mount volume", errE)
			}
		} else if errors.Is(errE, dto.ErrorDeviceNotFound) {
			return nil, huma.Error404NotFound("Device Not Found", errE)
		} else if errors.Is(errE, dto.ErrorInvalidParameter) {
			return nil, huma.Error406NotAcceptable("Invalid Parameter", errE)
		} else {
			return nil, huma.Error500InternalServerError("Unknown Error", errE)
		}
	}

	return &struct{ Body dto.MountPointData }{Body: mount_data}, nil
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

	// Disable all share services for this mount point
	_, errE := self.shareService.SetShareFromPathEnabled(mountPath, false)
	if errE != nil && !errors.Is(errE, dto.ErrorShareNotFound) {
		return nil, huma.Error500InternalServerError("Failed to disable share for mount point", err)
	}

	err = self.vservice.UnmountVolume(mountPath, input.Force, input.Lazy && !input.Force)
	if err != nil {
		return nil, huma.Error406NotAcceptable(fmt.Sprintf("%#v", err.Details()["Detail"]), err)
	}

	return nil, nil
}

/*
func (self *VolumeHandler) EjectDiskHandler(ctx context.Context, input *struct {
	DiskID string `path:"disk_id" doc:"The ID of the disk to eject (e.g., sda, sdb)"`
}) (*struct{ Status int }, error) {
	if self.apiContext.ReadOnlyMode {
		return nil, huma.Error403Forbidden("Cannot eject disk in read-only mode")
	}

	err := self.vservice.EjectDisk(input.DiskID)
	if err != nil {
		if errors.Is(err, dto.ErrorDeviceNotFound) {
			return nil, huma.Error404NotFound(fmt.Sprintf("Disk '%s' not found or not ejectable", input.DiskID), err)
		}
		if errors.Is(err, dto.ErrorInvalidParameter) {
			return nil, huma.Error400BadRequest(fmt.Sprintf("Invalid parameter for ejecting disk '%s'", input.DiskID), err)
		}
		// Error is already logged by the service layer
		// Return a more generic error to the client
		return nil, huma.Error500InternalServerError(fmt.Sprintf("Failed to eject disk '%s': %s", input.DiskID, err.Error()), err)
	}

	// Return 204 No Content on success
	return &struct{ Status int }{Status: http.StatusNoContent}, nil
}
*/

// PatchMountPointSettings handles PATCH requests to partially update the configuration of an existing mount point.
func (self *VolumeHandler) PatchMountPointSettings(ctx context.Context, input *struct {
	MountPathHash string             `path:"mount_path_hash"`
	Body          dto.MountPointData `required:"true"`
}) (*struct{ Body dto.MountPointData }, error) {
	if self.apiContext.ReadOnlyMode {
		return nil, huma.Error403Forbidden("Cannot update volume settings in read-only mode")
	}

	mountPath, errE := self.vservice.PathHashToPath(input.MountPathHash)
	if errE != nil {
		return nil, huma.Error404NotFound("No mount point found for the provided mount pathhash", nil)
	}

	updatedDto, serviceErr := self.vservice.PatchMountPointSettings(mountPath, input.Body)
	if serviceErr != nil {
		if errors.Is(serviceErr, dto.ErrorNotFound) {
			return nil, huma.Error404NotFound(serviceErr.Error())
		}
		// Error is already logged by the service layer
		return nil, huma.Error500InternalServerError("Failed to patch mount configuration", serviceErr)
	}

	return &struct{ Body dto.MountPointData }{Body: *updatedDto}, nil
}
