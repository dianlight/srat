package service

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/lsblk"
	"github.com/dianlight/srat/repository"
	"github.com/jaypipes/ghw"
	"github.com/jinzhu/copier"
	"github.com/kr/pretty"
	"github.com/pilebones/go-udev/netlink"
	"github.com/snapcore/snapd/osutil"
	"github.com/u-root/u-root/pkg/mount"
	ublock "github.com/u-root/u-root/pkg/mount/block"
	"github.com/ztrue/tracerr"
)

type VolumeServiceInterface interface {
	MountVolume(md dto.MountPointData) error
	UnmountVolume(id uint, force bool, lazy bool) error
	GetVolumesData() (*dto.BlockInfo, error)
	NotifyClient()
}

type VolumeService struct {
	ctx               context.Context
	volumesQueueMutex sync.RWMutex
	broascasting      BroadcasterServiceInterface
	mount_repo        repository.MountPointPathRepositoryInterface
}

func NewVolumeService(ctx context.Context, broascasting BroadcasterServiceInterface, mount_repo repository.MountPointPathRepositoryInterface) VolumeServiceInterface {
	p := &VolumeService{
		ctx:               ctx,
		broascasting:      broascasting,
		volumesQueueMutex: sync.RWMutex{},
		mount_repo:        mount_repo,
	}
	p.GetVolumesData()
	go p.udevEventHandler()
	return p
}

