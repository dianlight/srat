package service

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"

	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dbom/g"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/internal/darwinstubs/mount"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/tlog"
	"github.com/prometheus/procfs"
	"github.com/shomali11/util/xhashes"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/*
VolumeServiceInterface defines the interface for managing volumes and mount points.

Copilot file rules:
- Always validate input parameters for mount and unmount operations.
- Always update disks map puntually after an operation that changes state.
*/

type VolumeServiceInterface interface {
	MountVolume(md *dto.MountPointData) errors.E
	UnmountVolume(id string, force bool) errors.E
	GetVolumesData() []*dto.Disk
	GetDevicePathByDeviceID(deviceID string) (string, errors.E)
	PatchMountPointSettings(root string, path string, settingsPatch dto.MountPointData) (*dto.MountPointData, errors.E)
	// Test only
	MockSetProcfsGetMounts(f func() ([]*procfs.MountInfo, error))
}

type VolumeService struct {
	ctx             context.Context
	db              *gorm.DB
	hardwareClient  HardwareServiceInterface
	fs_service      FilesystemServiceInterface
	shareService    ShareServiceInterface
	state           *dto.ContextState
	sfGroup         singleflight.Group
	haService       HomeAssistantServiceInterface
	hdidleService   HDIdleServiceInterface
	eventBus        events.EventBusInterface
	convDto         converter.DtoToDbomConverterImpl
	mounter         VolumeMountManagerInterface
	procfsGetMounts func() ([]*procfs.MountInfo, error)
	disks           *dto.DiskMap
	refreshVersion  uint32
}

type VolumeServiceProps struct {
	fx.In
	Ctx context.Context
	Db  *gorm.DB
	//MountPointRepo    repository.MountPointPathRepositoryInterface
	HardwareClient    HardwareServiceInterface `optional:"true"`
	FilesystemService FilesystemServiceInterface
	ShareService      ShareServiceInterface
	State             *dto.ContextState
	HAService         HomeAssistantServiceInterface `optional:"true"`
	HDIdleService     HDIdleServiceInterface        `optional:"true"`
	EventBus          events.EventBusInterface
	Mounter           VolumeMountManagerInterface
	Disks             *dto.DiskMap
}

