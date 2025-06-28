package service

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	"fmt"
	"os/exec"
	"regexp"
	"syscall"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/lsblk"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/tlog"
	"github.com/pilebones/go-udev/netlink"
	"github.com/shomali11/util/xhashes"
	"github.com/snapcore/snapd/osutil"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/loop"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type VolumeServiceInterface interface {
	MountVolume(md *dto.MountPointData) errors.E
	UnmountVolume(id string, force bool, lazy bool) errors.E
	GetVolumesData() (*[]dto.Disk, error)
	PathHashToPath(pathhash string) (string, errors.E)
	EjectDisk(diskID string) error
	UpdateMountPointSettings(path string, settingsUpdate dto.MountPointData) (*dto.MountPointData, errors.E)
	PatchMountPointSettings(path string, settingsPatch dto.MountPointData) (*dto.MountPointData, errors.E)
	NotifyClient()
}

type VolumeService struct {
	ctx               context.Context
	volumesQueueMutex sync.RWMutex
	broascasting      BroadcasterServiceInterface
	mount_repo        repository.MountPointPathRepositoryInterface
	hardwareClient    hardware.ClientWithResponsesInterface
	lsblk             lsblk.LSBLKInterpreterInterface
	fs_service        FilesystemServiceInterface
	shareService      ShareServiceInterface // Added for disabling shares
	staticConfig      *dto.ContextState
	sfGroup           singleflight.Group
}

type VolumeServiceProps struct {
	fx.In
	Ctx               context.Context
	Broadcaster       BroadcasterServiceInterface
	MountPointRepo    repository.MountPointPathRepositoryInterface
	HardwareClient    hardware.ClientWithResponsesInterface `optional:"true"`
	LsblkInterpreter  lsblk.LSBLKInterpreterInterface
	FilesystemService FilesystemServiceInterface
	ShareService      ShareServiceInterface // Added
	StaticConfig      *dto.ContextState
}

func NewVolumeService(
	lc fx.Lifecycle,
	in VolumeServiceProps,
) VolumeServiceInterface {
	p := &VolumeService{
		ctx:               in.Ctx,
		broascasting:      in.Broadcaster,
		volumesQueueMutex: sync.RWMutex{},
		mount_repo:        in.MountPointRepo,
		hardwareClient:    in.HardwareClient,
		lsblk:             in.LsblkInterpreter,
		fs_service:        in.FilesystemService,
		staticConfig:      in.StaticConfig,
		shareService:      in.ShareService, // Added
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			p.ctx.Value("wg").(*sync.WaitGroup).Add(1)
			go func() {
				defer p.ctx.Value("wg").(*sync.WaitGroup).Done()
				p.udevEventHandler()
			}()
			return nil
		},
	})

	return p
}

