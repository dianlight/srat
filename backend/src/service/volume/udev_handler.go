// Package volume: udev and volume event reactions (partition/disk hotplug,
// partition and mount point synchronization, format follow-up).
package volume

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/tlog"
	"github.com/pilebones/go-udev/netlink"
	"github.com/prometheus/procfs"
	"gitlab.com/tozd/go/errors"
)

// Sentinel errors reported by ConsumeUdevChannels when the udev monitor
// goroutine has exited and its output channels have been closed. Callers use
// these to decide whether to reconnect (any non-nil return) or shut down
// cleanly (nil return on context cancellation).
var (
	ErrUdevQueueClosed     = errors.New("udev event queue closed: monitor exited")
	ErrUdevErrorChanClosed = errors.New("udev error channel closed: monitor exited")
)

// HardwareInvalidator clears cached hardware state (nil-allowed).
type HardwareInvalidator interface {
	InvalidateHardwareInfo()
}

// HandlerParams wires a UdevHandler. Hardware may be nil. Refresh
// re-synchronizes the volume cache; ResetRecheck restarts the provisional
// recheck budget; ProcfsMounts parses /proc/self/mountinfo (nil-allowed).
type HandlerParams struct {
	Ctx          context.Context
	Disks        *dto.DiskMap
	Hardware     HardwareInvalidator
	EventBus     events.EventBusInterface
	Repo         *MountRepository
	Orchestrator *MountOrchestrator
	Filesystem   FilesystemOps
	Refresh      func() errors.E
	ResetRecheck func()
	ProcfsMounts func() ([]*procfs.MountInfo, error)
}

// UdevHandler reacts to block-device uevents and volume events. It owns no
// persistence or mount syscalls itself: storage goes through MountRepository,
// mounts through MountOrchestrator.
type UdevHandler struct {
	ctx          context.Context
	disks        *dto.DiskMap
	hardware     HardwareInvalidator
	eventBus     events.EventBusInterface
	repo         *MountRepository
	orchestrator *MountOrchestrator
	fs           FilesystemOps
	refresh      func() errors.E
	resetRecheck func()
	procfsMounts func() ([]*procfs.MountInfo, error)

	// probe, when non-nil, is invoked by ProcessUdevEvent before the event
	// is dispatched. Tests use it to observe consumed uevents.
	probe func(netlink.UEvent)
}

// NewUdevHandler creates a UdevHandler.
func NewUdevHandler(in HandlerParams) *UdevHandler {
	return &UdevHandler{
		ctx:          in.Ctx,
		disks:        in.Disks,
		hardware:     in.Hardware,
		eventBus:     in.EventBus,
		repo:         in.Repo,
		orchestrator: in.Orchestrator,
		fs:           in.Filesystem,
		refresh:      in.Refresh,
		resetRecheck: in.ResetRecheck,
		procfsMounts: in.ProcfsMounts,
	}
}

// SetProbe installs the test observation hook for consumed uevents.
func (h *UdevHandler) SetProbe(probe func(netlink.UEvent)) {
	h.probe = probe
}

// SetHardware replaces the hardware invalidator (used by tests that swap the
// client after construction).
func (h *UdevHandler) SetHardware(hardware HardwareInvalidator) {
	h.hardware = hardware
}

// SetContext replaces the handler context (used by tests that swap the
// service context after construction).
func (h *UdevHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

// SetProcfsMounts replaces the procfs parser (used by tests).
func (h *UdevHandler) SetProcfsMounts(f func() ([]*procfs.MountInfo, error)) {
	h.procfsMounts = f
}

// MatchPartitionWithDevName reports whether a partition answers to a kernel
// device name in any of its spellings (full path, legacy path, bare name).
func MatchPartitionWithDevName(partition *dto.Partition, devName string) bool {
	if partition == nil || devName == "" {
		return false
	}

	fullDevName := devName
	if !strings.HasPrefix(devName, "/dev/") {
		fullDevName = filepath.Join("/dev", devName)
	}

	candidates := []string{}
	if partition.DevicePath != nil {
		candidates = append(candidates, *partition.DevicePath)
	}
	if partition.LegacyDevicePath != nil {
		candidates = append(candidates, *partition.LegacyDevicePath)
	}
	if partition.LegacyDeviceName != nil {
		candidates = append(candidates, *partition.LegacyDeviceName)
	}
	if partition.Id != nil {
		candidates = append(candidates, *partition.Id)
	}

	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		if trimmed == devName || trimmed == fullDevName {
			return true
		}
		if filepath.Base(trimmed) == devName {
			return true
		}
	}

	return false
}