func NewVolumeService(
	lc fx.Lifecycle,
	in VolumeServiceProps,
) VolumeServiceInterface {
	p := &VolumeService{
		ctx:             in.Ctx,
		db:              in.Db,
		hardwareClient:  in.HardwareClient,
		fs_service:      in.FilesystemService,
		state:           in.State,
		shareService:    in.ShareService,
		haService:       in.HAService,
		hdidleService:   in.HDIdleService,
		eventBus:        in.EventBus,
		convDto:         converter.DtoToDbomConverterImpl{},
		mounter:         in.Mounter,
		procfsGetMounts: procfs.GetMounts,
		disks:           in.Disks,
		refreshVersion:  0,
	}

	var unsubscribe [7]func()
	unsubscribe[0] = p.eventBus.OnPartition(p.handlePartitionEvent)
	unsubscribe[1] = p.eventBus.OnMountPoint(p.handleMountPointEvent)
	unsubscribe[2] = p.eventBus.OnHomeAssistant(func(ctx context.Context, hae events.HomeAssistantEvent) errors.E {
		tlog.DebugContext(ctx, "Home Assistant started event received, getVolumesData called")
		if hae.Type == events.EventTypes.START {
			err := p.getVolumesData()
			if err != nil {
				slog.ErrorContext(ctx, "Failed to refresh volumes data on Home Assistant start event", "err", err)
			}
		}
		return nil
	})
	unsubscribe[3] = p.eventBus.OnShare(func(ctx context.Context, se events.ShareEvent) errors.E {
		tlog.DebugContext(ctx, "Share event received update cache volumes", "event_type", se.Type, "share", se.Share)
		switch se.Type {
		case events.EventTypes.REMOVE:
			ok, disk := p.disks.RemoveMountPointShare(se.Share.Name)
			if !ok {
				slog.WarnContext(ctx, "Failed to remove share from mount point in cache", "share", se.Share.Name)
			} else {
				p.eventBus.EmitDisk(events.DiskEvent{
					Event: events.Event{
						Type: events.EventTypes.UPDATE,
					},
					Disk: disk,
				})
			}
		case events.EventTypes.ADD, events.EventTypes.UPDATE:
			disk, err := p.disks.AddMountPointShare(se.Share)
			if err != nil {
				if se.Share.Usage != "internal" {
					slog.WarnContext(ctx, "Failed to add/update share in mount point in cache", "share", se.Share, "err", err)
				}
				return nil
			}
			p.eventBus.EmitDisk(events.DiskEvent{
				Event: events.Event{
					Type: events.EventTypes.UPDATE,
				},
				Disk: disk,
			})
		}
		return nil
	})
	unsubscribe[4] = p.eventBus.OnSmart(func(ctx context.Context, se events.SmartEvent) errors.E {
		// Only update disk cache when the event carries SmartInfo (non-empty DiskId).
		// Self-test progress events carry SmartTestStatus with an empty SmartInfo.
		if se.SmartInfo.DiskId != "" {
			if err := p.disks.AddSmartInfo(&se.SmartInfo); err != nil {
				slog.WarnContext(ctx, "Failed to add SMART info to disk cache", "error", err)
			}
		}
		return nil
	})
	unsubscribe[5] = p.eventBus.OnPower(func(ctx context.Context, pe events.PowerEvent) errors.E {
		// Handle PowerEvent
		if err := p.disks.AddHDIdleDevice(&pe.PowerInfo); err != nil {
			slog.WarnContext(ctx, "Failed to add HDIdle device info to disk cache", "error", err)
		}
		return nil
	})
	unsubscribe[6] = p.eventBus.OnFilesystemTask(p.handleFilesystemTaskEvent)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			err := p.getVolumesData()
			if err != nil {
				return err
			}
			if wg, ok := p.ctx.Value(ctxkeys.WaitGroup).(*sync.WaitGroup); ok && wg != nil {
				wg.Go(func() {
					p.udevEventHandler()
				})
			}
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
	// close mounthpath loop before save
	if dbom_mount_data.ExportedShare != nil {
		dbom_mount_data.ExportedShare.MountPointData = *dbom_mount_data
		dbom_mount_data.ExportedShare.MountPointDataPath = &dbom_mount_data.Path
		dbom_mount_data.ExportedShare.MountPointDataRoot = dbom_mount_data.Root
	}

	//slog.DebugContext(self.ctx, "Persisting mount point to database", "mount_point", md.Path, "device_id", md.DeviceId, "is_mounted", md.IsMounted, "is_to_mount_at_startup", (md.IsToMountAtStartup != nil && *md.IsToMountAtStartup))
	tlog.TraceContext(self.ctx, "Mount point data", "data", spew.Sdump(dbom_mount_data))
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

	if md.Root == "" {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"DeviceId", md.DeviceId,
			"Root", md.Root,
			"Message", "Mount point root is empty",
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

	mountFsType := ""
	if md.FSType != nil {
		mountFsType = *md.FSType
	}

	// Final validation: ensure DevicePath is non-nil and exists on the OS before
	// delegating to the mount manager.
	if md.Partition.DevicePath == nil || *md.Partition.DevicePath == "" {
		return errors.WithDetails(dto.ErrorDeviceNotFound,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"Message", "Device path is nil or empty, cannot mount",
		)
	}
	if _, statErr := os.Stat(*md.Partition.DevicePath); statErr != nil {
		if os.IsPermission(statErr) {
			return errors.WithDetails(dto.ErrorOperationNotPermitted,
				"DeviceId", md.DeviceId,
				"Path", md.Path,
				"DevicePath", *md.Partition.DevicePath,
				"Message", "Permission denied to access device",
				"Error", statErr.Error(),
			)
		}
		return errors.WithDetails(dto.ErrorDeviceNotFound,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"DevicePath", *md.Partition.DevicePath,
			"Message", "Device path does not exist",
			"Error", statErr.Error(),
		)
	}

	if err := ms.mounter.Mount(md, flags, data, mountFsType); err != nil {
		return err
	}

	// Dismiss any existing failure notifications since the mount was successful.
	ms.dismissAutomountNotification(md.DeviceId, "automount_failure")
	ms.dismissAutomountNotification(md.DeviceId, "unmounted_partition")

	return nil
}

func (ms *VolumeService) UnmountVolume(path string, force bool) errors.E {
	// Look up mount point data from in-memory cache first
	md, ok := ms.disks.GetMountPointByPath(path)
	if ok && md.Share != nil && md.Share.Status.IsHAMounted {
		slog.DebugContext(ms.ctx, "Found mount point as HAMounted", "path", path)
		md.IsInvalid = true
		_ = ms.eventBus.EmitShare(events.ShareEvent{
			Event: events.Event{Type: events.EventTypes.REMOVE},
			Share: md.Share,
		})
	} else if !ok {
		slog.WarnContext(ms.ctx, "Mount point not found in cache, try to umount", "path", path)
		md = &dto.MountPointData{Path: path}
	}
	return ms.unmountVolume(md, force)
}

func (ms *VolumeService) unmountVolume(md *dto.MountPointData, force bool) errors.E {
	return ms.mounter.Unmount(md, force)
}

func (self *VolumeService) findPartitionByDevName(devName string) (*dto.Partition, string, bool) {
	if self.disks == nil || devName == "" {
		return nil, "", false
	}

	for diskID, disk := range *self.disks {
		if disk.Partitions == nil {
			continue
		}
		for _, partition := range *disk.Partitions {
			if matchPartitionWithDevName(&partition, devName) {
				p := partition
				return &p, diskID, true
			}
		}
	}

	return nil, "", false
}

func (self *VolumeService) handlePartitionUdevAddEvent(devName string) bool {
	partition, _, found := self.findPartitionByDevName(devName)
	if !found || partition == nil || partition.Id == nil || *partition.Id == "" {
		return false
	}

	if partition.MountPointData == nil || len(*partition.MountPointData) == 0 {
		return false
	}

	handled := false
	for _, mountPoint := range *partition.MountPointData {
		if mountPoint.IsToMountAtStartup == nil || !*mountPoint.IsToMountAtStartup || mountPoint.IsMounted {
			continue
		}

		mountCopy := mountPoint
		mountCopy.Partition = partition
		mountCopy.DeviceId = *partition.Id
		if mountCopy.Path == "" {
			continue
		}

		err := self.MountVolume(&mountCopy)
		if err != nil {
			slog.WarnContext(self.ctx, "Failed automount retry for partition add event", "devname", devName, "path", mountCopy.Path, "err", err)
			continue
		}

		handled = true
	}

	return handled
}

func (self *VolumeService) handlePartitionUdevRemoveEvent(devName string) {
	partition, diskID, found := self.findPartitionByDevName(devName)
	if !found || partition == nil || partition.Id == nil || *partition.Id == "" {
		if self.hardwareClient != nil {
			self.hardwareClient.InvalidateHardwareInfo()
		}
		if err := self.getVolumesData(); err != nil {
			slog.ErrorContext(self.ctx, "Failed to refresh volume cache after unknown partition removal", "devname", devName, "err", err)
		}
		return
	}

	if partition.MountPointData != nil {
		for _, mountPoint := range *partition.MountPointData {
			if mountPoint.Path == "" || !mountPoint.IsMounted {
				continue
			}
			mountCopy := mountPoint
			if err := self.unmountVolume(&mountCopy, true); err != nil {
				slog.WarnContext(self.ctx, "Failed to unmount path during partition remove handling", "path", mountCopy.Path, "devname", devName, "err", err)
			}
		}
	}

	removed := self.disks.RemovePartition(diskID, *partition.Id)
	if !removed {
		slog.DebugContext(self.ctx, "Partition removal event did not find cache entry to delete", "disk_id", diskID, "partition_id", *partition.Id, "devname", devName)
	}

	if self.hardwareClient != nil {
		self.hardwareClient.InvalidateHardwareInfo()
	}
	if err := self.getVolumesData(); err != nil {
		slog.ErrorContext(self.ctx, "Failed to refresh volume cache after partition remove event", "devname", devName, "err", err)
	}
}

func (self *VolumeService) GetVolumesData() []*dto.Disk {
	if len(*self.disks) == 0 {
		err := self.getVolumesData()
		if err != nil {
			slog.ErrorContext(self.ctx, "Failed to get volumes data in GetVolumesData", "err", err)
			return []*dto.Disk{}
		}
	}
	value := slices.Collect(maps.Values(*self.disks))
	return value
}

// loadMountPointFromDB loads mount point data from the database for a partition
func (self *VolumeService) loadMountPointFromDB(part *dto.Partition) (map[string]*dto.MountPointData, errors.E) {
	if part.Id == nil || *part.Id == "" {
		return nil, nil
	}

	dmp, err := gorm.G[dbom.MountPointPath](self.db).
		Preload("ExportedShare", nil).
		Where(g.MountPointPath.DeviceId.Eq(*part.Id)).
		Find(self.ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(dmp) == 0 {
		tlog.TraceContext(self.ctx, "No mount point records found in DB for device", "device", *part.Id, "name", part.Name)
		return make(map[string]*dto.MountPointData), nil
	}

	tlog.TraceContext(self.ctx, "Found mount point records in DB for device", "device", *part.Id, "name", part.Name, "count", len(dmp))
	mountData, convErr := self.convDto.MountPointPathsToMountPointDataMap(dmp)
	if convErr != nil {
		slog.ErrorContext(self.ctx, "Failed to convert mount point data", "device", *part.Id, "err", convErr)
		return nil, errors.WithStack(convErr)
	}

	tlog.TraceContext(self.ctx, "Loaded mount point from repository", "device", *part.Id, "mountData", mountData)
	return mountData, nil
}

// getVolumesData retrieves and synchronizes volume data with caching and concurrency control.
// Disks and partitions are read from the hardware client and enriched with local mount point data.
// It also syncs mount point data with database records, saving new entries and removing obsolete ones.
func (self *VolumeService) getVolumesData() errors.E {
	tlog.TraceContext(self.ctx, "Requesting GetVolumesData via singleflight...")

	_, err, _ := self.sfGroup.Do("GetVolumesData", func() (any, error) {
		self.refreshVersion++
		filesystemSupportCache := make(map[string]*dto.FilesystemInfo)

		tlog.TraceContext(self.ctx, "Executing GetVolumesData core logic (singleflight)...")

		// Skip hardware client if it's not initialized
		if self.hardwareClient == nil {
			slog.DebugContext(self.ctx, "Hardware client not initialized, continuing with empty disk list")
			return self.disks, nil
		}

		// Get Host Hardware
		hwDisks, errHw := self.hardwareClient.GetHardwareInfo()
		if errHw != nil {
			return nil, errHw
		}
		if hwDisks == nil {
			tlog.TraceContext(self.ctx, "Hardware client returned nil disks, continuing with empty disk list")
			return self.disks, nil
		}

		tlog.DebugContext(self.ctx, "Retrieved hardware disks from hardware client", "disk_count", len(hwDisks))
		// Disks processing
		for _, disk := range hwDisks {
			if disk.Partitions == nil {
				continue
			}
			tlog.TraceContext(self.ctx, "Processing disk from hardware client", "disk_id", *disk.Id, "partition_count", len(*disk.Partitions))
			disk.RefreshVersion = self.refreshVersion

			currentDisk, updateDisk := self.disks.Get(*disk.Id)

			err := self.disks.AddOrUpdate(&disk)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to update existing disk in cache", "disk_id", *disk.Id, "err", err)
			}

			for pid, part := range *disk.Partitions {
				if part.FsType != nil && *part.FsType != "" {
					if cached, ok := filesystemSupportCache[*part.FsType]; ok {
						part.FilesystemInfo = cached
					} else {
						info, err := self.fs_service.GetSupportAndInfo(self.ctx, *part.FsType)
						if err != nil || info == nil || info.Support == nil {
							part.FilesystemInfo = &dto.FilesystemInfo{}
						} else {
							part.FilesystemInfo = info
						}
						filesystemSupportCache[*part.FsType] = part.FilesystemInfo
					}
				}
				(*disk.Partitions)[pid] = part
				//			if err := self.processPartitionMountData(&disk, pid, part, true); err != nil {
				//				slog.WarnContext(self.ctx, "Failed to process partition mount data for new disk", "disk_id", *disk.Id, "partition_id", pid, "err", err)
				//				continue
				//			}
				if currentDisk != nil && updateDisk {
					self.eventBus.EmitPartition(events.PartitionEvent{
						Event:     events.Event{Type: events.EventTypes.UPDATE},
						Partition: &part,
						Disk:      &disk,
					})
				} else {
					self.eventBus.EmitPartition(events.PartitionEvent{
						Event:     events.Event{Type: events.EventTypes.ADD},
						Partition: &part,
						Disk:      &disk,
					})
				}
			}
		}
		return nil, nil
	})

	if err != nil {
		//slog.Error("Singleflight execution of GetVolumesData failed", "err", err, "shared", shared)
		return errors.WithStack(err)
	}

	return nil
}

func (self *VolumeService) handleFilesystemTaskEvent(ctx context.Context, e events.FilesystemTaskEvent) errors.E {
	if e.Task == nil || !strings.EqualFold(e.Task.Operation, "format") || !strings.EqualFold(e.Task.Status, "success") {
		return nil
	}

	slog.InfoContext(ctx, "Refreshing volume cache after successful format task", "device", e.Task.Device, "filesystem_type", e.Task.FilesystemType)

	if self.hardwareClient != nil {
		self.hardwareClient.InvalidateHardwareInfo()
	}

	if err := self.getVolumesData(); err != nil {
		slog.ErrorContext(ctx, "Failed to refresh volume cache after format success", "device", e.Task.Device, "err", err)
		return err
	}

	disk := self.findDiskForDevicePath(e.Task.Device)
	if disk == nil {
		slog.DebugContext(ctx, "No disk found to broadcast after format refresh", "device", e.Task.Device)
		return nil
	}

	self.eventBus.EmitDisk(events.DiskEvent{
		Event: events.Event{Type: events.EventTypes.UPDATE},
		Disk:  disk,
	})

	return nil
}

func (self *VolumeService) findDiskForDevicePath(devicePath string) *dto.Disk {
	if self.disks == nil || *self.disks == nil {
		return nil
	}

	normalizedDevice := strings.TrimSpace(devicePath)
	var fallback *dto.Disk
	for _, disk := range *self.disks {
		if fallback == nil {
			fallback = disk
		}
		if disk.Partitions == nil {
			continue
		}
		for _, partition := range *disk.Partitions {
			if strings.TrimSpace(self.disks.GetPartitionDevicePath(&partition)) == normalizedDevice {
				return disk
			}
		}
	}

	return fallback
}

func (self *VolumeService) handlePartitionEvent(ctx context.Context, e events.PartitionEvent) errors.E {

	tlog.TraceContext(ctx, "Processing partition event for mount data sync", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "event_type", e.Type)

	if e.Partition.DevicePath == nil || *e.Partition.DevicePath == "" {
		slog.DebugContext(ctx, "Skipping partition with nil or empty device path", "disk_id", *e.Disk.Id)
		return nil
	}
	if e.Partition.DiskId == nil || *e.Partition.DiskId == "" {
		e.Partition.DiskId = e.Disk.Id
	}

	mountData, err := self.loadMountPointFromDB(e.Partition)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to load mount point data from DB for partition", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "err", err)
		return err
	}
	// Add missing mount points from DB to in-memory cache
	for _, md := range mountData {
		err := self.disks.AddOrUpdateMountPoint(*e.Partition.DiskId, *e.Partition.Id, *md)
		if err != nil {
			slog.WarnContext(self.ctx, "Failed to add mount point to disk map during partition event handling", "disk_id", *e.Partition.DiskId, "partition_id", *e.Partition.Id, "mount_path", md.Path, "err", err)
			continue
		}
	}

	// Get current mount information from procfs
	mountInfos, errS := self.procfsGetMounts()
	if errS != nil {
		slog.ErrorContext(ctx, "Failed to get current mount information from procfs", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "err", errS)
		return errors.WithStack(errS)
	}

	// Update existing mount points with current mount info
	tlog.TraceContext(ctx, "Synchronizing mount points for partition", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "mount_data_count", len(mountData), "procfs_mounts_count", len(mountInfos))
	for _, prtstate := range mountInfos {
		iw := osutil.IsWritable(prtstate.MountPoint)
		if mountPoint, ok := self.disks.GetMountPoint(*e.Partition.DiskId, *e.Partition.Id, prtstate.MountPoint); ok {
			tlog.TraceContext(ctx, "Found existing mount point in cache for partition, updating state", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "mount_path", mountPoint.Path, "is_mounted", mountPoint.IsMounted)
			oldstate := mountPoint.IsMounted
			mountPoint.IsMounted = true
			mountPoint.Path = prtstate.MountPoint
			mountPoint.Root = prtstate.Root
			mountPoint.RefreshVersion = self.refreshVersion
			mountPoint.IsWriteSupported = &iw
			if err := mountPoint.Flags.Scan(prtstate.Options); err != nil {
				slog.WarnContext(ctx, "Failed to scan mount flags", "mount_path", prtstate.MountPoint, "error", err)
			}
			if err := mountPoint.CustomFlags.Scan(prtstate.SuperOptions); err != nil {
				slog.WarnContext(ctx, "Failed to scan custom mount flags", "mount_path", prtstate.MountPoint, "error", err)
			}
			mountPoint.FSType = &prtstate.FSType
			mountPoint.Type = "ADDON"
			err := self.disks.AddOrUpdateMountPoint(*e.Partition.DiskId, *e.Partition.Id, *mountPoint)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to add mount point to disk map", "disk_id", *e.Partition.DiskId, "partition_id", *e.Partition.Id, "mount_path", mountPoint.Path, "err", err)
				continue
			}
			if !oldstate {
				_ = self.eventBus.EmitMountPoint(events.MountPointEvent{
					Event:      events.Event{Type: events.EventTypes.UPDATE},
					MountPoint: mountPoint,
				})
			}
			continue
		} else if prtstate.Source == *e.Partition.DevicePath || (e.Partition.LegacyDevicePath != nil && prtstate.Source == *e.Partition.LegacyDevicePath) {
			// Found matching mount info for partition

			mountPoint := dto.MountPointData{
				Path:             prtstate.MountPoint,
				Root:             prtstate.Root,
				DeviceId:         *e.Partition.Id,
				IsWriteSupported: &iw,
				IsMounted:        true,
				Flags:            &dto.MountFlags{},
				CustomFlags:      &dto.MountFlags{},
				FSType:           &prtstate.FSType,
				Type:             "ADDON",
				Partition:        e.Partition,
				RefreshVersion:   self.refreshVersion,
			}
			if err := mountPoint.Flags.Scan(prtstate.Options); err != nil {
				slog.WarnContext(ctx, "Failed to scan mount flags", "mount_path", prtstate.MountPoint, "error", err)
			}
			if err := mountPoint.CustomFlags.Scan(prtstate.SuperOptions); err != nil {
				slog.WarnContext(ctx, "Failed to scan custom mount flags", "mount_path", prtstate.MountPoint, "error", err)
			}
			err := self.disks.AddOrUpdateMountPoint(*e.Partition.DiskId, *e.Partition.Id, mountPoint)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to add mount point to disk map", "disk_id", *e.Partition.DiskId, "partition_id", *e.Partition.Id, "mount_path", mountPoint.Path, "err", err)
				continue
			}
			_ = self.eventBus.EmitMountPoint(events.MountPointEvent{
				Event:      events.Event{Type: events.EventTypes.ADD},
				MountPoint: &mountPoint,
			})
			continue
		}
	}

	tlog.TraceContext(ctx, "Marking stale mount points as unmounted for partition", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id)
	for _, mountPoint := range self.disks.GetAllMountPoints() {
		tlog.TraceContext(ctx, " --> Checking mount point for staleness",
			"disk_id", *e.Disk.Id,
			"partition_id", *e.Partition.Id,
			"mount_path", mountPoint.Path,
			"refresh_version", mountPoint.RefreshVersion,
			"current_refresh_version", self.refreshVersion,
			"is_mounted", mountPoint.IsMounted,
			"is_to_mount_at_startup", (mountPoint.IsToMountAtStartup != nil && *mountPoint.IsToMountAtStartup),
		)
		if (mountPoint.RefreshVersion != self.refreshVersion) &&
			(mountPoint.IsMounted || (mountPoint.IsToMountAtStartup != nil && *mountPoint.IsToMountAtStartup)) {
			tlog.DebugContext(ctx, "Marking mount point as unmounted since not found in procfs mounts", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "mount_path", mountPoint.Path)
			oldtstate := mountPoint.IsMounted
			mountPoint.IsMounted = false
			mountPoint.RefreshVersion = self.refreshVersion
			err := self.disks.AddOrUpdateMountPoint(*e.Partition.DiskId, *e.Partition.Id, *mountPoint)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to add mount point to disk map", "disk_id", *e.Partition.DiskId, "partition_id", *e.Partition.Id, "mount_path", mountPoint.Path, "err", err)
				continue
			}
			if oldtstate || (mountPoint.IsToMountAtStartup != nil && *mountPoint.IsToMountAtStartup) {
				_ = self.eventBus.EmitMountPoint(events.MountPointEvent{
					Event:      events.Event{Type: events.EventTypes.UPDATE},
					MountPoint: mountPoint,
				})
			}
		}
	}
	tlog.TraceContext(ctx, "Done synchronizing mount points for partition", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id)

	return nil
}

