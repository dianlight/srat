package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/internal/darwinstubs/mount"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service/volume"
	"github.com/dianlight/tlog"
	"github.com/prometheus/procfs"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
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

// VolumeService is the thin facade over the volume subpackage: it owns the
// refresh cycle (GetVolumesData/getVolumesData) and event subscriptions,
// while mount persistence lives in volume.MountRepository, mount operations
// in volume.MountOrchestrator and event reactions in volume.UdevHandler.
// The facade issues no direct db.* queries and no os/exec calls.
type VolumeService struct {
	ctx             context.Context
	hardwareClient  HardwareServiceInterface
	fs_service      FilesystemServiceInterface
	state           *dto.ContextState
	sfGroup         singleflight.Group
	eventBus        events.EventBusInterface
	procfsGetMounts func() ([]*procfs.MountInfo, error)
	disks           *dto.DiskMap

	repo         *volume.MountRepository
	orchestrator *volume.MountOrchestrator
	udevHandler  *volume.UdevHandler

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
}

type VolumeServiceProps struct {
	fx.In
	Ctx               context.Context
	Db                *gorm.DB
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
		hardwareClient:  in.HardwareClient,
		fs_service:      in.FilesystemService,
		state:           in.State,
		eventBus:        in.EventBus,
		procfsGetMounts: procfs.GetMounts,
		disks:           in.Disks,

		recheckInterval:        15 * time.Second,
		maxProvisionalRechecks: 5,
	}

	mountRepo := repository.NewMountPointPathRepository(in.Db)
	p.repo = volume.NewMountRepository(in.Ctx, mountRepo, in.Disks, in.EventBus)
	p.orchestrator = volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx:        in.Ctx,
		Disks:      in.Disks,
		Filesystem: in.FilesystemService,
		Mounter:    in.Mounter,
		HA:         in.HAService,
		EventBus:   in.EventBus,
		Repo:       mountRepo,
		ProtectedMode: func() bool {
			return in.State.ProtectedMode
		},
		Volumes: func() ([]*dto.Disk, errors.E) {
			return p.GetVolumesData()
		},
		ResetAfter: 5 * time.Minute,
	})
	p.udevHandler = volume.NewUdevHandler(volume.HandlerParams{
		Ctx:          in.Ctx,
		Disks:        in.Disks,
		Hardware:     in.HardwareClient,
		EventBus:     in.EventBus,
		Repo:         p.repo,
		Orchestrator: p.orchestrator,
		Filesystem:   in.FilesystemService,
		Refresh:      p.getVolumesData,
		ResetRecheck: p.resetProvisionalRecheckBudget,
		ProcfsMounts: func() ([]*procfs.MountInfo, error) {
			return p.procfsGetMounts()
		},
	})

	var unsubscribe [7]func()
	unsubscribe[0] = p.eventBus.OnPartition(p.udevHandler.HandlePartitionEvent)
	unsubscribe[1] = p.eventBus.OnMountPoint(p.udevHandler.HandleMountPointEvent)
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
				if err := p.eventBus.EmitDisk(events.DiskEvent{
					Type: events.EventTypes.UPDATE,
					Disk: disk,
				}); err != nil {
					slog.WarnContext(ctx, "Failed to emit disk update event after removing share from mount point", "share", se.Share.Name, "err", err)
				}
			}
		case events.EventTypes.ADD, events.EventTypes.UPDATE:
			disk, err := p.disks.AddMountPointShare(se.Share)
			if err != nil {
				if se.Share.Usage != "internal" {
					slog.WarnContext(ctx, "Failed to add/update share in mount point in cache", "share", se.Share, "err", err)
				}
				return nil
			}
			if err := p.eventBus.EmitDisk(events.DiskEvent{
				Type: events.EventTypes.UPDATE,
				Disk: disk,
			}); err != nil {
				slog.WarnContext(ctx, "Failed to emit disk update event after adding/updating share in mount point", "share", se.Share.Name, "err", err)
			}
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
	unsubscribe[6] = p.eventBus.OnFilesystemTask(p.udevHandler.HandleFilesystemTaskEvent)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// H10: warm the volume cache at boot so the first HTTP request
			// never pays the hardware discovery cost; GetVolumesData keeps
			// its lazy path as fallback. A discovery failure must not fail
			// the app start.
			if err := p.getVolumesData(); err != nil {
				slog.WarnContext(p.ctx, "Failed to warm volume cache at startup", "err", err)
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

// MountVolume validates a mount request and delegates to the orchestrator.
func (s *VolumeService) MountVolume(md *dto.MountPointData) errors.E {
	return s.orchestrator.MountVolume(md)
}

// UnmountVolume resolves a mount point from the cache and unmounts it.
func (s *VolumeService) UnmountVolume(path string, force bool) errors.E {
	return s.orchestrator.UnmountVolume(path, force)
}

// PatchMountPointSettings applies a partial mount configuration update.
func (s *VolumeService) PatchMountPointSettings(root string, path string, patchData dto.MountPointData) (*dto.MountPointData, errors.E) {
	return s.orchestrator.PatchMountPointSettings(root, path, patchData)
}

// GetDevicePathByDeviceID resolves a partition device id to its OS path.
func (s *VolumeService) GetDevicePathByDeviceID(deviceID string) (string, errors.E) {
	return s.orchestrator.GetDevicePathByDeviceID(deviceID)
}

func (s *VolumeService) GetVolumesData() ([]*dto.Disk, errors.E) {
	if s.disks.Len() == 0 {
		if err := s.getVolumesData(); err != nil {
			slog.ErrorContext(s.ctx, "Failed to get volumes data in GetVolumesData", "err", err)
			return nil, errors.WithStack(err)
		}
	}
	return s.disks.All(), nil
}

// getVolumesData retrieves and synchronizes volume data with caching and concurrency control.
// Disks and partitions are read from the hardware client and enriched with local mount point data.
// It also syncs mount point data with database records, saving new entries and removing obsolete ones.
func (s *VolumeService) getVolumesData() errors.E {
	tlog.TraceContext(s.ctx, "Requesting GetVolumesData via singleflight...")

	_, err, _ := s.sfGroup.Do("GetVolumesData", func() (any, error) {
		refreshVersion := s.disks.NextRefreshVersion()
		filesystemSupportCache := make(map[string]*dto.FilesystemInfo)

		tlog.TraceContext(s.ctx, "Executing GetVolumesData core logic (singleflight)...")

		// Skip hardware client if it's not initialized
		if s.hardwareClient == nil {
			slog.DebugContext(s.ctx, "Hardware client not initialized, continuing with empty disk list")
			return s.disks, nil
		}

		// Get Host Hardware
		hwDisks, errHw := s.hardwareClient.GetHardwareInfo()
		if errHw != nil {
			return nil, errHw
		}
		if hwDisks == nil {
			tlog.TraceContext(s.ctx, "Hardware client returned nil disks, continuing with empty disk list")
			return s.disks, nil
		}

		tlog.DebugContext(s.ctx, "Retrieved hardware disks from hardware client", "disk_count", len(hwDisks))
		// H2: parse procfs once per refresh cycle, lazily, only when at least
		// one partition event will be emitted. The snapshot is shared by every
		// batched PartitionEvent of this cycle, so the handler never re-parses.
		var mountInfos []*procfs.MountInfo
		var mountInfosErr error
		// Disks processing
		for _, disk := range hwDisks {
			partitionCount := 0
			if disk.Partitions != nil {
				partitionCount = len(*disk.Partitions)
			}
			tlog.TraceContext(s.ctx, "Processing disk from hardware client", "disk_id", *disk.Id, "partition_count", partitionCount)
			disk.RefreshVersion = refreshVersion

			currentDisk, updateDisk := s.disks.Get(*disk.Id)

			// Build the enriched partition map locally before publishing the
			// disk to the cache. Publishing first and then enriching the
			// nested map in place would mutate a map that concurrent readers
			// (e.g. GetPartition under the read lock) may be iterating — a
			// data race. The disk is only published once it is fully built.
			var changedPartitions []*dto.Partition
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
				changedPartitions = make([]*dto.Partition, 0, len(*disk.Partitions))
				for pid, part := range *disk.Partitions {
					if part.FsType != nil && *part.FsType != "" {
						if cached, ok := filesystemSupportCache[*part.FsType]; ok {
							part.FilesystemInfo = cached
						} else {
							info, err := s.fs_service.GetSupportAndInfo(s.ctx, *part.FsType)
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
			}

			// Guard against synthesized whole-disk entries overwriting real
			// partitions during hardware flapping: when a USB disk's partitions
			// disappear momentarily, GetHardwareInfo may return a single
			// synthesized partition whose LegacyDeviceName matches the disk
			// name. If the cache already holds a version with real partitions
			// (more than one, or a single partition with a different name),
			// keep the cached version — the next udev ADD event will refresh
			// with accurate data.  This prevents the perpetual churn where
			// each getVolumesData cycle replaces real partitions with
			// synthesized ones and vice-versa.
			if currentDisk != nil && updateDisk && s.isWholeDiskSynthesized(&disk) && !s.isWholeDiskSynthesized(currentDisk) {
				tlog.DebugContext(s.ctx, "Skipping synthesized whole-disk downgrade — cached disk has real partitions",
					"disk_id", *disk.Id, "cached_partitions", len(*currentDisk.Partitions))
				// Touch the RefreshVersion so the eviction loop below
				// does not drop the disk from the map.
				currentDisk.RefreshVersion = refreshVersion
				continue
			}

			err := s.disks.AddOrUpdate(&disk)
			if err != nil {
				slog.WarnContext(s.ctx, "Failed to update existing disk in cache", "disk_id", *disk.Id, "err", err)
			}

			if len(changedPartitions) > 0 {
				if s.procfsGetMounts != nil && mountInfos == nil && mountInfosErr == nil {
					mountInfos, mountInfosErr = s.procfsGetMounts()
					if mountInfosErr != nil {
						slog.WarnContext(s.ctx, "Failed to get current mount information from procfs", "err", mountInfosErr)
					}
				}
				eventType := events.EventTypes.ADD
				if currentDisk != nil && updateDisk {
					eventType = events.EventTypes.UPDATE
				}
				if err := s.eventBus.EmitPartition(events.PartitionEvent{
					Event:      events.Event{Type: eventType},
					Disk:       &disk,
					Partitions: changedPartitions,
					MountInfos: mountInfos,
				}); err != nil {
					slog.WarnContext(s.ctx, "Failed to emit partition event during volume refresh", "disk_id", *disk.Id, "err", err)
				}
			}
		}

		// Evict disks that were not part of the latest hardware snapshot.
		// Every disk upserted above carries the current RefreshVersion, so
		// anything left with an older version was absent from the snapshot and
		// must be dropped. Without this, removed drives (and drives that
		// briefly vanish from a snapshot) would stay in the map as phantom
		// volumes until their next udev event.
		for id, disk := range s.disks.Snapshot() {
			if disk.RefreshVersion != refreshVersion {
				s.disks.Remove(id)
				if err := s.eventBus.EmitDisk(events.DiskEvent{
					Event: events.Event{Type: events.EventTypes.REMOVE},
					Disk:  disk,
				}); err != nil {
					slog.WarnContext(s.ctx, "Failed to emit disk removal event during volume refresh", "disk_id", id, "err", err)
				}
			}
		}

		s.manageProvisionalRechecks()
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
func (s *VolumeService) isWholeDiskSynthesized(d *dto.Disk) bool {
	if d == nil || d.Partitions == nil || len(*d.Partitions) != 1 || d.LegacyDeviceName == nil {
		return false
	}
	for _, part := range *d.Partitions {
		return part.LegacyDeviceName != nil && *part.LegacyDeviceName == *d.LegacyDeviceName
	}
	return false
}

func (s *VolumeService) hasWholeDiskSynthesizedDisks() bool {
	for _, disk := range s.disks.DeepCopyAll() {
		if s.isWholeDiskSynthesized(disk) {
			return true
		}
	}
	return false
}

// resetProvisionalRecheckBudget clears the pending recheck state so that the
// next call to manageProvisionalRechecks will start a fresh recheck chain.
// This is called when a udev partition ADD event arrives and the partition was
// not yet in the DiskMap, giving the bounded recheck loop another chance to
// settle the layout after the Supervisor API has had time to process its own
// udev events.
func (s *VolumeService) resetProvisionalRecheckBudget() {
	s.recheckMu.Lock()
	defer s.recheckMu.Unlock()
	if s.pendingRecheck != nil && s.pendingRecheck.timer != nil {
		s.pendingRecheck.timer.Stop()
	}
	s.pendingRecheck = nil
}

// manageProvisionalRechecks keeps a bounded, time-based reconciliation loop
// running while the disk map contains whole-disk synthesized entries.
// Event-driven invalidation alone cannot heal those entries when the boot race
// happened before the relevant udev events were observable, so a few forced
// refreshes are scheduled until the layout settles or the budget runs out.
func (s *VolumeService) manageProvisionalRechecks() {
	s.recheckMu.Lock()
	defer s.recheckMu.Unlock()

	if s.recheckInterval <= 0 {
		s.recheckInterval = 15 * time.Second
	}
	if s.maxProvisionalRechecks <= 0 {
		s.maxProvisionalRechecks = 5
	}

	if !s.hasWholeDiskSynthesizedDisks() {
		if s.pendingRecheck != nil && s.pendingRecheck.timer != nil {
			s.pendingRecheck.timer.Stop()
		}
		s.pendingRecheck = nil
		return
	}

	if s.pendingRecheck == nil {
		s.pendingRecheck = &provisionalRecheckState{attempts: s.maxProvisionalRechecks}
	}
	if s.pendingRecheck.attempts <= 0 {
		// The entry never settled; keep the exhausted state so the budget is
		// not silently restored on the next call, and stop probing.
		return
	}
	s.pendingRecheck.attempts--
	if s.pendingRecheck.timer != nil {
		s.pendingRecheck.timer.Stop()
	}
	s.pendingRecheck.timer = time.AfterFunc(s.recheckInterval, s.runProvisionalRecheck)
}

// runProvisionalRecheck forces a fresh hardware fetch and re-synchronizes the
// volume cache, letting a whole-disk synthesized entry be replaced by the real
// partition layout once the system has settled.
func (s *VolumeService) runProvisionalRecheck() {
	if s.ctx.Err() != nil {
		return
	}
	if s.hardwareClient != nil {
		s.hardwareClient.InvalidateHardwareInfo()
	}
	if err := s.getVolumesData(); err != nil {
		slog.ErrorContext(s.ctx, "Failed to refresh volume cache during provisional recheck", "err", err)
	}
}

func (s *VolumeService) MockSetProcfsGetMounts(f func() ([]*procfs.MountInfo, error)) {
	s.procfsGetMounts = f
}

// MockSetMountOps allows tests to override mount operations.
func (s *VolumeService) MockSetMountOps(
	tryMount func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
	mountFn func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
	unmountFn func(target string, force, lazy bool) error,
) {
	if fsSvc, ok := s.fs_service.(interface {
		MockSetMountOps(
			tryMount func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
			mountFn func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error),
			unmountFn func(target string, force, lazy bool) error,
		)
	}); ok {
		fsSvc.MockSetMountOps(tryMount, mountFn, unmountFn)
	}
}
