package service

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"fmt"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/tlog"
	"github.com/pilebones/go-udev/netlink"
	"github.com/prometheus/procfs"
	"github.com/shomali11/util/xhashes"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type VolumeServiceInterface interface {
	MountVolume(md *dto.MountPointData) errors.E
	UnmountVolume(id string, force bool, lazy bool) errors.E
	GetVolumesData() *[]dto.Disk
	PathHashToPath(pathhash string) (string, errors.E)
	//EjectDisk(diskID string) error
	UpdateMountPointSettings(path string, settingsUpdate dto.MountPointData) (*dto.MountPointData, errors.E)
	PatchMountPointSettings(path string, settingsPatch dto.MountPointData) (*dto.MountPointData, errors.E)
	//CreateAutomountFailureNotification(mountPath, device string, err errors.E)
	//CreateUnmountedPartitionNotification(mountPath, device string)
	//DismissAutomountNotification(mountPath string, notificationType string)
	//CheckUnmountedAutomountPartitions() errors.E
	// Test only
	MockSetProcfsGetMounts(f func() ([]*procfs.MountInfo, error))
	CreateBlockDevice(device string) error
}

type VolumeService struct {
	ctx               context.Context
	volumesQueueMutex sync.RWMutex
	refreshing        atomic.Bool
	broascasting      BroadcasterServiceInterface
	mount_repo        repository.MountPointPathRepositoryInterface
	hardwareClient    HardwareServiceInterface
	fs_service        FilesystemServiceInterface
	shareService      ShareServiceInterface
	issueService      IssueServiceInterface
	state             *dto.ContextState
	sfGroup           singleflight.Group
	haService         HomeAssistantServiceInterface
	hdidleService     HDIdleServiceInterface
	eventBus          events.EventBusInterface
	convDto           converter.DtoToDbomConverterImpl
	convMDto          converter.MountToDtoImpl
	procfsGetMounts   func() ([]*procfs.MountInfo, error)
	disks             *map[string]dto.Disk
	refreshVersion    uint32
}

type VolumeServiceProps struct {
	fx.In
	Ctx context.Context
	//Broadcaster       BroadcasterServiceInterface
	MountPointRepo    repository.MountPointPathRepositoryInterface
	HardwareClient    HardwareServiceInterface `optional:"true"`
	FilesystemService FilesystemServiceInterface
	ShareService      ShareServiceInterface
	IssueService      IssueServiceInterface
	State             *dto.ContextState
	HAService         HomeAssistantServiceInterface `optional:"true"`
	HDIdleService     HDIdleServiceInterface        `optional:"true"`
	EventBus          events.EventBusInterface
}

func NewVolumeService(
	lc fx.Lifecycle,
	in VolumeServiceProps,
) VolumeServiceInterface {
	p := &VolumeService{
		ctx: in.Ctx,
		//broascasting:      in.Broadcaster,
		volumesQueueMutex: sync.RWMutex{},
		mount_repo:        in.MountPointRepo,
		hardwareClient:    in.HardwareClient,
		fs_service:        in.FilesystemService,
		state:             in.State,
		shareService:      in.ShareService,
		issueService:      in.IssueService,
		haService:         in.HAService,
		hdidleService:     in.HDIdleService,
		eventBus:          in.EventBus,
		convDto:           converter.DtoToDbomConverterImpl{},
		convMDto:          converter.MountToDtoImpl{},
		procfsGetMounts:   procfs.GetMounts,
		disks:             &map[string]dto.Disk{},
		refreshVersion:    0,
	}

	var unsubscribe func()
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			unsubscribe = p.eventBus.OnMountPoint(func(mpe events.MountPointEvent) {
				// Avoid recursive refresh loops: skip handling mount events while we are refreshing volumes
				if p.refreshing.Load() {
					return
				}
				err := p.persistMountPoint(mpe.MountPoint)
				if err != nil {
					slog.Error("Failed to persist mount point on event", "mount_point", mpe.MountPoint, "err", err)
				}
				if mpe.MountPoint.Partition != nil && mpe.MountPoint.Partition.Id != nil {
					slog.Info("MountPointEvent received", "type", mpe.Type, "mount_point", mpe.MountPoint.Path, "device_id", *mpe.MountPoint.Partition.Id, "is_mounted", mpe.MountPoint.IsMounted)
					if mpe.MountPoint.Partition.DiskId != nil {
						disktmp, ok := (*p.disks)[*mpe.MountPoint.Partition.DiskId]
						if ok {
							parttmp, ok := (*disktmp.Partitions)[*mpe.MountPoint.Partition.Id]
							if ok {
								(*parttmp.MountPointData)[mpe.MountPoint.Path] = *mpe.MountPoint
							}
						}
					}
				}
				err = p.getVolumesData()
				if err != nil {
					slog.Error("Failed to refresh volumes data on mount point event", "err", err)
				}
				if !mpe.MountPoint.IsMounted && mpe.Type == events.EventTypes.ADD && mpe.MountPoint.IsToMountAtStartup != nil && *mpe.MountPoint.IsToMountAtStartup {
					err = p.MountVolume(mpe.MountPoint)
					if err != nil {
						slog.Error("Failed to mount volume on event", "mount_point", mpe.MountPoint, "err", err)
						p.createAutomountFailureNotification(mpe.MountPoint.Path, mpe.MountPoint.DeviceId, err)
					}
				}
			})
			err := p.getVolumesData()
			if err != nil {
				return err
			}
			p.ctx.Value("wg").(*sync.WaitGroup).Add(1)
			go func() {
				defer p.ctx.Value("wg").(*sync.WaitGroup).Done()
				p.udevEventHandler()
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if unsubscribe != nil {
				unsubscribe()
			}
			return nil
		},
	})

	return p
}