func (ms *VolumeService) MountVolume(md *dto.MountPointData) errors.E {

	if md.Path == "" {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"Device", md.Device,
			"Path", md.Path,
			"Message", "Mount point path is empty",
		)
	}
	if md.Device == "" {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"Device", md.Device,
			"Path", md.Path,
			"Message", "Source device name is empty in request",
		)
	}

	dbom_mount_data, err := ms.mount_repo.FindByPath(md.Path)
	if err != nil {
		// If not found, create a new one based on input
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dbom_mount_data = &dbom.MountPointPath{}
		} else {
			return errors.WithStack(err) // Other DB error
		}
	}

	var conv converter.DtoToDbomConverterImpl
	err = conv.MountPointDataToMountPointPath(*md, dbom_mount_data)
	if err != nil {
		return errors.WithStack(err)
	}

	if dbom_mount_data.Device == "" {
		return errors.WithDetails(dto.ErrorDeviceNotFound,
			"Device", dbom_mount_data.Device,
			"Path", dbom_mount_data.Path,
			"Message", "Source device name is empty in request/DB record",
		)
	}

	if dbom_mount_data.Path == "" {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"Device", dbom_mount_data.Device,
			"Path", dbom_mount_data.Path,
			"Message", "Mount point path is empty",
		)
	}

	// --- Start Device Existence Check ---
	var real_device string
	deviceName := dbom_mount_data.Device
	fullDevicePath := "/dev/" + deviceName

	// Check 1: Does the raw device name exist (e.g., a loop device path)?
	fi, errStatRaw := os.Stat(deviceName)
	if errStatRaw == nil {
		real_device = deviceName // Raw name exists
		if fi.Mode().IsRegular() {
			slog.Debug("Device found using raw name", "device", real_device)
			loopd, err := loop.FindDevice()
			if err != nil {
				return errors.WithDetails(dto.ErrorMountFail, "Detail", "Error finding loop device", "Device", real_device, "Error", err)
			}

			err = ms.createBlockDevice(loopd)
			if err != nil {
				slog.Error("Error setting loop device", "device", real_device, "loop_device", loopd, "err", err)
			}

			err = loop.SetFile(loopd, deviceName)
			if err != nil {
				slog.Error("Error setting loop device", "device", real_device, "loop_device", loopd, "err", err)
				return errors.WithDetails(dto.ErrorMountFail, "Detail", "Error setting loop device", "Device", real_device, "Error", err)
			}
			real_device = loopd // Update the device to the loop device
			dbom_mount_data.Device = loopd
			dbom_mount_data.Flags.Add(dbom.MounDataFlag{
				Name: "ro",
			})
		}
		slog.Debug("Device found using raw name", "device", real_device, "type", fi.Mode().Type())
	} else if os.IsNotExist(errStatRaw) {
		// Check 2: Does the /dev/ path exist?
		_, errStatFull := os.Stat(fullDevicePath)
		if errStatFull == nil {
			real_device = fullDevicePath // /dev/ path exists
			slog.Debug("Device found using full path", "device", real_device)
		} else if os.IsNotExist(errStatFull) {
			// Neither exists
			slog.Error("Device not found", "raw_path", deviceName, "full_path", fullDevicePath)
			return errors.WithDetails(dto.ErrorDeviceNotFound,
				"Device", deviceName,
				"CheckedPaths", []string{deviceName, fullDevicePath},
				"Message", "Source device does not exist on the system",
			)
		} else {
			// Some other error checking the full path (e.g., permissions)
			slog.Error("Error checking device existence (full path)", "path", fullDevicePath, "err", errStatFull)
			return errors.WithDetails(dto.ErrorMountFail, "Detail", "Error checking device existence",
				"Path", fullDevicePath, "Error", errStatFull,
			)
		}
	} else {
		// Some other error checking the raw path (e.g., permissions)
		slog.Error("Error checking device existence (raw path)", "path", deviceName, "err", errStatRaw)
		return errors.WithDetails(dto.ErrorMountFail, "Detail", "Error checking device existence",
			"Path", deviceName, "Error", errStatRaw,
		)
	}
	// --- End Device Existence Check ---

	ok, err := osutil.IsMounted(dbom_mount_data.Path)
	if err != nil {
		// Note: IsMounted might fail if the path doesn't exist yet, which is fine before mounting.
		// Consider if this check needs refinement based on expected state.
		// For now, we proceed assuming an error here might be ignorable if ok is false.
		if ok { // Only return error if it claims to be mounted but check failed
			slog.Error("Error checking if path is mounted", "path", dbom_mount_data.Path, "err", err)
			return errors.WithDetails(dto.ErrorMountFail, "Detail", "Error checking mount status", "Path", dbom_mount_data.Path, "Error", err)
		}
		slog.Debug("osutil.IsMounted check failed, but path not mounted, proceeding", "path", dbom_mount_data.Path, "err", err)
		ok = false // Ensure ok is false if IsMounted errored
	}

	if ok {
		slog.Warn("Volume already mounted according to OS check", "device", real_device, "path", dbom_mount_data.Path)
		return errors.WithDetails(dto.ErrorAlreadyMounted,
			"Device", real_device,
			"Path", dbom_mount_data.Path,
			"Message", "Volume is already mounted",
		)
	}

	// Rename logic if path is already mounted (even if DB state was inconsistent)
	orgPath := dbom_mount_data.Path
	if ok { // Only rename if osutil.IsMounted returned true
		slog.Info("Attempting to rename mount path due to conflict", "original_path", orgPath)
		for i := 1; ; i++ {
			dbom_mount_data.Path = orgPath + "_(" + strconv.Itoa(i) + ")"
			okCheck, errCheck := osutil.IsMounted(dbom_mount_data.Path)
			if errCheck != nil {
				// Similar to above, error might be okay if path doesn't exist yet
				if okCheck {
					slog.Error("Error checking renamed path mount status", "path", dbom_mount_data.Path, "err", errCheck)
					return errors.WithDetails(dto.ErrorMountFail, "Detail", "Error checking renamed mount status", "Path", dbom_mount_data.Path, "Error", errCheck)
				}
				okCheck = false // Treat error as not mounted
			}
			if !okCheck {
				slog.Info("Found available renamed path", "new_path", dbom_mount_data.Path)
				break // Found an unused path
			}
			if i > 100 { // Safety break
				slog.Error("Could not find available renamed mount path after 100 attempts", "original_path", orgPath)
				return errors.WithDetails(dto.ErrorMountFail, "Path", orgPath, "Message", "Could not find available renamed mount path")
			}
		}
	}

	conv.MountPointPathToMountPointData(*dbom_mount_data, md)

	flags, data, err := ms.fs_service.MountFlagsToSyscallFlagAndData(*md.Flags)
	if err != nil {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"Device", real_device,
			"Path", md.Path,
			"Message", "Invalid Flags",
			"Error", err,
		)
	}

	slog.Debug("Attempting to mount volume", "device", real_device, "path", dbom_mount_data.Path, "fstype", dbom_mount_data.FSType, "flags", flags, "data", data)

	var mp *mount.MountPoint
	mountFunc := func() error { return os.MkdirAll(dbom_mount_data.Path, 0o666) }

	if dbom_mount_data.FSType == "" {
		// Use TryMount if FSType is not specified
		mp, err = mount.TryMount(real_device, dbom_mount_data.Path, data, flags, mountFunc)
	} else {
		// Use Mount if FSType is specified
		mp, err = mount.Mount(real_device, dbom_mount_data.Path, dbom_mount_data.FSType, data, flags, mountFunc)
	}

	if err != nil {
		slog.Error("Failed to mount volume", "device", real_device, "fstype", dbom_mount_data.FSType, "path", dbom_mount_data.Path, "flags", flags, "err", err, "mountpoint_details", mp)
		// Attempt to clean up directory if we created it and mount failed? Optional.
		// os.Remove(dbom_mount_data.Path)
		return errors.WithDetails(dto.ErrorMountFail,
			"Detail", "Mount command failed",
			"Device", real_device,
			"Path", dbom_mount_data.Path,
			"FSType", dbom_mount_data.FSType,
			"Flags", flags,
			"Error", err,
		)
	} else {
		slog.Info("Successfully mounted volume", "device", real_device, "path", dbom_mount_data.Path, "fstype", dbom_mount_data.FSType, "flags", mp.Flags, "data", mp.Data)
		var convm converter.MountToDbomImpl
		// Update dbom_mount_data with details from the actual mount point if available
		err = convm.MountToMountPointPath(mp, dbom_mount_data)
		if err != nil {
			// Log error but proceed, as mount succeeded
			slog.Warn("Failed to convert mount details back to DBOM", "err", err)
			// Don't return error here, mount was successful
		}

		mflags, errE := ms.fs_service.SyscallFlagToMountFlag(mp.Flags)
		if errE != nil {
			slog.Warn("Failed to convert mount flags back to DTO", "err", errE)
		} else {
			*dbom_mount_data.Flags = conv.MountFlagsToMountDataFlags(mflags)
		}
		// Use the validated real_device path in the DB record
		dbom_mount_data.Device = real_device // Store the original name, not the /dev/ path potentially

		err = ms.mount_repo.Save(dbom_mount_data)
		if err != nil {
			// Critical: Mount succeeded but DB save failed. State is inconsistent.
			// Attempt to unmount?
			slog.Error("CRITICAL: Mount succeeded but failed to save state to DB. Attempting unmount.", "device", real_device, "path", dbom_mount_data.Path, "save_error", err)
			unmountErr := mount.Unmount(dbom_mount_data.Path, true, false) // Force unmount
			if unmountErr != nil {
				slog.Error("Failed to auto-unmount after DB save failure", "path", dbom_mount_data.Path, "unmount_error", unmountErr)
				// Return original save error, but add context
				return errors.WithDetails(dto.ErrorDatabaseError, "Detail", "Failed to save mount state after successful mount, and auto-unmount failed",
					"Device", real_device, "Path", dbom_mount_data.Path, "Error", err, "UnmountError", unmountErr.Error())

			}
			// Return original save error
			return errors.WithDetails(dto.ErrorDatabaseError, "Detail", "Failed to save mount state after successful mount. Volume has been unmounted.",
				"Device", real_device, "Path", dbom_mount_data.Path, "Error", err)
		}
		go ms.NotifyClient()
		conv.MountPointPathToMountPointData(*dbom_mount_data, md)
	}
	return nil
}

