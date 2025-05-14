package service

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	"fmt"
	"regexp"
	"syscall"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/hardware"
	"github.com/dianlight/srat/lsblk"
	"github.com/dianlight/srat/repository"
	"github.com/pilebones/go-udev/netlink"
	"github.com/snapcore/snapd/osutil"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/loop"
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
	hardwareClient    hardware.ClientWithResponsesInterface
	lsblk             lsblk.LSBLKInterpreterInterface
}

func NewVolumeService(ctx context.Context, broascasting BroadcasterServiceInterface, mount_repo repository.MountPointPathRepositoryInterface, hardwareClient hardware.ClientWithResponsesInterface, lsblk lsblk.LSBLKInterpreterInterface) VolumeServiceInterface {
	p := &VolumeService{
		ctx:               ctx,
		broascasting:      broascasting,
		volumesQueueMutex: sync.RWMutex{},
		mount_repo:        mount_repo,
		hardwareClient:    hardwareClient,
		lsblk:             lsblk,
	}
	//p.GetVolumesData()
	ctx.Value("wg").(*sync.WaitGroup).Add(1)
	go func() {
		defer ctx.Value("wg").(*sync.WaitGroup).Done()
		p.udevEventHandler()
	}()
	return p
}

func (ms *VolumeService) MountVolume(md dto.MountPointData) errors.E {

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
	err = conv.MountPointDataToMountPointPath(md, dbom_mount_data)
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
			dbom_mount_data.Flags.Add(dbom.MS_RDONLY)
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

	if dbom_mount_data.IsMounted && ok {
		slog.Warn("Volume already mounted according to DB and OS check", "device", real_device, "path", dbom_mount_data.Path)
		return errors.WithDetails(dto.ErrorMountFail,
			"Device", real_device,
			"Path", dbom_mount_data.Path,
			"Message", "Volume is already mounted",
		)
	}

	// Handle potential mount point conflicts if OS says it's mounted but DB doesn't
	if ok && !dbom_mount_data.IsMounted {
		slog.Warn("Mount point path is already in use by another mount, but not tracked in DB for this record.", "path", dbom_mount_data.Path)
		// Option 1: Fail
		// return errors.WithDetails(dto.ErrorMountFail, "Path", dbom_mount_data.Path, "Message", "Mount point path already in use")
		// Option 2: Try renaming (as below) - Keep existing logic
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

	flags, err := dbom_mount_data.Flags.Value()
	if err != nil {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"Device", real_device,
			"Path", dbom_mount_data.Path,
			"Message", "Invalid Flags",
			"Error", err,
		)
	}

	slog.Debug("Attempting to mount volume", "device", real_device, "path", dbom_mount_data.Path, "fstype", dbom_mount_data.FSType, "flags", flags)

	var mp *mount.MountPoint
	mountFunc := func() error { return os.MkdirAll(dbom_mount_data.Path, 0o666) }

	if dbom_mount_data.FSType == "" {
		// Use TryMount if FSType is not specified
		mp, err = mount.TryMount(real_device, dbom_mount_data.Path, "" /*data*/, uintptr(flags.(int64)), mountFunc)
	} else {
		// Use Mount if FSType is specified
		mp, err = mount.Mount(real_device, dbom_mount_data.Path, dbom_mount_data.FSType, "" /*data*/, uintptr(flags.(int64)), mountFunc)
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
		slog.Info("Successfully mounted volume", "device", real_device, "path", dbom_mount_data.Path, "fstype", dbom_mount_data.FSType)
		var convm converter.MountToDbomImpl
		// Update dbom_mount_data with details from the actual mount point if available
		err = convm.MountToMountPointPath(mp, dbom_mount_data)
		if err != nil {
			// Log error but proceed, as mount succeeded
			slog.Warn("Failed to convert mount details back to DBOM", "err", err)
			// Don't return error here, mount was successful
		}
		dbom_mount_data.IsMounted = true
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
		ms.NotifyClient()
	}
	return nil
}

// ... rest of the file remains the same ...

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
		dbom_mount_data.IsMounted = false
		err = ms.mount_repo.Save(dbom_mount_data)
		if err != nil {
			// Log error, but unmount succeeded. State might be inconsistent in DB.
			slog.Error("Unmount succeeded but failed to update DB state", "path", path, "err", err)
			// Don't return error here, as the primary operation (unmount) succeeded.
			// However, the DB is now potentially out of sync.
		}
	}
	ms.NotifyClient()
	return nil

}

