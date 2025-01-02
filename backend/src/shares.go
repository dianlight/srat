package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"dario.cat/mergo"
	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/data"
	"github.com/gorilla/mux"
)

var (
	sharesQueue      = map[string](chan *config.Shares){}
	sharesQueueMutex = sync.RWMutex{}
)

// ListShares godoc
//
//	@Summary		List all configured shares
//	@Description	List all configured shares
//	@Tags			share
//	@Produce		json
//	@Success		200	{object}	config.Shares
//	@Failure		405	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/shares [get]
func listShares(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	DoResponse(http.StatusOK, w, data.Config.Shares)
	//	   DoResponse(http.StatusOK, w, config.ExportedShare{}.All())
}

// GetShare godoc
//
//	@Summary		Get a share
//	@Description	get share by Name
//	@Tags			share
//	@Produce		json
//	@Param			share_name	path		string	true	"Name"
//	@Success		200			{object}	config.Share
//	@Failure		405			{object}	dto.ResponseError
//	@Failure		500			{object}	dto.ResponseError
//	@Router			/share/{share_name} [get]
func getShare(w http.ResponseWriter, r *http.Request) {
	shareName := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	data, ok := data.Config.Shares[shareName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		jsonResponse, jsonError := json.Marshal(data)

		if jsonError != nil {
			fmt.Println("Unable to encode JSON")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(jsonError.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(jsonResponse)
		}

	}

}

// CreateShare godoc
//
//	@Summary		Create a share
//	@Description	create e new share
//	@Tags			share
//	@Accept			json
//	@Produce		json
//	@Param			share	body		config.Share	true	"Create model"
//	@Success		201		{object}	config.Share
//	@Failure		400		{object}	dto.ResponseError
//	@Failure		405		{object}	dto.ResponseError
//	@Failure		409		{object}	dto.ResponseError
//	@Failure		500		{object}	dto.ResponseError
//	@Router			/share [post]
func createShare(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var share config.Share

	err := json.NewDecoder(r.Body).Decode(&share)
	if err != nil {
		DoResponseError(http.StatusBadRequest, w, "Invalid JSON", err)
		return
	}

	if share.Name == "" {
		DoResponseError(http.StatusBadRequest, w, "No Name in data", nil)
		return
	}

	fshare, ok := data.Config.Shares[share.Name]
	if ok {
		DoResponseError(http.StatusConflict, w, "Share already exists", fshare)
	} else {

		data.Config.Shares[share.Name] = share
		data.DirtySectionState.Shares = true

		notifyClient()
		DoResponse(http.StatusCreated, w, share)
	}
}

// notifyClient sends the current shares configuration to all connected clients.
// It iterates through the sharesQueue, sending the Config.Shares to each client's channel.
// This function is used to broadcast updates to all clients when the shares configuration changes.
// The function uses a read lock to ensure thread-safe access to the sharesQueue.
func notifyClient() {
	sharesQueueMutex.RLock()
	for _, v := range sharesQueue {
		v <- &data.Config.Shares
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
//	@Param			share_name	path		string			true	"Name"
//	@Param			share		body		config.Share	true	"Update model"
//	@Success		200			{object}	config.Share
//	@Failure		400			{object}	dto.ResponseError
//	@Failure		405			{object}	dto.ResponseError
//	@Failure		404			{object}	dto.ResponseError
//	@Failure		500			{object}	dto.ResponseError
//	@Router			/share/{share_name} [put]
//	@Router			/share/{share_name} [patch]
func updateShare(w http.ResponseWriter, r *http.Request) {
	share_name := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	adata, ok := data.Config.Shares[share_name]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		var share config.Share

		err := json.NewDecoder(r.Body).Decode(&share)
		if err != nil {
			DoResponseError(http.StatusBadRequest, w, "Invalid JSON", err)
			return
		}

		err2 := mergo.MapWithOverwrite(&adata, share)
		if err2 != nil {
			DoResponseError(http.StatusInternalServerError, w, "Internal error", err2)
			return
		}
		data.Config.Shares[share_name] = adata
		data.DirtySectionState.Shares = true
		notifyClient()
		DoResponse(http.StatusOK, w, adata)
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
func deleteShare(w http.ResponseWriter, r *http.Request) {
	share := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	_, ok := data.Config.Shares[share]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {

		delete(data.Config.Shares, share)
		data.DirtySectionState.Shares = true
		notifyClient()

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
func SharesWsHandler(ctx context.Context, request WebSocketMessageEnvelope, c chan *WebSocketMessageEnvelope) {
	sharesQueueMutex.Lock()
	if sharesQueue[request.Uid] == nil {
		sharesQueue[request.Uid] = make(chan *config.Shares, 10)
	}
	sharesQueue[request.Uid] <- &data.Config.Shares
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
			smessage := &WebSocketMessageEnvelope{
				Event: EventShare,
				Uid:   request.Uid,
				Data:  <-queue,
			}
			c <- smessage
		}
	}
}