func (ms *VolumeService) PathHashToPath(pathhash string) (string, errors.E) {
	dbom_mount_data, err := ms.mount_repo.All()
	if err != nil {
		return "", errors.WithStack(err)
	}
	for _, mount_data := range dbom_mount_data {
		if xhashes.MD5(mount_data.Path) == pathhash {
			return mount_data.Path, nil
		}
	}
	return "", errors.New("PathHash not found")
}

func (ms *VolumeService) UnmountVolume(path string, force bool, lazy bool) errors.E {
	dbom_mount_data, err := ms.mount_repo.FindByPath(path)
	if err != nil {
		// If not found in DB, maybe still try to unmount the path?
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Warn("Mount path not found in DB, attempting unmount anyway", "path", path)
			// Create a temporary record for logging/unmount call if needed, or just use path
		} else {
			return errors.WithStack(err)
		}
		// If not found, proceed to unmount using the path directly
	} else {
		// If found in DB, use the path from the record (might differ slightly if renamed)
		path = dbom_mount_data.Path
	}

	slog.Debug("Attempting to unmount volume", "path", path, "force", force, "lazy", lazy)
	err = mount.Unmount(path, force, lazy)
	if err != nil {
		slog.Error("Failed to unmount volume", "path", path, "err", err)
		// Check if it's already unmounted
		ok, checkErr := osutil.IsMounted(path)
		if checkErr == nil && !ok {
			slog.Warn("Unmount command failed, but volume is already unmounted.", "path", path)
			// Proceed to update DB state as unmounted
		} else {
			return errors.WithDetails(dto.ErrorUnmountFail, "Detail", "Unmount command failed", "Path", path, "Error", err)
		}
	} else {
		slog.Info("Successfully unmounted volume", "path", path, "device", dbom_mount_data.Device)
		if strings.HasPrefix(dbom_mount_data.Device, "/dev/loop") {
			// If the device is a loop device, remove the loop device
			err = loop.ClearFile(dbom_mount_data.Device)
			if err != nil {
				slog.Error("Failed to remove loop device", "device", dbom_mount_data.Device, "err", err)
				// Log but don't return error, as unmount succeeded
			} else {
				slog.Debug("Successfully removed loop device", "device", dbom_mount_data.Device)
			}
		}
		err = os.Remove(dbom_mount_data.Path) // Remove the mount point directory
		if err != nil {
			slog.Error("Failed to remove mount point directory", "path", dbom_mount_data.Path, "err", err)
		} else {
			slog.Debug("Removed mount point directory", "path", dbom_mount_data.Path)
		}
	}

	// Update DB state only if the record was found initially
	if dbom_mount_data != nil {
		err = ms.mount_repo.Save(dbom_mount_data)
		if err != nil {
			// Log error, but unmount succeeded. State might be inconsistent in DB.
			slog.Error("Unmount succeeded but failed to update DB state", "path", path, "err", err)
			// Don't return error here, as the primary operation (unmount) succeeded.
			// However, the DB is now potentially out of sync.
		}
	}
	go ms.NotifyClient()
	return nil

}