func (ms *VolumeService) MountVolume(md dto.MountPointData) error {
	dbom_mount_data, err := ms.mount_repo.FindByID(md.ID)
	if err != nil {
		return tracerr.Wrap(err)
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.MountPointDataToMountPointPath(md, dbom_mount_data)
	if err != nil {
		return tracerr.Wrap(err)
	}

	if dbom_mount_data.Source == "" {
		return tracerr.Wrap(dto.NewErrorInfo(dto.ErrorCodes.MOUNT_FAIL, map[string]any{
			"Device":  dbom_mount_data.Source,
			"Path":    dbom_mount_data.Path,
			"Message": "Source device is empty",
		}, nil))
	}

	if dbom_mount_data.Path == "" {
		return tracerr.Wrap(dto.NewErrorInfo(dto.ErrorCodes.MOUNT_FAIL, map[string]any{
			"Device":  dbom_mount_data.Source,
			"Path":    dbom_mount_data.Path,
			"Message": "Mount point path is empty",
		}, nil))
	}

	ok, err := osutil.IsMounted(dbom_mount_data.Path)
	if err != nil {
		return tracerr.Wrap(err)
	}

	if dbom_mount_data.IsMounted && ok {
		return dto.NewErrorInfo(dto.ErrorCodes.MOUNT_FAIL, map[string]any{
			"Device":  dbom_mount_data.Source,
			"Path":    dbom_mount_data.Path,
			"Message": "Volume is already mounted",
		}, nil)
	}

	orgPath := dbom_mount_data.Path
	for i := 1; ok; i++ {
		dbom_mount_data.Path = orgPath + "_(" + strconv.Itoa(i) + ")"
		ok, err = osutil.IsMounted(dbom_mount_data.Path)
		if err != nil {
			return tracerr.Wrap(err)
		}
	}

	flags, err := dbom_mount_data.Flags.Value()
	if err != nil {
		return tracerr.Wrap(dto.NewErrorInfo(dto.ErrorCodes.MOUNT_FAIL, map[string]any{
			"Device":  dbom_mount_data.Source,
			"Path":    dbom_mount_data.Path,
			"Message": "Invalid Flags",
		}, err))
	}
	var mp *mount.MountPoint
	if dbom_mount_data.FSType == "" {
		mp, err = mount.TryMount(dbom_mount_data.Source, dbom_mount_data.Path, "" /*mount_data.Data*/, uintptr(flags.(int64)), func() error { return os.MkdirAll(dbom_mount_data.Path, 0o666) })
	} else {
		mp, err = mount.Mount(dbom_mount_data.Source, dbom_mount_data.Path, dbom_mount_data.FSType, "" /*mount_data.Data*/, uintptr(flags.(int64)), func() error { return os.MkdirAll(dbom_mount_data.Path, 0o666) })
	}
	if err != nil {
		slog.Error("Failed to mount volume:", "source", dbom_mount_data.Source, "fstype", dbom_mount_data.FSType, "path", dbom_mount_data.Path, "flags", flags, "mp", mp)
		return tracerr.Wrap(dto.NewErrorInfo(dto.ErrorCodes.MOUNT_FAIL, map[string]any{
			"Device":  dbom_mount_data.Source,
			"Path":    dbom_mount_data.Path,
			"Message": "Mount failed",
		}, err))
	} else {
		var convm converter.MountToDbomImpl
		err = convm.MountToMountPointPath(mp, dbom_mount_data)
		if err != nil {
			return tracerr.Wrap(err)
		}
		dbom_mount_data.IsMounted = true
		err = ms.mount_repo.Save(dbom_mount_data)
		if err != nil {
			return tracerr.Wrap(err)
		}
		ms.NotifyClient()
	}
	return nil
}

func (ms *VolumeService) UnmountVolume(id uint, force bool, lazy bool) error {
	dbom_mount_data, err := ms.mount_repo.FindByID(id)
	if err != nil {
		return tracerr.Wrap(err)
	}
	err = mount.Unmount(dbom_mount_data.Path, force, lazy)
	if err != nil {
		return tracerr.Wrap(err)
	}
	dbom_mount_data.IsMounted = false
	err = ms.mount_repo.Save(dbom_mount_data)
	if err != nil {
		return tracerr.Wrap(err)
	}
	ms.NotifyClient()
	return nil

}

func (self *VolumeService) udevEventHandler() {
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
	/*
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		go func() {
			<-signals
			slog.Debug("Exiting monitor mode...")
			close(quit)
			// os.Exit(0)
		}()
	*/

	// Handling message from queue
	for {
		select {
		case <-self.ctx.Done():
			close(quit)
			slog.Info("Run process closed", "err", tracerr.SprintSourceColor(self.ctx.Err()))
			return
		case uevent := <-queue:
			slog.Info("Handle", "event", pretty.Sprint(uevent))
			if uevent.Action == "add" {
				self.NotifyClient()
			} else if uevent.Action == "remove" {
				self.NotifyClient()
			}
		case err := <-errors:
			slog.Error("ERROR:", "err", tracerr.SprintSourceColor(err))
		}
	}

}

func (self *VolumeService) GetVolumesData() (*dto.BlockInfo, error) {
	blockInfo, err := ghw.Block()
	retBlockInfo := &dto.BlockInfo{}

	copier.Copy(retBlockInfo, blockInfo)

	//pretty.Print(blockInfo)
	//pretty.Print(retBlockInfo)

	if err == nil {
		for _, v := range blockInfo.Disks {
			if len(v.Partitions) == 0 {

				rblock, err := ublock.Device(v.Name)
				if err != nil {
					slog.Warn("Error getting block device for device", "dev", v.Name, "err", err)
					continue
				}

				var partition = &dto.BlockPartition{
					Name: rblock.Name,
					Type: rblock.FSType,
					UUID: rblock.FsUUID,
				}

				lsbkInfo, err := lsblk.GetInfoFromDevice(v.Name)
				if err != nil {
					slog.Debug("GetLabelsFromDevice failed", "err", err)
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
			slog.Warn("Error converting partition to mount point data", "err", err)
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
		err = self.mount_repo.Save(mount_data)
		if err != nil {
			slog.Warn("Error saving mount point data", "err", err)
			continue
		}
		conv.MountPointPathToMountPointData(*mount_data, &retBlockInfo.Partitions[i].MountPointData)
	}

	//pretty.Print(retBlockInfo)
	return retBlockInfo, err
}

func (self *VolumeService) NotifyClient() {
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
	self.broascasting.BroadcastMessage(&event)
}
