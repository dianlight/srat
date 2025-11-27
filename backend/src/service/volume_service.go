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
	"github.com/dianlight/srat/dbom/g"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/tlog"
	"github.com/pilebones/go-udev/netlink"
	"github.com/prometheus/procfs"
	"github.com/shomali11/util/xhashes"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VolumeServiceInterface interface {
	MountVolume(md *dto.MountPointData) errors.E
	UnmountVolume(id string, force bool, lazy bool) errors.E
	GetVolumesData() *[]dto.Disk
	PathHashToPath(pathhash string) (string, errors.E)
	//EjectDisk(diskID string) error
	PatchMountPointSettings(path string, settingsPatch dto.MountPointData) (*dto.MountPointData, errors.E)
	// Test only
	MockSetProcfsGetMounts(f func() ([]*procfs.MountInfo, error))
	CreateBlockDevice(device string) error
}

type VolumeService struct {
	ctx               context.Context
	db                *gorm.DB
	volumesQueueMutex sync.RWMutex
	refreshing        atomic.Bool
	broascasting      BroadcasterServiceInterface
	//mount_repo        repository.MountPointPathRepositoryInterface
	hardwareClient  HardwareServiceInterface
	fs_service      FilesystemServiceInterface
	shareService    ShareServiceInterface
	issueService    IssueServiceInterface
	state           *dto.ContextState
	sfGroup         singleflight.Group
	haService       HomeAssistantServiceInterface
	hdidleService   HDIdleServiceInterface
	eventBus        events.EventBusInterface
	convDto         converter.DtoToDbomConverterImpl
	convMDto        converter.MountToDtoImpl
	procfsGetMounts func() ([]*procfs.MountInfo, error)
	disks           *dto.DiskMap
	refreshVersion  uint32
	// test hooks for mount operations
	tryMountFunc func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error)
	doMountFunc  func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error)
	unmountFunc  func(target string, force, lazy bool) error
}

