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

	"github.com/dianlight/srat/lsblk"
	"github.com/gorilla/mux"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	"github.com/jinzhu/copier"
	"github.com/kr/pretty"
	"github.com/pilebones/go-udev/netlink"
	"github.com/u-root/u-root/pkg/mount"
	ublock "github.com/u-root/u-root/pkg/mount/block"
	"golang.org/x/sys/unix"
)

var invalidCharactere = regexp.MustCompile(`[^a-zA-Z0-9-]`)
var extractDeviceName = regexp.MustCompile(`/dev/(\w+)\d+`)
var extractBlockName = regexp.MustCompile(`/dev/(\w+\d+)`)

var (
	volumesQueue      = map[string](chan *block.Info){}
	volumesQueueMutex = sync.RWMutex{}
)

func GetVolumesData() (*block.Info, error) {
	blockInfo, err := ghw.Block()

	if err == nil {
		for _, v := range blockInfo.Disks {
			if len(v.Partitions) == 0 {

				fs, _, err := mount.FSFromBlock("/dev/" + v.Name)
				if err != nil {
					log.Printf("Error getting filesystem for device /dev/%s: %v", v.Name, err)
					continue
				}

				rblock, errb := ublock.Device(v.Name)
				if errb != nil {
					log.Printf("Error getting block device for device /dev/%s: %v", v.Name, errb)
					continue
				}

				label, partlabel, mountpoint, err := lsblk.GetLabelsFromDevice(v.Name)
				if err != nil {
					log.Printf("GetLabelsFromDevice failed: %v", err)
					continue
				}
				if *label == "" && *partlabel == "" {
					*label = "unknown"
					*partlabel = "unknown"
				} else if *label == "" {
					*label = "unknown"
				} else if *partlabel == "" {
					*partlabel = *label
				}

				var partition = &block.Partition{
					Disk:            v,
					Name:            v.Name,
					Label:           *partlabel,
					FilesystemLabel: *label,
					UUID:            rblock.FsUUID,
					SizeBytes:       v.SizeBytes,
					Type:            fs,
					MountPoint:      *mountpoint,
					IsReadOnly:      false,
				}
				v.Partitions = append(v.Partitions, partition)

			} else {
				for _, d := range v.Partitions {
					if d.Label == "unknown" && d.FilesystemLabel == "unknown" {
						d.Label = d.UUID
						label, partlabel, _, err := lsblk.GetLabelsFromDevice(d.Name)
						if err == nil && (*partlabel != "" || *label != "") {
							d.FilesystemLabel = *label
							d.Label = *partlabel
						}
					}
					if d.Label == "unknown" || d.Label == "" {
						d.Label = d.FilesystemLabel
					}
				}
			}
		}
	}
	return blockInfo, err
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

type MounDataFlag int

const (
	MS_RDONLY                   MounDataFlag = unix.MS_RDONLY
	MS_BIND                     MounDataFlag = unix.MS_BIND
	MS_LAZYTIME                 MounDataFlag = unix.MS_LAZYTIME
	MS_NOEXEC                   MounDataFlag = unix.MS_NOEXEC
	MS_NOSUID                   MounDataFlag = unix.MS_NOSUID
	MS_NOUSER                   MounDataFlag = unix.MS_NOUSER
	MS_RELATIME                 MounDataFlag = unix.MS_RELATIME
	MS_SYNC                     MounDataFlag = unix.MS_SYNC
	MS_NOATIME                  MounDataFlag = unix.MS_NOATIME
	ReadOnlyMountPoindDataFlags MounDataFlag = unix.MS_RDONLY | unix.MS_NOATIME
)

var MounDataFlags = []MounDataFlag{
	MS_RDONLY,
	MS_BIND,
	MS_LAZYTIME,
	MS_NOEXEC,
	MS_NOSUID,
	MS_NOUSER,
	MS_RELATIME,
	MS_SYNC,
	MS_NOATIME,
}

type MountPointData struct {
	Path   string         `json:"path"`
	Device string         `json:"device"`
	FSType string         `json:"fstype"`
	Flags  []MounDataFlag `json:"flags"`
	Data   string         `json:"data"`
}

// MountVolume godoc
//
//	@Summary		mount an existing volume
//	@Description	mount an existing volume
//	@Tags			volume
//	@Accept			json
//	@Produce		json
//	@Param			volume_label	path		string			true	"Volume Label to Mount"
//	@Param			mount_data		body		MountPointData	true	"Mount data"
//	@Success		201				{object}	MountPointData
//	@Failure		400				{object}	ResponseError
//	@Failure		405				{object}	ResponseError
//	@Failure		409				{object}	ResponseError
//	@Failure		500				{object}	ResponseError
//	@Router			/volume/{volume_label}/mount [post]
func mountVolume(w http.ResponseWriter, r *http.Request) {
	//volume_label := mux.Vars(r)["volume_label"]
	w.Header().Set("Content-Type", "application/json")

	var mount_data MountPointData

	err := json.NewDecoder(r.Body).Decode(&mount_data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var flags = 0
	for _, flag := range mount_data.Flags {
		flags |= int(flag)
	}

	if mp, err := mount.Mount(mount_data.Device, mount_data.Path, mount_data.FSType, mount_data.Data, uintptr(flags), func() error { return os.MkdirAll(mount_data.Path, 0o666) }); err != nil {
		log.Printf("TryMount(%s) = %v, want nil", mount_data.Device, err)
		DoResponseError(http.StatusConflict, w, "Error Mounting volume", err)
		return
	} else {
		mounted_data := MountPointData{}
		copier.Copy(&mounted_data, mp)
		for _, flags := range MounDataFlags {
			if mp.Flags&uintptr(flags) != 0 {
				mounted_data.Flags = append(mounted_data.Flags, flags)
			}
		}

		var data, _ = GetVolumesData()
		notifyVolumeClient(data)

		jsonResponse, jsonError := json.Marshal(mounted_data)

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

// notifyVolumeClient sends updated volume information to all registered clients.
//
// This function iterates through all registered volume queues and sends the
// provided volume information to each of them. It uses a read lock to ensure
// thread-safe access to the shared volumesQueue.
//
// Parameters:
//   - volumes: A pointer to block.Info containing the updated volume information
//     to be sent to all clients.
//
// The function does not return any value.
func notifyVolumeClient(volumes *block.Info) {
	volumesQueueMutex.RLock()
	for _, v := range volumesQueue {
		v <- volumes
	}
	volumesQueueMutex.RUnlock()
}

// UmountVolume godoc
//
//	@Summary		Umount the selected volume
//	@Description	Umount the selected volume
//	@Tags			volume
//	@Param			volume_label	path	string	true	"Label of the volume to be unmounted"
//	@Param			force			query	bool	true	"Umount forcefully - forces an unmount regardless of currently open or otherwise used files within the file system to be unmounted."
//	@Param			lazy			query	bool	true	"Umount lazily - disallows future uses of any files below path -- i.e. it hides the file system mounted at path, but the file system itself is still active and any currently open files can continue to be used. When all references to files from this file system are gone, the file system will actually be unmounted."
//	@Success		204
//	@Failure		404	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/volume/{volume_label}/mount [delete]
func umountVolume(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	volume := mux.Vars(r)["volume_label"]
	force := r.URL.Query().Get("force")
	lazy := r.URL.Query().Get("lazy")

	volumes, err := GetVolumesData()
	if err != nil {
		DoResponseError(http.StatusInternalServerError, w, "Error fetching volumes", err)
		return
	}

	for _, v := range volumes.Disks {
		for _, d := range v.Partitions {
			if d.Label == volume {
				err := mount.Unmount(d.MountPoint, force == "true", lazy == "true")
				if err != nil {
					DoResponseError(http.StatusInternalServerError, w, fmt.Sprintf("Error unmounting %s", d.MountPoint), err)
					return
				}
				var data, _ = GetVolumesData()
				notifyVolumeClient(data)

				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
	}
	DoResponseError(http.StatusNotFound, w, fmt.Sprintf("No mount on %s found!", volume), "")
}

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
