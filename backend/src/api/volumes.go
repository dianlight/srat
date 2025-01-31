package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
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

type VolumeHandler struct {
	ctx               context.Context
	volumesQueueMutex sync.RWMutex
}

func NewVolumeHandler(ctx context.Context) *VolumeHandler {
	p := new(VolumeHandler)
	p.ctx = ctx
	//p.sharesQueue = map[string](chan *[]dto.SharedResource){}
	p.volumesQueueMutex = sync.RWMutex{}
	StateFromContext(p.ctx).SSEBroker.AddOpenConnectionListener(func(broker BrokerInterface) error {
		p.notifyClient()
		return nil
	})
	go p.VolumesEventHandler()
	return p
}

func (self *VolumeHandler) GetVolumesData() (*dto.BlockInfo, error) {
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
func (self *VolumeHandler) ListVolumes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	volumes, err := self.GetVolumesData()
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
func (self *VolumeHandler) MountVolume(w http.ResponseWriter, r *http.Request) {
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

	volumes, err := self.GetVolumesData()
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
		self.notifyClient()
		//		context_state := (&dto.Status{}).FromContext(r.Context())
		context_state := StateFromContext(r.Context())
		context_state.DataDirtyTracker.Volumes = true
		HttpJSONReponse(w, mounted_data, nil)
	}
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
func (self *VolumeHandler) UmountVolume(w http.ResponseWriter, r *http.Request) {
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
	self.notifyClient()
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
func (self *VolumeHandler) VolumesEventHandler() {
	slog.Debug("Monitoring UEvent kernel message to user-space...")

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		slog.Error("Unable to connect to Netlink Kobject UEvent socket", "err", tracerr.SprintSourceColor(err))
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
		slog.Debug("Exiting monitor mode...")
		close(quit)
		// os.Exit(0)
	}()

	// Handling message from queue
	for {
		select {
		case uevent := <-queue:
			slog.Info("Handle", "event", pretty.Sprint(uevent))
			if uevent.Action == "add" {
				self.notifyClient()
			} else if uevent.Action == "remove" {
				self.notifyClient()
			}
		case err := <-errors:
			slog.Error("ERROR:", "err", tracerr.SprintSourceColor(err))
		}
	}

}

func (self *VolumeHandler) notifyClient() {
	self.volumesQueueMutex.Lock()
	defer self.volumesQueueMutex.Unlock()

	var data, err = self.GetVolumesData()
	if err != nil {
		log.Printf("Unable to fetch volumes: %v", err)
		return
	}

	var event dto.EventMessageEnvelope
	event.Event = dto.EventVolumes
	event.Data = data
	StateFromContext(self.ctx).SSEBroker.BroadcastMessage(&event)
}