func (self *VolumeService) udevEventHandler() {
	tlog.Trace("Starting Udev event handler...") // Changed log level

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		tlog.Error("Unable to connect to Netlink Kobject UEvent socket", "err", err)
		return // Exit goroutine if connection fails
	}
	defer conn.Close()

	queue := make(chan netlink.UEvent, 10) // Added buffer to queue
	errorChan := make(chan error, 1)       // Renamed and buffered error channel
	quit := conn.Monitor(queue, errorChan, nil /*matcher*/)
	slog.Info("Udev monitor started successfully.")

	// Handling message from queue
	for {
		select {
		case <-self.ctx.Done():
			slog.Info("Udev event handler stopping due to context cancellation.", "err", self.ctx.Err())
			close(quit) // Signal monitor to stop
			// Drain remaining events? Maybe not necessary.
			return
		case uevent := <-queue:
			// Filter events - only interested in block devices for now
			if subsystem, ok := uevent.Env["SUBSYSTEM"]; ok && subsystem == "block" {
				action := uevent.Action
				devName, _ := uevent.Env["DEVNAME"]
				devType, _ := uevent.Env["DEVTYPE"]                                                               // disk, partition
				slog.Debug("Received Udev block event", "action", action, "devname", devName, "devtype", devType) // Changed log level

				// Trigger notification on add/remove/change events for block devices
				if action == "add" || action == "remove" || action == "change" {
					slog.Info("Relevant Udev event detected, triggering client notification.", "action", action, "devname", devName)
					go self.NotifyClient()
				}
			}
			// Optional: Log other events at debug level if needed
			// else {
			//  slog.Debug("Ignoring Udev event from other subsystem", "subsystem", subsystem)
			// }
		case err := <-errorChan:
			slog.Error("Error received from Udev monitor", "err", err)
			// Decide if error is fatal. If e.g. permission error, might need to stop.
			// For now, just log and continue. If monitor stops, loop will exit via quit channel closure.
		}
	}
	// slog.Info("Udev event handler finished.") // This might not be reached if ctx cancels
}

