package service

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

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
	GetVolumesData() ([]*dto.Disk, errors.E)
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

	// Provisional recheck of whole-disk synthesized entries. When a snapshot
	// catches a partitioned drive before its partition children are visible,
	// the hardware client synthesizes a whole-disk filesystem entry (e.g. a
	// partition named "sda" on disk "sda"). That snapshot can stay cached long
	// past the boot race, so we re-fetch a few times until the entry settles
	// (or the attempt budget is exhausted).
	recheckInterval        time.Duration
	maxProvisionalRechecks int
	pendingRecheck         *provisionalRecheckState
	recheckMu              sync.Mutex

	// Automount retry guard: bounds repeated mount attempts for the same
	// mount path. When the OS-level mount succeeds but the converter or
	// cache write fails (see volumeMountManager.Mount), the emitted event can
	// carry a stale IsMounted=false, which re-enters handleMountPointEvent
	// and would otherwise loop forever with one mount(2) per iteration.
	// We cap attempts per path and back off exponentially between failures.
	automountBackoffBase time.Duration
	maxAutomountAttempts int
	automountRetryMu     sync.Mutex
	automountRetries     map[string]automountRetryState
}

// automountRetryState tracks how many times an automount attempt for a given
// mount path has failed and when the next attempt is allowed.
type automountRetryState struct {
	attempts    int
	nextRetryAt time.Time
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

		recheckInterval:        15 * time.Second,
		maxProvisionalRechecks: 5,

		automountBackoffBase: 2 * time.Second,
		maxAutomountAttempts: 5,
		automountRetries:     map[string]automountRetryState{},
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
	root := "/"
	if md.Root != "" {
		root = md.Root
	}

	// Load the existing record (if any) and convert INTO it so that fields
	// absent from the event payload (automount flag, flags, custom flags)
	// never wipe persisted configuration. The share association lives on the
	// exported_shares side (MountPointDataPath/MountPointDataRoot), so it is
	// preserved regardless of the payload's Share value.
	existing, err := gorm.G[dbom.MountPointPath](self.db).
		Where(g.MountPointPath.Path.Eq(md.Path), g.MountPointPath.Root.Eq(root)).
		First(self.ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(err)
	}
	dbom_mount_data := existing // zero value when no record exists yet

	err = self.convDto.MountPointDataToMountPointPath(*md, &dbom_mount_data)
	if err != nil {
		return errors.WithStack(err)
	}
	// close mounthpath loop before save
	if dbom_mount_data.ExportedShare != nil {
		dbom_mount_data.ExportedShare.MountPointData = dbom_mount_data
		dbom_mount_data.ExportedShare.MountPointDataPath = &dbom_mount_data.Path
		dbom_mount_data.ExportedShare.MountPointDataRoot = dbom_mount_data.Root
	}

	//slog.DebugContext(self.ctx, "Persisting mount point to database", "mount_point", md.Path, "device_id", md.DeviceId, "is_mounted", md.IsMounted, "is_to_mount_at_startup", (md.IsToMountAtStartup != nil && *md.IsToMountAtStartup))
	tlog.TraceContext(self.ctx, "Mount point data", "data", spew.Sdump(dbom_mount_data))
	err = self.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).
		Create(&dbom_mount_data).Error
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
		for _, disk := range ms.disks.Snapshot() {
			if disk.Partitions == nil {
				continue
			}
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

	// Final validation: ensure DevicePath exists on the OS before
	// delegating to the mount manager. (The nil/empty check already
	// returned above — this only verifies OS-level existence.)
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
	// Early validation of required fields
	if ms.state.ProtectedMode {
		return errors.WithDetails(dto.ErrorOperationNotPermittedInProtectedMode,
			"Operation", "UnmountVolume",
			"Detail", "Unmount operation is not permitted when ProtectedMode is enabled.",
		)
	}

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

	for diskID, disk := range self.disks.Snapshot() {
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

		if allowed, exhausted := self.allowAutomountAttempt(mountPoint.Path); !allowed {
			if exhausted {
				slog.WarnContext(self.ctx, "Automount attempts exhausted for partition add event, giving up",
					"devname", devName, "path", mountPoint.Path, "max_attempts", self.maxAutomountAttempts)
			} else {
				slog.DebugContext(self.ctx, "Skipping automount retry for partition add event, backing off",
					"devname", devName, "path", mountPoint.Path)
			}
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
			if errors.Is(err, dto.ErrorAlreadyMounted) {
				slog.InfoContext(self.ctx, "Mount point already mounted during partition add automount retry", "devname", devName, "path", mountCopy.Path)
				self.clearAutomountRetry(mountCopy.Path)
				handled = true
				continue
			}
			slog.WarnContext(self.ctx, "Failed automount retry for partition add event", "devname", devName, "path", mountCopy.Path, "err", err)
			self.recordAutomountFailure(mountCopy.Path)
			continue
		}

		self.clearAutomountRetry(mountCopy.Path)
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

func (self *VolumeService) GetVolumesData() ([]*dto.Disk, errors.E) {
	if self.disks.Len() == 0 {
		if err := self.getVolumesData(); err != nil {
			slog.ErrorContext(self.ctx, "Failed to get volumes data in GetVolumesData", "err", err)
			return nil, errors.WithStack(err)
		}
	}
	return self.disks.All(), nil
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

// loadMountPointFromDBByPath loads a single mount point configuration from the
// database by its primary key (path, root) — independently of the partition
// device id — and converts it to a DTO. The second return value reports
// whether a record was found.
func (self *VolumeService) loadMountPointFromDBByPath(path string, root string) (*dto.MountPointData, bool, errors.E) {
	if path == "" {
		return nil, false, nil
	}
	dbMPs, err := gorm.G[dbom.MountPointPath](self.db).
		Where(g.MountPointPath.Path.Eq(path), g.MountPointPath.Root.Eq(root)).
		Find(self.ctx)
	if err != nil {
		return nil, false, errors.WithStack(err)
	}
	if len(dbMPs) == 0 {
		return nil, false, nil
	}
	dtoMP := dto.MountPointData{}
	if convErr := self.convDto.MountPointPathToMountPointData(dbMPs[0], &dtoMP, nil); convErr != nil {
		return nil, false, errors.WithStack(convErr)
	}
	return &dtoMP, true, nil
}

// getVolumesData retrieves and synchronizes volume data with caching and concurrency control.
// Disks and partitions are read from the hardware client and enriched with local mount point data.
// It also syncs mount point data with database records, saving new entries and removing obsolete ones.
func (self *VolumeService) getVolumesData() errors.E {
	tlog.TraceContext(self.ctx, "Requesting GetVolumesData via singleflight...")

	_, err, _ := self.sfGroup.Do("GetVolumesData", func() (any, error) {
		refreshVersion := self.disks.NextRefreshVersion()
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
		// H2: parse procfs once per refresh cycle, lazily, only when at least
		// one partition event will be emitted. The snapshot is shared by every
		// batched PartitionEvent of this cycle, so the handler never re-parses.
		var mountInfos []*procfs.MountInfo
		var mountInfosErr error
		// Disks processing
		for _, disk := range hwDisks {
			tlog.TraceContext(self.ctx, "Processing disk from hardware client", "disk_id", *disk.Id, "partition_count", len(*disk.Partitions))
			disk.RefreshVersion = refreshVersion

			currentDisk, updateDisk := self.disks.Get(*disk.Id)

			err := self.disks.AddOrUpdate(&disk)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to update existing disk in cache", "disk_id", *disk.Id, "err", err)
			}

			if disk.Partitions != nil {
				// H8: the hardware cache (30-min TTL) hands out the same
				// *map[string]Partition to every caller. Enriching it in place
				// below would mutate that shared map — a concurrent map
				// read/write panic risk with the HDIdle HTTP handler (which
				// iterates the same cache on another goroutine) and cross-
				// service cache pollution. Copy the map container first; the
				// partition values are value types, so only the map itself is
				// shared. The hardware cache keeps its raw, un-enriched shape.
				copiedPartitions := make(map[string]dto.Partition, len(*disk.Partitions))
				for pid, part := range *disk.Partitions {
					copiedPartitions[pid] = part
				}
				disk.Partitions = &copiedPartitions

				// H2: batch all changed partitions of this disk into a single
				// PartitionEvent sharing one procfs snapshot, instead of one
				// event per partition (each of which re-parsed procfs inside
				// the handler). Reduces P events to one per disk.
				changedPartitions := make([]*dto.Partition, 0, len(*disk.Partitions))
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
					} else {
						// The filesystem type is unknown (e.g. a raw disk or an
						// unreadable/unformatted partition). Formatting does not depend
						// on the current filesystem, so enable the format action while
						// keeping check/label actions disabled (they require a known
						// filesystem). The frontend format dialog defaults to a known
						// format-capable filesystem type in this case.
						part.FilesystemInfo = &dto.FilesystemInfo{
							Support: &dto.FilesystemSupport{
								CanFormat: true,
							},
						}
					}
					(*disk.Partitions)[pid] = part
					changedPartitions = append(changedPartitions, &part)
				}
				if len(changedPartitions) > 0 {
					if self.procfsGetMounts != nil && mountInfos == nil && mountInfosErr == nil {
						mountInfos, mountInfosErr = self.procfsGetMounts()
						if mountInfosErr != nil {
							slog.WarnContext(self.ctx, "Failed to get current mount information from procfs", "err", mountInfosErr)
						}
					}
					eventType := events.EventTypes.ADD
					if currentDisk != nil && updateDisk {
						eventType = events.EventTypes.UPDATE
					}
					self.eventBus.EmitPartition(events.PartitionEvent{
						Event:      events.Event{Type: eventType},
						Disk:       &disk,
						Partitions: changedPartitions,
						MountInfos: mountInfos,
					})
				}
			}
		}

		// Evict disks that were not part of the latest hardware snapshot.
		// Every disk upserted above carries the current RefreshVersion, so
		// anything left with an older version was absent from the snapshot and
		// must be dropped. Without this, removed drives (and drives that
		// briefly vanish from a snapshot) would stay in the map as phantom
		// volumes until their next udev event.
		for id, disk := range self.disks.Snapshot() {
			if disk.RefreshVersion != refreshVersion {
				self.disks.Remove(id)
				self.eventBus.EmitDisk(events.DiskEvent{
					Event: events.Event{Type: events.EventTypes.REMOVE},
					Disk:  disk,
				})
			}
		}

		self.manageProvisionalRechecks()
		return nil, nil
	})

	if err != nil {
		//slog.Error("Singleflight execution of GetVolumesData failed", "err", err, "shared", shared)
		return errors.WithStack(err)
	}

	return nil
}

