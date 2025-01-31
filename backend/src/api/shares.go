package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

type ShareHandler struct {
	ctx              context.Context
	sharesQueueMutex sync.RWMutex
	//sharesQueue      map[string]chan *[]dto.SharedResource
}

func NewShareHandler(ctx context.Context) *ShareHandler {
	p := new(ShareHandler)
	p.ctx = ctx
	//p.sharesQueue = map[string](chan *[]dto.SharedResource){}
	p.sharesQueueMutex = sync.RWMutex{}
	StateFromContext(p.ctx).SSEBroker.AddOpenConnectionListener(func(broker BrokerInterface) error {
		p.notifyClient()
		return nil
	})
	return p
}

/*
func GetSharedResources(ctx context.Context) (*dto.SharedResources, error) {
	var shares dto.SharedResources
	var dbshares dbom.ExportedShare
	err := dbshares.GetAll()
	if err!= nil {

        return nil, err
    }
	shares.From(dbshares)

	// Check all ID Matching and sign dirty!
	for _, sdto := range shares {
		if sdto.ID == nil {
			ckdb := dbom.ExportedShare{}
			err := ckdb.FromNameOrPath(sdto.Name, sdto.Path)
			if err != nil {
				return nil, err
			}
			if ckdb.Name != "" {
				sdto.ID = &ckdb.ID
				sdto.DirtyStatus = true
			}
			shares[sdto.Name] = sdto
		} else {
			dbshare := dbom.ExportedShare{}
			dbshare.ID = *sdto.ID
			err := dbshare.Get()
			if err != nil {
				return nil, err
			}
			nsdto := dto.SharedResource{}
			nsdto.From(dbshare)

			if !reflect.DeepEqual(nsdto, sdto) {
				sdto.DirtyStatus = true
				shares[sdto.Name] = sdto
			}
		}
		// Check if mounted and wich path

		if sdto.DeviceId == nil {
			sstat := syscall.Stat_t{}
			err := syscall.Stat(sdto.Path, &sstat)
			if err != nil {
				// check if error is not such file or directory
				if os.IsNotExist(err) {
					sdto.Invalid = true
				} else {
					return nil, err
				}
			} else {
				sdto.DeviceId = &sstat.Dev
			}
		}
		shares[sdto.Name] = sdto
	}

	// TODO: Popolate missing share and set to delete!

	return &shares, nil
}
*/

// ListShares godoc
//
//	@Summary		List all configured shares
//	@Description	List all configured shares
//	@Tags			share
//	@Produce		json
//	@Success		200	{object}	[]dto.SharedResource
//	@Failure		405	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/shares [get]
func (self *ShareHandler) ListShares(w http.ResponseWriter, r *http.Request) {
	var shares []dto.SharedResource
	var dbshares dbom.ExportedShares
	err := dbshares.Load()
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

// GetShare godoc
//
//	@Summary		Get a share
//	@Description	get share by Name
//	@Tags			share
//	@Produce		json
//	@Param			share_name	path		string	true	"Name"
//	@Success		200			{object}	dto.SharedResource
//	@Failure		405			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/share/{share_name} [get]
func (self *ShareHandler) GetShare(w http.ResponseWriter, r *http.Request) {
	shareName := mux.Vars(r)["share_name"]

	dbshare := dbom.ExportedShare{Name: shareName}
	err := dbshare.FromName(shareName)
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
	err = conv.ExportedShareToSharedResource(dbshare, &share)
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
//	@Failure		400		{object}	ErrorResponse
//	@Failure		405		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
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
	err = dbshare.Save()
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

	//context_state := (&dto.Status{}).FromContext(r.Context())
	context_state := StateFromContext(r.Context())
	err = conv.ExportedShareToSharedResource(*dbshare, &share)
	//err = mapper.Map(context.Background(), &share, dbshare)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	context_state.DataDirtyTracker.Shares = true
	go self.notifyClient()
	HttpJSONReponse(w, share, &Options{
		Code: http.StatusCreated,
	})
}

func (self *ShareHandler) notifyClient() {
	self.sharesQueueMutex.RLock()
	defer self.sharesQueueMutex.RUnlock()
	var shares []dto.SharedResource
	var dbshares = dbom.ExportedShares{}
	err := dbshares.Load()
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

	var event dto.EventMessageEnvelope
	event.Event = dto.EventShare
	event.Data = shares
	StateFromContext(self.ctx).SSEBroker.BroadcastMessage(&event)
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
//	@Failure		400			{object}	ErrorResponse
//	@Failure		405			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/share/{share_name} [put]
func (self *ShareHandler) UpdateShare(w http.ResponseWriter, r *http.Request) {
	share_name := mux.Vars(r)["share_name"]

	var share dto.SharedResource
	err := HttpJSONRequest(&share, w, r)
	if err != nil {
		return
	}

	dbshare := &dbom.ExportedShare{
		Name: share_name,
	}
	err = dbshare.FromName(share_name)
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
	err = dbshare.Save()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	context_state := StateFromContext(r.Context())
	context_state.DataDirtyTracker.Shares = true
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
//	@Failure		400	{object}	ErrorResponse
//	@Failure		405	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/share/{share_name} [delete]
func (self *ShareHandler) DeleteShare(w http.ResponseWriter, r *http.Request) {
	share_name := mux.Vars(r)["share_name"]

	var share dto.SharedResource
	dbshare := &dbom.ExportedShare{
		Name: share_name,
	}
	err := dbshare.FromName(share_name)
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
	err = dbshare.Delete()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	context_state := StateFromContext(r.Context())
	context_state.DataDirtyTracker.Shares = true
	go self.notifyClient()
	HttpJSONReponse(w, nil, &Options{
		Code: http.StatusNoContent,
	})
}