func (self *VolumeService) GetVolumesData() (*[]dto.Disk, error) {
	slog.Debug("Requesting GetVolumesData via singleflight...")

	const sfKey = "GetVolumesData"

	v, err, shared := self.sfGroup.Do(sfKey, func() (interface{}, error) {
		self.volumesQueueMutex.Lock()
		defer self.volumesQueueMutex.Unlock()

		slog.Debug("Executing GetVolumesData core logic (singleflight)...")

		ret := []dto.Disk{}
		conv := converter.HaHardwareToDtoImpl{}
		dbconv := converter.DtoToDbomConverterImpl{}
		lsblkconv := converter.LsblkToDtoConverterImpl{}

		if self.staticConfig.SupervisorURL == "demo" {
			ret = append(ret, dto.Disk{
				Id: pointer.String("DemoDisk"),
				Partitions: &[]dto.Partition{
					{
						Id:     pointer.String("DemoPartition"),
						Device: pointer.String("/dev/bogus"),
						System: pointer.Bool(false),
						MountPointData: &[]dto.MountPointData{
							{
								Path:      "/mnt/bogus",
								FSType:    pointer.String("ext4"),
								IsMounted: false,
							},
						},
					},
				},
			})
			return &ret, nil
		}

		hwser, errHw := self.hardwareClient.GetHardwareInfoWithResponse(self.ctx)
		if errHw != nil || hwser == nil {
			slog.Error("Failed to get hardware info from Home Assistant Supervisor", "err", errHw)
			return nil, errors.Wrap(errHw, "failed to get hardware info from HA Supervisor")
		}

		if hwser.StatusCode() != 200 || hwser.JSON200 == nil || hwser.JSON200.Data == nil || hwser.JSON200.Data.Drives == nil {
			errMsg := "Received invalid hardware info response from HA Supervisor"
			slog.Error(errMsg, "status_code", hwser.StatusCode(), "response_body", string(hwser.Body))
			return nil, errors.New(errMsg)
		}

		slog.Debug("Processing drives from HA Supervisor", "drive_count", len(*hwser.JSON200.Data.Drives))
		for i, drive := range *hwser.JSON200.Data.Drives {
			if drive.Filesystems == nil || len(*drive.Filesystems) == 0 {
				slog.Debug("Skipping drive with no filesystems", "drive_index", i, "drive_id", drive.Id)
				continue
			}
			var diskDto dto.Disk
			errConvDrive := conv.DriveToDisk(drive, &diskDto)
			if errConvDrive != nil {
				slog.Warn("Error converting drive to disk DTO", "drive_index", i, "drive_id", drive.Id, "err", errConvDrive)
				continue
			}
			if diskDto.Partitions == nil || len(*diskDto.Partitions) == 0 {
				slog.Debug("Skipping drive DTO with no partitions after conversion", "drive_index", i, "drive_id", drive.Id)
				continue
			}

			ret = append(ret, diskDto)
		}

		slog.Debug("Syncing mount point data with database", "disk_count", len(ret))
		for diskIdx := range ret {
			disk := &ret[diskIdx]
			if disk.Partitions == nil {
				continue
			}
			for partIdx := range *disk.Partitions {
				partition := &(*disk.Partitions)[partIdx]
				if partition.MountPointData == nil || len(*partition.MountPointData) == 0 {
					info, errLsblk := self.lsblk.GetInfoFromDevice(*partition.Device)
					if errLsblk != nil {
						slog.Warn("Error getting info from device", "device", *partition.Device, "err", errLsblk)
						continue
					}
					mountPointDto := &dto.MountPointData{}
					errConvLsblk := lsblkconv.LsblkInfoToMountPointData(info, mountPointDto)
					if errConvLsblk != nil {
						slog.Warn("Error converting Lsblk info to MountPointData", "device", *partition.Device, "err", errConvLsblk)
						continue
					}
					partition.MountPointData = &[]dto.MountPointData{*mountPointDto}
				}

				for mpIdx := range *partition.MountPointData {
					mountPointDto := &(*partition.MountPointData)[mpIdx]

					mountPointPathDB, errRepoFind := self.mount_repo.FindByPath(mountPointDto.Path)
					if errRepoFind != nil && !errors.Is(errRepoFind, gorm.ErrRecordNotFound) {
						slog.Warn("Error searching for mount point in DB", "path", mountPointDto.Path, "err", errRepoFind)
						mountPointDto.IsInvalid = true
						invalidError := errors.Wrap(errRepoFind, "DB find error").Error()
						mountPointDto.InvalidError = &invalidError
						continue
					}

					isNewRecord := errors.Is(errRepoFind, gorm.ErrRecordNotFound)
					if isNewRecord {
						mountPointPathDB = &dbom.MountPointPath{}
						slog.Debug("Mount point not found in DB, will create new record", "path", mountPointDto.Path)
					} else {
						slog.Debug("Found existing mount point in DB", "path", mountPointDto.Path)
					}

					errConvDtoToDbom := dbconv.MountPointDataToMountPointPath(*mountPointDto, mountPointPathDB)
					if errConvDtoToDbom != nil {
						slog.Warn("Error converting DTO mount point data to DBOM", "path", mountPointDto.Path, "err", errConvDtoToDbom)
						mountPointDto.IsInvalid = true
						invalidError := errors.Wrap(errConvDtoToDbom, "DTO to DBOM conversion error").Error()
						mountPointDto.InvalidError = &invalidError
						continue
					}

					if mountPointPathDB.Type == "ADDON" {
						errRepoSave := self.mount_repo.Save(mountPointPathDB)
						if errRepoSave != nil {
							slog.Warn("Error saving mount point data to DB", "path", mountPointPathDB.Path, "data", mountPointPathDB, "err", errRepoSave)
							mountPointDto.IsInvalid = true
							invalidError := errors.Wrap(errRepoSave, "DB save error").Error()
							mountPointDto.InvalidError = &invalidError
							continue
						}
					}

					errConvDbomToDto := dbconv.MountPointPathToMountPointData(*mountPointPathDB, mountPointDto)
					if errConvDbomToDto != nil {
						slog.Error("Error converting DBOM mount point data back to DTO", "path", mountPointPathDB.Path, "err", errConvDbomToDto)
						mountPointDto.IsInvalid = true
						invalidError := errors.Wrap(errConvDbomToDto, "DBOM to DTO conversion error").Error()
						mountPointDto.InvalidError = &invalidError
						continue
					}
					slog.Debug("Successfully synced mount point with DB", "path", mountPointDto.Path, "is_mounted", mountPointDto.IsMounted)

					(*partition.MountPointData)[mpIdx] = *mountPointDto
				}

			}
		}

		for i, disk := range ret {
			for j, volume := range *disk.Partitions {
				if volume.MountPointData == nil {
					continue
				}
				for k, mountPoint := range *volume.MountPointData {
					sharedData, errShare := self.shareService.GetShareFromPath(mountPoint.Path)
					if errShare != nil {
						if errors.Is(errShare, dto.ErrorShareNotFound) {
							continue
						} else {
							return nil, errShare
						}
					}
					if sharedData != nil {
						shares := (*(*(ret)[i].Partitions)[j].MountPointData)[k].Shares
						if shares == nil {
							shares = make([]dto.SharedResource, 0)
						}
						shares = append(shares, *sharedData)
						(*(*(ret)[i].Partitions)[j].MountPointData)[k].Shares = shares
					}
				}
			}
		}

		slog.Debug("Finished getting and syncing volumes data (core logic).")
		return &ret, nil
	})

	if err != nil {
		slog.Error("Singleflight execution of GetVolumesData failed", "err", err, "shared", shared)
		return nil, err
	}

	slog.Debug("Singleflight execution of GetVolumesData successful", "shared", shared)

	result, ok := v.(*[]dto.Disk)
	if !ok {
		slog.Error("Singleflight returned unexpected type for GetVolumesData", "type", fmt.Sprintf("%T", v))
		return nil, errors.New("internal error: singleflight returned unexpected type")
	}

	return result, nil
}

