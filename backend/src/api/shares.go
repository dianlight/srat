package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

var (
	sharesQueue      = map[string](chan *dto.SharedResources){}
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
//	@Success		200	{object}	dto.SharedResources
//	@Failure		405	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/shares [get]
func ListShares(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var shares dto.SharedResources
	var dbshares dbom.ExportedShares
	err := dbshares.Load()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	err = shares.From(dbshares)
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err.Error())
		return
	}
	shares.ToResponse(http.StatusOK, w)
}

// GetShare godoc
//
//	@Summary		Get a share
//	@Description	get share by Name
//	@Tags			share
//	@Produce		json
//	@Param			share_name	path		string	true	"Name"
//	@Success		200			{object}	dto.SharedResource
//	@Failure		405			{object}	dto.ResponseError
//	@Failure		500			{object}	dto.ResponseError
//	@Router			/share/{share_name} [get]
func GetShare(w http.ResponseWriter, r *http.Request) {
	shareName := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	dbshare := dbom.ExportedShare{Name: shareName}
	err := dbshare.Get()
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		dto.ResponseError{}.ToResponseError(http.StatusNotFound, w, "Share not found", nil)
	} else if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	share := dto.SharedResource{}
	err = share.From(dbshare)
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}

	share.ToResponse(http.StatusOK, w)

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
//	@Failure		400		{object}	dto.ResponseError
//	@Failure		405		{object}	dto.ResponseError
//	@Failure		409		{object}	dto.ResponseError
//	@Failure		500		{object}	dto.ResponseError
//	@Router			/share [post]
func CreateShare(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var share dto.SharedResource
	share.FromJSONBody(w, r)

	dbshare := &dbom.ExportedShare{
		//		Model: gorm.Model{ID: *share.ID},
		Name: share.Name,
	}
	err := dbshare.Get()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		dto.ResponseError{}.ToResponseError(http.StatusConflict, w, "Share already exists", dbshare)
		return
	} else if err == nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}

	share.To(dbshare)
	err = dbshare.Save()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	share.From(dbshare)
	context_state.DataDirtyTracker.Shares = true
	var shares dto.SharedResources
	err = shares.From(dbshare)
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	notifyClient(&shares)
	share.ToResponse(http.StatusCreated, w)
}

// notifyClient sends the current shares configuration to all connected clients.
// It iterates through the sharesQueue, sending the Config.Shares to each client's channel.
// This function is used to broadcast updates to all clients when the shares configuration changes.
// The function uses a read lock to ensure thread-safe access to the sharesQueue.
func notifyClient(in *dto.SharedResources) {
	sharesQueueMutex.RLock()
	for _, v := range sharesQueue {
		v <- in
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
//	@Failure		400			{object}	dto.ResponseError
//	@Failure		405			{object}	dto.ResponseError
//	@Failure		404			{object}	dto.ResponseError
//	@Failure		500			{object}	dto.ResponseError
//	@Router			/share/{share_name} [put]
func UpdateShare(w http.ResponseWriter, r *http.Request) {
	share_name := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	var share dto.SharedResource
	share.FromIgnoreEmpty(share)

	dbshare := &dbom.ExportedShare{
		Model: gorm.Model{ID: *share.ID},
		Name:  share_name,
	}
	err := dbshare.Get()
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		dto.ResponseError{}.ToResponseError(http.StatusNotFound, w, "Share not exists", dbshare)
		return
	} else if err == nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	share.ToIgnoreEmpty(dbshare)
	err = dbshare.Save()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	context_state.DataDirtyTracker.Shares = true
	var shares dto.SharedResources
	err = shares.From(dbshare)
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	notifyClient(&shares)
	share.ToResponse(http.StatusCreated, w)
}

// DeleteShare godoc
//
//	@Summary		Delere a share
//	@Description	delere a share
//	@Tags			share
//	@Param			share_name	path	string	true	"Name"
//	@Success		204
//	@Failure		400	{object}	dto.ResponseError
//	@Failure		405	{object}	dto.ResponseError
//	@Failure		404	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/share/{share_name} [delete]
func DeleteShare(w http.ResponseWriter, r *http.Request) {
	share_name := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	var share dto.SharedResource
	share.FromIgnoreEmpty(share)

	dbshare := &dbom.ExportedShare{
		Model: gorm.Model{ID: *share.ID},
		Name:  share_name,
	}
	err := dbshare.Get()
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		dto.ResponseError{}.ToResponseError(http.StatusNotFound, w, "Share not exists", dbshare)
		return
	} else if err == nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	share.ToIgnoreEmpty(dbshare)
	err = dbshare.Delete()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}

	context_state := (&dto.ContextState{}).FromContext(r.Context())
	context_state.DataDirtyTracker.Shares = true
	var shares dto.SharedResources
	err = shares.From(dbshare)
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}
	notifyClient(&shares)
	share.ToResponse(http.StatusCreated, w)
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
		sharesQueue[request.Uid] = make(chan *dto.SharedResources, 10)
	}
	var dbshare dbom.ExportedShare
	var shares dto.SharedResources
	err := shares.From(dbshare)
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