func (self *VolumeService) persistMountPoint(md *dto.MountPointData) errors.E {
	dbom_mount_data := &dbom.MountPointPath{}
	err := self.convDto.MountPointDataToMountPointPath(*md, dbom_mount_data)
	if err != nil {
		return errors.WithStack(err)
	}
	err = self.mount_repo.Save(dbom_mount_data)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (ms *VolumeService) MountVolume(md *dto.MountPointData) errors.E {
	// Early validation of required fields
	if ms.state.ProtectedMode {
		return errors.WithDetails(dto.ErrorOperationNotPermittedInProtectedMode,
			"Operation", "MountVolume",
			"Detail", "Mount operation is not permitted when ProtectedMode is enabled.",
		)
	}

	if md == nil {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"Message", "MountPointData is nil",
		)
	}

	if md.Path == "" {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"Message", "Mount point path is empty",
		)
	}

	if md.DeviceId == "" {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"Message", "Source device name is empty in request",
		)
	}

	if md.Partition == nil || md.Partition.Id == nil || *md.Partition.Id == "" {
		for _, disk := range *ms.disks {
			for _, part := range *disk.Partitions {
				if *part.Id == md.DeviceId {
					md.Partition = &part
					break
				}
			}
		}
	}
	/*
		// Validate Partition data
		if md.Partition == nil {
			// Populate partition from disk
			disks, err := ms.hardwareClient.GetHardwareInfo()
			if err != nil {
				return errors.WithStack(err)
			}

			for _, disk := range disks {
				for _, part := range *disk.Partitions {
					if *part.Id == md.DeviceId {
						md.Partition = &part
						break
					}
				}
			}
		}
	*/
	if md.Partition == nil {
		return errors.WithDetails(dto.ErrorDeviceNotFound,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"Message", "Source device does not exist on the system",
		)
	}

	if md.Partition.DevicePath == nil || *md.Partition.DevicePath == "" {
		return errors.WithDetails(dto.ErrorDeviceNotFound,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"Message", "Source device does not exist on the system",
		)
	}

	ok, errS := osutil.IsMounted(md.Path)
	if errS != nil {
		// Note: IsMounted might fail if the path doesn't exist yet, which is fine before mounting.
		// Consider if this check needs refinement based on expected state.
		// For now, we proceed assuming an error here might be ignorable if ok is false.
		if ok { // Only return error if it claims to be mounted but check failed
			//slog.Error("Error checking if path is mounted", "path", dbom_mount_data.Path, "err", errS)
			return errors.WithDetails(dto.ErrorMountFail, "Detail", "Error checking mount status", "Path", md.Path, "Error", errS)
		}
		slog.Debug("osutil.IsMounted check failed, but path not mounted, proceeding", "path", md.Path, "err", errS)
		ok = false // Ensure ok is false if IsMounted errored
	}

	if ok {
		slog.Warn("Volume already mounted according to OS check", "device", md.DeviceId, "path", md.Path)
		return errors.WithDetails(dto.ErrorAlreadyMounted,
			"Device", md.DeviceId,
			"Path", md.Path,
			"Message", "Volume is already mounted",
		)
	}

	/*
		// Rename logic if path is already mounted (even if DB state was inconsistent)
		orgPath := md.Path
		if ok { // Only rename if osutil.IsMounted returned true
			slog.Info("Attempting to rename mount path due to conflict", "original_path", orgPath)
			for i := 1; ; i++ {
				md.Path = orgPath + "_(" + strconv.Itoa(i) + ")"
				okCheck, errCheck := osutil.IsMounted(md.Path)
				if errCheck != nil {
					// Similar to above, error might be okay if path doesn't exist yet
					if okCheck {
						slog.Error("Error checking renamed path mount status", "path", md.Path, "err", errCheck)
						return errors.WithDetails(dto.ErrorMountFail, "Detail", "Error checking renamed mount status", "Path", md.Path, "Error", errCheck)
					}
					okCheck = false // Treat error as not mounted
				}
				if !okCheck {
					slog.Info("Found available renamed path", "new_path", md.Path)
					break // Found an unused path
				}
				if i > 100 { // Safety break
					slog.Error("Could not find available renamed mount path after 100 attempts", "original_path", orgPath)
					return errors.WithDetails(dto.ErrorMountFail, "Path", orgPath, "Message", "Could not find available renamed mount path")
				}
			}
		}
	*/

	// Initialize flags if nil to avoid nil pointer dereference
	if md.Flags == nil {
		md.Flags = &dto.MountFlags{}
		slog.Debug("Initialized nil Flags to empty MountFlags", "device", md.DeviceId, "path", md.Path)
	}

	flags, data, err := ms.fs_service.MountFlagsToSyscallFlagAndData(*md.Flags)
	if err != nil {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"Device", md.DeviceId,
			"Path", md.Path,
			"Message", "Invalid Flags",
			"Error", err,
		)
	}

	slog.Debug("Attempting to mount volume", "device", md.DeviceId, "path", md.Path, "fstype", md.FSType, "flags", flags, "data", data)

	var mp *mount.MountPoint
	// Ensure secure directory permissions when creating mount point
	mountFunc := func() error { return os.MkdirAll(md.Path, 0o750) }

	// Final validation before mount - ensure DevicePath is not nil
	if md.Partition.DevicePath == nil || *md.Partition.DevicePath == "" {
		return errors.WithDetails(dto.ErrorDeviceNotFound,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"Message", "Device path is nil or empty, cannot mount",
		)
	}

	if md.FSType == nil || *md.FSType == "" {
		// Use TryMount if FSType is not specified
		mp, errS = mount.TryMount(*md.Partition.DevicePath, md.Path, data, flags, mountFunc)
	} else {
		// Use Mount if FSType is specified
		mp, errS = mount.Mount(*md.Partition.DevicePath, md.Path, *md.FSType, data, flags, mountFunc)
	}

	if errS != nil {
		// Provide detailed error message with all context

		fsTypeStr := "auto"
		if md.FSType != nil {
			fsTypeStr = *md.FSType
		}

		slog.Error("Failed to mount volume",
			"device_id", md.DeviceId,
			"device_path", *md.Partition.DevicePath,
			"fstype", fsTypeStr,
			"mount_path", md.Path,
			"flags", flags,
			"data", data,
			"mount_error", errS,
			"mountpoint_details", mp)

		// Attempt to clean up directory if we created it and mount failed
		if _, statErr := os.Stat(md.Path); statErr == nil {
			// Directory exists, try to remove it
			if removeErr := os.Remove(md.Path); removeErr != nil {
				slog.Warn("Failed to cleanup mount directory after mount failure",
					"path", md.Path,
					"cleanup_error", removeErr)
			}
		}

		return errors.WithDetails(dto.ErrorMountFail,
			"Detail", fmt.Sprintf("Mount command failed: %v", errS),
			"DeviceId", md.DeviceId,
			"DevicePath", *md.Partition.DevicePath,
			"MountPath", md.Path,
			"FSType", fsTypeStr,
			"Flags", flags,
			"Data", data,
			"Error", errS.Error(),
		)
	} else {
		slog.Info("Successfully mounted volume", "device", md.DeviceId, "path", md.Path, "fstype", md.FSType, "flags", mp.Flags, "data", mp.Data)
		mount_data := &dto.MountPointData{}
		// Update dbom_mount_data with details from the actual mount point if available
		errS = ms.convMDto.MountToMountPointData(mp, mount_data, *ms.GetVolumesData())
		if errS != nil {
			// Log error but proceed, as mount succeeded
			slog.Warn("Failed to convert mount details back to DTO", "err", errS)
			// Don't return error here, mount was successful
		}
		/* Gia fatto in converter?
		mflags, errE := ms.fs_service.SyscallFlagToMountFlag(mp.Flags)
		if errE != nil {
			slog.Warn("Failed to convert mount flags back to DTO", "err", errE)
		} else {
			fl := ms.convDto.MountFlagsToMountDataFlags(mflags)
			dbom_mount_data.Flags = &fl
		}
		*/
		ms.eventBus.EmitMountPoint(events.MountPointEvent{
			Event:      events.Event{Type: events.EventTypes.UPDATE},
			MountPoint: mount_data,
		})

		/*
			err = ms.mount_repo.Save(dbom_mount_data)
			if err != nil {
				// Critical: Mount succeeded but DB save failed. State is inconsistent.
				// Attempt to unmount?
				slog.Error("CRITICAL: Mount succeeded but failed to save state to DB. Attempting unmount.", "device", md.DeviceId, "data", dbom_mount_data, "mp", mp, "save_error", err)
				unmountErr := mount.Unmount(dbom_mount_data.Path, true, false) // Force unmount
				if unmountErr != nil {
					slog.Error("Failed to auto-unmount after DB save failure", "path", dbom_mount_data.Path, "unmount_error", unmountErr)
					// Return original save error, but add context
					return errors.WithDetails(dto.ErrorDatabaseError, "Detail", "Failed to save mount state after successful mount, and auto-unmount failed",
						"Device", dbom_mount_data.DeviceId, "Path", dbom_mount_data.Path, "Error", err, "UnmountError", unmountErr.Error())

				}
				// Return original save error
				return errors.WithDetails(dto.ErrorDatabaseError, "Detail", "Failed to save mount state after successful mount. Volume has been unmounted.",
					"Device", dbom_mount_data.DeviceId, "Path", dbom_mount_data.Path, "Error", err)
			}
		*/
	}

	// Dismiss any existing failure notifications since the mount was successful
	ms.dismissAutomountNotification(md.DeviceId, "automount_failure")
	ms.dismissAutomountNotification(md.DeviceId, "unmounted_partition")

	return nil
}

