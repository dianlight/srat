package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"

	"dario.cat/mergo"
	"github.com/dianlight/srat/lsblk"
	"github.com/gorilla/mux"
	"github.com/shirou/gopsutil/v4/disk"
)

var invalidCharactere = regexp.MustCompile(`[[:^ascii:]]|\W`)
var extractDeviceName = regexp.MustCompile(`/dev/(\w+)\d+`)

var (
	volumesQueue      = map[string](chan *[]Volume){}
	volumesQueueMutex = sync.RWMutex{}
)

type Volume struct {
	Label        string         `json:"label"`
	SerialNumber string         `json:"serial_number"`
	Stats        disk.UsageStat `json:"stats"`
	Lsbk         lsblk.Device   `json:"lsbk"`
	disk.PartitionStat
	// IOStats disk.IOCountersStat `json:"io_stats"`
}

// ListVolumes godoc
//
//	@Summary		List all available volumes
//	@Description	List all available volumes
//	@Tags			volume
//	@Produce		json
//	@Success		200	{object}	[]Volume
//	@Failure		405	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/volumes [get]
func listVolumes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_partitions, err := disk.Partitions(false)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	// Falbak Mode - Get also lsblk data!
	_devices, errd := lsblk.ListDevices()
	if errd != nil {
		log.Println("lsblk not available", errd)
	}
	//log.Println(_devices)

	var partitions = make([]Volume, 0)

	for _, partition := range _partitions {
		volume := Volume{PartitionStat: partition}
		stats, e1 := disk.Usage(partition.Mountpoint)
		if e1 != nil {
			log.Println(e1)
		} else {
			mergo.Merge(&volume, &Volume{Stats: *stats})
		}
		device, e12 := _devices[extractDeviceName.FindStringSubmatch(partition.Device)[1]]
		if !e12 {
			log.Printf("Unmapped device %s", extractDeviceName.FindStringSubmatch(partition.Device))
		} else {
			child := slices.IndexFunc(device.Children, func(a lsblk.Device) bool {
				return a.Name == strings.TrimPrefix(partition.Device, "/dev/")
			})
			if child == -1 {
				log.Printf("Unmapped child device %s of %s", device.Name, strings.TrimPrefix(partition.Device, "/dev/"))
			} else {
				volume.Lsbk = device.Children[child]
			}
		}
		//IOStats, e2 := disk.IOCounters( /*partition.Device*/ )
		//if e2 != nil {
		//	log.Println(e2)
		//} else {
		//	log.Printf("IOStats %+v", IOStats)
		//		mergo.Merge(&volume, &Volume{IOStats: IOStats[partition.Device]})
		//}

		volumeSerialNumber, e4 := disk.SerialNumber(partition.Device)
		if e4 != nil {
			log.Println("Reading Serial Number:", partition.Device, e4)
			volume.SerialNumber = strings.ToUpper(invalidCharactere.ReplaceAllLiteralString(volume.Device, ""))
		} else if volumeSerialNumber == "" {
			//volumeSerialNumber, e5 := lsblk.getSerial()
			volume.SerialNumber = volume.Lsbk.UUID
		} else {
			volume.SerialNumber = volumeSerialNumber

		}
		volumeLabel, e3 := disk.Label(partition.Device)
		if e3 != nil {
			//log.Println("Reading Label:", partition.Device, e3)
			volume.Label = strings.ToUpper(invalidCharactere.ReplaceAllLiteralString(volume.SerialNumber, ""))
		} else {
			volume.Label = volumeLabel
		}

		partitions = append(partitions, volume)
	}

	/*
			di = DiskInfo()
		disks = di.get_disk_list(sorting=True)
		regex = r"Name:\s+(\w+)\s.*\n"
		for d in disks:
		    if d.get_partition_table_type() == "":
		        continue
		    plist = d.get_partition_list()
		    for item in plist:
		        label = item.get_fs_label()
		        if label.startswith("hassos"):
		            continue
		        elif label != "":
		            print(item.get_fs_label())
		        elif item.get_fs_type() == "apfs":
		            print("id:{uuid}".format(uuid=item.get_fs_uuid()))
		#        print(item.get_fs_label()," ",item.get_fs_type()," ",item.get_part_uuid()," ",item.get_name())

	*/

	jsonResponse, jsonError := json.Marshal(partitions)

	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}

}

// GetVolume godoc
//
//	@Summary		Get a volume
//	@Description	get a volume by Name
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
func getVolume(w http.ResponseWriter, r *http.Request) {
	share := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	data, ok := config.Shares[share]
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

func notifyVolumeClient() {
	sharesQueueMutex.RLock()
	for _, v := range sharesQueue {
		v <- &config.Shares
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

func VolumesWsHandler(request WebSocketMessageEnvelope, c chan *WebSocketMessageEnvelope) {
	sharesQueueMutex.Lock()
	if sharesQueue[request.Uid] == nil {
		sharesQueue[request.Uid] = make(chan *Shares, 10)
	}
	sharesQueue[request.Uid] <- &config.Shares
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