func (self *VolumeService) NotifyClient() {
	slog.Debug("Notifying client about volume changes...")

	var data, err = self.GetVolumesData()
	if err != nil {
		slog.Error("Unable to fetch volumes data for notification", "err", err)
		// Optionally, broadcast an error message or specific state?
		// self.broascasting.BroadcastMessage(map[string]string{"error": "Failed to get volume data"})
		return // Don't broadcast potentially stale or empty data on error
	}

	slog.Debug("Broadcasting updated volumes data", "disk_count", len(*data))
	_, broadcastErr := self.broascasting.BroadcastMessage(data)
	if broadcastErr != nil {
		// Log the error from broadcasting
		slog.Error("Failed to broadcast volume data update", "err", broadcastErr)
	}
}

func (self *VolumeService) createBlockDevice(device string) error {
	// Controlla se il dispositivo esiste già
	if _, err := os.Stat(device); !os.IsNotExist(err) {
		slog.Warn("Loop device already exists", "device", device)
		return nil
	}

	// Estrai i numeri major e minor dal nome del dispositivo
	major, minor, err := extractMajorMinor(device)
	if err != nil {
		return errors.Errorf("errore durante l'estrazione dei numeri major e minor: %v", err)
	}

	// Crea il dispositivo di blocco usando la syscall mknod
	dev := (major << 8) | minor
	err = syscall.Mknod(device, syscall.S_IFBLK|0660, dev)
	if err != nil {
		return errors.Errorf("errore durante la creazione del dispositivo di blocco: %v", err)
	}

	return nil
}

