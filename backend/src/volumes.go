package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	"github.com/kr/pretty"
	"github.com/pilebones/go-udev/netlink"
)

var invalidCharactere = regexp.MustCompile(`[^a-zA-Z0-9-]`)
var extractDeviceName = regexp.MustCompile(`/dev/(\w+)\d+`)
var extractBlockName = regexp.MustCompile(`/dev/(\w+\d+)`)

var (
	volumesQueue      = map[string](chan *block.Info){}
	volumesQueueMutex = sync.RWMutex{}
)

func GetVolumesData() (*block.Info, error) {
	return ghw.Block()
}

// ListVolumes godoc
//
//	@Summary		List all available volumes
//	@Description	List all available volumes
//	@Tags			volume
//	@Produce		json
//	@Success		200	{object}	block.Info
//	@Failure		405	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/volumes [get]
func listVolumes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	volumes, err := GetVolumesData()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error fetching volumes: %v", err)))
		return
	}

	jsonResponse, jsonError := json.Marshal(volumes)

	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

/*
// MountVolume godoc
//
//	@Summary		mount an existing volume
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

	func mountVolume(w http.ResponseWriter, r *http.Request) {
		share := mux.Vars(r)["share_name"]
		w.Header().Set("Content-Type", "application/json")

		data, ok := config.Shares[share]
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
			var share Share

			err := json.NewDecoder(r.Body).Decode(&share)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// TODO: Create Share

			notifyVolumeClient()

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
*/
func notifyVolumeClient(volumes *block.Info) {
	volumesQueueMutex.RLock()
	for _, v := range volumesQueue {
		v <- volumes
	}
	volumesQueueMutex.RUnlock()
}

/*
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
func updateVolume(w http.ResponseWriter, r *http.Request) {
	share := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	data, ok := config.Shares[share]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		var share Share

		err := json.NewDecoder(r.Body).Decode(&share)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mergo.Merge(&share, data)

		// TODO: Save share as new data!

		notifyClient()

		jsonResponse, jsonError := json.Marshal(share)

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
func umountVolume(w http.ResponseWriter, r *http.Request) {
	share := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	_, ok := config.Shares[share]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {

		// TODO: Delete share

		notifyClient()

		w.WriteHeader(http.StatusNoContent)

	}

}
*/

func VolumesEventHandler() {
	log.Println("Monitoring UEvent kernel message to user-space...")

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		log.Fatalln("Unable to connect to Netlink Kobject UEvent socket")
	}
	defer conn.Close()

	queue := make(chan netlink.UEvent)
	errors := make(chan error)
	quit := conn.Monitor(queue, errors, nil /*matcher*/)

	// Signal handler to quit properly monitor mode
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-signals
		log.Println("Exiting monitor mode...")
		close(quit)
		// os.Exit(0)
	}()

	// Handling message from queue
	for {
		select {
		case uevent := <-queue:
			log.Println("Handle", pretty.Sprint(uevent))
			if uevent.Action == "add" {
				var data, _ = GetVolumesData()
				notifyVolumeClient(data)
			} else if uevent.Action == "remove" {
				var data, _ = GetVolumesData()
				notifyVolumeClient(data)
			}
		case err := <-errors:
			log.Println("ERROR:", err)
		}
	}

}

func VolumesWsHandler(ctx context.Context, request WebSocketMessageEnvelope, c chan *WebSocketMessageEnvelope) {
	volumesQueueMutex.Lock()
	if volumesQueue[request.Uid] == nil {
		volumesQueue[request.Uid] = make(chan *block.Info, 10)
	}

	var data, err = GetVolumesData()
	if err != nil {
		log.Printf("Unable to fetch volumes: %v", err)
		return
	} else {
		volumesQueue[request.Uid] <- data
		volumesQueueMutex.Unlock()
		log.Printf("Handle recv: %s %s %d", request.Event, request.Uid, len(volumesQueue))
	}
	var queue = volumesQueue[request.Uid]
	go VolumesEventHandler()
	for {
		select {
		case <-ctx.Done():
			volumesQueueMutex.Lock()
			delete(volumesQueue, request.Uid)
			volumesQueueMutex.Unlock()
			return
		default:
			smessage := &WebSocketMessageEnvelope{
				Event: EventVolumes,
				Uid:   request.Uid,
				Data:  <-queue,
			}
			//log.Printf("Handle send: %s %s %d", smessage.Event, smessage.Uid, len(c))
			c <- smessage
		}
	}
}