// provisionalRecheckState tracks the bounded recheck chain for whole-disk
// synthesized entries.
type provisionalRecheckState struct {
	attempts int
	timer    *time.Timer
}

// isWholeDiskSynthesized reports whether the disk carries a single partition
// whose legacy device name equals the disk's own name (e.g. a partition "sda"
// on disk "sda"). Real partition children always append a number ("sda1"), so
// this shape only occurs for whole-disk filesystems: either a genuine
// superfloppy drive or a snapshot taken before the drive's partition table was
// enumerated.
func (self *VolumeService) isWholeDiskSynthesized(d *dto.Disk) bool {
	if d == nil || d.Partitions == nil || len(*d.Partitions) != 1 || d.LegacyDeviceName == nil {
		return false
	}
	for _, part := range *d.Partitions {
		return part.LegacyDeviceName != nil && *part.LegacyDeviceName == *d.LegacyDeviceName
	}
	return false
}

func (self *VolumeService) hasWholeDiskSynthesizedDisks() bool {
	for _, disk := range self.disks.All() {
		if self.isWholeDiskSynthesized(disk) {
			return true
		}
	}
	return false
}

// manageProvisionalRechecks keeps a bounded, time-based reconciliation loop
// running while the disk map contains whole-disk synthesized entries.
// Event-driven invalidation alone cannot heal those entries when the boot race
// happened before the relevant udev events were observable, so a few forced
// refreshes are scheduled until the layout settles or the budget runs out.
func (self *VolumeService) manageProvisionalRechecks() {
	self.recheckMu.Lock()
	defer self.recheckMu.Unlock()

	if self.recheckInterval <= 0 {
		self.recheckInterval = 15 * time.Second
	}
	if self.maxProvisionalRechecks <= 0 {
		self.maxProvisionalRechecks = 5
	}

	if !self.hasWholeDiskSynthesizedDisks() {
		if self.pendingRecheck != nil && self.pendingRecheck.timer != nil {
			self.pendingRecheck.timer.Stop()
		}
		self.pendingRecheck = nil
		return
	}

	if self.pendingRecheck == nil {
		self.pendingRecheck = &provisionalRecheckState{attempts: self.maxProvisionalRechecks}
	}
	if self.pendingRecheck.attempts <= 0 {
		// The entry never settled; keep the current state and stop probing.
		self.pendingRecheck = nil
		return
	}
	self.pendingRecheck.attempts--
	self.pendingRecheck.timer = time.AfterFunc(self.recheckInterval, self.runProvisionalRecheck)
}