// FindPartitionByDevName scans the disk map for the partition behind a
// kernel device name.
func (h *UdevHandler) FindPartitionByDevName(devName string) (*dto.Partition, string, bool) {
	if h.disks == nil || devName == "" {
		return nil, "", false
	}

	for diskID, disk := range h.disks.Snapshot() {
		if disk.Partitions == nil {
			continue
		}
		for _, partition := range *disk.Partitions {
			if MatchPartitionWithDevName(&partition, devName) {
				p := partition
				return &p, diskID, true
			}
		}
	}

	return nil, "", false
}

// ProcessUdevEvent dispatches one block-device uevent to the matching
// add/remove handler. Non-block subsystems and non-disk/partition devices are
// dropped here so the platform-specific monitor loop stays minimal.
func (h *UdevHandler) ProcessUdevEvent(uevent netlink.UEvent) {
	if h.probe != nil {
		h.probe(uevent)
	}

	subsystem, ok := uevent.Env["SUBSYSTEM"]
	if !ok || subsystem != "block" {
		return
	}
	action := uevent.Action
	devName := uevent.Env["DEVNAME"]
	devType := uevent.Env["DEVTYPE"]

	slog.DebugContext(h.ctx, "Received Udev block event", "action", action, "devname", devName, "devtype", devType, "env", uevent.Env)

	if devType != "disk" && devType != "partition" {
		slog.DebugContext(h.ctx, "Ignoring Udev event for non-disk/partition block device", "devname", devName, "devtype", devType)
		return
	}

	switch {
	case devType == "disk" && action == netlink.REMOVE:
		// Removal is handled by invalidating the hardware cache and
		// re-synchronizing the volume map; reconciliation evicts the
		// disk because it is absent from the new snapshot.
		h.HandleDiskUdevRemoveEvent(devName)
	case devType == "disk" && action == netlink.ADD:
		slog.InfoContext(h.ctx, "Processing block device event", "action", action, "devname", devName)
		if h.hardware != nil {
			h.hardware.InvalidateHardwareInfo()
		}
		if err := h.refresh(); err != nil {
			slog.ErrorContext(h.ctx, "Failed to get volumes data after udev event", "err", err)
		}
	case devType == "disk" && action == netlink.CHANGE:
		slog.InfoContext(h.ctx, "Ignore: Processing block device change event", "action", action, "devname", devName)
	case devType == "partition" && action == netlink.ADD:
		slog.InfoContext(h.ctx, "Processing partition addition event", "action", action, "devname", devName)
		if h.HandlePartitionUdevAddEvent(devName) {
			return
		}
		if h.hardware != nil {
			h.hardware.InvalidateHardwareInfo()
		}
		if err := h.refresh(); err != nil {
			slog.ErrorContext(h.ctx, "Failed to refresh volume cache after partition add event", "devname", devName, "err", err)
		}
		// The Supervisor API may not have processed its own udev events yet
		// when a flapping device's partition ADD event arrives, causing the
		// initial refresh to return stale data with a synthetic whole-disk
		// entry. Schedule a delayed retry so the Supervisor has time to
		// pick up the real partitions, and reset the provisional recheck
		// budget so the bounded recheck loop gets another chance.
		h.SchedulePartitionAddRetry(devName)
	case devType == "partition" && action == netlink.REMOVE:
		slog.InfoContext(h.ctx, "Processing partition removal event", "action", action, "devname", devName)
		h.HandlePartitionUdevRemoveEvent(devName)
	}
}

