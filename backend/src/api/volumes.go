package api

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

var invalidCharactere = regexp.MustCompile(`[^a-zA-Z0-9-]`)
var extractDeviceName = regexp.MustCompile(`/dev/(\w+)\d+`)
var extractBlockName = regexp.MustCompile(`/dev/(\w+\d+)`)

type VolumeHandler struct {
	apiContext *ContextState
	vservice   service.VolumeServiceInterface
	mount_repo repository.MountPointPathRepositoryInterface
}

func NewVolumeHandler(vservice service.VolumeServiceInterface, mount_repo repository.MountPointPathRepositoryInterface, apiContext *ContextState) *VolumeHandler {
	p := new(VolumeHandler)
	p.vservice = vservice
	p.mount_repo = mount_repo
	p.apiContext = apiContext
	return p
}

func (broker *VolumeHandler) Patterns() []server.RouteDetail {
	return []server.RouteDetail{
		{Pattern: "/volumes", Method: "GET", Handler: broker.ListVolumes},
		{Pattern: "/volume/{id}/mount", Method: "POST", Handler: broker.MountVolume},
		{Pattern: "/volume/{id}/mount", Method: "DELETE", Handler: broker.UmountVolume},
	}
}

// ListVolumes godoc
//
//	@Summary		List all available volumes
//	@Description	List all available volumes
//	@Tags			volume
//	@Produce		json
//	@Success		200	{object}	dto.BlockInfo
//	@Failure		405	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/volumes [get]
func (self *VolumeHandler) ListVolumes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	volumes, err := self.vservice.GetVolumesData()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	HttpJSONReponse(w, volumes, nil)
}

// MountVolume godoc
//
//	@Summary		mount an existing volume
//	@Description	mount an existing volume
//	@Tags			volume
//	@Accept			json
//	@Produce		json
//	@Param			id			path		uint				true	"id of the mountpoint to be mounted"
//	@Param			mount_data	body		dto.MountPointData	true	"Mount data"
//	@Success		201			{object}	dto.MountPointData
//	@Failure		400			{object}	ErrorResponse
//	@Failure		405			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/volume/{id}/mount [post]
func (self *VolumeHandler) MountVolume(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 32)
	if err != nil {
		HttpJSONReponse(w, ErrorResponse{Error: "Invalid ID", Body: err}, &Options{
			Code: http.StatusBadRequest,
		})
		return
	}

	var mount_data dto.MountPointData
	err = HttpJSONRequest(&mount_data, w, r)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	if mount_data.ID != 0 && mount_data.ID != uint(id) {
		HttpJSONReponse(w, ErrorResponse{Error: "ID conflict", Body: nil}, &Options{
			Code: http.StatusBadRequest,
		})
		return
	}

	mount_data.ID = uint(id)

	err = self.vservice.MountVolume(mount_data)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	dbom_mount_data, err := self.mount_repo.FindByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HttpJSONReponse(w, ErrorResponse{Code: 404, Error: "MountPoint not found", Body: nil}, &Options{
				Code: http.StatusNotFound,
			})
			return
		}
		HttpJSONReponse(w, err, nil)
		return
	}
	var conv converter.DtoToDbomConverterImpl
	mounted_data := dto.MountPointData{}
	err = conv.MountPointPathToMountPointData(*dbom_mount_data, &mounted_data)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	//self.vservice.NotifyClient()
	//		context_state := (&dto.Status{}).FromContext(r.Context())
	//context_state := StateFromContext(r.Context())
	self.apiContext.DataDirtyTracker.Volumes = true
	HttpJSONReponse(w, mounted_data, nil)
	//}
}

// UmountVolume godoc
//
//	@Summary		Umount the selected volume
//	@Description	Umount the selected volume
//	@Tags			volume
//	@Param			id		path	uint	true	"id of the mountpoint to be mounted"
//	@Param			force	query	bool	true	"Umount forcefully - forces an unmount regardless of currently open or otherwise used files within the file system to be unmounted."
//	@Param			lazy	query	bool	true	"Umount lazily - disallows future uses of any files below path -- i.e. it hides the file system mounted at path, but the file system itself is still active and any currently open files can continue to be used. When all references to files from this file system are gone, the file system will actually be unmounted."
//	@Success		204
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/volume/{id}/mount [delete]
func (self *VolumeHandler) UmountVolume(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 32)
	if err != nil {
		HttpJSONReponse(w, ErrorResponse{Code: 500, Error: "Invalid ID", Body: nil}, &Options{
			Code: http.StatusBadRequest,
		})
		return
	}
	force := r.URL.Query().Get("force")
	lazy := r.URL.Query().Get("lazy")

	if !self.mount_repo.Exists(uint(id)) {
		HttpJSONReponse(w, ErrorResponse{Code: 404, Error: "MountPoint not found", Body: nil}, &Options{
			Code: http.StatusNotFound,
		})
		return
	}

	err = self.vservice.UnmountVolume(uint(id), force == "true", lazy == "true")
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	//context_state := StateFromContext(r.Context())
	self.apiContext.DataDirtyTracker.Volumes = true

	HttpJSONReponse(w, nil, nil)
	return
}
