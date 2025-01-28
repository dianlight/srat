package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/lsblk"
	"github.com/gorilla/mux"
	"github.com/jaypipes/ghw"
	"github.com/jinzhu/copier"
	"github.com/kr/pretty"
	"github.com/pilebones/go-udev/netlink"
	"github.com/u-root/u-root/pkg/mount"
	ublock "github.com/u-root/u-root/pkg/mount/block"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

var invalidCharactere = regexp.MustCompile(`[^a-zA-Z0-9-]`)
var extractDeviceName = regexp.MustCompile(`/dev/(\w+)\d+`)
var extractBlockName = regexp.MustCompile(`/dev/(\w+\d+)`)

var (
	volumesQueue      = map[string](chan *dto.BlockInfo){}
	volumesQueueMutex = sync.RWMutex{}
)

func GetVolumesData() (*dto.BlockInfo, error) {
	blockInfo, err := ghw.Block()
	retBlockInfo := &dto.BlockInfo{}

	copier.Copy(retBlockInfo, blockInfo)

	//pretty.Print(blockInfo)
	//pretty.Print(retBlockInfo)

	if err == nil {
		for _, v := range blockInfo.Disks {
			if len(v.Partitions) == 0 {

				rblock, errb := ublock.Device(v.Name)
				if errb != nil {
					log.Printf("Error getting block device for device /dev/%s: %v", v.Name, errb)
					continue
				}

				var partition = &dto.BlockPartition{
					Name: rblock.Name,
					Type: rblock.FSType,
					UUID: rblock.FsUUID,
				}

				lsbkInfo, err := lsblk.GetInfoFromDevice(v.Name)
				if err != nil {
					log.Printf("GetLabelsFromDevice failed: %v", err)
					continue
				}

				if lsbkInfo.Partlabel == "unknown" {
					lsbkInfo.Partlabel = lsbkInfo.Label
				}

				fs, flags, err := mount.FSFromBlock("/dev/" + v.Name)
				if err != nil {
					partition.Type = lsbkInfo.Fstype
					if partition.Type == "unknown" && rblock.FSType != "" {
						partition.Type = rblock.FSType
					}
					partition.PartitionFlags = []dto.MounDataFlag{}
				} else {
					partition.Type = fs
					partition.PartitionFlags.Scan(flags)
				}

				if partition.MountPoint != "" {
					stat := syscall.Statfs_t{}
					err := syscall.Statfs(partition.MountPoint, &stat)
					if err == nil {
						partition.MountFlags.Scan(stat.Flags)
					}
				}

				if partition.Type == "unknown" || partition.Type == "swap" || partition.Type == "" {
					continue
				}

				partition.Label = lsbkInfo.Partlabel
				partition.FilesystemLabel = lsbkInfo.Label
				partition.SizeBytes = v.SizeBytes
				partition.MountPoint = lsbkInfo.Mountpoint

				retBlockInfo.Partitions = append(retBlockInfo.Partitions, partition)

			} else {
				for _, d := range v.Partitions {
					var partition = &dto.BlockPartition{}
					copier.Copy(&partition, &d)

					if partition.Label == "unknown" || partition.FilesystemLabel == "unknown" || partition.Type == "unknown" {
						lsbkInfo, err := lsblk.GetInfoFromDevice(d.Name)
						if err == nil {
							if lsbkInfo.Label != "unknown" {
								partition.FilesystemLabel = strings.Replace(partition.FilesystemLabel, "unknown", lsbkInfo.Label, 1)
							}
							if lsbkInfo.Partlabel != "unknown" {
								partition.Label = strings.Replace(partition.Label, "unknown", lsbkInfo.Partlabel, 1)
							}
							if lsbkInfo.Fstype != "unknown" {
								partition.Type = strings.Replace(partition.Type, "unknown", lsbkInfo.Fstype, 1)
							}
						}
					}

					if partition.Label == "unknown" {
						partition.Label = partition.FilesystemLabel
					}
					if partition.Label == "unknown" && partition.FilesystemLabel == "unknown" {
						partition.Label = partition.UUID
					}
					fs, flags, err := mount.FSFromBlock("/dev/" + v.Name)
					if err == nil {
						partition.Type = strings.Replace(d.Type, "unknown", fs, 1)
						partition.PartitionFlags.Scan(flags)
					}

					if partition.MountPoint != "" {
						stat := syscall.Statfs_t{}
						err := syscall.Statfs(partition.MountPoint, &stat)
						if err == nil {
							partition.MountFlags.Scan(stat.Flags)
						}
					}

					if partition.Type == "unknown" || partition.Type == "swap" || partition.Type == "" {
						continue
					}

					retBlockInfo.Partitions = append(retBlockInfo.Partitions, partition)
				}
			}
		}
	}

	// Popolate MountPoints from partitions
	for i, partition := range retBlockInfo.Partitions {
		var conv converter.DtoToDbomConverterImpl
		var mount_data = &dbom.MountPointPath{}
		err = conv.BlockPartitionToMountPointPath(*partition, mount_data)
		if err != nil {
			log.Printf("Error converting partition to mount point data: %v", err)
			continue
		}
		if mount_data.Path == "" {
			if partition.FilesystemLabel != "unknown" {
				mount_data.Path = "/mnt/" + partition.FilesystemLabel
			} else if partition.Label != "unknown" {
				mount_data.Path = "/mnt/" + partition.Label
			} else if partition.UUID != "" {
				mount_data.Path = "/mnt/" + partition.UUID
			} else {
				mount_data.Path = "/mnt/" + partition.Name
			}
		}
		err = mount_data.Save()
		if err != nil {
			log.Printf("Error saving mount point data: %v", err)
			continue
		}
		conv.MountPointPathToMountPointData(*mount_data, &retBlockInfo.Partitions[i].MountPointData)
	}

	//pretty.Print(retBlockInfo)
	return retBlockInfo, err
}