// SchedulePartitionAddRetry schedules a delayed retry of the volume refresh
// after a partition ADD event that was not found in the DiskMap. The
// Supervisor API may not have processed its own udev events yet when a
// flapping device's partition ADD event arrives, so the initial refresh can
// return stale data containing only a synthetic whole-disk entry. The 500ms
// delay gives the Supervisor time to pick up the real partitions; the retry
// also resets the provisional recheck budget so the bounded recheck loop gets
// another chance to settle the layout.
func (h *UdevHandler) SchedulePartitionAddRetry(devName string) {
	time.AfterFunc(500*time.Millisecond, func() {
		if h.ctx.Err() != nil {
			return
		}
		slog.DebugContext(h.ctx, "Delayed retry: refreshing volume cache for partition", "devname", devName)
		if h.resetRecheck != nil {
			h.resetRecheck()
		}
		if h.hardware != nil {
			h.hardware.InvalidateHardwareInfo()
		}
		if err := h.refresh(); err != nil {
			slog.ErrorContext(h.ctx, "Delayed retry: failed to refresh volume cache for partition", "devname", devName, "err", err)
		}
	})
}

// logUdevMonitorError reports a non-fatal monitor error. Malformed uevents
// (e.g. non-standard kernel formatting) are logged at debug so a single bad
// message does not spam the addon log; other errors stay at error level.
func (h *UdevHandler) logUdevMonitorError(err error) {
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "unable to parse uevent") {
		if strings.Contains(err.Error(), "invalid env data") {
			slog.DebugContext(h.ctx, "Ignoring malformed uevent with invalid env data",
				"err", err,
				"detail", "This can occur when kernel sends events with non-standard formatting")
			return
		}
		slog.DebugContext(h.ctx, "Failed to parse uevent, skipping",
			"err", err,
			"detail", "Event format not recognized or incompatible")
		return
	}
	slog.ErrorContext(h.ctx, "Error received from Udev monitor", "err", err)
}

// ConsumeUdevChannels reads from the monitor's queue and error channels until
// one of them closes or the handler context is cancelled.
//
// The go-udev monitor goroutine exits permanently on fatal receive errors
// (e.g. ENOBUFS when the kernel netlink socket buffer fills up during a udev
// burst from a flapping USB device) and then closes both channels. Reading
// from a closed channel returns the zero value immediately, so without the
// ok-idiom checks below the loop would spin forever on empty UEvents and
// never observe another hardware event. Returning a sentinel error instead
// lets the caller (the platform-specific udevEventHandler) reconnect with
// backoff, which is what eventually re-populates the disk map once the
// hardware has settled.
//
// A nil return means the handler context was cancelled (clean shutdown).
func (h *UdevHandler) ConsumeUdevChannels(queue <-chan netlink.UEvent, errorChan <-chan error) error {
	for {
		select {
		case <-h.ctx.Done():
			return nil
		case uevent, ok := <-queue:
			if !ok {
				return errors.WithStack(ErrUdevQueueClosed)
			}
			h.ProcessUdevEvent(uevent)
		case err, ok := <-errorChan:
			if !ok {
				return errors.WithStack(ErrUdevErrorChanClosed)
			}
			h.logUdevMonitorError(err)
		}
	}
}

