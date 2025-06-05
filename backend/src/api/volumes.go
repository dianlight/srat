package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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
	shareService service.ShareServiceInterface
	mount_repo   repository.MountPointPathRepositoryInterface
	dirtyservice service.DirtyDataServiceInterface
}

func NewVolumeHandler(vservice service.VolumeServiceInterface, shareService service.ShareServiceInterface, mount_repo repository.MountPointPathRepositoryInterface, apiContext *dto.ContextState, dirtyservice service.DirtyDataServiceInterface) *VolumeHandler {
	p := new(VolumeHandler)
	p.vservice = vservice
	p.shareService = shareService
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
	huma.Put(api, "/volume/{mount_path_hash}/settings", self.UpdateVolumeSettings, huma.OperationTags("volume"))
	huma.Patch(api, "/volume/{mount_path_hash}/settings", self.PatchVolumeSettings, huma.OperationTags("volume"))
	huma.Post(api, "/volume/disk/{disk_id}/eject", self.EjectDiskHandler, huma.OperationTags("volume"))
}

func (self *VolumeHandler) ListVolumes(ctx context.Context, input *struct{}) (*struct{ Body *[]dto.Disk }, error) {
	volumes, err := self.vservice.GetVolumesData()
	if err != nil {
		return nil, err
	}
	// Integrate Disk with share status
	for i, disk := range *volumes {
		for j, volume := range *disk.Partitions {
			if volume.MountPointData == nil {
				continue
			}
			for k, mountPoint := range *volume.MountPointData {
				// Load additional data fro mountada from DB
				dbom_mount_data, err := self.mount_repo.FindByDevice(mountPoint.Device)
				if err != nil {
					if !errors.Is(err, gorm.ErrRecordNotFound) {
						return nil, err
					}
				} else {
					var conv converter.DtoToDbomConverterImpl
					conv.MountPointPathToMountPointData(*dbom_mount_data, &mountPoint)
					(*volume.MountPointData)[k] = mountPoint
				}

				// Get Shares
				shared, err := self.shareService.GetShareFromPath(mountPoint.Path)
				if err != nil {
					if errors.Is(err, dto.ErrorShareNotFound) {
						continue
					} else {
						// Some other error occurred, return it
						return nil, err
					}
				}
				shares := (*(*(*volumes)[i].Partitions)[j].MountPointData)[k].Shares
				shares = append(shares, *shared)
				(*(*(*volumes)[i].Partitions)[j].MountPointData)[k].Shares = shares
			}
		}
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

	errE := self.vservice.MountVolume(&mount_data)
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
	/*
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

		err = self.mount_repo.Save(dbom_mount_data)
		if err != nil {
			slog.Warn("Unamble to save mount point data","err",err,"mount_path",mount_data.Path)
		}
	*/
	self.dirtyservice.SetDirtyVolumes()

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
	_, errE := self.shareService.DisableShareFromPath(mountPath)
	if errE != nil && !errors.Is(errE, dto.ErrorShareNotFound) {
		return nil, huma.Error500InternalServerError("Failed to disable share for mount point", err)
	}

	err = self.vservice.UnmountVolume(mountPath, input.Force, input.Lazy && !input.Force)
	if err != nil {
		return nil, huma.Error406NotAcceptable(fmt.Sprintf("%v", err.Details()["Detail"]), err)
	}

	self.dirtyservice.SetDirtyVolumes()
	return nil, nil
}

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
		// Log the full error for server-side debugging
		slog.Error("Failed to eject disk", "disk_id", input.DiskID, "error", fmt.Sprintf("%+v", err)) // Log with stack trace if available
		// Return a more generic error to the client
		return nil, huma.Error500InternalServerError(fmt.Sprintf("Failed to eject disk '%s': %s", input.DiskID, err.Error()), err)
	}

	// Return 204 No Content on success
	return &struct{ Status int }{Status: http.StatusNoContent}, nil
}

// UpdateVolumeSettings handles PUT requests to update the configuration of an existing mount point.
func (self *VolumeHandler) UpdateVolumeSettings(ctx context.Context, input *struct {
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

	dbMountData, err := self.mount_repo.FindByPath(mountPath)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, huma.Error404NotFound(fmt.Sprintf("Mount configuration with hash %s not found", input.MountPathHash))
		}
		slog.Error("Error fetching mount configuration by hash for PUT", "hash", input.MountPathHash, "error", err)
		return nil, huma.Error500InternalServerError("Failed to retrieve mount configuration")
	}

	// Apply updates from input.Body
	if input.Body.FSType != nil {
		dbMountData.FSType = *input.Body.FSType
	}
	var conv converter.DtoToDbomConverterImpl

	if input.Body.Flags != nil {
		*dbMountData.Flags = conv.MountFlagsToMountDataFlags(*input.Body.Flags)
	}
	if input.Body.CustomFlags != nil {
		*dbMountData.Data = conv.MountFlagsToMountDataFlags(*input.Body.CustomFlags)
	}
	if input.Body.IsToMountAtStartup != nil {
		dbMountData.IsToMountAtStartup = input.Body.IsToMountAtStartup
	}

	if err := self.mount_repo.Save(dbMountData); err != nil {
		slog.Error("Error saving updated mount configuration", "hash", input.MountPathHash, "error", err)
		return nil, huma.Error500InternalServerError("Failed to save mount configuration")
	}

	updatedDto := dto.MountPointData{}
	if err := conv.MountPointPathToMountPointData(*dbMountData, &updatedDto); err != nil {
		slog.Error("Error converting updated mount configuration to DTO", "hash", input.MountPathHash, "error", err)
		return nil, huma.Error500InternalServerError("Failed to process updated mount configuration")
	}

	self.dirtyservice.SetDirtyVolumes()
	return &struct{ Body dto.MountPointData }{Body: updatedDto}, nil
}

// PatchVolumeSettings handles PATCH requests to partially update the configuration of an existing mount point.
func (self *VolumeHandler) PatchVolumeSettings(ctx context.Context, input *struct {
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

	dbMountData, err := self.mount_repo.FindByPath(mountPath)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, huma.Error404NotFound(fmt.Sprintf("Mount configuration with hash %s not found", input.MountPathHash))
		}
		slog.Error("Error fetching mount configuration by hash for PATCH", "hash", input.MountPathHash, "error", err)
		return nil, huma.Error500InternalServerError("Failed to retrieve mount configuration")
	}

	// Apply partial updates
	if input.Body.IsToMountAtStartup != nil {
		dbMountData.IsToMountAtStartup = input.Body.IsToMountAtStartup
	}
	// Add other patchable fields here if MountPointSettingsUpdate is extended

	if err := self.mount_repo.Save(dbMountData); err != nil {
		slog.Error("Error saving patched mount configuration", "hash", input.MountPathHash, "error", err)
		return nil, huma.Error500InternalServerError("Failed to save mount configuration")
	}

	var conv converter.DtoToDbomConverterImpl
	updatedDto := dto.MountPointData{}
	conv.MountPointPathToMountPointData(*dbMountData, &updatedDto) // Error handling for conversion can be added if complex

	self.dirtyservice.SetDirtyVolumes()
	return &struct{ Body dto.MountPointData }{Body: updatedDto}, nil
}