// ListVolumes godoc
//
//	@Summary		List all available volumes
//	@Description	List all available volumes
//	@Tags			volume
//	@Produce		json
//	@Success		200	{object}	dto.BlockInfo
//	@Failure		405	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/volumes [get]
func ListVolumes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	volumes, err := GetVolumesData()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	HttpJSONReponse(w, volumes, nil)
}

// MountVolume godoc
//
//	@Summary		mount an existing volume
//	@Description	mount an existing volume
//	@Tags			volume
//	@Accept			json
//	@Produce		json
//	@Param			id			path		uint				true	"id of the mountpoint to be mounted"
//	@Param			mount_data	body		dto.MountPointData	true	"Mount data"
//	@Success		201			{object}	dto.MountPointData
//	@Failure		400			{object}	ErrorResponse
//	@Failure		405			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/volume/{id}/mount [post]
func MountVolume(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 32)
	if err != nil {
		HttpJSONReponse(w, ErrorResponse{Error: "Invalid ID", Body: err}, &Options{
			Code: http.StatusBadRequest,
		})
		return
	}

	var mount_data dto.MountPointData
	err = HttpJSONRequest(&mount_data, w, r)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	if mount_data.ID != 0 && mount_data.ID != uint(id) {
		HttpJSONReponse(w, ErrorResponse{Error: "ID conflict", Body: nil}, &Options{
			Code: http.StatusBadRequest,
		})
		return
	}

	var dbom_mount_data dbom.MountPointPath
	err = dbom_mount_data.FromID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HttpJSONReponse(w, ErrorResponse{Code: 404, Error: "MountPoint not found", Body: nil}, &Options{
				Code: http.StatusNotFound,
			})
			return
		}
		HttpJSONReponse(w, err, nil)
		return
	}

	var conv converter.DtoToDbomConverterImpl
	err = conv.MountPointDataToMountPointPath(mount_data, &dbom_mount_data)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	flags, err := dbom_mount_data.Flags.Value()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	// Check if mount_data.Path is already mounted

	volumes, err := GetVolumesData()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	var c = 0
	for _, d := range volumes.Partitions {
		if d.MountPoint != "" && strings.HasPrefix(dbom_mount_data.Path, d.MountPoint) {
			c++
		}
	}
	if c > 0 {
		dbom_mount_data.Path += fmt.Sprintf("_(%d)", c)
	}

	// Save/Update MountPointData
	err = dbom_mount_data.Save()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	/* 	err = conv.MountPointPathToMountPointData(dbom_mount_data, &mount_data)
	   	if err != nil {
	   		HttpJSONReponse(w, err, nil)
	   		return
	   	} */

	var mp *mount.MountPoint
	if dbom_mount_data.FSType == "" {
		mp, err = mount.TryMount(dbom_mount_data.Source, dbom_mount_data.Path, "" /*mount_data.Data*/, uintptr(flags.(int64)), func() error { return os.MkdirAll(dbom_mount_data.Path, 0o666) })
	} else {
		mp, err = mount.Mount(dbom_mount_data.Source, dbom_mount_data.Path, dbom_mount_data.FSType, "" /*mount_data.Data*/, uintptr(flags.(int64)), func() error { return os.MkdirAll(dbom_mount_data.Path, 0o666) })
	}
	if err != nil {
		log.Printf("(Try)Mount(%s) = %s", dbom_mount_data.Source, tracerr.SprintSourceColor(err))
		HttpJSONReponse(w, err, nil)
		return
	} else {
		var convm converter.MountToDbomImpl
		err = convm.MountToMountPointPath(mp, &dbom_mount_data)
		if err != nil {
			HttpJSONReponse(w, err, nil)
			return
		}
		mounted_data := dto.MountPointData{}
		err = conv.MountPointPathToMountPointData(dbom_mount_data, &mounted_data)
		if err != nil {
			HttpJSONReponse(w, err, nil)
			return
		}
		var ndata, _ = GetVolumesData()
		notifyVolumeClient(ndata)
		//		context_state := (&dto.Status{}).FromContext(r.Context())
		context_state := StateFromContext(r.Context())
		context_state.DataDirtyTracker.Volumes = true
		HttpJSONReponse(w, mounted_data, nil)
	}
}

