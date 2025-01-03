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
	"strings"
	"sync"
	"syscall"

	"github.com/dianlight/srat/data"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/lsblk"
	"github.com/gobeam/stringy"
	"github.com/gorilla/mux"
	"github.com/jaypipes/ghw"
	"github.com/jinzhu/copier"
	"github.com/kr/pretty"
	"github.com/pilebones/go-udev/netlink"
	"github.com/u-root/u-root/pkg/mount"
	ublock "github.com/u-root/u-root/pkg/mount/block"
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
					partition.PartitionFlags = []data.MounDataFlag{}
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

	// Enrich the data with mount point information form DB ( previously saved state if mount point is not present)
	for i, partition := range retBlockInfo.Partitions {
		if partition.MountPoint == "" {
			var mp dbom.MountPointData
			//log.Printf("\nAttempting to %#v\n", partition)
			err := mp.FromName(partition.Name)
			if err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					log.Printf("Error fetching mount point data for device /dev/%s: %v", partition.Name, err)
				}
				continue
			}
			partition.DefaultMountPoint = mp.Path
			partition.MountData = mp.Data
			partition.MountFlags.Scan(mp.Flags)
			partition.DeviceId = &mp.DeviceId
			retBlockInfo.Partitions[i] = partition
		} else if partition.DeviceId == nil {
			sstat := syscall.Stat_t{}
			err := syscall.Stat(partition.MountPoint, &sstat)
			if err != nil {
				return nil, err
			}
			partition.DeviceId = &sstat.Dev
			retBlockInfo.Partitions[i] = partition
		}
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
//	@Failure		405	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/volumes [get]
func ListVolumes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	volumes, err := GetVolumesData()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Error fetching volumes", err)
		return
	}
	volumes.ToResponse(http.StatusOK, w)
}

// MountVolume godoc
//
//	@Summary		mount an existing volume
//	@Description	mount an existing volume
//	@Tags			volume
//	@Accept			json
//	@Produce		json
//	@Param			volume_name	path		string				true	"Volume Name to Mount"
//	@Param			mount_data	body		dto.MountPointData	true	"Mount data"
//	@Success		201			{object}	dto.MountPointData
//	@Failure		400			{object}	dto.ResponseError
//	@Failure		405			{object}	dto.ResponseError
//	@Failure		409			{object}	dto.ResponseError
//	@Failure		500			{object}	dto.ResponseError
//	@Router			/volume/{volume_name}/mount [post]
func MountVolume(w http.ResponseWriter, r *http.Request) {
	volume_name := mux.Vars(r)["volume_name"]
	w.Header().Set("Content-Type", "application/json")

	var mount_data dto.MountPointData
	mount_data.FromJSONBody(w, r)

	if mount_data.Name != "" && mount_data.Name != volume_name {
		dto.ResponseError{}.ToResponseError(http.StatusBadRequest, w, "Name conflict", nil)
		return
	} else if mount_data.Name == "" {
		mount_data.Name = volume_name
	}

	volumes, err := GetVolumesData()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Error fetching volumes", err)
		return
	}

	if mount_data.FSType == "" || mount_data.Label == "" || mount_data.Path == "" {

		for _, d := range volumes.Partitions {
			if d.Name == volume_name {
				if mount_data.FSType == "" {
					mount_data.FSType = d.Type
				}
				if mount_data.Label == "" {
					mount_data.Label = d.Label
				}
				if mount_data.Path == "" {
					mount_data.Path = d.MountPoint
				}
				break
			}
		}
	}

	if mount_data.Path == "" && mount_data.Label != "" {
		mount_data.Path = "/mnt/" + stringy.New(mount_data.Label).RemoveSpecialCharacter()
	} else if mount_data.Path == "" {
		mount_data.Path = "/mnt/" + mount_data.Name
	}

	var flags, err4 = mount_data.Flags.Value()
	if err4 != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Error parsing flags", err4)
		return
	}

	mount_data.Path = stringy.New(mount_data.Path).SnakeCase().Get()

	if !strings.HasPrefix(mount_data.Name, "/dev/") {
		mount_data.Name = "/dev/" + mount_data.Name
	}

	// Check if mount_data.Path is already mounted

	var c = 0
	for _, d := range volumes.Partitions {
		if d.MountPoint != "" && strings.HasPrefix(mount_data.Path, d.MountPoint) {
			c++
		}
	}
	if c > 0 {
		mount_data.Path += fmt.Sprintf("_(%d)", c)
	}

	var mp *mount.MountPoint
	if mount_data.FSType == "" {
		mp, err = mount.TryMount(mount_data.Name, mount_data.Path, mount_data.Data, uintptr(flags.(int64)), func() error { return os.MkdirAll(mount_data.Path, 0o666) })
	} else {
		mp, err = mount.Mount(mount_data.Name, mount_data.Path, mount_data.FSType, mount_data.Data, uintptr(flags.(int64)), func() error { return os.MkdirAll(mount_data.Path, 0o666) })
	}
	if err != nil {
		log.Printf("(Try)Mount(%s) = %v, want nil", mount_data.Name, err)
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Error Mounting volume", err)
		return
	} else {
		mounted_data := dto.MountPointData{}
		copier.Copy(&mounted_data, mp)
		//		mounted_data.Flags.Scan(mp.Flags)
		// log.Printf("---------------------   mp %v mounted_data = %v", mp, mounted_data)
		//		for _, flags := range config.MounDataFlags {
		//			if mp.Flags&uintptr(flags) != 0 {
		//				mounted_data.Flags = append(mounted_data.Flags, flags)
		//			}
		//		}

		var ndata, _ = GetVolumesData()
		notifyVolumeClient(ndata)
		data_dirty_tracker := r.Context().Value("data_dirty_tracker").(*dto.DataDirtyTracker)
		data_dirty_tracker.Volumes = true

		mounted_data.ToResponse(http.StatusCreated, w)
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
//	@Param			volume_name	path	string	true	"Name of the volume to be unmounted"
//	@Param			force		query	bool	true	"Umount forcefully - forces an unmount regardless of currently open or otherwise used files within the file system to be unmounted."
//	@Param			lazy		query	bool	true	"Umount lazily - disallows future uses of any files below path -- i.e. it hides the file system mounted at path, but the file system itself is still active and any currently open files can continue to be used. When all references to files from this file system are gone, the file system will actually be unmounted."
//	@Success		204
//	@Failure		404	{object}	dto.ResponseError
//	@Failure		500	{object}	dto.ResponseError
//	@Router			/volume/{volume_name}/mount [delete]
func UmountVolume(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	volume_name := mux.Vars(r)["volume_name"]
	force := r.URL.Query().Get("force")
	lazy := r.URL.Query().Get("lazy")

	volumes, err := GetVolumesData()
	if err != nil {
		dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, "Error fetching volumes", err)
		return
	}

	for _, d := range volumes.Partitions {
		if d.Name == volume_name {
			err := mount.Unmount(d.MountPoint, force == "true", lazy == "true")
			if err != nil {
				dto.ResponseError{}.ToResponseError(http.StatusInternalServerError, w, fmt.Sprintf("Error unmounting %s", d.MountPoint), err)
				return
			}
			var ndata, _ = GetVolumesData()
			notifyVolumeClient(ndata)
			data_dirty_tracker := r.Context().Value("data_dirty_tracker").(*dto.DataDirtyTracker)
			data_dirty_tracker.Volumes = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	dto.ResponseError{}.ToResponseError(http.StatusNotFound, w, fmt.Sprintf("No mount on %s found!", volume_name), "")
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
