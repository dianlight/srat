package service

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"sync"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/repository"
	"github.com/kr/pretty"
	"github.com/pilebones/go-udev/netlink"
	"github.com/snapcore/snapd/osutil"
	"github.com/u-root/u-root/pkg/mount"
	"gitlab.com/tozd/go/errors"
	"gorm.io/gorm"
)

type VolumeServiceInterface interface {
	MountVolume(md dto.MountPointData) errors.E
	UnmountVolume(id string, force bool, lazy bool) errors.E
	GetVolumesData() (*[]dto.Disk, error)
	NotifyClient()
}

type VolumeService struct {
	ctx               context.Context
	volumesQueueMutex sync.RWMutex
	broascasting      BroadcasterServiceInterface
	mount_repo        repository.MountPointPathRepositoryInterface
	hardwareClient    *hardware.ClientWithResponses
}

func NewVolumeService(ctx context.Context, broascasting BroadcasterServiceInterface, mount_repo repository.MountPointPathRepositoryInterface, hardwareClient *hardware.ClientWithResponses) VolumeServiceInterface {
	p := &VolumeService{
		ctx:               ctx,
		broascasting:      broascasting,
		volumesQueueMutex: sync.RWMutex{},
		mount_repo:        mount_repo,
		hardwareClient:    hardwareClient,
	}
	p.GetVolumesData()
	go p.udevEventHandler()
	return p
}

func (ms *VolumeService) MountVolume(md dto.MountPointData) errors.E {
	dbom_mount_data, err := ms.mount_repo.FindByPath(md.Path)
	if err != nil {
		return errors.WithStack(err)
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.MountPointDataToMountPointPath(md, dbom_mount_data)
	if err != nil {
		return errors.WithStack(err)
	}

	if dbom_mount_data.Device == "" {
		return errors.WithDetails(dto.ErrorDeviceNotFound,
			"Device", dbom_mount_data.Device,
			"Path", dbom_mount_data.Path,
			"Message", "Source device is empty",
		)
	}

	if dbom_mount_data.Path == "" {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"Device", dbom_mount_data.Device,
			"Path", dbom_mount_data.Path,
			"Message", "Mount point path is empty",
		)
	}

	ok, err := osutil.IsMounted(dbom_mount_data.Path)
	if err != nil {
		return errors.WithStack(err)
	}

	if dbom_mount_data.IsMounted && ok {
		return errors.WithDetails(dto.ErrorMountFail,
			"Device", dbom_mount_data.Device,
			"Path", dbom_mount_data.Path,
			"Message", "Volume is already mounted",
		)
	}

	orgPath := dbom_mount_data.Path
	for i := 1; ok; i++ {
		dbom_mount_data.Path = orgPath + "_(" + strconv.Itoa(i) + ")"
		ok, err = osutil.IsMounted(dbom_mount_data.Path)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	flags, err := dbom_mount_data.Flags.Value()
	if err != nil {
		return errors.WithDetails(errors.Basef("%w Invalid Flags %w", dto.ErrorInvalidParameter, err),
			"Device", dbom_mount_data.Device,
			"Path", dbom_mount_data.Path,
			"Message", "Invalid Flags",
		)
	}
	var mp *mount.MountPoint
	if dbom_mount_data.FSType == "" {
		mp, err = mount.TryMount("/dev/"+dbom_mount_data.Device, dbom_mount_data.Path, "" /*mount_data.Data*/, uintptr(flags.(int64)), func() error { return os.MkdirAll(dbom_mount_data.Path, 0o666) })
	} else {
		mp, err = mount.Mount("/dev/"+dbom_mount_data.Device, dbom_mount_data.Path, dbom_mount_data.FSType, "" /*mount_data.Data*/, uintptr(flags.(int64)), func() error { return os.MkdirAll(dbom_mount_data.Path, 0o666) })
	}
	if err != nil {
		slog.Error("Failed to mount volume:", "source", dbom_mount_data.Device, "fstype", dbom_mount_data.FSType, "path", dbom_mount_data.Path, "flags", flags, "mp", mp)
		return errors.WithDetails(errors.Basef("%w Mount Error %w", dto.ErrorMountFail, err),
			"Device", dbom_mount_data.Device,
			"Path", dbom_mount_data.Path,
			"Message", "Mount failed",
		)
	} else {
		var convm converter.MountToDbomImpl
		err = convm.MountToMountPointPath(mp, dbom_mount_data)
		if err != nil {
			return errors.WithStack(err)
		}
		dbom_mount_data.IsMounted = true
		err = ms.mount_repo.Save(dbom_mount_data)
		if err != nil {
			return errors.WithStack(err)
		}
		ms.NotifyClient()
	}
	return nil
}

func (ms *VolumeService) UnmountVolume(path string, force bool, lazy bool) errors.E {
	dbom_mount_data, err := ms.mount_repo.FindByPath(path)
	if err != nil {
		return errors.WithStack(err)
	}
	err = mount.Unmount(dbom_mount_data.Path, force, lazy)
	if err != nil {
		return errors.WithStack(err)
	}
	dbom_mount_data.IsMounted = false
	err = ms.mount_repo.Save(dbom_mount_data)
	if err != nil {
		return errors.WithStack(err)
	}
	ms.NotifyClient()
	return nil

}

func (self *VolumeService) udevEventHandler() {
	slog.Debug("Monitoring UEvent kernel message to user-space...")

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		slog.Error("Unable to connect to Netlink Kobject UEvent socket", "err", err)
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
			slog.Info("Run process closed", "err", self.ctx.Err())
			return
		case uevent := <-queue:
			slog.Info("Handle", "event", pretty.Sprint(uevent))
			if uevent.Action == "add" {
				self.NotifyClient()
			} else if uevent.Action == "remove" {
				self.NotifyClient()
			}
		case err := <-errors:
			slog.Error("ERROR:", "err", err)
		}
	}

}