func (ms *VolumeService) PathHashToPath(pathhash string) (string, errors.E) {
	dbom_mount_data, err := ms.mount_repo.All()
	if err != nil {
		return "", errors.WithStack(err)
	}
	for _, mount_data := range dbom_mount_data {
		if xhashes.SHA1(mount_data.Path) == pathhash {
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
	md := &dto.MountPointData{}
	errS := ms.convDto.MountPointPathToMountPointData(*dbom_mount_data, md, *ms.GetVolumesData())
	if errS != nil {
		return errors.WithStack(errS)
	}
	defer func() {
		ms.eventBus.EmitMountPoint(events.MountPointEvent{
			Event:      events.Event{Type: events.EventTypes.UPDATE},
			MountPoint: md,
		})
	}()

	slog.Debug("Attempting to unmount volume", "path", path, "force", force, "lazy", lazy)
	err = errors.WithStack(mount.Unmount(path, force, lazy))
	if err != nil {
		slog.Error("Failed to unmount volume", "path", path, "err", err)
		return errors.WithDetails(dto.ErrorUnmountFail, "Detail", err.Error(), "Path", path, "Error", err)
	} else {
		slog.Info("Successfully unmounted volume", "path", path, "device", dbom_mount_data.DeviceId)
		md.IsMounted = false
		/*
			if strings.HasPrefix(dbom_mount_data.Device, "/dev/loop") {
				// If the device is a loop device, remove the loop device
				err = errors.WithStack(loop.ClearFile(dbom_mount_data.Device))
				if err != nil {
					slog.Error("Failed to remove loop device", "device", dbom_mount_data.Device, "err", err)
					// Log but don't return error, as unmount succeeded
				} else {
					slog.Debug("Successfully removed loop device", "device", dbom_mount_data.Device)
				}
			}
		*/
		err = errors.WithStack(os.Remove(dbom_mount_data.Path)) // Remove the mount point directory
		if err != nil {
			slog.Error("Failed to remove mount point directory", "path", dbom_mount_data.Path, "err", err)
		} else {
			slog.Debug("Removed mount point directory", "path", dbom_mount_data.Path)
		}
	}

	/*
		// Update DB state only if the record was found initially
		if dbom_mount_data != nil {
			err = ms.mount_repo.Save(dbom_mount_data)
			if err != nil {
				// Log error, but unmount succeeded. State might be inconsistent in DB.
				slog.Error("Unmount succeeded but failed to update DB state", "path", path, "err", err)
				// Don't return error here, as the primary operation (unmount) succeeded.
				// However, the DB is now potentially out of sync.
			}

			// If this partition was marked for automount but is now unmounted, create a notification
			if dbom_mount_data.IsToMountAtStartup != nil && *dbom_mount_data.IsToMountAtStartup {
				ms.CreateUnmountedPartitionNotification(dbom_mount_data.Path, dbom_mount_data.DeviceId)
			}
		}
	*/
	return nil

}

func (self *VolumeService) udevEventHandler() {
	tlog.Trace("Starting Udev event handler...")

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		tlog.Error("Unable to connect to Netlink Kobject UEvent socket", "err", err)
		return // Exit goroutine if connection fails
	}
	defer conn.Close()

	queue := make(chan netlink.UEvent, 10)
	errorChan := make(chan error, 1)
	quit := conn.Monitor(queue, errorChan, nil)
	tlog.Trace("Udev monitor started successfully.")

	// Handling message from queue
	for {
		select {
		case <-self.ctx.Done():
			slog.Info("Udev event handler stopping due to context cancellation.", "err", self.ctx.Err())
			close(quit)
			return
		case uevent := <-queue:
			// Filter events - only interested in block devices for now
			if subsystem, ok := uevent.Env["SUBSYSTEM"]; ok && subsystem == "block" {
				action := uevent.Action
				devName, _ := uevent.Env["DEVNAME"]
				devType, _ := uevent.Env["DEVTYPE"]
				slog.Debug("Received Udev block event", "action", action, "devname", devName, "devtype", devType)

				// Process block device events
				if action == "add" || action == "remove" || action == "change" {
					slog.Info("Processing block device event", "action", action, "devname", devName)

					// Get current volumes data
					self.hardwareClient.InvalidateHardwareInfo()
					err := self.getVolumesData()
					if err != nil {
						slog.Error("Failed to get volumes data after udev event", "err", err)
						continue
					}
					/*
						// Create a map of currently mounted paths
						currentlyMounted := make(map[string]bool)
						if volumesData != nil {
							for _, disk := range *volumesData {
								if disk.Partitions != nil {
									for _, partition := range *disk.Partitions {
										if partition.MountPointData != nil {
											for _, mp := range *partition.MountPointData {
												if mp.IsMounted {
													currentlyMounted[mp.Path] = true
												}
											}
										}
									}
								}
							}
						}

						// Check all shares
						mountPoints, errE := self.mount_repo.All()
						if errE != nil {
							slog.Warn("Failed to get mount points from repository", "err", errE)
							continue
						}

						for _, mp := range mountPoints {
							// Skip if the mount point doesn't have any shares
							if len(mp.Shares) == 0 {
								continue
							}

							// Check if path is currently mounted
							isMounted := currentlyMounted[mp.Path]

							// Get existing mounted status from osutil
							wasMounted, err := osutil.IsMounted(mp.Path)
							if err != nil {
								slog.Warn("Failed to check mount status", "path", mp.Path, "err", err)
								wasMounted = false
							}

							if wasMounted && !isMounted {
								// Create an issue for unmounted volume
								issue := &dto.Issue{
									Title:       fmt.Sprintf("Volume unmounted: %s", mp.Path),
									Description: fmt.Sprintf("The volume at path %s was unexpectedly unmounted. This may affect shared resources.", mp.Path),
									DetailLink:  fmt.Sprintf("/storage/volumes?path=%s", mp.Path),
								}

								// Save the issue using issue repository
								if err := self.issueService.Create(issue); err != nil {
									slog.Error("Failed to create issue for unmounted volume", "path", mp.Path, "err", err)
								}

								// Disable shares for the unmounted volume
								_, err := self.shareService.DisableShareFromPath(mp.Path)
								if err != nil && !errors.Is(err, dto.ErrorShareNotFound) {
									slog.Error("Failed to disable share for unmounted volume", "path", mp.Path, "err", err)
								}
							}

							// Update mount point data if needed
							if err := self.mount_repo.Save(&mp); err != nil {
								slog.Error("Failed to update mount point", "path", mp.Path, "err", err)
							}
						}

						// Notify clients of changes
						go self.NotifyClient()
					*/
				}
			}
		case err := <-errorChan:
			// Parse errors from malformed uevents are expected and should not be treated as critical
			// These can occur from kernel/driver events with non-standard formatting
			if err != nil && strings.Contains(err.Error(), "unable to parse uevent") {
				// Extract more context if available
				errMsg := err.Error()
				if strings.Contains(errMsg, "invalid env data") {
					slog.Debug("Ignoring malformed uevent with invalid env data",
						"err", err,
						"detail", "This can occur when kernel sends events with non-standard formatting")
				} else {
					slog.Debug("Failed to parse uevent, skipping",
						"err", err,
						"detail", "Event format not recognized or incompatible")
				}
			} else {
				// Other errors (connection issues, etc.) are more serious
				slog.Error("Error received from Udev monitor", "err", err)
			}
		}
	}
}

func (self *VolumeService) GetVolumesData() *[]dto.Disk {
	if len(*self.disks) == 0 {
		err := self.getVolumesData()
		if err != nil {
			slog.Error("Failed to get volumes data in GetVolumesData", "err", err)
			return &[]dto.Disk{}
		}
	}
	value := slices.Collect(maps.Values(*self.disks))
	return &value
}

// getVolumesData retrieves and synchronizes volume data with caching and concurrency control.
// Disks and partitions are read from the hardware client and enriched with local mount point data.
// It also syncs mount point data with database records, saving new entries and removing obsolete ones.
func (self *VolumeService) getVolumesData() errors.E {
	tlog.Trace("Requesting GetVolumesData via singleflight...")

	v, err, _ := self.sfGroup.Do("GetVolumesData", func() (interface{}, error) {
		// Mark that a refresh cycle is in progress to avoid recursive event-triggered refreshes
		self.refreshing.Store(true)
		defer self.refreshing.Store(false)
		self.volumesQueueMutex.Lock()
		defer self.volumesQueueMutex.Unlock()
		self.refreshVersion++

		tlog.Trace("Executing GetVolumesData core logic (singleflight)...")

		ret := []dto.Disk{}
		//dbconv := converter.DtoToDbomConverterImpl{}

		// Use mock data in demo mode or when SRAT_MOCK is true
		if self.state.SupervisorURL == "demo" || os.Getenv("SRAT_MOCK") == "true" {
			demoParts := map[string]dto.Partition{
				"DemoPartition": {
					Id:         pointer.String("DemoPartition"),
					DevicePath: pointer.String("/dev/bogus"),
					System:     pointer.Bool(false),
					MountPointData: &map[string]dto.MountPointData{
						"/mnt/bogus": {
							Path:      "/mnt/bogus",
							FSType:    pointer.String("ext4"),
							IsMounted: false,
						},
					},
				},
			}
			ret = append(ret, dto.Disk{
				Id:         pointer.String("DemoDisk"),
				Partitions: &demoParts,
			})
			return &ret, nil
		}

		// Skip hardware client if it's not initialized
		if self.hardwareClient == nil {
			slog.Debug("Hardware client not initialized, continuing with empty disk list")
			return &ret, nil
		}

		// Get Host Hardware
		ret, errHw := self.hardwareClient.GetHardwareInfo()
		if errHw != nil {
			return nil, errHw
		}

		// Disks processing
		for _, disk := range ret {
			if disk.Partitions == nil {
				continue
			}
			if _, ok := (*self.disks)[*disk.Id]; !ok {
				// New disk, add it
				disk.RefreshVersion = self.refreshVersion
				(*self.disks)[*disk.Id] = disk
				for pid, part := range *disk.Partitions {
					if part.DiskId == nil || *part.DiskId == "" {
						part.DiskId = disk.Id
					}
					if part.DevicePath == nil || *part.DevicePath == "" {
						slog.Debug("Skipping partition with nil or empty device path", "disk_id", *disk.Id)
						continue
					}
					if part.MountPointData == nil {
						part.MountPointData = &map[string]dto.MountPointData{}
					}
					dmp, err := self.mount_repo.FindByDevice(*part.DevicePath)
					if err != nil {
						if !errors.Is(err, gorm.ErrRecordNotFound) {
							slog.Error("Failed to get mount point from repository", "device", *part.DevicePath, "err", err)
							continue
						}
					} else {
						// Existing mount point found, add to partition
						mountData := dto.MountPointData{}
						err := self.convDto.MountPointPathToMountPointData(*dmp, &mountData, *self.GetVolumesData())
						if err != nil {
							slog.Error("Failed to convert mount point data", "device", *part.DevicePath, "err", err)
							continue
						}
						(*part.MountPointData)[mountData.Path] = mountData
						(*disk.Partitions)[pid] = part
						self.eventBus.EmitMountPoint(events.MountPointEvent{
							Event:      events.Event{Type: events.EventTypes.ADD},
							MountPoint: &mountData,
						})
					}
				}
				self.eventBus.EmitDisk(events.DiskEvent{
					Event: events.Event{Type: events.EventTypes.ADD},
					Disk:  &disk,
				})
			} else {
				// Existing disk, update refresh version
				existing := (*self.disks)[*disk.Id]
				existing.RefreshVersion = self.refreshVersion
				(*self.disks)[*disk.Id] = existing
			}

		}

		mountInfos, err := self.procfsGetMounts()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// Partition processing
		for diskName, disk := range *self.disks {
			if disk.RefreshVersion != self.refreshVersion {
				// Disk not present in current scan, remove it
				removedDisk := disk
				delete(*self.disks, diskName)
				self.eventBus.EmitDisk(events.DiskEvent{
					Event: events.Event{Type: events.EventTypes.REMOVE},
					Disk:  &removedDisk,
				})
			}

			for pidx, part := range *disk.Partitions {
				if part.Id == nil || *part.Id == "" {
					slog.Debug("Skipping partition with nil or empty device id", "disk_id", diskName, "partition_index", pidx)
					continue
				}

				if part.MountPointData == nil {
					part.MountPointData = &map[string]dto.MountPointData{}
				}

				for _, prtstate := range mountInfos {
					found := false
					var foundPath string
					for path, mpd := range *part.MountPointData {
						if mpd.Path == prtstate.MountPoint {
							found = true
							foundPath = path
							break
						}
					}
					if found {
						mpd := (*part.MountPointData)[foundPath]
						if !mpd.IsMounted {
							mpd.IsMounted = true
							(*part.MountPointData)[foundPath] = mpd
							self.eventBus.EmitMountPoint(events.MountPointEvent{
								Event:      events.Event{Type: events.EventTypes.UPDATE},
								MountPoint: &mpd,
							})
						}
						mpd.RefreshVersion = self.refreshVersion
						(*part.MountPointData)[foundPath] = mpd
						continue
					}

					if prtstate.Source == *part.DevicePath || (part.LegacyDevicePath != nil && prtstate.Source == *part.LegacyDevicePath) {
						// Found matching mount info for partition
						mountPoint := dto.MountPointData{
							Path:           prtstate.MountPoint,
							DeviceId:       *part.Id,
							IsMounted:      true,
							Flags:          &dto.MountFlags{},
							CustomFlags:    &dto.MountFlags{},
							FSType:         pointer.String(prtstate.FSType),
							Type:           "ADDON",
							Partition:      &part,
							RefreshVersion: self.refreshVersion,
						}
						mountPoint.Flags.Scan(prtstate.Options)
						mountPoint.CustomFlags.Scan(prtstate.SuperOptions)
						(*part.MountPointData)[prtstate.MountPoint] = mountPoint
						self.eventBus.EmitMountPoint(events.MountPointEvent{
							Event:      events.Event{Type: events.EventTypes.ADD},
							MountPoint: &mountPoint,
						})
						continue
					}

				}

				for key, mpd := range *part.MountPointData {
					if mpd.RefreshVersion != self.refreshVersion {
						// Mark as not mounted and update refresh version in-place in the map
						mpd.IsMounted = false
						mpd.RefreshVersion = self.refreshVersion
						(*part.MountPointData)[key] = mpd
						self.eventBus.EmitMountPoint(events.MountPointEvent{
							Event:      events.Event{Type: events.EventTypes.UPDATE},
							MountPoint: &mpd,
						})
						continue
					}
				}
			}
		}

		/*
			// Add AddonMountPointsData and check from DB (Save New and Update Old )
			mountPointDataToSave := make([]dbom.MountPointPath, 0)
			//mountPointDataToDelete := make([]dbom.MountPointPath, 0)
			for idx, disk := range ret {
				if disk.Partitions == nil {
					continue
				}
				for pidx, part := range *disk.Partitions {
					if part.Id == nil || *part.Id == "" {
						slog.Debug("Skipping partition with nil or empty device id", "disk_id", disk.Id, "partition_index", pidx)
						continue
					}

					mountInfos, err := self.procfsGetMounts()
					if err != nil {
						slog.Error("Failed to get local mount points", "err", err)
						continue
					}
					//slog.Debug("Prtstatus entries found", "count", len(prtstatus))
					for _, prtstate := range mountInfos {
						//slog.Debug("Check", "legacy_device", *part.LegacyDevicePath, "part_device", prtstate.Device, "part_mountpoint", prtstate.Mountpoint, "db_device", *part.Id)
						if part.LegacyDevicePath != nil && part.DevicePath != nil &&
							(prtstate.Source == *part.LegacyDevicePath || prtstate.Source == *part.DevicePath) &&
							prtstate.MountPoint != "" {
							// Local mountpoint for partition found
							tlog.Trace("Checking partition", "part_device", prtstate.Source, "part_mountpoint", prtstate.MountPoint, "db_device", *part.Id, "legacy_device", *part.LegacyDeviceName)
							mountPoint := dto.MountPointData{}
							if mountPointPath, ok := existingDBmountPoints[*part.Id]; ok {
								// Existing mount point in DB, update details
								errConv := dbconv.MountPointPathToMountPointData(mountPointPath, &mountPoint, nil)
								if errConv != nil {
									slog.Error("Failed to convert mount point data", "err", errConv)
								}
							}
							//slog.Debug("Opts found for mount point", "mountPoint", mountPoint, "opts", prtstate.Options, "fstype", prtstate.FSType, "superOpts", prtstate.SuperOptions)
							mountPoint.Path = prtstate.MountPoint
							mountPoint.DeviceId = *part.Id
							mountPoint.IsMounted = true
							mountPoint.Flags = &dto.MountFlags{}
							mountPoint.Flags.Scan(prtstate.Options)
							mountPoint.CustomFlags = &dto.MountFlags{}
							mountPoint.CustomFlags.Scan(prtstate.SuperOptions)
							mountPoint.FSType = &prtstate.FSType
							mountPoint.Type = "ADDON"
							mountPoint.Partition = &part
							if (*disk.Partitions)[pidx].MountPointData == nil {
								(*disk.Partitions)[pidx].MountPointData = &[]dto.MountPointData{}
							}
							*(*disk.Partitions)[pidx].MountPointData = append(*(*disk.Partitions)[pidx].MountPointData, mountPoint)
							delete(existingDBmountPoints, *part.Id)
							tlog.Trace("Added mount point data to partition", "device_id", *part.Id, "prtstate", prtstate, "mountpoint", mountPoint)

							mountPointPath := &dbom.MountPointPath{}
							errConv := self.convDto.MountPointDataToMountPointPath(mountPoint, mountPointPath)
							if errConv != nil {
								slog.Error("Failed to convert mount point data for saving", "err", errConv)
							}
							mountPointDataToSave = append(mountPointDataToSave, *mountPointPath)
						}
					}
				}
				ret[idx] = disk
			}
		*/

		/*
			// Populate existing Shares
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

			// Save mountPointDataToSave
			if len(mountPointDataToSave) > 0 {
				tlog.Trace("Saving updated mount points to DB", "count", len(mountPointDataToSave))
				for _, mp := range mountPointDataToSave {
					tlog.Trace("Saving mount point to DB", "path", mp.Path, "device_id", mp.DeviceId)
					err = self.mount_repo.Save(&mp)
					if err != nil {
						slog.Error("Failed to save mount point to DB", "path", mp.Path, "device_id", mp.DeviceId, "err", err)
					}
				}
			}

			// Remove mountPointDataToDelete
			if len(existingDBmountPoints) > 0 {
				tlog.Trace("Cleaning up mount points in DB that no longer exist on system", "count", len(existingDBmountPoints))
				for _, mp := range existingDBmountPoints {
					tlog.Trace("Removing stale mount point from DB", "path", mp.Path, "device_id", mp.DeviceId)
					err = self.mount_repo.Delete(mp.Path)
					if err != nil {
						slog.Error("Failed to delete stale mount point from DB", "path", mp.Path, "device_id", mp.DeviceId, "err", err)
					}
				}
			}

			// Enrich disks with HDIdle status/config snapshot
			if self.hdidleService != nil {
				for i := range ret {
					if ret[i].Id == nil || *ret[i].Id == "" {
						continue
					}
					cfg, cerr := self.hdidleService.GetDeviceConfig(*ret[i].Id)
					if cerr != nil {
						slog.Debug("Failed to get HDIdle device config for disk", "disk_id", *ret[i].Id, "err", cerr)
						continue
					}
					if cfg != nil {
						ret[i].HDIdleStatus = cfg
					}
				}
			}
		*/

		//slog.Debug("Finished getting and syncing volumes data (core logic).")
		return &ret, nil
	})

	if err != nil {
		//slog.Error("Singleflight execution of GetVolumesData failed", "err", err, "shared", shared)
		return errors.WithStack(err)
	}

	_, ok := v.(*[]dto.Disk)
	if !ok {
		slog.Error("Singleflight returned unexpected type for GetVolumesData", "type", fmt.Sprintf("%T", v))
		return errors.New("internal error: singleflight returned unexpected type")
	}

	return nil
}

/*
func (self *VolumeService) NotifyClient() {
	slog.Debug("Notifying client about volume changes...")

	var data = self.GetVolumesData()

	slog.Debug("Broadcasting updated volumes data", "disk_count", len(*data))
	self.broascasting.BroadcastMessage(data)
}
*/

func (self *VolumeService) CreateBlockDevice(device string) error {
	// Controlla se il dispositivo esiste già
	if _, err := os.Stat(device); !os.IsNotExist(err) {
		slog.Warn("Loop device already exists", "device", device)
		return nil
	}

	// Estrai i numeri major e minor dal nome del dispositivo
	major, minor, err := self.extractMajorMinor(device)
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

func (self *VolumeService) extractMajorMinor(device string) (int, int, error) {
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

/*
func (self *VolumeService) EjectDisk(diskID string) error {
	slog.Info("Attempting to eject disk", "disk_id", diskID)

	defer func() {
		go self.NotifyClient()
	}()

	allDisks := self.GetVolumesData()

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
	return nil
}
*/

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
	if convErr := conv.MountPointPathToMountPointData(*dbMountData, &updatedDto, nil); convErr != nil {
		return nil, errors.WithStack(convErr)
	}
	ms.eventBus.EmitMountPoint(events.MountPointEvent{
		Event:      events.Event{Type: events.EventTypes.UPDATE},
		MountPoint: &updatedDto,
	})
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

	currentDto := dto.MountPointData{}
	if convErr := conv.MountPointPathToMountPointData(*dbMountData, &currentDto, nil); convErr != nil {
		return nil, errors.WithStack(convErr)
	}
	ms.eventBus.EmitMountPoint(events.MountPointEvent{
		Event:      events.Event{Type: events.EventTypes.UPDATE},
		MountPoint: &currentDto,
	})
	return &currentDto, nil
}

// createAutomountFailureNotification creates a persistent notification for failed automount operations
func (self *VolumeService) createAutomountFailureNotification(mountPath, device string, err errors.E) {
	if self.haService == nil {
		slog.Debug("Home Assistant service not available, skipping automount failure notification")
		return
	}

	notificationID := fmt.Sprintf("srat_automount_failure_%s", xhashes.SHA1(mountPath))
	title := "Automount Failed"

	var message string
	if errors.Is(err, dto.ErrorDeviceNotFound) {
		message = fmt.Sprintf("Device '%s' for mount point '%s' not found during startup. The device may have been removed or disconnected.", device, mountPath)
	} else if errors.Is(err, dto.ErrorMountFail) {
		message = fmt.Sprintf("Failed to mount device '%s' to '%s' during startup. Check device filesystem and permissions.", device, mountPath)
	} else {
		message = fmt.Sprintf("Automount failed for device '%s' to '%s': %s", device, mountPath, err.Error())
	}

	notifyErr := self.haService.CreatePersistentNotification(notificationID, title, message)
	if notifyErr != nil {
		slog.Error("Failed to create automount failure notification", "mount_path", mountPath, "device", device, "err", notifyErr)
	} else {
		slog.Info("Created automount failure notification", "mount_path", mountPath, "device", device, "notification_id", notificationID)
	}
}

/*
// CreateUnmountedPartitionNotification creates a persistent notification for unmounted partitions that are marked for automount
func (self *VolumeService) CreateUnmountedPartitionNotification(deviceId, legacyDevice string) {
	if self.haService == nil {
		slog.Debug("Home Assistant service not available, skipping unmounted partition notification")
		return
	}

	notificationID := fmt.Sprintf("srat_unmounted_partition_%s", xhashes.SHA1(deviceId))
	title := "Unmounted Partition with Automount Enabled"
	message := fmt.Sprintf("Partition '%s' (device: %s) is configured for automount but is currently unmounted. This may indicate a device issue or the device is not connected.", deviceId, legacyDevice)

	notifyErr := self.haService.CreatePersistentNotification(notificationID, title, message)
	if notifyErr != nil {
		slog.Error("Failed to create unmounted partition notification", "mount_path", deviceId, "device", legacyDevice, "err", notifyErr)
	} else {
		tlog.Trace("Created unmounted partition notification", "mount_path", deviceId, "device", legacyDevice, "notification_id", notificationID)
	}
}
*/

// DismissAutomountNotification dismisses an automount-related notification
func (self *VolumeService) dismissAutomountNotification(deviceId string, notificationType string) {
	if self.haService == nil {
		return
	}

	notificationID := fmt.Sprintf("srat_%s_%s", notificationType, xhashes.SHA1(deviceId))

	notifyErr := self.haService.DismissPersistentNotification(notificationID)
	if notifyErr != nil {
		slog.Warn("Failed to dismiss automount notification", "mount_path", deviceId, "notification_type", notificationType, "err", notifyErr)
	} else {
		slog.Debug("Dismissed automount notification", "mount_path", deviceId, "notification_type", notificationType, "notification_id", notificationID)
	}
}

/*

// CheckUnmountedAutomountPartitions checks for partitions marked for automount that are not currently mounted
func (self *VolumeService) CheckUnmountedAutomountPartitions() errors.E {
	// Get all mount points from the database that are marked for automount
	allMountPoints, err := self.mount_repo.All()
	if err != nil {
		return errors.Wrap(err, "failed to get all mount points from database")
	}

	for _, mountPoint := range allMountPoints {
		// Skip if not marked for automount
		if mountPoint.IsToMountAtStartup == nil || !*mountPoint.IsToMountAtStartup {
			continue
		}

		// Check if the path is currently mounted
		isMounted, mountCheckErr := osutil.IsMounted(mountPoint.Path)
		if mountCheckErr != nil {
			slog.Warn("Failed to check mount status for automount partition",
				"path", mountPoint.Path,
				"device", mountPoint.DeviceId,
				"err", mountCheckErr)
			continue
		}

		if !isMounted {
			tlog.Trace("Found unmounted partition marked for automount",
				"device", mountPoint.DeviceId,
				"mount_path", mountPoint.Path)

			// Create a notification for the unmounted partition
			self.CreateUnmountedPartitionNotification(mountPoint.DeviceId, mountPoint.FSType)
		} else {
			// If it's mounted, dismiss any existing unmounted partition notifications
			self.DismissAutomountNotification(mountPoint.DeviceId, "unmounted_partition")
		}
	}

	return nil
}
*/

func (ms *VolumeService) MockSetProcfsGetMounts(f func() ([]*procfs.MountInfo, error)) {
	ms.procfsGetMounts = f
}
