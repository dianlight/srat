package main

import (
	"encoding/json"
	"fmt"
	"log"
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
//
// _Accept       json
//
//	@Produce		json
//
// _Param        id   path      int  true  "Account ID"
//
//	@Success		200	{object}	Shares
//
// _Failure      400  {object}  ResponseError
//
//	@Failure		405	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/shares [get]
func listShares(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	jsonResponse, jsonError := json.Marshal(data.Config.Shares)

	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}

}

// GetShare godoc
//
//	@Summary		Get a share
//	@Description	get share by Name
//	@Tags			share
//
// _Accept       json
//
//	@Produce		json
//	@Param			share_name	path		string	true	"Name"
//	@Success		200			{object}	Share
//	@Failure		405			{object}	ResponseError
//	@Failure		500			{object}	ResponseError
//	@Router			/share/{share_name} [get]
func getShare(w http.ResponseWriter, r *http.Request) {
	share := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	data, ok := data.Config.Shares[share]
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
//	@Param			share_name	path		string	true	"Name"
//	@Param			share		body		Share	true	"Create model"
//	@Success		201			{object}	Share
//	@Failure		400			{object}	ResponseError
//	@Failure		405			{object}	ResponseError
//	@Failure		409			{object}	ResponseError
//	@Failure		500			{object}	ResponseError
//	@Router			/share/{share_name} [post]
func createShare(w http.ResponseWriter, r *http.Request) {
	share := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	data, ok := data.Config.Shares[share]
	if ok {
		w.WriteHeader(http.StatusConflict)
		jsonResponse, jsonError := json.Marshal(ResponseError{Error: "Share already exists", Body: data})

		if jsonError != nil {
			fmt.Println("Unable to encode JSON")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(jsonError.Error()))
		} else {
			w.Write(jsonResponse)
		}
	} else {
		var share config.Share

		err := json.NewDecoder(r.Body).Decode(&share)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: Create Share

		notifyClient()

		jsonResponse, jsonError := json.Marshal(share)

		if jsonError != nil {
			fmt.Println("Unable to encode JSON")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(jsonError.Error()))
		} else {
			w.WriteHeader(http.StatusCreated)
			w.Write(jsonResponse)
		}

	}
}

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
//	@Param			share_name	path		string	true	"Name"
//	@Param			share		body		Share	true	"Update model"
//	@Success		200			{object}	Share
//	@Failure		400			{object}	ResponseError
//	@Failure		405			{object}	ResponseError
//	@Failure		404			{object}	ResponseError
//	@Failure		500			{object}	ResponseError
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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err2 := mergo.MapWithOverwrite(&adata, share)
		if err2 != nil {
			http.Error(w, err2.Error(), http.StatusInternalServerError)
			return
		}
		data.Config.Shares[share_name] = adata
		notifyClient()

		jsonResponse, jsonError := json.Marshal(adata)

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

// DeleteShare godoc
//
//	@Summary		Delere a share
//	@Description	delere a share
//	@Tags			share
//
// _Accept       json
// _Produce      json
//
//	@Param			share_name	path	string	true	"Name"
//	@Success		204
//	@Failure		400	{object}	ResponseError
//	@Failure		405	{object}	ResponseError
//	@Failure		404	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/share/{share_name} [delete]
func deleteShare(w http.ResponseWriter, r *http.Request) {
	share := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	_, ok := data.Config.Shares[share]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {

		delete(data.Config.Shares, share)

		notifyClient()

		w.WriteHeader(http.StatusNoContent)

	}

}

func SharesWsHandler(request WebSocketMessageEnvelope, c chan *WebSocketMessageEnvelope) {
	sharesQueueMutex.Lock()
	if sharesQueue[request.Uid] == nil {
		sharesQueue[request.Uid] = make(chan *config.Shares, 10)
	}
	sharesQueue[request.Uid] <- &data.Config.Shares
	var queue = sharesQueue[request.Uid]
	sharesQueueMutex.Unlock()
	log.Printf("Handle recv: %s %s %d", request.Event, request.Uid, len(sharesQueue))
	for {
		smessage := &WebSocketMessageEnvelope{
			Event: "shares",
			Uid:   request.Uid,
			Data:  <-queue,
		}
		log.Printf("Handle send: %s %s %d", smessage.Event, smessage.Uid, len(c))
		c <- smessage
	}
}