type VolumeServiceProps struct {
	fx.In
	Ctx context.Context
	Db  *gorm.DB
	//MountPointRepo    repository.MountPointPathRepositoryInterface
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
		//mount_repo:        in.MountPointRepo,
		db:              in.Db,
		hardwareClient:  in.HardwareClient,
		fs_service:      in.FilesystemService,
		state:           in.State,
		shareService:    in.ShareService,
		issueService:    in.IssueService,
		haService:       in.HAService,
		hdidleService:   in.HDIdleService,
		eventBus:        in.EventBus,
		convDto:         converter.DtoToDbomConverterImpl{},
		convMDto:        converter.MountToDtoImpl{},
		procfsGetMounts: procfs.GetMounts,
		disks:           &dto.DiskMap{},
		refreshVersion:  0,
		tryMountFunc:    mount.TryMount,
		doMountFunc:     mount.Mount,
		unmountFunc:     mount.Unmount,
	}

	var unsubscribe [2]func()
	unsubscribe[0] = p.eventBus.OnMountPoint(func(ctx context.Context, mpe events.MountPointEvent) errors.E {
		// Avoid recursive refresh loops: skip handling mount events while we are refreshing volumes
		if p.refreshing.Load() {
			return nil
		}
		err := p.persistMountPoint(mpe.MountPoint)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to persist mount point on event", "mount_point", mpe.MountPoint, "err", err)
			return err
		}
		if mpe.MountPoint.Partition != nil && mpe.MountPoint.Partition.Id != nil {
			slog.InfoContext(ctx, "MountPointEvent received", "type", mpe.Type, "mount_point", mpe.MountPoint.Path, "device_id", *mpe.MountPoint.Partition.Id, "is_mounted", mpe.MountPoint.IsMounted, "is_to_mount_at_startup", mpe.MountPoint.IsToMountAtStartup)
			if mpe.MountPoint.Partition.DiskId != nil {
				err := p.disks.AddMountPoint(*mpe.MountPoint.Partition.DiskId, *mpe.MountPoint.Partition.Id, *mpe.MountPoint)
				if err != nil {
					slog.WarnContext(ctx, "Failed to add mount point to disk map", "err", err)
				}
			}
		}
		if !mpe.MountPoint.IsMounted && (mpe.Type == events.EventTypes.ADD || mpe.Type == events.EventTypes.UPDATE) && mpe.MountPoint.IsToMountAtStartup != nil && *mpe.MountPoint.IsToMountAtStartup {
			err = p.MountVolume(mpe.MountPoint)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to mount volume on event", "mount_point", mpe.MountPoint, "err", err)
				p.createAutomountFailureNotification(mpe.MountPoint.Path, mpe.MountPoint.DeviceId, err)
			}
		}
		//		err = p.getVolumesData()
		//		if err != nil {
		//			slog.ErrorContext(ctx, "Failed to refresh volumes data on mount point event", "err", err)
		//			return err
		//		}
		return nil
	})
	unsubscribe[1] = p.eventBus.OnHomeAssistant(func(ctx context.Context, hae events.HomeAssistantEvent) errors.E {
		if hae.Type == events.EventTypes.START {
			err := p.getVolumesData()
			if err != nil {
				slog.ErrorContext(ctx, "Failed to refresh volumes data on Home Assistant start event", "err", err)
			}
		}
		return nil
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
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
			for _, unsub := range unsubscribe {
				if unsub != nil {
					unsub()
				}
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
	err = self.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).
		Create(dbom_mount_data).Error
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
		slog.DebugContext(ms.ctx, "osutil.IsMounted check failed, but path not mounted, proceeding", "path", md.Path, "err", errS)
		ok = false // Ensure ok is false if IsMounted errored
	}

	if ok {
		slog.WarnContext(ms.ctx, "Volume already mounted according to OS check", "device", md.DeviceId, "path", md.Path)
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
			slog.InfoContext(ms.ctx, "Attempting to rename mount path due to conflict", "original_path", orgPath)
			for i := 1; ; i++ {
				md.Path = orgPath + "_(" + strconv.Itoa(i) + ")"
				okCheck, errCheck := osutil.IsMounted(md.Path)
				if errCheck != nil {
					// Similar to above, error might be okay if path doesn't exist yet
					if okCheck {
								slog.ErrorContext(ms.ctx, "Error checking renamed path mount status", "path", md.Path, "err", errCheck)
						return errors.WithDetails(dto.ErrorMountFail, "Detail", "Error checking renamed mount status", "Path", md.Path, "Error", errCheck)
					}
					okCheck = false // Treat error as not mounted
				}
				if !okCheck {
					slog.InfoContext(ms.ctx, "Found available renamed path", "new_path", md.Path)
					break // Found an unused path
				}
				if i > 100 { // Safety break
					slog.ErrorContext(ms.ctx, "Could not find available renamed mount path after 100 attempts", "original_path", orgPath)
					return errors.WithDetails(dto.ErrorMountFail, "Path", orgPath, "Message", "Could not find available renamed mount path")
				}
			}
		}
	*/

	// Initialize flags if nil to avoid nil pointer dereference
	if md.Flags == nil {
		md.Flags = &dto.MountFlags{}
		slog.DebugContext(ms.ctx, "Initialized nil Flags to empty MountFlags", "device", md.DeviceId, "path", md.Path)
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

	slog.DebugContext(ms.ctx, "Attempting to mount volume", "device", md.DeviceId, "path", md.Path, "fstype", md.FSType, "flags", flags, "data", data)

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
		mp, errS = ms.tryMountFunc(*md.Partition.DevicePath, md.Path, data, flags, mountFunc)
	} else {
		// Use Mount if FSType is specified
		mp, errS = ms.doMountFunc(*md.Partition.DevicePath, md.Path, *md.FSType, data, flags, mountFunc)
	}

	if errS != nil {
		// Provide detailed error message with all context

		fsTypeStr := "auto"
		if md.FSType != nil {
			fsTypeStr = *md.FSType
		}

		slog.ErrorContext(ms.ctx, "Failed to mount volume",
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
				slog.WarnContext(ms.ctx, "Failed to cleanup mount directory after mount failure",
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
		slog.InfoContext(ms.ctx, "Successfully mounted volume", "device", md.DeviceId, "path", md.Path, "fstype", md.FSType, "flags", mp.Flags, "data", mp.Data)
		mount_data := &dto.MountPointData{}
		// Update dbom_mount_data with details from the actual mount point if available
		errS = ms.convMDto.MountToMountPointData(mp, mount_data, ms.disks)
		if errS != nil {
			// Log error but proceed, as mount succeeded
			slog.WarnContext(ms.ctx, "Failed to convert mount details back to DTO", "err", errS)
			// Don't return error here, mount was successful
		}
		ms.eventBus.EmitMountPoint(events.MountPointEvent{
			Event:      events.Event{Type: events.EventTypes.UPDATE},
			MountPoint: mount_data,
		})
		/*
			// Emit VolumeEvent for mount operation
			ms.eventBus.EmitVolume(events.VolumeEvent{
				Event:      events.Event{Type: events.EventTypes.UPDATE},
				MountPoint: mount_data,
				Operation:  "mount",
			})
		*/
	}

	// Dismiss any existing failure notifications since the mount was successful
	ms.dismissAutomountNotification(md.DeviceId, "automount_failure")
	ms.dismissAutomountNotification(md.DeviceId, "unmounted_partition")

	return nil
}

func (ms *VolumeService) PathHashToPath(pathhash string) (string, errors.E) {
	dbom_mount_data, err := gorm.G[dbom.MountPointPath](ms.db).Find(ms.ctx)
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
	// Look up mount point data from in-memory cache first
	md, ok := ms.disks.GetMountPointByPath(path)
	if !ok {
		slog.WarnContext(ms.ctx, "Mount path not found in cache, attempting unmount anyway", "path", path)
		// Proceed with unmount using the path directly even if not in cache
		md = dto.MountPointData{
			Path:      path,
			IsMounted: true,
		}
	} else {
		// Use the cached path (might differ if renamed)
		path = md.Path
	}

	// Use the in-memory mount point data, but keep a reference for deferred events
	mountPointData := md

	defer func() {
		ms.eventBus.EmitMountPoint(events.MountPointEvent{
			Event:      events.Event{Type: events.EventTypes.UPDATE},
			MountPoint: &mountPointData,
		})
		// Emit VolumeEvent for unmount operation
		if !mountPointData.IsMounted {
			ms.eventBus.EmitVolume(events.VolumeEvent{
				Event:      events.Event{Type: events.EventTypes.UPDATE},
				MountPoint: &mountPointData,
				Operation:  "unmount",
			})
		}
	}()

	slog.DebugContext(ms.ctx, "Attempting to unmount volume", "path", path, "force", force, "lazy", lazy)
	unmountErr := errors.WithStack(ms.unmountFunc(path, force, lazy))
	if unmountErr != nil {
		slog.ErrorContext(ms.ctx, "Failed to unmount volume", "path", path, "err", unmountErr)
		return errors.WithDetails(dto.ErrorUnmountFail, "Detail", unmountErr.Error(), "Path", path, "Error", unmountErr)
	}

	// Unmount succeeded
	slog.InfoContext(ms.ctx, "Successfully unmounted volume", "path", path, "device", md.DeviceId)
	mountPointData.IsMounted = false

	// Remove the mount point directory
	if err := os.Remove(path); err != nil {
		slog.ErrorContext(ms.ctx, "Failed to remove mount point directory", "path", path, "err", err)
	} else {
		slog.DebugContext(ms.ctx, "Removed mount point directory", "path", path)
	}

	// Update the cached mount point data with unmounted state
	// If Partition reference is not available, try to find it from the disks map using DeviceId
	partitionInfo := md.Partition
	if partitionInfo == nil && md.DeviceId != "" {
		// Try to find the partition from disks map
		for _, disk := range *ms.disks {
			if disk.Partitions != nil {
				for _, part := range *disk.Partitions {
					if part.Id != nil && *part.Id == md.DeviceId {
						partitionInfo = &part
						break
					}
				}
			}
		}
	}

	// Update cache if we have the partition information
	if partitionInfo != nil && partitionInfo.DiskId != nil && partitionInfo.Id != nil {
		mountPointData.IsMounted = false
		mountPointData.Partition = partitionInfo
		err := ms.disks.AddMountPoint(*partitionInfo.DiskId, *partitionInfo.Id, mountPointData)
		if err != nil {
			slog.WarnContext(ms.ctx, "Failed to update mount point in cache after unmount", "path", path, "err", err)
		}
	}

	return nil
}

func (self *VolumeService) udevEventHandler() {
	tlog.TraceContext(self.ctx, "Starting Udev event handler...")

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		tlog.ErrorContext(self.ctx, "Unable to connect to Netlink Kobject UEvent socket", "err", err)
		return // Exit goroutine if connection fails
	}
	defer conn.Close()

	queue := make(chan netlink.UEvent, 10)
	errorChan := make(chan error, 1)
	quit := conn.Monitor(queue, errorChan, nil)
	tlog.TraceContext(self.ctx, "Udev monitor started successfully.")

	// Handling message from queue
	for {
		select {
		case <-self.ctx.Done():
			slog.InfoContext(self.ctx, "Udev event handler stopping due to context cancellation.", "err", self.ctx.Err())
			close(quit)
			return
		case uevent := <-queue:
			// Filter events - only interested in block devices for now
			if subsystem, ok := uevent.Env["SUBSYSTEM"]; ok && subsystem == "block" {
				action := uevent.Action
				devName, _ := uevent.Env["DEVNAME"]
				devType, _ := uevent.Env["DEVTYPE"]
				slog.DebugContext(self.ctx, "Received Udev block event", "action", action, "devname", devName, "devtype", devType)

				// Process block device events
				if action == "add" || action == "remove" || action == "change" {
					slog.InfoContext(self.ctx, "Processing block device event", "action", action, "devname", devName)

					// Get current volumes data
					self.hardwareClient.InvalidateHardwareInfo()
					err := self.getVolumesData()
					if err != nil {
						slog.ErrorContext(self.ctx, "Failed to get volumes data after udev event", "err", err)
						continue
					}
				}
			}
		case err := <-errorChan:
			// Parse errors from malformed uevents are expected and should not be treated as critical
			// These can occur from kernel/driver events with non-standard formatting
			if err != nil && strings.Contains(err.Error(), "unable to parse uevent") {
				// Extract more context if available
				errMsg := err.Error()
				if strings.Contains(errMsg, "invalid env data") {
					slog.DebugContext(self.ctx, "Ignoring malformed uevent with invalid env data",
						"err", err,
						"detail", "This can occur when kernel sends events with non-standard formatting")
				} else {
					slog.DebugContext(self.ctx, "Failed to parse uevent, skipping",
						"err", err,
						"detail", "Event format not recognized or incompatible")
				}
			} else {
				// Other errors (connection issues, etc.) are more serious
				slog.ErrorContext(self.ctx, "Error received from Udev monitor", "err", err)
			}
		}
	}
}

func (self *VolumeService) GetVolumesData() *[]dto.Disk {
	if len(*self.disks) == 0 {
		err := self.getVolumesData()
		if err != nil {
			slog.ErrorContext(self.ctx, "Failed to get volumes data in GetVolumesData", "err", err)
			return &[]dto.Disk{}
		}
	}
	value := slices.Collect(maps.Values(*self.disks))
	return &value
}

// loadMountPointFromDB loads mount point data from the database for a partition
func (self *VolumeService) loadMountPointFromDB(part *dto.Partition) ([]*dto.MountPointData, errors.E) {
	if part.Id == nil || *part.Id == "" {
		return nil, nil
	}

	dmp, err := gorm.G[dbom.MountPointPath](self.db).
		Where(g.MountPointPath.DeviceId.Eq(*part.Id)).
		Find(self.ctx)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.ErrorContext(self.ctx, "Failed to get mount point from repository", "device", *part.DevicePath, "err", err)
			return nil, errors.WithStack(err)
		}
		return nil, nil
	}
	tlog.DebugContext(self.ctx, "Found mount point records in DB for device", "device", *part.DevicePath, "count", len(dmp))

	mountData, convErr := self.convDto.MountPointPathsToMountPointDatas(dmp)
	if convErr != nil {
		slog.ErrorContext(self.ctx, "Failed to convert mount point data", "device", *part.DevicePath, "err", convErr)
		return nil, errors.WithStack(convErr)
	}

	slog.DebugContext(self.ctx, "Loaded mount point from repository", "device", *part.DevicePath, "mountData", mountData)

	return mountData, nil
}

