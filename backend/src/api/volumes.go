package api

import (
	"errors"
	"regexp"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
	"github.com/ztrue/tracerr"
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

func (p *VolumeHandler) Routers(srv *fuego.Server) error {
	fuego.Get(srv, "/volumes", p.ListVolumes, option.Description("List all available volumes"), option.Tags("volume"))
	fuego.Post(srv, "/volume/{id}/mount", p.MountVolume, option.Description("mount an existing volume"), option.Tags("volume"))
	fuego.Delete(srv, "/volume/{id}/mount", p.UmountVolume, option.Description("Umount the selected volume"),
		option.Tags("volume"),
		option.QueryBool("force", "Force Umount"),
		option.QueryBool("lazy", "Lazy Umount "))
	return nil
}

func (self *VolumeHandler) ListVolumes(c fuego.ContextNoBody) (*dto.BlockInfo, error) {

	volumes, err := self.vservice.GetVolumesData()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	return volumes, nil
}

func (self *VolumeHandler) MountVolume(c fuego.ContextWithBody[dto.MountPointData]) (*dto.MountPointData, error) {
	mount_data, err := c.Body()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	id, err := c.PathParamIntErr("id")
	if err != nil {
		return nil, fuego.PathParamInvalidTypeError{
			Err:          tracerr.Wrap(err),
			ParamName:    "id",
			ExpectedType: "uint",
		}
	}

	if mount_data.ID != 0 && mount_data.ID != uint(id) {
		return nil, fuego.PathParamInvalidTypeError{
			Err:          tracerr.Errorf("Inconsistent ID provided in the request"),
			ParamName:    "id",
			ExpectedType: "uint",
		}
	}

	mount_data.ID = uint(id)

	err = self.vservice.MountVolume(mount_data)
	if err != nil {
		if einfo, ok := err.(*dto.ErrorInfo); ok {
			switch einfo.Code {
			case dto.ErrorCodes.INVALID_PARAMETER:
				return nil, fuego.BadRequestError{}
			case dto.ErrorCodes.DEVICE_NOT_FOUND:
				return nil, fuego.NotFoundError{}
			case dto.ErrorCodes.MOUNT_FAIL:
				return nil, fuego.NotAcceptableError{
					Title: einfo.Data["Message"].(string),
				}
			default:
				return nil, tracerr.Wrap(einfo)
			}
		}
		return nil, tracerr.Wrap(err)
	}

	dbom_mount_data, err := self.mount_repo.FindByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fuego.NotFoundError{
				Err: dto.NewErrorInfo(dto.ErrorCodes.MOUNT_FAIL, map[string]any{
					"Device":  dbom_mount_data.Source,
					"Path":    dbom_mount_data.Path,
					"Message": "Mount Record Not Found",
				}, err),
			}
		}
		return nil, tracerr.Wrap(err)
	}
	var conv converter.DtoToDbomConverterImpl
	mounted_data := dto.MountPointData{}
	err = conv.MountPointPathToMountPointData(*dbom_mount_data, &mounted_data)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	self.dirtyservice.SetDirtyVolumes()
	return &mounted_data, nil
}

func (self *VolumeHandler) UmountVolume(c fuego.ContextNoBody) (bool, error) {
	id, err := c.PathParamIntErr("id")
	if err != nil {
		return false, fuego.PathParamInvalidTypeError{
			Err:          tracerr.Wrap(err),
			ParamName:    "id",
			ExpectedType: "uint",
		}
	}

	force := c.QueryParamBool("force")
	lazy := c.QueryParamBool("lazy")

	if ok, _ := self.mount_repo.Exists(uint(id)); !ok {
		return false, fuego.NotFoundError{
			Err:   tracerr.Errorf("No mount point found for the provided ID"),
			Title: "No mount point found for the provided ID",
		}
	}

	err = self.vservice.UnmountVolume(uint(id), force, lazy)
	if err != nil {
		return false, tracerr.Wrap(err)
	}

	self.dirtyservice.SetDirtyVolumes()

	return true, nil
}