func (self *VolumeService) GetVolumesData() (*[]dto.Disk, error) {

	ret := []dto.Disk{}
	conv := converter.HaHardwareToDtoImpl{}
	dbconv := converter.DtoToDbomConverterImpl{}
	//lsconv := converter.LsblkToDbomConverterImpl{}

	hwser, err := self.hardwareClient.GetHardwareInfoWithResponse(self.ctx, nil)
	if err != nil {
		slog.Warn("Error getting hardware info fallback to direct system!", "err", err)
		/*
			blockInfo, err := ghw.Block()
			if err != nil {
				return nil, errors.WithStack(err)
			}

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
							partition.PartitionFlags = []string{}
						} else {
							partition.Type = fs
							tmp := dto.MountFlags{}
							tmp.Scan(flags)
							partition.PartitionFlags = tmp.Strings()
						}

						if partition.MountPoint != "" {
							stat := syscall.Statfs_t{}
							err := syscall.Statfs(partition.MountPoint, &stat)
							if err == nil {
								tmp := dto.MountFlags{}
								tmp.Scan(stat.Flags)
								partition.MountFlags = tmp.Strings()
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
								tmp := dto.MountFlags{}
								tmp.Scan(flags)
								partition.PartitionFlags = tmp.Strings()
							}

							if partition.MountPoint != "" {
								stat := syscall.Statfs_t{}
								err := syscall.Statfs(partition.MountPoint, &stat)
								if err == nil {
									tmp := dto.MountFlags{}
									tmp.Scan(stat.Flags)
									partition.MountFlags = tmp.Strings()
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
		*/
	} else {

		for _, drive := range *hwser.JSON200.Data.Drives {
			if drive.Filesystems == nil || len(*drive.Filesystems) == 0 {
				continue
			}
			var diskDto dto.Disk
			err = conv.DriveToDisk(drive, &diskDto)
			if err != nil {
				slog.Warn("Error converting drive to disk", "err", err)
				continue
			}
			if diskDto.Partitions == nil || len(*diskDto.Partitions) == 0 {
				continue
			}

			ret = append(ret, diskDto)
		}
	}

	for _, disk := range ret {
		if disk.Partitions == nil || len(*disk.Partitions) == 0 {
			continue
		}
		for i, partition := range *disk.Partitions {
			for j, mountPoint := range *partition.MountPointData {
				mountPointPath, err := self.mount_repo.FindByPath(mountPoint.Path)
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					slog.Warn("Error search for a mount directory", "err", err)
					continue
				}

				if mountPointPath == nil {
					mountPointPath = &dbom.MountPointPath{}
				}

				err = dbconv.MountPointDataToMountPointPath(mountPoint, mountPointPath)
				if err != nil {
					slog.Warn("Error converting partition to mount point data", "err", err)
					continue
				}

				err = self.mount_repo.Save(mountPointPath)
				if err != nil {
					if errors.Is(err, gorm.ErrDuplicatedKey) {
						slog.Warn("Duplicate Key!", "data", mountPointPath, "err", err)
					} else {
						slog.Warn("Error saving mount point data", "data", mountPointPath, "err", err)
					}
					mountPointPath.IsInvalid = true
					continue
				}

				dbconv.MountPointPathToMountPointData(*mountPointPath, &(*(*disk.Partitions)[i].MountPointData)[j])
			}
		}
	}

	/*
		// Popolate MountPoints from partitions
		for i, partition := range retBlockInfo.Partitions {
			var conv converter.DtoToDbomConverterImpl
			var mount_data = &dbom.MountPointPath{}
			err = conv.BlockPartitionToMountPointPath(*partition, mount_data)
			//		slog.Debug("1.lags", "flapartition", partition.MountFlags, "mount_data", mount_data.Flags)

			if err != nil {
				slog.Warn("Error converting partition to mount point data", "err", err)
				continue
			}
			if mount_data.MountPoint == "" {
				if partition.MountPoint != "" {
					mount_data.MountPoint = partition.MountPoint
				} else if partition.FilesystemLabel != "unknown" {
					mount_data.MountPoint = "/mnt/" + partition.FilesystemLabel
				} else if partition.Label != "unknown" {
					mount_data.MountPoint = "/mnt/" + partition.Label
				} else if partition.UUID != "" {
					mount_data.MountPoint = "/mnt/" + partition.UUID
				} else {
					mount_data.MountPoint = "/mnt/" + partition.Name
				}
			}

			if partition.MountPoint == "" {
				orgPath := mount_data.MountPoint
				for i := 1; i < 20; i++ {
					ok, err := self.mount_repo.FindByPath(mount_data.MountPoint)
					if errors.Is(err, gorm.ErrRecordNotFound) {
						break
					}
					if err != nil {
						slog.Warn("Error search for a mount directory", "err", err)
						continue
					}
					if ok.Device == partition.Name {
						break
					}
					mount_data.MountPoint = orgPath + "_(" + strconv.Itoa(i) + ")"
				}
			}

			err = self.mount_repo.Save(mount_data)
			if err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					slog.Warn("Duplicate Key!", "data", mount_data, "err", err)
				} else {
					slog.Warn("Error saving mount point data", "data", mount_data, "err", err)
				}
				mount_data.IsInvalid = true
				continue
			}
			conv.MountPointPathToMountPointData(*mount_data, &retBlockInfo.Partitions[i].MountPointData)
			//		slog.Debug("2.lags", "mount_data", mount_data.Flags, "flapartition", retBlockInfo.Partitions[i].MountPointData.Flags)

		}
	*/

	//pretty.Print(retBlockInfo)
	return &ret, err
}

func (self *VolumeService) NotifyClient() {
	slog.Debug("Notifying client about changes...")
	self.volumesQueueMutex.Lock()
	defer self.volumesQueueMutex.Unlock()

	var data, err = self.GetVolumesData()
	if err != nil {
		slog.Error("Unable to fetch volumes", "err", err)
		return
	}

	self.broascasting.BroadcastMessage(data)
}
