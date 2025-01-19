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
	"github.com/dianlight/srat/mapper"
	"github.com/gorilla/mux"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

var (
	sharesQueue      = map[string](chan *[]dto.SharedResource){}
	sharesQueueMutex = sync.RWMutex{}
)

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
func ListShares(w http.ResponseWriter, r *http.Request) {
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
		share.CheckValidity()
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
func GetShare(w http.ResponseWriter, r *http.Request) {
	shareName := mux.Vars(r)["share_name"]

	dbshare := dbom.ExportedShare{Name: shareName}
	err := dbshare.Get()
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		HttpJSONReponse(w, fmt.Errorf("Share not found"), &Options{
			Code: http.StatusNotFound,
		})
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
func CreateShare(w http.ResponseWriter, r *http.Request) {

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

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	err = conv.ExportedShareToSharedResource(*dbshare, &share)
	//err = mapper.Map(context.Background(), &share, dbshare)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	context_state.DataDirtyTracker.Shares = true
	notifyClient()
	HttpJSONReponse(w, share, &Options{
		Code: http.StatusCreated,
	})
}

// notifyClient sends the current shares configuration to all connected clients.
// It iterates through the sharesQueue, sending the Config.Shares to each client's channel.
// This function is used to broadcast updates to all clients when the shares configuration changes.
// The function uses a read lock to ensure thread-safe access to the sharesQueue.
func notifyClient() {
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
	//	err = mapper.Map(context.Background(), &shares, dbshares)
	//	if err != nil {
	//		log.Fatal(err)
	//		return
	//	}
	sharesQueueMutex.RLock()
	for _, v := range sharesQueue {
		v <- &shares
	}
	sharesQueueMutex.RUnlock()
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
func UpdateShare(w http.ResponseWriter, r *http.Request) {
	share_name := mux.Vars(r)["share_name"]

	var share dto.SharedResource
	err := HttpJSONRequest(&share, w, r)
	if err != nil {
		return
	}

	dbshare := &dbom.ExportedShare{
		Name: share_name,
	}
	err = dbshare.Get()
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

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	context_state.DataDirtyTracker.Shares = true
	notifyClient()
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
func DeleteShare(w http.ResponseWriter, r *http.Request) {
	share_name := mux.Vars(r)["share_name"]

	var share dto.SharedResource
	dbshare := &dbom.ExportedShare{
		Name: share_name,
	}
	err := dbshare.Get()
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

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	context_state.DataDirtyTracker.Shares = true
	notifyClient()
	HttpJSONReponse(w, nil, &Options{
		Code: http.StatusNoContent,
	})
}

// SharesWsHandler handles WebSocket connections for share updates.
// It manages a queue for each client and sends share configuration updates.
//
// Parameters:
//   - ctx: The context for handling cancellation and timeouts.
//   - request: The WebSocket message envelope containing client information.
//   - c: A channel for sending WebSocket messages back to the client.
//
// The function doesn't return any value, it runs until the context is cancelled.
func SharesWsHandler(ctx context.Context, request dto.WebSocketMessageEnvelope, c chan *dto.WebSocketMessageEnvelope) {
	sharesQueueMutex.Lock()
	if sharesQueue[request.Uid] == nil {
		sharesQueue[request.Uid] = make(chan *[]dto.SharedResource, 10)
	}
	var dbshare dbom.ExportedShare
	var shares []dto.SharedResource
	err := mapper.Map(context.Background(), &shares, dbshare)
	if err != nil {
		log.Println(err)
		return
	}
	sharesQueue[request.Uid] <- &shares

	var queue = sharesQueue[request.Uid]
	sharesQueueMutex.Unlock()
	for {
		select {
		case <-ctx.Done():
			sharesQueueMutex.Lock()
			delete(sharesQueue, request.Uid)
			sharesQueueMutex.Unlock()
			return
		default:
			smessage := &dto.WebSocketMessageEnvelope{
				Event: dto.EventShare,
				Uid:   request.Uid,
				Data:  <-queue,
			}
			c <- smessage
		}
	}
}
