package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"
	"github.com/gorilla/mux"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

type ShareHandler struct {
	sharesQueueMutex    sync.RWMutex
	broadcaster         service.BroadcasterServiceInterface
	apiContext          *dto.ContextState
	dirtyservice        service.DirtyDataServiceInterface
	exported_share_repo repository.ExportedShareRepositoryInterface
}

func NewShareHandler(broadcaster service.BroadcasterServiceInterface,
	apiContext *dto.ContextState,
	dirtyService service.DirtyDataServiceInterface,
	exported_share_repo repository.ExportedShareRepositoryInterface,
) *ShareHandler {
	p := new(ShareHandler)
	p.broadcaster = broadcaster
	p.apiContext = apiContext
	p.dirtyservice = dirtyService
	p.exported_share_repo = exported_share_repo
	p.sharesQueueMutex = sync.RWMutex{}
	return p
}

func (broker *ShareHandler) Patterns() []server.RouteDetail {
	return []server.RouteDetail{
		{Pattern: "/shares", Method: "GET", Handler: broker.ListShares},
		{Pattern: "/shares/usages", Method: "GET", Handler: broker.ListShareUsages},
		{Pattern: "/share/{share_name}", Method: "GET", Handler: broker.GetShare},
		{Pattern: "/share", Method: "POST", Handler: broker.CreateShare},
		{Pattern: "/share/{share_name}", Method: "PUT", Handler: broker.UpdateShare},
		{Pattern: "/share/{share_name}", Method: "DELETE", Handler: broker.DeleteShare},
	}
}

// ListShares godoc
//
//	@Summary		List all configured shares
//	@Description	List all configured shares
//	@Tags			share
//	@Produce		json
//	@Success		200	{object}	[]dto.SharedResource
//	@Failure		405	{object}	dto.ErrorInfo
//	@Failure		500	{object}	dto.ErrorInfo
//	@Router			/shares [get]
func (self *ShareHandler) ListShares(w http.ResponseWriter, r *http.Request) {
	var shares []dto.SharedResource
	var dbshares []dbom.ExportedShare
	err := self.exported_share_repo.All(&dbshares)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var conv converter.DtoToDbomConverterImpl
	for _, dbshare := range dbshares {
		var share dto.SharedResource
		err = conv.ExportedShareToSharedResource(dbshare, &share)
		if err != nil {
			HttpJSONReponse(w, err, nil)
			return
		}
		shares = append(shares, share)
	}
	HttpJSONReponse(w, shares, nil)
}

// ListShareUsages godoc
//
//	@Summary		List all available usages for shares
//	@Description	List all available usages for shares
//	@Tags			share
//	@Produce		json
//	@Success		200	{object}	[]dto.HAMountUsage
//	@Router			/shares/usages [get]
func (self *ShareHandler) ListShareUsages(w http.ResponseWriter, r *http.Request) {
	//HttpJSONReponse(w, dto.HAMountUsages.All(), nil)
}

// GetShare godoc
//
//	@Summary		Get a share
//	@Description	get share by Name
//	@Tags			share
//	@Produce		json
//	@Param			share_name	path		string	true	"Name"
//	@Success		200			{object}	dto.SharedResource
//	@Failure		405			{object}	dto.ErrorInfo
//	@Failure		500			{object}	dto.ErrorInfo
//	@Router			/share/{share_name} [get]
func (self *ShareHandler) GetShare(w http.ResponseWriter, r *http.Request) {
	shareName := mux.Vars(r)["share_name"]

	dbshare, err := self.exported_share_repo.FindByName(shareName)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		HttpJSONReponse(w, fmt.Errorf("Share not found"), &Options{
			Code: http.StatusNotFound,
		})
		return
	} else if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	share := dto.SharedResource{}
	var conv converter.DtoToDbomConverterImpl
	err = conv.ExportedShareToSharedResource(*dbshare, &share)
	//err = mapper.Map(context.Background(), &share, dbshare)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	HttpJSONReponse(w, share, nil)
}