func (self *VolumeService) handleMountPointEvent(ctx context.Context, e events.MountPointEvent) errors.E {
	if e.MountPoint.Type == "" {
		e.MountPoint.Type = inferMountPointType(e.MountPoint)
		slog.WarnContext(ctx, "Mount point type missing, defaulting", "mount_point", e.MountPoint.Path, "type", e.MountPoint.Type)
	}
	tlog.DebugContext(ctx, "Processing mount point event for persistence",
		"mount_point", e.MountPoint.Path,
		"device_id", e.MountPoint.DeviceId,
		"event_type", e.Type,
		"mount_point_type", e.MountPoint.Type,
		"is_mounted", e.MountPoint.IsMounted,
		"is_to_mount_at_startup", (e.MountPoint.IsToMountAtStartup != nil && *e.MountPoint.IsToMountAtStartup),
	)
	err := self.persistMountPoint(e.MountPoint)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to persist mount point on event", "mount_point", e.MountPoint, "err", err)
		return err
	}
	if (e.Type == events.EventTypes.ADD || e.Type == events.EventTypes.UPDATE) && !e.MountPoint.IsMounted && e.MountPoint.IsToMountAtStartup != nil && *e.MountPoint.IsToMountAtStartup {
		slog.InfoContext(ctx, "New mount point added and not mounted, attempting to mount", "mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId)
		err = self.MountVolume(e.MountPoint)
		if err != nil {
			if errors.Is(err, dto.ErrorAlreadyMounted) {
				slog.InfoContext(ctx, "Mount point already mounted during automount attempt", "mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId)
				return nil
			}
			slog.ErrorContext(ctx, "Failed to mount volume on event", "mount_point", e.MountPoint, "err", err)
			self.createAutomountFailureNotification(e.MountPoint.Path, e.MountPoint.DeviceId, err)
		}
	}
	return nil
}

func inferMountPointType(mountPoint *dto.MountPointData) string {
	if mountPoint == nil {
		return "ADDON"
	}
	path := mountPoint.Path
	if path == "" {
		path = mountPoint.Root
	}
	if path == "" {
		return "ADDON"
	}
	if strings.HasPrefix(path, "/mnt") {
		return "ADDON"
	}
	return "HOST"
}

func (ms *VolumeService) PatchMountPointSettings(root string, path string, patchData dto.MountPointData) (*dto.MountPointData, errors.E) {

	dbMountData, err := gorm.G[dbom.MountPointPath](ms.db).
		Where(g.MountPointPath.Root.Eq(root), g.MountPointPath.Path.Eq(path)).First(ms.ctx)
	if err != nil {
		return nil, errors.Wrapf(dto.ErrorNotFound, "mount configuration with root %s and path %s not found", root, path)
	}

	err = ms.convDto.MountPointDataToMountPointPath(patchData, &dbMountData)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	affected, err := gorm.G[*dbom.MountPointPath](ms.db).
		Where(g.MountPointPath.Root.Eq(root), g.MountPointPath.Path.Eq(path)).
		Updates(ms.ctx, &dbMountData)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(dto.ErrorNotFound, "mount configuration with root %s and path %s not found", root, path)
		}
		return nil, errors.WithStack(err)
	}
	if affected == 0 {
		return nil, errors.Wrapf(dto.ErrorNotFound, "mount configuration with root %s and path %s not found", root, path)
	}

	currentDto := dto.MountPointData{}
	if convErr := ms.convDto.MountPointPathToMountPointData(dbMountData, &currentDto, ms.GetVolumesData()); convErr != nil {
		return nil, errors.WithStack(convErr)
	}
	// Update cached mount point data
	if currentDto.Partition != nil && currentDto.Partition.DiskId != nil && currentDto.Partition.Id != nil {
		err := ms.disks.AddOrUpdateMountPoint(*currentDto.Partition.DiskId, *currentDto.Partition.Id, currentDto)
		if err != nil {
			slog.WarnContext(ms.ctx, "Failed to update mount point in cache", "err", err)
		}
	} else {
		// Fallback: partition could not be resolved.
		updated := false
		if existing, ok := ms.disks.GetMountPointByPath(path); ok {
			if existing.Partition != nil && existing.Partition.DiskId != nil && existing.Partition.Id != nil {
				existing.IsToMountAtStartup = currentDto.IsToMountAtStartup
				err := ms.disks.AddOrUpdateMountPoint(*existing.Partition.DiskId, *existing.Partition.Id, *existing)
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
						err := ms.disks.AddOrUpdateMountPoint(dk, pid, existing)
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
	_ = ms.eventBus.EmitMountPoint(events.MountPointEvent{
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
	if fsSvc, ok := ms.fs_service.(interface {
		MockSetMountOps(
			tryMount func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
			mountFn func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
			unmountFn func(target string, force, lazy bool) error,
		)
	}); ok {
		fsSvc.MockSetMountOps(tryMount, mountFn, unmountFn)
	}
}

func (ms *VolumeService) GetDevicePathByDeviceID(deviceID string) (string, errors.E) {
	md, ok := ms.disks.Get(deviceID)
	if !ok {
		return "", errors.WithDetails(dto.ErrorNotFound, "Message", "mount point not found", "DeviceId", deviceID)
	}
	return *md.DevicePath, nil
}