// processPartitionMountData loads mount data from DB and initializes partition mount point data
func (self *VolumeService) processPartitionMountData(disk *dto.Disk, pid string, part dto.Partition, emitEvents bool) error {
	if part.DiskId == nil || *part.DiskId == "" {
		part.DiskId = disk.Id
	}

	if part.DevicePath == nil || *part.DevicePath == "" {
		slog.DebugContext(self.ctx, "Skipping partition with nil or empty device path", "disk_id", *disk.Id)
		return nil
	}

	if part.MountPointData == nil {
		part.MountPointData = &map[string]dto.MountPointData{}
	}

	mountData, err := self.loadMountPointFromDB(&part)
	if err != nil {
		return err
	}

	if len(mountData) > 0 {
		for _, mountD := range mountData {
			if mountD1, ok := (*part.MountPointData)[mountD.Path]; ok {
				mountD.IsToMountAtStartup = mountD1.IsToMountAtStartup
				mountD.RefreshVersion = self.refreshVersion
				(*part.MountPointData)[mountD.Path] = mountD1
				if emitEvents {
					self.eventBus.EmitMountPoint(events.MountPointEvent{
						Event:      events.Event{Type: events.EventTypes.UPDATE},
						MountPoint: mountD,
					})
				}
			} else {
				mountD.RefreshVersion = self.refreshVersion
				(*part.MountPointData)[mountD.Path] = *mountD
				if emitEvents {
					self.eventBus.EmitMountPoint(events.MountPointEvent{
						Event:      events.Event{Type: events.EventTypes.ADD},
						MountPoint: mountD,
					})
				}
			}
		}
		(*disk.Partitions)[pid] = part
	}

	return nil
}