// CreateShare godoc
//
//	@Summary		Create a share
//	@Description	create e new share
//	@Tags			share
//	@Accept			json
//	@Produce		json
//	@Param			share	body		dto.SharedResource	true	"Create model"
//	@Success		201		{object}	dto.SharedResource
//	@Failure		400		{object}	dto.ErrorInfo
//	@Failure		405		{object}	dto.ErrorInfo
//	@Failure		409		{object}	dto.ErrorInfo
//	@Failure		500		{object}	dto.ErrorInfo
//	@Router			/share [post]
func (self *ShareHandler) CreateShare(w http.ResponseWriter, r *http.Request) {

	var share dto.SharedResource
	err := HttpJSONRequest(&share, w, r)
	if err != nil {
		return
	}

	dbshare := &dbom.ExportedShare{
		Name: share.Name,
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.SharedResourceToExportedShare(share, dbshare)
	//err = mapper.Map(context.Background(), &dbshare, share)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	if len(dbshare.Users) == 0 {
		adminUser := dbom.SambaUser{}
		err = adminUser.GetAdmin()
		if err != nil {
			HttpJSONReponse(w, err, nil)
			return
		}
		dbshare.Users = append(dbshare.Users, adminUser)
	}
	err = self.exported_share_repo.Save(dbshare)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			HttpJSONReponse(w, fmt.Errorf("Share already exists"), &Options{
				Code: http.StatusConflict,
			})
			return
		}
		HttpJSONReponse(w, err, nil)
		return
	}

	err = conv.ExportedShareToSharedResource(*dbshare, &share)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	self.dirtyservice.SetDirtyShares()
	go self.notifyClient()
	HttpJSONReponse(w, share, &Options{
		Code: http.StatusCreated,
	})
}

func (self *ShareHandler) notifyClient() {
	self.sharesQueueMutex.RLock()
	defer self.sharesQueueMutex.RUnlock()
	var shares []dto.SharedResource
	var dbshares = []dbom.ExportedShare{}
	err := self.exported_share_repo.All(&dbshares)
	if err != nil {
		log.Fatal(err)
		return
	}
	var conv converter.DtoToDbomConverterImpl
	for _, dbshare := range dbshares {
		var share dto.SharedResource
		err = conv.ExportedShareToSharedResource(dbshare, &share)
		if err != nil {
			log.Fatal(err)
			return
		}
		shares = append(shares, share)
	}
	self.broadcaster.BroadcastMessage(shares)
}

// UpdateShare godoc
//
//	@Summary		Update a share
//	@Description	update e new share
//	@Tags			share
//	@Accept			json
//	@Produce		json
//	@Param			share_name	path		string				true	"Name"
//	@Param			share		body		dto.SharedResource	true	"Update model"
//	@Success		200			{object}	dto.SharedResource
//	@Failure		400			{object}	dto.ErrorInfo
//	@Failure		405			{object}	dto.ErrorInfo
//	@Failure		404			{object}	dto.ErrorInfo
//	@Failure		409			{object}	dto.ErrorInfo
//	@Failure		500			{object}	dto.ErrorInfo
//	@Router			/share/{share_name} [put]
func (self *ShareHandler) UpdateShare(w http.ResponseWriter, r *http.Request) {
	share_name := mux.Vars(r)["share_name"]

	var share dto.SharedResource
	err := HttpJSONRequest(&share, w, r)
	if err != nil {
		return
	}

	dbshare, err := self.exported_share_repo.FindByName(share_name)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		HttpJSONReponse(w, tracerr.New("Share not found"), &Options{
			Code: http.StatusNotFound,
		})
		return
	} else if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.SharedResourceToExportedShare(share, dbshare)
	//	err = mapper.Map(context.Background(), &dbshare, share)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	if share_name != dbshare.Name {
		err = self.exported_share_repo.UpdateName(share_name, dbshare.Name)
		if err != nil {
			HttpJSONReponse(w, err,
				&Options{
					Code: http.StatusConflict,
				})
			return
		}
	}

	err = self.exported_share_repo.Save(dbshare)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	err = conv.ExportedShareToSharedResource(*dbshare, &share)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	self.dirtyservice.SetDirtyShares()
	go self.notifyClient()
	HttpJSONReponse(w, share, nil)
}

// DeleteShare godoc
//
//	@Summary		Delere a share
//	@Description	delere a share
//	@Tags			share
//	@Param			share_name	path	string	true	"Name"
//	@Success		204
//	@Failure		400	{object}	dto.ErrorInfo
//	@Failure		405	{object}	dto.ErrorInfo
//	@Failure		404	{object}	dto.ErrorInfo
//	@Failure		500	{object}	dto.ErrorInfo
//	@Router			/share/{share_name} [delete]
func (self *ShareHandler) DeleteShare(w http.ResponseWriter, r *http.Request) {
	share_name := mux.Vars(r)["share_name"]

	var share dto.SharedResource
	dbshare, err := self.exported_share_repo.FindByName(share_name)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		HttpJSONReponse(w, fmt.Errorf("Share not found"), &Options{
			Code: http.StatusNotFound,
		})
		return
	} else if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.SharedResourceToExportedShare(share, dbshare)
	//err = mapper.Map(context.Background(), &dbshare, share)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	err = self.exported_share_repo.Delete(dbshare.Name)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	self.dirtyservice.SetDirtyShares()
	go self.notifyClient()
	HttpJSONReponse(w, nil, &Options{
		Code: http.StatusNoContent,
	})
}