// runProvisionalRecheck forces a fresh hardware fetch and re-synchronizes the
// volume cache, letting a whole-disk synthesized entry be replaced by the real
// partition layout once the system has settled.
func (self *VolumeService) runProvisionalRecheck() {
	if self.ctx.Err() != nil {
		return
	}
	if self.hardwareClient != nil {
		self.hardwareClient.InvalidateHardwareInfo()
	}
	if err := self.getVolumesData(); err != nil {
		slog.ErrorContext(self.ctx, "Failed to refresh volume cache during provisional recheck", "err", err)
	}
}

// handleDiskUdevRemoveEvent reacts to a block device being removed from the
// host. The hardware snapshot is invalidated and the volume cache refreshed;
// reconciliation then evicts the disk because it is absent from the new
// snapshot (see the pruning step in getVolumesData).
func (self *VolumeService) handleDiskUdevRemoveEvent(devName string) {
	slog.InfoContext(self.ctx, "Processing block device removal event", "devname", devName)
	if self.hardwareClient != nil {
		self.hardwareClient.InvalidateHardwareInfo()
	}
	if err := self.getVolumesData(); err != nil {
		slog.ErrorContext(self.ctx, "Failed to refresh volume cache after disk removal event", "devname", devName, "err", err)
	}
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
	if self.disks == nil || self.disks.Len() == 0 {
		return nil
	}

	normalizedDevice := strings.TrimSpace(devicePath)
	for _, disk := range self.disks.All() {
		if disk.Partitions == nil {
			continue
		}
		for _, partition := range *disk.Partitions {
			if strings.TrimSpace(self.disks.GetPartitionDevicePath(&partition)) == normalizedDevice {
				return disk
			}
		}
	}

	return nil
}