// processNewDisk adds a new disk to the cache and loads its partition mount data from DB
func (self *VolumeService) processNewDisk(disk dto.Disk) error {
	disk.RefreshVersion = self.refreshVersion
	err := self.disks.Add(disk)
	if err != nil {
		return err
	}

	if disk.Partitions != nil {
		for pid, part := range *disk.Partitions {
			if err := self.processPartitionMountData(&disk, pid, part, true); err != nil {
				slog.WarnContext(self.ctx, "Failed to process partition mount data for new disk", "disk_id", *disk.Id, "partition_id", pid, "err", err)
				continue
			}
		}
		self.eventBus.EmitDiskAndPartition(events.DiskEvent{
			Event: events.Event{Type: events.EventTypes.ADD},
			Disk:  &disk,
		})
	}

	return nil
}

// processExistingDisk updates an existing disk's refresh version and reloads mount data from DB
func (self *VolumeService) processExistingDisk(diskId string) error {
	existing, ok := self.disks.Get(diskId)
	if !ok {
		return errors.WithDetails(dto.ErrorNotFound, "Message", "disk not found", "DiskId", diskId)
	}
	existing.RefreshVersion = self.refreshVersion
	err := self.disks.Add(existing)
	if err != nil {
		return err
	}
	// Refresh mount data from DB for all partitions to ensure is_to_mount_at_startup is up to date
	if existing.Partitions != nil {
		for pid, part := range *existing.Partitions {
			if err := self.processPartitionMountData(&existing, pid, part, true); err != nil {
				slog.WarnContext(self.ctx, "Failed to process partition mount data for new disk", "disk_id", *existing.Id, "partition_id", pid, "err", err)
				continue
			}
		}
		self.eventBus.EmitDiskAndPartition(events.DiskEvent{
			Event: events.Event{Type: events.EventTypes.UPDATE},
			Disk:  &existing,
		})
	}

	return nil
}