// notifyVolumeClient sends updated volume information to all registered clients.
//
// This function iterates through all registered volume queues and sends the
// provided volume information to each of them. It uses a read lock to ensure
// thread-safe access to the shared volumesQueue.
//
// Parameters:
//   - volumes: A pointer to BlockInfo containing the updated volume information
//     to be sent to all clients.
//
// The function does not return any value.
func notifyVolumeClient(volumes *dto.BlockInfo) {
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
//	@Param			id		path	uint	true	"id of the mountpoint to be mounted"
//	@Param			force	query	bool	true	"Umount forcefully - forces an unmount regardless of currently open or otherwise used files within the file system to be unmounted."
//	@Param			lazy	query	bool	true	"Umount lazily - disallows future uses of any files below path -- i.e. it hides the file system mounted at path, but the file system itself is still active and any currently open files can continue to be used. When all references to files from this file system are gone, the file system will actually be unmounted."
//	@Success		204
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/volume/{id}/mount [delete]
func UmountVolume(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 32)
	if err != nil {
		HttpJSONReponse(w, ErrorResponse{Code: 500, Error: "Invalid ID", Body: nil}, &Options{
			Code: http.StatusBadRequest,
		})
		return
	}
	force := r.URL.Query().Get("force")
	lazy := r.URL.Query().Get("lazy")

	var dbom_mount_data dbom.MountPointPath
	err = dbom_mount_data.FromID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			HttpJSONReponse(w, ErrorResponse{Code: 404, Error: "MountPoint not found", Body: nil}, &Options{
				Code: http.StatusNotFound,
			})
			return
		}
		HttpJSONReponse(w, err, nil)
		return
	}

	err = mount.Unmount(dbom_mount_data.Path, force == "true", lazy == "true")
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}
	var ndata, _ = GetVolumesData()
	notifyVolumeClient(ndata)
	//	context_state := (&dto.Status{}).FromContext(r.Context())
	context_state := StateFromContext(r.Context())
	context_state.DataDirtyTracker.Volumes = true
	w.WriteHeader(http.StatusNoContent)
	return
}

// VolumesEventHandler monitors and handles UEvent kernel messages related to volume changes.
// It sets up a connection to receive UEvents, processes them, and notifies clients of volume updates.
// The function runs indefinitely, handling events and errors, until interrupted by a termination signal.
//
// This function does not take any parameters.
//
// The function does not return any value, but it will log errors and volume changes,
// and call notifyVolumeClient when volumes are added or removed.
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

// VolumesWsHandler handles WebSocket connections for volume-related events.
// It sets up a queue for the client, fetches initial volume data, and continuously
// listens for updates to send to the client.
//
// Parameters:
//   - ctx: A context.Context for handling cancellation and timeouts.
//   - request: A WebSocketMessageEnvelope containing the client's request information.
//   - c: A channel for sending WebSocketMessageEnvelope messages back to the client.
//
// The function doesn't return any value, but it continues to run until the context is cancelled,
// sending volume updates to the client through the provided channel.
func VolumesWsHandler(ctx context.Context, request dto.WebSocketMessageEnvelope, c chan *dto.WebSocketMessageEnvelope) {
	volumesQueueMutex.Lock()
	if volumesQueue[request.Uid] == nil {
		volumesQueue[request.Uid] = make(chan *dto.BlockInfo, 10)
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
			smessage := &dto.WebSocketMessageEnvelope{
				Event: dto.EventVolumes,
				Uid:   request.Uid,
				Data:  <-queue,
			}
			//log.Printf("Handle send: %s %s %d", smessage.Event, smessage.Uid, len(c))
			c <- smessage
		}
	}
}