func (self *VolumeService) udevEventHandler() {
	slog.Debug("Starting Udev event handler...") // Changed log level

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		slog.Error("Unable to connect to Netlink Kobject UEvent socket", "err", err)
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
					self.NotifyClient()
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
	slog.Debug("Getting volumes data...") // Added log
	ret := []dto.Disk{}
	conv := converter.HaHardwareToDtoImpl{}
	dbconv := converter.DtoToDbomConverterImpl{}
	lsblkconv := converter.LsblkToDtoConverterImpl{}

	hwser, err := self.hardwareClient.GetHardwareInfoWithResponse(self.ctx)
	if err != nil {
		// Log clearly that we are falling back or failing
		slog.Error("Failed to get hardware info from Home Assistant Supervisor", "err", err)
		// Decide on fallback: return error, return empty, or try alternative method?
		// For now, return the error as the primary source failed.
		return nil, errors.Wrap(err, "failed to get hardware info from HA Supervisor")
		// --- Fallback logic removed as per original code structure ---
	}

	// Check response status and result field
	if hwser.StatusCode() != 200 || hwser.JSON200 == nil || hwser.JSON200.Data == nil || hwser.JSON200.Data.Drives == nil {
		errMsg := "Received invalid hardware info response from HA Supervisor"
		slog.Error(errMsg, "status_code", hwser.StatusCode(), "response_body", string(hwser.Body))
		// Attempt to parse potential error message if available (e.g., JSON400, JSON500)
		// For now, return a generic error
		return nil, errors.New(errMsg)
	}

	slog.Debug("Processing drives from HA Supervisor", "drive_count", len(*hwser.JSON200.Data.Drives))
	for i, drive := range *hwser.JSON200.Data.Drives {
		if drive.Filesystems == nil || len(*drive.Filesystems) == 0 {
			slog.Debug("Skipping drive with no filesystems", "drive_index", i, "drive_id", drive.Id)
			continue
		}
		var diskDto dto.Disk
		err = conv.DriveToDisk(drive, &diskDto)
		if err != nil {
			slog.Warn("Error converting drive to disk DTO", "drive_index", i, "drive_id", drive.Id, "err", err)
			continue // Skip this drive
		}
		if diskDto.Partitions == nil || len(*diskDto.Partitions) == 0 {
			slog.Debug("Skipping drive DTO with no partitions after conversion", "drive_index", i, "drive_id", drive.Id)
			continue // Skip if conversion resulted in no partitions
		}

		ret = append(ret, diskDto)
	}

	slog.Debug("Syncing mount point data with database", "disk_count", len(ret))
	// Iterate through the DTOs and sync with DB
	for diskIdx := range ret { // Use index to modify slice elements
		disk := &ret[diskIdx] // Get pointer to modify
		if disk.Partitions == nil {
			continue
		}
		for partIdx := range *disk.Partitions {
			partition := &(*disk.Partitions)[partIdx] // Get pointer
			if partition.MountPointData == nil || len(*partition.MountPointData) == 0 {
				info, err := self.lsblk.GetInfoFromDevice(*partition.Device)
				if err != nil {
					slog.Warn("Error getting info from device", "device", *partition.Device, "err", err)
					continue
				}
				mountPointDto := &dto.MountPointData{}
				err = lsblkconv.LsblkInfoToMountPointData(info, mountPointDto)
				if err != nil {
					slog.Warn("Error converting Lsblk info to MountPointData", "device", *partition.Device, "err", err)
					continue
				}
				partition.MountPointData = &[]dto.MountPointData{*mountPointDto} // Wrap in slice
			}

			for mpIdx := range *partition.MountPointData {
				mountPointDto := &(*partition.MountPointData)[mpIdx] // Get pointer

				// Find existing DB record by path
				mountPointPathDB, err := self.mount_repo.FindByPath(mountPointDto.Path)
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					slog.Warn("Error searching for mount point in DB", "path", mountPointDto.Path, "err", err)
					mountPointDto.IsInvalid = true // Mark DTO as invalid due to DB error
					invalidError := errors.Wrap(err, "DB find error").Error()
					mountPointDto.InvalidError = &invalidError
					continue // Skip this mount point
				}

				isNewRecord := errors.Is(err, gorm.ErrRecordNotFound)
				if isNewRecord {
					mountPointPathDB = &dbom.MountPointPath{} // Create new DB object
					slog.Debug("Mount point not found in DB, will create new record", "path", mountPointDto.Path)
				} else {
					slog.Debug("Found existing mount point in DB", "path", mountPointDto.Path)
				}

				// Convert DTO data (from HA Supervisor) to DB object
				// This updates mountPointPathDB with latest info from DTO
				err = dbconv.MountPointDataToMountPointPath(*mountPointDto, mountPointPathDB)
				if err != nil {
					slog.Warn("Error converting DTO mount point data to DBOM", "path", mountPointDto.Path, "err", err)
					mountPointDto.IsInvalid = true                                           // Mark DTO as invalid due to conversion error
					invalidError := errors.Wrap(err, "DTO to DBOM conversion error").Error() // Updated to use Error() method
					mountPointDto.InvalidError = &invalidError
					continue // Skip this mount point
				}

				if mountPointPathDB.Type == "ADDON" {
					// Check OS mount status *before* saving, update IsMounted in DB object
					isMountedOS, osCheckErr := osutil.IsMounted(mountPointPathDB.Path)
					if osCheckErr != nil {
						// Log error but proceed, maybe path doesn't exist yet for new mounts
						slog.Warn("Error checking OS mount status", "path", mountPointPathDB.Path, "err", osCheckErr)
						// Keep IsMounted as potentially set by DTO conversion unless OS definitively says not mounted
						if !isMountedOS {
							mountPointPathDB.IsMounted = false
						}
					} else {
						// Trust the OS check
						mountPointPathDB.IsMounted = isMountedOS
						slog.Debug("OS mount status check", "path", mountPointPathDB.Path, "is_mounted", isMountedOS)
					}

					// Save the updated DB object (Create or Update)
					err = self.mount_repo.Save(mountPointPathDB)
					if err != nil {
						slog.Warn("Error saving mount point data to DB", "path", mountPointPathDB.Path, "data", mountPointPathDB, "err", err)
						mountPointDto.IsInvalid = true                            // Mark DTO as invalid due to DB save error
						invalidError := errors.Wrap(err, "DB save error").Error() // Updated to use Error() method
						mountPointDto.InvalidError = &invalidError
						continue // Skip this mount point
					}
				}

				// Convert the final DB state (after save and OS check) back to the DTO
				// This ensures the DTO reflects the actual saved state including IsMounted
				err = dbconv.MountPointPathToMountPointData(*mountPointPathDB, mountPointDto)
				if err != nil {
					// This conversion should ideally not fail if the reverse worked
					slog.Error("Error converting DBOM mount point data back to DTO", "path", mountPointPathDB.Path, "err", err)
					mountPointDto.IsInvalid = true                                           // Mark DTO as invalid
					invalidError := errors.Wrap(err, "DBOM to DTO conversion error").Error() // Updated to use Error() method
					mountPointDto.InvalidError = &invalidError                               // Set the invalid error
					// Continue, but DTO might be slightly inconsistent
					continue
				}
				slog.Debug("Successfully synced mount point with DB", "path", mountPointDto.Path, "is_mounted", mountPointDto.IsMounted)
			}

		}
	}

	slog.Debug("Finished getting and syncing volumes data.")
	return &ret, nil // Return nil error on success
}

func (self *VolumeService) NotifyClient() {
	slog.Debug("Notifying client about volume changes...")
	self.volumesQueueMutex.Lock()
	defer self.volumesQueueMutex.Unlock()

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
		return fmt.Errorf("il dispositivo %s esiste già", device)
	}

	// Estrai i numeri major e minor dal nome del dispositivo
	major, minor, err := extractMajorMinor(device)
	if err != nil {
		return fmt.Errorf("errore durante l'estrazione dei numeri major e minor: %v", err)
	}

	// Crea il dispositivo di blocco usando la syscall mknod
	dev := (major << 8) | minor
	err = syscall.Mknod(device, syscall.S_IFBLK|0660, dev)
	if err != nil {
		return fmt.Errorf("errore durante la creazione del dispositivo di blocco: %v", err)
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