func (self *VolumeService) handlePartitionEvent(ctx context.Context, e events.PartitionEvent) errors.E {
	if len(e.Partitions) > 0 {
		// Batch mode (H2): all changed partitions of one disk share a single
		// procfs snapshot carried by the event. If the emitter provided none
		// (e.g. an external emitter), parse once here and reuse it.
		mountInfos := e.MountInfos
		if mountInfos == nil {
			parsed, errS := self.procfsGetMounts()
			if errS != nil {
				slog.ErrorContext(ctx, "Failed to get current mount information from procfs", "disk_id", *e.Disk.Id, "err", errS)
				return errors.WithStack(errS)
			}
			mountInfos = parsed
		}
		for _, part := range e.Partitions {
			if err := self.syncPartitionMountData(ctx, e.Disk, part, mountInfos); err != nil {
				slog.WarnContext(ctx, "Failed to sync partition mount data", "disk_id", *e.Disk.Id, "partition_id", *part.Id, "err", err)
			}
		}
		return nil
	}

	// Single-partition mode (legacy emitters, e.g. udev events, tests)
	mountInfos := e.MountInfos
	if mountInfos == nil {
		parsed, errS := self.procfsGetMounts()
		if errS != nil {
			slog.ErrorContext(ctx, "Failed to get current mount information from procfs", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "err", errS)
			return errors.WithStack(errS)
		}
		mountInfos = parsed
	}
	return self.syncPartitionMountData(ctx, e.Disk, e.Partition, mountInfos)
}