// processMountInfos updates partition mount states based on current procfs mount information
func (self *VolumeService) processMountInfos(mountInfos []*procfs.MountInfo) {
	for diskName, disk := range *self.disks {
		if disk.RefreshVersion != self.refreshVersion {
			// Disk not present in current scan, remove it
			removedDisk := disk
			self.disks.Remove(diskName)
			self.eventBus.EmitDiskAndPartition(events.DiskEvent{
				Event: events.Event{Type: events.EventTypes.REMOVE},
				Disk:  &removedDisk,
			})
			continue
		}

		for pidx, part := range *disk.Partitions {
			if part.Id == nil || *part.Id == "" {
				slog.DebugContext(self.ctx, "Skipping partition with nil or empty device id", "disk_id", diskName, "partition_index", pidx)
				continue
			}

			if part.MountPointData == nil {
				part.MountPointData = &map[string]dto.MountPointData{}
			}

			// Update existing mount points with current mount info
			for _, prtstate := range mountInfos {
				if mpd, ok := (*part.MountPointData)[prtstate.MountPoint]; ok {
					if !mpd.IsMounted {
						mpd.IsMounted = true
						(*part.MountPointData)[prtstate.MountPoint] = mpd
						self.eventBus.EmitMountPoint(events.MountPointEvent{
							Event:      events.Event{Type: events.EventTypes.UPDATE},
							MountPoint: &mpd,
						})
					}
					mpd.RefreshVersion = self.refreshVersion
					(*part.MountPointData)[prtstate.MountPoint] = mpd
					continue
				}

				if prtstate.Source == *part.DevicePath || (part.LegacyDevicePath != nil && prtstate.Source == *part.LegacyDevicePath) {
					// Found matching mount info for partition
					iw := osutil.IsWritable(prtstate.MountPoint)
					mountPoint := dto.MountPointData{
						Path:             prtstate.MountPoint,
						DeviceId:         *part.Id,
						PathHash:         xhashes.SHA1(prtstate.MountPoint),
						IsWriteSupported: pointer.Bool(iw),
						IsMounted:        true,
						Flags:            &dto.MountFlags{},
						CustomFlags:      &dto.MountFlags{},
						FSType:           pointer.String(prtstate.FSType),
						Type:             "ADDON",
						Partition:        &part,
						RefreshVersion:   self.refreshVersion,
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

			// Mark mount points not found in procfs as unmounted
			for key, mpd := range *part.MountPointData {
				if mpd.RefreshVersion != self.refreshVersion {
					mpd.IsMounted = false
					mpd.RefreshVersion = self.refreshVersion
					(*part.MountPointData)[key] = mpd
					self.eventBus.EmitMountPoint(events.MountPointEvent{
						Event:      events.Event{Type: events.EventTypes.UPDATE},
						MountPoint: &mpd,
					})
				}
			}
			err := self.disks.AddPartition(*disk.Id, part)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to update partition in disk map", "disk_id", *disk.Id, "partition_id", *part.Id, "err", err)
			}
		}
	}
}

// getVolumesData retrieves and synchronizes volume data with caching and concurrency control.
// Disks and partitions are read from the hardware client and enriched with local mount point data.
// It also syncs mount point data with database records, saving new entries and removing obsolete ones.
func (self *VolumeService) getVolumesData() errors.E {
	tlog.TraceContext(self.ctx, "Requesting GetVolumesData via singleflight...")

	_, err, _ := self.sfGroup.Do("GetVolumesData", func() (interface{}, error) {
		// Mark that a refresh cycle is in progress to avoid recursive event-triggered refreshes
		self.refreshing.Store(true)
		defer self.refreshing.Store(false)
		self.volumesQueueMutex.Lock()
		defer self.volumesQueueMutex.Unlock()
		self.refreshVersion++

		tlog.TraceContext(self.ctx, "Executing GetVolumesData core logic (singleflight)...")

		ret := self.disks

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

			(*ret)["DemoDisk"] = dto.Disk{
				Id:         pointer.String("DemoDisk"),
				Partitions: &demoParts,
			}
			return &ret, nil
		}

		// Skip hardware client if it's not initialized
		if self.hardwareClient == nil {
			slog.DebugContext(self.ctx, "Hardware client not initialized, continuing with empty disk list")
			return &ret, nil
		}

		// Get Host Hardware
		hwDisks, errHw := self.hardwareClient.GetHardwareInfo()
		if errHw != nil {
			return nil, errHw
		}

		// Disks processing
		for _, disk := range hwDisks {
			if disk.Partitions == nil {
				continue
			}

			if _, ok := self.disks.Get(*disk.Id); !ok {
				// New disk, add it
				if err := self.processNewDisk(disk); err != nil {
					slog.WarnContext(self.ctx, "Failed to process new disk", "disk_id", *disk.Id, "err", err)
					continue
				}
			} else {
				// Existing disk, update refresh version and reload mount data from DB
				if err := self.processExistingDisk(*disk.Id); err != nil {
					slog.WarnContext(self.ctx, "Failed to process existing disk", "disk_id", *disk.Id, "err", err)
					continue
				}
			}
		}

		// Get current mount information from procfs
		mountInfos, err := self.procfsGetMounts()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		tlog.TraceContext(self.ctx, "Mount infos retrieved from procfs", "count", len(mountInfos))

		// Update partition mount states based on current mount info
		self.processMountInfos(mountInfos)

		// Build return slice from current disks map (order not guaranteed)
		return ret, nil
	})

	if err != nil {
		//slog.Error("Singleflight execution of GetVolumesData failed", "err", err, "shared", shared)
		return errors.WithStack(err)
	}

	return nil
}

func (self *VolumeService) CreateBlockDevice(device string) error {
	// Controlla se il dispositivo esiste già
	if _, err := os.Stat(device); !os.IsNotExist(err) {
		slog.WarnContext(self.ctx, "Loop device already exists", "device", device)
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

func (ms *VolumeService) PatchMountPointSettings(path string, patchData dto.MountPointData) (*dto.MountPointData, errors.E) {

	dbMountData, err := gorm.G[dbom.MountPointPath](ms.db).
		Where(g.MountPointPath.Path.Eq(path)).First(ms.ctx)
	if err != nil {
		return nil, errors.Wrapf(dto.ErrorNotFound, "mount configuration with path %s not found", path)
	}

	err = ms.convDto.MountPointDataToMountPointPath(patchData, &dbMountData)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	affected, err := gorm.G[*dbom.MountPointPath](ms.db).
		Where(g.MountPointPath.Path.Eq(path)).
		Updates(ms.ctx, &dbMountData)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(dto.ErrorNotFound, "mount configuration with path %s not found", path)
		}
		return nil, errors.WithStack(err)
	}
	if affected == 0 {
		return nil, errors.Wrapf(dto.ErrorNotFound, "mount configuration with path %s not found", path)
	}

	currentDto := dto.MountPointData{}
	if convErr := ms.convDto.MountPointPathToMountPointData(dbMountData, &currentDto, *ms.GetVolumesData()); convErr != nil {
		return nil, errors.WithStack(convErr)
	}
	// Update cached mount point data
	if currentDto.Partition != nil && currentDto.Partition.DiskId != nil && currentDto.Partition.Id != nil {
		err := ms.disks.AddMountPoint(*currentDto.Partition.DiskId, *currentDto.Partition.Id, currentDto)
		if err != nil {
			slog.WarnContext(ms.ctx, "Failed to update mount point in cache", "err", err)
		}
	} else {
		// Fallback: partition could not be resolved.
		updated := false
		if existing, ok := ms.disks.GetMountPointByPath(path); ok {
			if existing.Partition != nil && existing.Partition.DiskId != nil && existing.Partition.Id != nil {
				existing.IsToMountAtStartup = currentDto.IsToMountAtStartup
				err := ms.disks.AddMountPoint(*existing.Partition.DiskId, *existing.Partition.Id, existing)
				if err != nil {
					slog.WarnContext(ms.ctx, "Failed to update mount point in fallback cache update", "err", err)
				}
				updated = true
			}
		}
		if !updated {
			for dk, d := range *ms.disks {
				if d.Partitions == nil {
					continue
				}
				for pid, part := range *d.Partitions {
					if part.MountPointData == nil {
						continue
					}
					if existing, ok := (*part.MountPointData)[path]; ok {
						existing.IsToMountAtStartup = currentDto.IsToMountAtStartup
						err := ms.disks.AddMountPoint(dk, pid, existing)
						if err != nil {
							slog.WarnContext(ms.ctx, "Failed to update mount point in fallback cache update", "err", err)
						}
						updated = true
						break
					}
				}
				if updated {
					break
				}
			}
		}
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
		slog.DebugContext(self.ctx, "Home Assistant service not available, skipping automount failure notification")
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
		slog.ErrorContext(self.ctx, "Failed to create automount failure notification", "mount_path", mountPath, "device", device, "err", notifyErr)
	} else {
		slog.InfoContext(self.ctx, "Created automount failure notification", "mount_path", mountPath, "device", device, "notification_id", notificationID)
	}
}

// DismissAutomountNotification dismisses an automount-related notification
func (self *VolumeService) dismissAutomountNotification(deviceId string, notificationType string) {
	if self.haService == nil {
		return
	}

	notificationID := fmt.Sprintf("srat_%s_%s", notificationType, xhashes.SHA1(deviceId))

	notifyErr := self.haService.DismissPersistentNotification(notificationID)
	if notifyErr != nil {
		slog.WarnContext(self.ctx, "Failed to dismiss automount notification", "mount_path", deviceId, "notification_type", notificationType, "err", notifyErr)
	} else {
		slog.DebugContext(self.ctx, "Dismissed automount notification", "mount_path", deviceId, "notification_type", notificationType, "notification_id", notificationID)
	}
}

func (ms *VolumeService) MockSetProcfsGetMounts(f func() ([]*procfs.MountInfo, error)) {
	ms.procfsGetMounts = f
}

// MockSetMountOps allows tests to override mount operations.
func (ms *VolumeService) MockSetMountOps(
	tryMount func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
	mountFn func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
	unmountFn func(target string, force, lazy bool) error,
) {
	if tryMount != nil {
		ms.tryMountFunc = tryMount
	}
	if mountFn != nil {
		ms.doMountFunc = mountFn
	}
	if unmountFn != nil {
		ms.unmountFunc = unmountFn
	}
}