// HandlePartitionUdevAddEvent retries automount for startup-marked,
// unmounted mount points of an added partition.
func (h *UdevHandler) HandlePartitionUdevAddEvent(devName string) bool {
	partition, _, found := h.FindPartitionByDevName(devName)
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

		if allowed, exhausted := h.orchestrator.AllowAutomountAttempt(mountPoint.Path); !allowed {
			if exhausted {
				slog.WarnContext(h.ctx, "Automount attempts exhausted for partition add event, giving up",
					"devname", devName, "path", mountPoint.Path, "max_attempts", h.orchestrator.maxAutomountAttempts)
			} else {
				slog.DebugContext(h.ctx, "Skipping automount retry for partition add event, backing off",
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

		err := h.orchestrator.MountVolume(&mountCopy)
		if err != nil {
			if errors.Is(err, dto.ErrorAlreadyMounted) {
				slog.InfoContext(h.ctx, "Mount point already mounted during partition add automount retry", "devname", devName, "path", mountCopy.Path)
				h.orchestrator.ClearAutomountRetry(mountCopy.Path)
				handled = true
				continue
			}
			slog.WarnContext(h.ctx, "Failed automount retry for partition add event", "devname", devName, "path", mountCopy.Path, "err", err)
			h.orchestrator.RecordAutomountFailure(mountCopy.Path)
			continue
		}

		h.orchestrator.ClearAutomountRetry(mountCopy.Path)
		handled = true
	}

	return handled
}

// HandlePartitionUdevRemoveEvent force-unmounts a removed partition's mount
// points, drops it from the cache and refreshes the volume map.
func (h *UdevHandler) HandlePartitionUdevRemoveEvent(devName string) {
	partition, diskID, found := h.FindPartitionByDevName(devName)
	if !found || partition == nil || partition.Id == nil || *partition.Id == "" {
		// Partition was never tracked in the DiskMap — its removal doesn't
		// change our state.  Do NOT invalidate the hardware cache here;
		// during hardware flapping (constant remove/add cycles) this would
		// cause perpetual cache churn and the disk can settle on stale data.
		slog.DebugContext(h.ctx, "Ignoring partition remove for untracked partition", "devname", devName)
		return
	}

	if partition.MountPointData != nil {
		for _, mountPoint := range *partition.MountPointData {
			if mountPoint.Path == "" || !mountPoint.IsMounted {
				continue
			}
			mountCopy := mountPoint
			if err := h.orchestrator.unmountVolume(&mountCopy, true); err != nil {
				slog.WarnContext(h.ctx, "Failed to unmount path during partition remove handling", "path", mountCopy.Path, "devname", devName, "err", err)
			}
		}
	}

	removed := h.disks.RemovePartition(diskID, *partition.Id)
	if !removed {
		slog.DebugContext(h.ctx, "Partition removal event did not find cache entry to delete", "disk_id", diskID, "partition_id", *partition.Id, "devname", devName)
	}

	if h.hardware != nil {
		h.hardware.InvalidateHardwareInfo()
	}
	if err := h.refresh(); err != nil {
		slog.ErrorContext(h.ctx, "Failed to refresh volume cache after partition remove event", "devname", devName, "err", err)
	}
}

// HandleDiskUdevRemoveEvent reacts to a block device being removed from the
// host. The hardware snapshot is invalidated and the volume cache refreshed;
// reconciliation then evicts the disk because it is absent from the new
// snapshot (see the pruning step in the facade refresh).
func (h *UdevHandler) HandleDiskUdevRemoveEvent(devName string) {
	slog.InfoContext(h.ctx, "Processing block device removal event", "devname", devName)
	if h.hardware != nil {
		h.hardware.InvalidateHardwareInfo()
	}
	if err := h.refresh(); err != nil {
		slog.ErrorContext(h.ctx, "Failed to refresh volume cache after disk removal event", "devname", devName, "err", err)
	}
}

// HandlePartitionEvent syncs one or many changed partitions against the
// procfs snapshot carried by the event (parsed once here when absent).
func (h *UdevHandler) HandlePartitionEvent(ctx context.Context, e events.PartitionEvent) errors.E {
	if len(e.Partitions) > 0 {
		// Batch mode (H2): all changed partitions of one disk share a single
		// procfs snapshot carried by the event. If the emitter provided none
		// (e.g. an external emitter), parse once here and reuse it.
		mountInfos := e.MountInfos
		if mountInfos == nil && h.procfsMounts != nil {
			parsed, errS := h.procfsMounts()
			if errS != nil {
				slog.ErrorContext(ctx, "Failed to get current mount information from procfs", "disk_id", *e.Disk.Id, "err", errS)
				return errors.WithStack(errS)
			}
			mountInfos = parsed
		}
		var syncErr errors.E
		for _, part := range e.Partitions {
			if err := h.SyncPartitionMountData(ctx, e.Disk, part, mountInfos); err != nil {
				slog.WarnContext(ctx, "Failed to sync partition mount data", "disk_id", *e.Disk.Id, "partition_id", *part.Id, "err", err)
				// H9: propagate the first failure back through the
				// synchronous bus so the emitter sees the DB persist failure
				// instead of a false success. Remaining partitions are still
				// synced; the first error is what the caller observes.
				if syncErr == nil {
					syncErr = err
				}
			}
		}
		return syncErr
	}

	// Single-partition mode (legacy emitters, e.g. udev events, tests)
	mountInfos := e.MountInfos
	if mountInfos == nil && h.procfsMounts != nil {
		parsed, errS := h.procfsMounts()
		if errS != nil {
			slog.ErrorContext(ctx, "Failed to get current mount information from procfs", "disk_id", *e.Disk.Id, "partition_id", *e.Partition.Id, "err", errS)
			return errors.WithStack(errS)
		}
		mountInfos = parsed
	}
	return h.SyncPartitionMountData(ctx, e.Disk, e.Partition, mountInfos)
}

// SyncPartitionMountData reconciles a partition's mount points with the
// current procfs snapshot: it loads persisted mount points from the
// repository, adds missing ones to the in-memory cache, updates mount points
// present in procfs and marks stale ones as unmounted. Cache mutations always
// precede their emits (#971).
func (h *UdevHandler) SyncPartitionMountData(ctx context.Context, disk *dto.Disk, part *dto.Partition, mountInfos []*procfs.MountInfo) errors.E {
	tlog.TraceContext(ctx, "Processing partition event for mount data sync", "disk_id", *disk.Id, "partition_id", *part.Id)

	if part.DevicePath == nil || *part.DevicePath == "" {
		slog.DebugContext(ctx, "Skipping partition with nil or empty device path", "disk_id", *disk.Id)
		return nil
	}
	if part.DiskId == nil || *part.DiskId == "" {
		part.DiskId = disk.Id
	}

	mountData, err := h.repo.LoadByDevice(part)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to load mount point data from DB for partition", "disk_id", *disk.Id, "partition_id", *part.Id, "err", err)
		return err
	}
	// Add missing mount points from DB to in-memory cache
	for _, md := range mountData {
		err := h.disks.AddOrUpdateMountPoint(*part.DiskId, *part.Id, *md)
		if err != nil {
			slog.WarnContext(h.ctx, "Failed to add mount point to disk map during partition event handling", "disk_id", *part.DiskId, "partition_id", *part.Id, "mount_path", md.Path, "err", err)
			continue
		}
	}

	// Update existing mount points with current mount info
	tlog.TraceContext(ctx, "Synchronizing mount points for partition", "disk_id", *disk.Id, "partition_id", *part.Id, "mount_data_count", len(mountData), "procfs_mounts_count", len(mountInfos))
	for _, prtstate := range mountInfos {
		iw := osutil.IsWritable(prtstate.MountPoint)
		if mountPoint, ok := h.disks.GetMountPoint(*part.DiskId, *part.Id, prtstate.MountPoint); ok {
			tlog.TraceContext(ctx, "Found existing mount point in cache for partition, updating state", "disk_id", *disk.Id, "partition_id", *part.Id, "mount_path", mountPoint.Path, "is_mounted", mountPoint.IsMounted)
			oldstate := mountPoint.IsMounted
			mountPoint.IsMounted = true
			mountPoint.Path = prtstate.MountPoint
			mountPoint.Root = prtstate.Root
			mountPoint.RefreshVersion = h.disks.CurrentRefreshVersion()
			mountPoint.IsWriteSupported = &iw
			if err := mountPoint.Flags.Scan(prtstate.Options); err != nil {
				slog.WarnContext(ctx, "Failed to scan mount flags", "mount_path", prtstate.MountPoint, "error", err)
			}
			if err := mountPoint.CustomFlags.Scan(prtstate.SuperOptions); err != nil {
				slog.WarnContext(ctx, "Failed to scan custom mount flags", "mount_path", prtstate.MountPoint, "error", err)
			}
			mountPoint.FSType = &prtstate.FSType
			mountPoint.Type = "ADDON"
			err := h.disks.AddOrUpdateMountPoint(*part.DiskId, *part.Id, *mountPoint)
			if err != nil {
				slog.WarnContext(h.ctx, "Failed to add mount point to disk map", "disk_id", *part.DiskId, "partition_id", *part.Id, "mount_path", mountPoint.Path, "err", err)
				continue
			}
			if !oldstate {
				_ = h.eventBus.EmitMountPoint(events.MountPointEvent{
					Type:       events.EventTypes.UPDATE,
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
				RefreshVersion:   h.disks.CurrentRefreshVersion(),
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
			if existingMP, ok := h.disks.GetMountPointByPath(prtstate.MountPoint); ok {
				mountPoint.IsToMountAtStartup = existingMP.IsToMountAtStartup
				mountPoint.Flags = existingMP.Flags
				mountPoint.CustomFlags = existingMP.CustomFlags
			} else if dbMP, found, dbErr := h.repo.LoadByPath(prtstate.MountPoint, prtstate.Root); dbErr != nil {
				slog.WarnContext(ctx, "Failed to load persisted mount point configuration", "mount_path", prtstate.MountPoint, "err", dbErr)
			} else if found {
				mountPoint.IsToMountAtStartup = dbMP.IsToMountAtStartup
				mountPoint.Flags = dbMP.Flags
				mountPoint.CustomFlags = dbMP.CustomFlags
			}
			err := h.disks.AddOrUpdateMountPoint(*part.DiskId, *part.Id, mountPoint)
			if err != nil {
				slog.WarnContext(h.ctx, "Failed to add mount point to disk map", "disk_id", *part.DiskId, "partition_id", *part.Id, "mount_path", mountPoint.Path, "err", err)
				continue
			}
			_ = h.eventBus.EmitMountPoint(events.MountPointEvent{
				Type:       events.EventTypes.ADD,
				MountPoint: &mountPoint,
			})
			continue
		}
	}

	tlog.TraceContext(ctx, "Marking stale mount points as unmounted for partition", "disk_id", *disk.Id, "partition_id", *part.Id)
	for _, mountPoint := range h.disks.GetMountPointsForPartition(*part.DiskId, *part.Id) {
		tlog.TraceContext(ctx, " --> Checking mount point for staleness",
			"disk_id", *disk.Id,
			"partition_id", *part.Id,
			"mount_path", mountPoint.Path,
			"refresh_version", mountPoint.RefreshVersion,
			"current_refresh_version", h.disks.CurrentRefreshVersion(),
			"is_mounted", mountPoint.IsMounted,
			"is_to_mount_at_startup", (mountPoint.IsToMountAtStartup != nil && *mountPoint.IsToMountAtStartup),
		)
		if (mountPoint.RefreshVersion != h.disks.CurrentRefreshVersion()) &&
			(mountPoint.IsMounted || (mountPoint.IsToMountAtStartup != nil && *mountPoint.IsToMountAtStartup)) {
			tlog.DebugContext(ctx, "Marking mount point as unmounted since not found in procfs mounts", "disk_id", *disk.Id, "partition_id", *part.Id, "mount_path", mountPoint.Path)
			oldtstate := mountPoint.IsMounted
			mountPoint.IsMounted = false
			mountPoint.RefreshVersion = h.disks.CurrentRefreshVersion()
			err := h.disks.AddOrUpdateMountPoint(*part.DiskId, *part.Id, *mountPoint)
			if err != nil {
				slog.WarnContext(h.ctx, "Failed to add mount point to disk map", "disk_id", *part.DiskId, "partition_id", *part.Id, "mount_path", mountPoint.Path, "err", err)
				continue
			}
			if oldtstate || (mountPoint.IsToMountAtStartup != nil && *mountPoint.IsToMountAtStartup) {
				_ = h.eventBus.EmitMountPoint(events.MountPointEvent{
					Type:       events.EventTypes.UPDATE,
					MountPoint: mountPoint,
				})
			}
		}
	}
	tlog.TraceContext(ctx, "Done synchronizing mount points for partition", "disk_id", *disk.Id, "partition_id", *part.Id)

	return nil
}

// HandleMountPointEvent persists a mount point event and automounts
// newly added, startup-marked, unmounted mount points. Persistence precedes
// the mount attempt; the automount guard bounds failure retries.
func (h *UdevHandler) HandleMountPointEvent(ctx context.Context, e events.MountPointEvent) errors.E {
	if e.MountPoint.Type == "" {
		e.MountPoint.Type = InferMountPointType(e.MountPoint)
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
	err := h.repo.Persist(e.MountPoint)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to persist mount point on event", "mount_point", e.MountPoint, "err", err)
		return err
	}
	if (e.Type == events.EventTypes.ADD || e.Type == events.EventTypes.UPDATE) && !e.MountPoint.IsMounted && e.MountPoint.IsToMountAtStartup != nil && *e.MountPoint.IsToMountAtStartup {
		if allowed, exhausted := h.orchestrator.AllowAutomountAttempt(e.MountPoint.Path); !allowed {
			if exhausted {
				slog.WarnContext(ctx, "Automount attempts exhausted for mount point, giving up",
					"mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId, "max_attempts", h.orchestrator.maxAutomountAttempts)
			} else {
				slog.DebugContext(ctx, "Skipping automount attempt, backing off",
					"mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId)
			}
			return nil
		}
		slog.InfoContext(ctx, "New mount point added and not mounted, attempting to mount", "mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId)
		err = h.orchestrator.MountVolume(e.MountPoint)
		if err != nil {
			if errors.Is(err, dto.ErrorAlreadyMounted) {
				slog.InfoContext(ctx, "Mount point already mounted during automount attempt", "mount_point", e.MountPoint.Path, "device_id", e.MountPoint.DeviceId)
				h.orchestrator.ClearAutomountRetry(e.MountPoint.Path)
				return nil
			}
			slog.ErrorContext(ctx, "Failed to mount volume on event", "mount_point", e.MountPoint, "err", err)
			h.orchestrator.RecordAutomountFailure(e.MountPoint.Path)
			h.orchestrator.createAutomountFailureNotification(e.MountPoint.Path, e.MountPoint.DeviceId, err)
		} else {
			h.orchestrator.ClearAutomountRetry(e.MountPoint.Path)
		}
	}
	return nil
}

// HandleFilesystemTaskEvent refreshes the volume cache after a successful
// format task, patches the cached partition metadata and broadcasts the
// updated disk. The broadcast follows the cache patch (#971).
func (h *UdevHandler) HandleFilesystemTaskEvent(ctx context.Context, e events.FilesystemTaskEvent) errors.E {
	if e.Task == nil || !strings.EqualFold(e.Task.Operation, "format") || !strings.EqualFold(e.Task.Status, "success") {
		return nil
	}

	slog.InfoContext(ctx, "Refreshing volume cache after successful format task", "device", e.Task.Device, "filesystem_type", e.Task.FilesystemType)

	if h.hardware != nil {
		h.hardware.InvalidateHardwareInfo()
	}

	if err := h.refresh(); err != nil {
		slog.ErrorContext(ctx, "Failed to refresh volume cache after format success", "device", e.Task.Device, "err", err)
		return err
	}

	h.patchPartitionAfterFormat(ctx, e.Task.Device, e.Task.FilesystemType, e.Task.Label)

	disk := h.findDiskForDevicePath(e.Task.Device)
	if disk == nil {
		slog.DebugContext(ctx, "No disk found to broadcast after format refresh", "device", e.Task.Device)
		return nil
	}

	if err := h.eventBus.EmitDisk(events.DiskEvent{
		Type: events.EventTypes.UPDATE,
		Disk: disk,
	}); err != nil {
		slog.WarnContext(ctx, "Failed to emit disk update event after format refresh", "device", e.Task.Device, "err", err)
	}

	return nil
}

// patchPartitionAfterFormat overwrites the cached partition name and filesystem
// type after a successful format. The hardware inventory can stay stale for
// minutes, so the DiskMap cannot rely on it alone: the requested label carried
// by the format task is authoritative, with a live GetPartitionLabel read as
// fallback for empty-label formats.
func (h *UdevHandler) patchPartitionAfterFormat(ctx context.Context, devicePath, fsType, label string) {
	if h.disks == nil || strings.TrimSpace(devicePath) == "" {
		return
	}
	diskID, partitionID := h.findDiskAndPartitionForDevicePath(devicePath)
	if diskID == "" || partitionID == "" {
		slog.DebugContext(ctx, "No partition found to patch after format", "device", devicePath)
		return
	}
	part, ok := h.disks.GetPartition(diskID, partitionID)
	if !ok {
		return
	}
	patched := part
	if strings.TrimSpace(fsType) != "" {
		patched.FsType = new(fsType)
	}
	resolvedLabel := strings.TrimSpace(label)
	if resolvedLabel == "" && h.fs != nil {
		lookupFsType := strings.TrimSpace(fsType)
		if lookupFsType == "" && part.FsType != nil {
			lookupFsType = strings.TrimSpace(*part.FsType)
		}
		if lookupFsType != "" {
			if liveLabel, err := h.fs.GetPartitionLabel(ctx, strings.TrimSpace(devicePath), lookupFsType); err == nil {
				resolvedLabel = strings.TrimSpace(liveLabel)
			} else {
				slog.WarnContext(ctx, "Failed to re-read partition label after format, keeping cached name", "device", devicePath, "err", err)
			}
		}
	}
	// Only overwrite the cached name with a non-empty authoritative label.
	// Empty-label formats keep the hardware value (which falls back to the
	// partition entry name) to avoid clearing to a wrong empty state.
	if resolvedLabel != "" {
		patched.Name = new(resolvedLabel)
	}
	// Skip the write when nothing changed to avoid needless cache churn.
	if (patched.Name == nil && part.Name == nil || patched.Name != nil && part.Name != nil && *patched.Name == *part.Name) &&
		(patched.FsType == nil && part.FsType == nil || patched.FsType != nil && part.FsType != nil && *patched.FsType == *part.FsType) {
		return
	}
	if err := h.disks.AddPartition(diskID, patched); err != nil {
		slog.WarnContext(ctx, "Failed to patch partition name after format", "device", devicePath, "disk", diskID, "partition", partitionID, "err", err)
	}
}

func (h *UdevHandler) findDiskAndPartitionForDevicePath(devicePath string) (string, string) {
	if h.disks == nil {
		return "", ""
	}
	normalizedDevice := strings.TrimSpace(devicePath)
	for diskID, disk := range h.disks.Snapshot() {
		if disk == nil || disk.Partitions == nil {
			continue
		}
		for partitionID, partition := range *disk.Partitions {
			if strings.TrimSpace(h.disks.GetPartitionDevicePath(&partition)) == normalizedDevice {
				return diskID, partitionID
			}
		}
	}
	return "", ""
}

func (h *UdevHandler) findDiskForDevicePath(devicePath string) *dto.Disk {
	if h.disks == nil || h.disks.Len() == 0 {
		return nil
	}

	normalizedDevice := strings.TrimSpace(devicePath)
	for _, disk := range h.disks.DeepCopyAll() {
		if disk.Partitions == nil {
			continue
		}
		for _, partition := range *disk.Partitions {
			if strings.TrimSpace(h.disks.GetPartitionDevicePath(&partition)) == normalizedDevice {
				return disk
			}
		}
	}

	return nil
}

// FindDiskForDevicePath resolves an OS device path to its cached disk.
func (h *UdevHandler) FindDiskForDevicePath(devicePath string) *dto.Disk {
	return h.findDiskForDevicePath(devicePath)
}
