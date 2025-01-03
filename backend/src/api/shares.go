package api

import (
	"context"
	"net/http"
	"os"
	"reflect"
	"sync"
	"syscall"

	"dario.cat/mergo"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/gorilla/mux"
)

var (
	sharesQueue      = map[string](chan *dto.SharedResources){}
	sharesQueueMutex = sync.RWMutex{}
)

func GetSharedResources(ctx context.Context) (*dto.SharedResources, error) {

	var shares dto.SharedResources
	addon_config := ctx.Value("addon_config").(*config.Config)
	err := shares.From(addon_config.Shares)
	if err != nil {
		return nil, err
	}

	// Check all ID Matching and sign dirty!
	for _, sdto := range shares {
		if sdto.ID == nil {
			ckdb := dbom.ExportedShare{}
			err = ckdb.FromNameOrPath(sdto.Name, sdto.Path)
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
			err = dbshare.Get()
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

	shares, err := GetSharedResources(r.Context())
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

	shares, err := GetSharedResources(r.Context())
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
		return
	}

	data, ok := shares.Get(shareName)
	if !ok {
		dto.ResponseError{}.ToResponseError(http.StatusNotFound, w, "Share not found", nil)
	} else {
		var share dto.SharedResource
		share.From(data)
		share.ToResponse(http.StatusOK, w)
	}

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

	addon_config := r.Context().Value("addon_config").(*config.Config)
	fshare, ok := addon_config.Shares[share.Name]
	if ok {
		dto.ResponseError{}.ToResponseError(http.StatusConflict, w, "Share already exists", fshare)
	} else {
		var share2 config.Share
		share.To(&share2)
		data_dirty_tracker := r.Context().Value("data_dirty_tracker").(*dto.DataDirtyTracker)
		data_dirty_tracker.Shares = true
		notifyClient(&addon_config.Shares)
		share.ToResponse(http.StatusCreated, w)
	}
}

// notifyClient sends the current shares configuration to all connected clients.
// It iterates through the sharesQueue, sending the Config.Shares to each client's channel.
// This function is used to broadcast updates to all clients when the shares configuration changes.
// The function uses a read lock to ensure thread-safe access to the sharesQueue.
func notifyClient(in *config.Shares) {
	var in2 dto.SharedResources
	in2.From(in)
	sharesQueueMutex.RLock()
	for _, v := range sharesQueue {
		v <- &in2
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
//	@Router			/share/{share_name} [patch]
func UpdateShare(w http.ResponseWriter, r *http.Request) {
	share_name := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	addon_config := r.Context().Value("addon_config").(*config.Config)

	adata, ok := addon_config.Shares[share_name]
	if !ok {
		dto.ResponseError{}.ToResponseError(http.StatusNotFound, w, "Share not found", nil)
	} else {
		var share dto.SharedResource
		share.FromJSONBody(w, r)
		var cshare dto.SharedResource
		cshare.From(adata)

		err2 := mergo.MapWithOverwrite(&cshare, share)
		if err2 != nil {
			dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err2)
			return
		}
		err := cshare.To(&adata)
		if err != nil {
			dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Internal error", err)
			return
		}
		addon_config.Shares[share_name] = adata
		data_dirty_tracker := r.Context().Value("data_dirty_tracker").(*dto.DataDirtyTracker)
		data_dirty_tracker.Shares = true
		notifyClient(&addon_config.Shares)
		//	log.Println(pretty.Sprint(cshare, share))
		cshare.ToResponse(http.StatusOK, w)
	}
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
	share := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	addon_config := r.Context().Value("addon_config").(*config.Config)

	_, ok := addon_config.Shares[share]
	if !ok {
		dto.ResponseError{}.ToResponseError(http.StatusNotFound, w, "Share not found", nil)
	} else {

		delete(addon_config.Shares, share)
		data_dirty_tracker := r.Context().Value("data_dirty_tracker").(*dto.DataDirtyTracker)
		data_dirty_tracker.Shares = true
		notifyClient(&addon_config.Shares)
		w.WriteHeader(http.StatusNoContent)
	}

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
	addon_config := ctx.Value("addon_config").(*config.Config)
	var currentShares dto.SharedResources
	currentShares.From(&addon_config.Shares)
	sharesQueue[request.Uid] <- &currentShares

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