func extractMajorMinor(device string) (int, int, error) {
	re := regexp.MustCompile(`/dev/loop(\d+)`)
	matches := re.FindStringSubmatch(device)
	if len(matches) != 2 {
		return 0, 0, fmt.Errorf("formato del dispositivo non valido")
	}

	minor, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, fmt.Errorf("errore durante la conversione del numero minor: %v", err)
	}

	// Il numero major per i dispositivi di loop è generalmente 7
	major := 7

	return major, minor, nil
}

func (self *VolumeService) EjectDisk(diskID string) error {
	slog.Info("Attempting to eject disk", "disk_id", diskID)

	allDisks, err := self.GetVolumesData()
	if err != nil {
		return errors.Wrapf(err, "failed to get volume data before ejecting disk %s", diskID)
	}

	var targetDisk *dto.Disk
	for i, d := range *allDisks {
		if d.Id != nil && *d.Id == diskID {
			targetDisk = &(*allDisks)[i]
			break
		}
	}

	if targetDisk == nil {
		return errors.WithDetails(dto.ErrorDeviceNotFound, "DiskID", diskID, "Message", "Disk not found")
	}

	if targetDisk.Removable == nil || !*targetDisk.Removable {
		return errors.WithDetails(dto.ErrorInvalidParameter, "DiskID", diskID, "Message", "Disk is not removable")
	}

	// Unmount all mounted partitions of this disk
	if targetDisk.Partitions != nil {
		for _, partition := range *targetDisk.Partitions {
			if partition.MountPointData != nil {
				for _, mpd := range *partition.MountPointData {
					if mpd.IsMounted {
						slog.Info("Disabling shares for path before unmount during eject", "path", mpd.Path, "disk_id", diskID)
						_, shareErr := self.shareService.DisableShareFromPath(mpd.Path)
						if shareErr != nil && !errors.Is(shareErr, dto.ErrorShareNotFound) {
							slog.Warn("Failed to disable share during eject, proceeding with unmount", "path", mpd.Path, "error", shareErr)
							// Not returning error here, will attempt unmount anyway
						}

						slog.Info("Unmounting partition during eject", "path", mpd.Path, "disk_id", diskID)
						unmountErr := self.UnmountVolume(mpd.Path, true, true) // Force and lazy unmount
						if unmountErr != nil {
							// If unmount fails, we should probably stop and not try to eject.
							return errors.Wrapf(unmountErr, "failed to unmount partition %s during eject of disk %s", mpd.Path, diskID)
						}
					}
				}
			}
		}
	}

	// Eject the physical disk
	devicePath := "/dev/" + *targetDisk.Id // Assuming Disk.Id is the device name like "sda"
	slog.Info("Executing eject command", "device_path", devicePath)
	cmd := exec.Command("eject", devicePath)
	if ejectErr := cmd.Run(); ejectErr != nil {
		return errors.Wrapf(ejectErr, "failed to eject disk %s using command", devicePath)
	}

	slog.Info("Disk ejected successfully", "disk_id", diskID, "device_path", devicePath)
	go self.NotifyClient() // Notify clients of the change
	return nil
}