// syncPartitionMountData reconciles a partition's mount points with the
// current procfs snapshot: it loads persisted mount points from the DB, adds
// missing ones to the in-memory cache, updates mount points present in procfs
// and marks stale ones as unmounted. Used by both the single-partition and the
// batched event paths; mountInfos is the shared procfs snapshot.
func (self *VolumeService) syncPartitionMountData(ctx context.Context, disk *dto.Disk, part *dto.Partition, mountInfos []*procfs.MountInfo) errors.E {
	tlog.TraceContext(ctx, "Processing partition event for mount data sync", "disk_id", *disk.Id, "partition_id", *part.Id)

	if part.DevicePath == nil || *part.DevicePath == "" {
		slog.DebugContext(ctx, "Skipping partition with nil or empty device path", "disk_id", *disk.Id)
		return nil
	}
	if part.DiskId == nil || *part.DiskId == "" {
		part.DiskId = disk.Id
	}

	mountData, err := self.loadMountPointFromDB(part)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to load mount point data from DB for partition", "disk_id", *disk.Id, "partition_id", *part.Id, "err", err)
		return err
	}
	// Add missing mount points from DB to in-memory cache
	for _, md := range mountData {
		err := self.disks.AddOrUpdateMountPoint(*part.DiskId, *part.Id, *md)
		if err != nil {
			slog.WarnContext(self.ctx, "Failed to add mount point to disk map during partition event handling", "disk_id", *part.DiskId, "partition_id", *part.Id, "mount_path", md.Path, "err", err)
			continue
		}
	}

	// Update existing mount points with current mount info
	tlog.TraceContext(ctx, "Synchronizing mount points for partition", "disk_id", *disk.Id, "partition_id", *part.Id, "mount_data_count", len(mountData), "procfs_mounts_count", len(mountInfos))
	for _, prtstate := range mountInfos {
		iw := osutil.IsWritable(prtstate.MountPoint)
		if mountPoint, ok := self.disks.GetMountPoint(*part.DiskId, *part.Id, prtstate.MountPoint); ok {
			tlog.TraceContext(ctx, "Found existing mount point in cache for partition, updating state", "disk_id", *disk.Id, "partition_id", *part.Id, "mount_path", mountPoint.Path, "is_mounted", mountPoint.IsMounted)
			oldstate := mountPoint.IsMounted
			mountPoint.IsMounted = true
			mountPoint.Path = prtstate.MountPoint
			mountPoint.Root = prtstate.Root
			mountPoint.RefreshVersion = self.disks.CurrentRefreshVersion()
			mountPoint.IsWriteSupported = &iw
			if err := mountPoint.Flags.Scan(prtstate.Options); err != nil {
				slog.WarnContext(ctx, "Failed to scan mount flags", "mount_path", prtstate.MountPoint, "error", err)
			}
			if err := mountPoint.CustomFlags.Scan(prtstate.SuperOptions); err != nil {
				slog.WarnContext(ctx, "Failed to scan custom mount flags", "mount_path", prtstate.MountPoint, "error", err)
			}
			mountPoint.FSType = &prtstate.FSType
			mountPoint.Type = "ADDON"
			err := self.disks.AddOrUpdateMountPoint(*part.DiskId, *part.Id, *mountPoint)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to add mount point to disk map", "disk_id", *part.DiskId, "partition_id", *part.Id, "mount_path", mountPoint.Path, "err", err)
				continue
			}
			if !oldstate {
				_ = self.eventBus.EmitMountPoint(events.MountPointEvent{
					Event:      events.Event{Type: events.EventTypes.UPDATE},
					MountPoint: mountPoint,
				})
			}
			continue
		} else if prtstate.Source == *part.DevicePath || (part.LegacyDevicePath != nil && prtstate.Source == *part.LegacyDevicePath) {
			// Found matching mount info for partition

			mountPoint := dto.MountPointData{
				Path:             prtstate.MountPoint,
				Root:             prtstate.Root,
				DeviceId:         *part.Id,
				IsWriteSupported: &iw,
				IsMounted:        true,
				Flags:            &dto.MountFlags{},
				CustomFlags:      &dto.MountFlags{},
				FSType:           &prtstate.FSType,
				Type:             "ADDON",
				Partition:        part,
				RefreshVersion:   self.disks.CurrentRefreshVersion(),
			}
			if err := mountPoint.Flags.Scan(prtstate.Options); err != nil {
				slog.WarnContext(ctx, "Failed to scan mount flags", "mount_path", prtstate.MountPoint, "error", err)
			}
			if err := mountPoint.CustomFlags.Scan(prtstate.SuperOptions); err != nil {
				slog.WarnContext(ctx, "Failed to scan custom mount flags", "mount_path", prtstate.MountPoint, "error", err)
			}
			// Merge persisted configuration (automount flag, flags, custom flags)
			// into the freshly discovered mount point so that persisting the ADD
			// event cannot wipe user settings. The share association is owned by
			// the share service and is intentionally not merged here.
			if existingMP, ok := self.disks.GetMountPointByPath(prtstate.MountPoint); ok {
				mountPoint.IsToMountAtStartup = existingMP.IsToMountAtStartup
				mountPoint.Flags = existingMP.Flags
				mountPoint.CustomFlags = existingMP.CustomFlags
			} else if dbMP, found, dbErr := self.loadMountPointFromDBByPath(prtstate.MountPoint, prtstate.Root); dbErr != nil {
				slog.WarnContext(ctx, "Failed to load persisted mount point configuration", "mount_path", prtstate.MountPoint, "err", dbErr)
			} else if found {
				mountPoint.IsToMountAtStartup = dbMP.IsToMountAtStartup
				mountPoint.Flags = dbMP.Flags
				mountPoint.CustomFlags = dbMP.CustomFlags
			}
			err := self.disks.AddOrUpdateMountPoint(*part.DiskId, *part.Id, mountPoint)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to add mount point to disk map", "disk_id", *part.DiskId, "partition_id", *part.Id, "mount_path", mountPoint.Path, "err", err)
				continue
			}
			_ = self.eventBus.EmitMountPoint(events.MountPointEvent{
				Event:      events.Event{Type: events.EventTypes.ADD},
				MountPoint: &mountPoint,
			})
			continue
		}
	}

	tlog.TraceContext(ctx, "Marking stale mount points as unmounted for partition", "disk_id", *disk.Id, "partition_id", *part.Id)
	for _, mountPoint := range self.disks.GetMountPointsForPartition(*part.DiskId, *part.Id) {
		tlog.TraceContext(ctx, " --> Checking mount point for staleness",
			"disk_id", *disk.Id,
			"partition_id", *part.Id,
			"mount_path", mountPoint.Path,
			"refresh_version", mountPoint.RefreshVersion,
			"current_refresh_version", self.disks.CurrentRefreshVersion(),
			"is_mounted", mountPoint.IsMounted,
			"is_to_mount_at_startup", (mountPoint.IsToMountAtStartup != nil && *mountPoint.IsToMountAtStartup),
		)
		if (mountPoint.RefreshVersion != self.disks.CurrentRefreshVersion()) &&
			(mountPoint.IsMounted || (mountPoint.IsToMountAtStartup != nil && *mountPoint.IsToMountAtStartup)) {
			tlog.DebugContext(ctx, "Marking mount point as unmounted since not found in procfs mounts", "disk_id", *disk.Id, "partition_id", *part.Id, "mount_path", mountPoint.Path)
			oldtstate := mountPoint.IsMounted
			mountPoint.IsMounted = false
			mountPoint.RefreshVersion = self.disks.CurrentRefreshVersion()
			err := self.disks.AddOrUpdateMountPoint(*part.DiskId, *part.Id, *mountPoint)
			if err != nil {
				slog.WarnContext(self.ctx, "Failed to add mount point to disk map", "disk_id", *part.DiskId, "partition_id", *part.Id, "mount_path", mountPoint.Path, "err", err)
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
	tlog.TraceContext(ctx, "Done synchronizing mount points for partition", "disk_id", *disk.Id, "partition_id", *part.Id)

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
		if allowed, exhausted := self.allowAutomountAttempt(e.MountPoint.Path); !allowed {
			if exhausted {
				slog.WarnContext(ctx, "Automount attempts exhausted for mount point, giving up",
					"mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId, "max_attempts", self.maxAutomountAttempts)
			} else {
				slog.DebugContext(ctx, "Skipping automount attempt, backing off",
					"mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId)
			}
			return nil
		}
		slog.InfoContext(ctx, "New mount point added and not mounted, attempting to mount", "mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId)
		err = self.MountVolume(e.MountPoint)
		if err != nil {
			if errors.Is(err, dto.ErrorAlreadyMounted) {
				slog.InfoContext(ctx, "Mount point already mounted during automount attempt", "mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId)
				self.clearAutomountRetry(e.MountPoint.Path)
				return nil
			}
			slog.ErrorContext(ctx, "Failed to mount volume on event", "mount_point", e.MountPoint, "err", err)
			self.recordAutomountFailure(e.MountPoint.Path)
			self.createAutomountFailureNotification(e.MountPoint.Path, e.MountPoint.DeviceId, err)
		} else {
			self.clearAutomountRetry(e.MountPoint.Path)
		}
	}
	return nil
}

// allowAutomountAttempt reports whether a new mount attempt for the given
// path is currently permitted. It returns (false, true) when the attempt
// budget is exhausted (terminal state) and (false, false) while backing off.
func (self *VolumeService) allowAutomountAttempt(path string) (allowed bool, exhausted bool) {
	self.automountRetryMu.Lock()
	defer self.automountRetryMu.Unlock()

	st, ok := self.automountRetries[path]
	if !ok {
		return true, false
	}
	if st.attempts >= self.maxAutomountAttempts {
		return false, true
	}
	if time.Now().Before(st.nextRetryAt) {
		return false, false
	}
	return true, false
}

// recordAutomountFailure registers one failed mount attempt for the given
// path and schedules the next allowed attempt with exponential backoff.
func (self *VolumeService) recordAutomountFailure(path string) {
	self.automountRetryMu.Lock()
	defer self.automountRetryMu.Unlock()

	if self.automountRetries == nil {
		self.automountRetries = map[string]automountRetryState{}
	}
	st := self.automountRetries[path]
	st.attempts++
	backoff := self.automountBackoffBase << (st.attempts - 1)
	if backoff <= 0 {
		backoff = self.automountBackoffBase
	}
	st.nextRetryAt = time.Now().Add(backoff)
	self.automountRetries[path] = st
}

// clearAutomountRetry removes the retry state for the given path after a
// successful mount attempt.
func (self *VolumeService) clearAutomountRetry(path string) {
	self.automountRetryMu.Lock()
	defer self.automountRetryMu.Unlock()
	delete(self.automountRetries, path)
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
		slog.DebugContext(ms.ctx, "PatchMountPointSettings: no fields changed (no-op)", "root", root, "path", path)
	}

	// Reload the record from the DB. Struct-based Updates skips zero-value
	// (nil pointer) fields, so the converter's unconditional assignments
	// (Flags, Data, ExportedShare) are NOT written back when the patch omits
	// them — but the in-memory dbMountData still holds those nils. Converting
	// it directly would drop the persisted flags from the response DTO and
	// from the disk cache below, so re-read the row to get the true state.
	// An all-nil patch on an existing record therefore degrades to a
	// successful no-op instead of a misleading 404 "not found".
	if dbMountData, err = gorm.G[dbom.MountPointPath](ms.db).
		Where(g.MountPointPath.Root.Eq(root), g.MountPointPath.Path.Eq(path)).First(ms.ctx); err != nil {
		return nil, errors.Wrapf(dto.ErrorNotFound, "mount configuration with root %s and path %s not found", root, path)
	}

	currentDto := dto.MountPointData{}
	// The converter uses the volume cache only to resolve the partition from
	// the DeviceId; on load failure pass nil so the fallback cache-update path
	// below still keeps the persisted state consistent.
	disksContext, errVolumes := ms.GetVolumesData()
	if errVolumes != nil {
		slog.WarnContext(ms.ctx, "PatchMountPointSettings: failed to load volumes data for partition resolution", "err", errVolumes)
		disksContext = nil
	}
	if convErr := ms.convDto.MountPointPathToMountPointData(dbMountData, &currentDto, disksContext); convErr != nil {
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
			for dk, d := range ms.disks.Snapshot() {
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
	// DeviceId identifies a partition (see MountVolume, which matches
	// *part.Id == md.DeviceId), so resolve it through the partition map rather
	// than the disk map: disk IDs must not match here.
	part, _, ok := ms.disks.GetPartitionByID(deviceID)
	if !ok {
		return "", errors.WithDetails(dto.ErrorNotFound,
			"Message", "partition not found",
			"DeviceId", deviceID,
		)
	}
	path := ms.disks.GetPartitionDevicePath(part)
	if path == "" {
		return "", errors.WithDetails(dto.ErrorDeviceNotFound,
			"Message", "device path not available",
			"DeviceId", deviceID,
		)
	}
	return path, nil
}