func (ms *VolumeService) UpdateMountPointSettings(path string, updates dto.MountPointData) (*dto.MountPointData, errors.E) {
	dbMountData, err := ms.mount_repo.FindByPath(path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(dto.ErrorNotFound, "mount configuration with path %s not found", path)
		}
		return nil, errors.WithStack(err)
	}

	var conv converter.DtoToDbomConverterImpl

	// Apply updates
	if updates.FSType != nil {
		dbMountData.FSType = *updates.FSType
	}
	if updates.Flags != nil {
		if dbMountData.Flags == nil {
			dbMountData.Flags = &dbom.MounDataFlags{}
		}
		*dbMountData.Flags = conv.MountFlagsToMountDataFlags(*updates.Flags)
	}
	if updates.CustomFlags != nil {
		if dbMountData.Data == nil {
			dbMountData.Data = &dbom.MounDataFlags{}
		}
		*dbMountData.Data = conv.MountFlagsToMountDataFlags(*updates.CustomFlags)
	}
	if updates.IsToMountAtStartup != nil {
		dbMountData.IsToMountAtStartup = updates.IsToMountAtStartup
	}

	if err := ms.mount_repo.Save(dbMountData); err != nil {
		return nil, errors.WithStack(err)
	}

	updatedDto := dto.MountPointData{}
	if convErr := conv.MountPointPathToMountPointData(*dbMountData, &updatedDto); convErr != nil {
		return nil, errors.WithStack(convErr)
	}
	// Consider if a notification is needed after settings change
	// go ms.notifyClient()
	return &updatedDto, nil
}

func (ms *VolumeService) PatchMountPointSettings(path string, patchData dto.MountPointData) (*dto.MountPointData, errors.E) {
	dbMountData, err := ms.mount_repo.FindByPath(path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(dto.ErrorNotFound, "mount configuration with path %s not found", path)
		}
		return nil, errors.WithStack(err)
	}

	var conv converter.DtoToDbomConverterImpl
	changed := false

	if patchData.FSType != nil {
		if dbMountData.FSType != *patchData.FSType {
			dbMountData.FSType = *patchData.FSType
			changed = true
		}
	}

	if patchData.Flags != nil {
		if dbMountData.Flags == nil {
			dbMountData.Flags = &dbom.MounDataFlags{}
		}
		*dbMountData.Flags = conv.MountFlagsToMountDataFlags(*patchData.Flags)
		changed = true
	}

	if patchData.CustomFlags != nil {
		if dbMountData.Data == nil {
			dbMountData.Data = &dbom.MounDataFlags{}
		}
		*dbMountData.Data = conv.MountFlagsToMountDataFlags(*patchData.CustomFlags)
		changed = true
	}

	if patchData.IsToMountAtStartup != nil {
		if dbMountData.IsToMountAtStartup == nil || *dbMountData.IsToMountAtStartup != *patchData.IsToMountAtStartup {
			dbMountData.IsToMountAtStartup = patchData.IsToMountAtStartup
			changed = true
		}
	}

	if changed {
		if err := ms.mount_repo.Save(dbMountData); err != nil {
			return nil, errors.WithStack(err)
		}
	}
	go ms.NotifyClient() // Consider if settings changes should trigger a volume data broadcast

	currentDto := dto.MountPointData{}
	if convErr := conv.MountPointPathToMountPointData(*dbMountData, &currentDto); convErr != nil {
		return nil, errors.WithStack(convErr)
	}
	return &currentDto, nil
}
