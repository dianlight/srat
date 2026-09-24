// Package volume: mount orchestration (validation, mount/unmount delegation,
// automount retry guard and mount settings patches).
package volume

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/repository"
	"github.com/shomali11/util/xhashes"
	"gitlab.com/tozd/go/errors"
)

// FilesystemOps is the subset of FilesystemServiceInterface used by the
// orchestrator: flag conversion for mount syscalls and label reads.
type FilesystemOps interface {
	MountFlagsToSyscallFlagAndData(inputFlags []dto.MountFlag) (uintptr, string, errors.E)
	GetPartitionLabel(ctx context.Context, devicePath, fsType string) (string, errors.E)
}

// Mounter performs the OS-level mount work (VolumeMountManagerInterface
// method set, redeclared here so this package never imports service).
type Mounter interface {
	Mount(md *dto.MountPointData, flags uintptr, data, mountFsType string) errors.E
	Unmount(md *dto.MountPointData, force bool) errors.E
}

// Notifier raises Home Assistant persistent notifications (nil-allowed).
type Notifier interface {
	CreatePersistentNotification(notificationID, title, message string) error
	DismissPersistentNotification(notificationID string) error
}

// automountRetryState tracks how many times an automount attempt for a given
// mount path has failed and when the next attempt is allowed.
type automountRetryState struct {
	attempts    int
	nextRetryAt time.Time
}

// OrchestratorParams wires a MountOrchestrator. HA may be nil (notifications
// are skipped). ResetAfter zero disables retry-budget reset (used by tests).
type OrchestratorParams struct {
	Ctx           context.Context
	Disks         *dto.DiskMap
	Filesystem    FilesystemOps
	Mounter       Mounter
	HA            Notifier
	EventBus      events.EventBusInterface
	Repo          repository.MountPointPathRepositoryInterface
	ProtectedMode func() bool
	Volumes       func() ([]*dto.Disk, errors.E)
	BackoffBase   time.Duration
	MaxAttempts   int
	ResetAfter    time.Duration
}

// MountOrchestrator owns mount/unmount operations, the automount retry guard
// and mount settings patches. It never touches *gorm.DB: writes go through
// the repository, cache state through *dto.DiskMap.
type MountOrchestrator struct {
	ctx      context.Context
	disks    *dto.DiskMap
	fs       FilesystemOps
	mounter  Mounter
	ha       Notifier
	eventBus events.EventBusInterface
	repo     repository.MountPointPathRepositoryInterface
	conv     converter.DtoToDbomConverterImpl

	protectedMode func() bool
	volumes       func() ([]*dto.Disk, errors.E)

	// Automount retry guard: bounds repeated mount attempts for the same
	// mount path. When the OS-level mount succeeds but the converter or
	// cache write fails, the emitted event can carry a stale IsMounted=false,
	// which re-enters the mount point handler and would otherwise loop
	// forever with one mount(2) per iteration. Attempts per path are capped
	// with exponential backoff between failures.
	automountBackoffBase time.Duration
	maxAutomountAttempts int
	// automountResetAfter is the cooldown after which an exhausted path's retry
	// budget is released, so a transient failure burst does not disable
	// automount for that path until the process restarts. Zero disables it.
	automountResetAfter time.Duration
	automountRetryMu    sync.Mutex
	automountRetries    map[string]automountRetryState
}

// NewMountOrchestrator creates a MountOrchestrator, applying production
// retry defaults for non-positive BackoffBase/MaxAttempts.
func NewMountOrchestrator(in OrchestratorParams) *MountOrchestrator {
	backoff := in.BackoffBase
	if backoff <= 0 {
		backoff = 2 * time.Second
	}
	maxAttempts := in.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	return &MountOrchestrator{
		ctx:                  in.Ctx,
		disks:                in.Disks,
		fs:                   in.Filesystem,
		mounter:              in.Mounter,
		ha:                   in.HA,
		eventBus:             in.EventBus,
		repo:                 in.Repo,
		protectedMode:        in.ProtectedMode,
		volumes:              in.Volumes,
		automountBackoffBase: backoff,
		maxAutomountAttempts: maxAttempts,
		automountResetAfter:  in.ResetAfter,
		automountRetries:     map[string]automountRetryState{},
	}
}

// MountVolume validates a mount request and delegates the OS work to the
// mounter. Validation runs before any state mutation; the mounter updates
// the cache and emits only after the mount completes (#971).
func (o *MountOrchestrator) MountVolume(md *dto.MountPointData) errors.E {
	// Early validation of required fields
	if o.protectedMode != nil && o.protectedMode() {
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

	// Reject paths the database cannot persist before any OS work (#1091).
	// Without this guard a colon path mounts at the OS level, returns
	// success, then fails async persistence and loses state on restart.
	if err := dbom.ValidateMountPointPath(md.Path); err != nil {
		return errors.WithDetails(dto.ErrorInvalidParameter,
			"DeviceId", md.DeviceId,
			"Path", md.Path,
			"Message", err.Error(),
			"SuggestedPath", dbom.SuggestSanitizedMountPointPath(md.Path),
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
		for _, disk := range o.disks.Snapshot() {
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
			return errors.WithDetails(dto.ErrorMountFail, "Detail", "Error checking mount status", "Path", md.Path, "Error", errS)
		}
		slog.DebugContext(o.ctx, "osutil.IsMounted check failed, but path not mounted, proceeding", "path", md.Path, "err", errS)
		ok = false // Ensure ok is false if IsMounted errored
	}

	if ok {
		slog.WarnContext(o.ctx, "Volume already mounted according to OS check", "device", md.DeviceId, "path", md.Path)
		return errors.WithDetails(dto.ErrorAlreadyMounted,
			"Device", md.DeviceId,
			"Path", md.Path,
			"Message", "Volume is already mounted",
		)
	}

	// Initialize flags if nil to avoid nil pointer dereference
	if md.Flags == nil {
		md.Flags = &dto.MountFlags{}
		slog.DebugContext(o.ctx, "Initialized nil Flags to empty MountFlags", "device", md.DeviceId, "path", md.Path)
	}

	flags, data, err := o.fs.MountFlagsToSyscallFlagAndData(*md.Flags)
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

	if err := o.mounter.Mount(md, flags, data, mountFsType); err != nil {
		return err
	}

	// Dismiss any existing failure notifications since the mount was successful.
	o.dismissAutomountNotification(md.DeviceId, "automount_failure")
	o.dismissAutomountNotification(md.DeviceId, "unmounted_partition")

	return nil
}

// UnmountVolume resolves a mount point from the cache and unmounts it. A
// Home Assistant–mounted share is released via a Share REMOVE event instead.
// Cache invalidation precedes every emit (#971).
func (o *MountOrchestrator) UnmountVolume(path string, force bool) errors.E {
	// Early validation of required fields
	if o.protectedMode != nil && o.protectedMode() {
		return errors.WithDetails(dto.ErrorOperationNotPermittedInProtectedMode,
			"Operation", "UnmountVolume",
			"Detail", "Unmount operation is not permitted when ProtectedMode is enabled.",
		)
	}

	// Look up mount point data from in-memory cache first
	md, ok := o.disks.GetMountPointByPath(path)
	if ok && md.Share != nil && md.Share.Status.IsHAMounted {
		slog.DebugContext(o.ctx, "Found mount point as HAMounted", "path", path)
		md.IsInvalid = true
		_ = o.eventBus.EmitShare(events.ShareEvent{
			Type:  events.EventTypes.REMOVE,
			Share: md.Share,
		})
	} else if !ok {
		slog.WarnContext(o.ctx, "Mount point not found in cache, try to umount", "path", path)
		md = &dto.MountPointData{Path: path}
	}
	return o.unmountVolume(md, force)
}

func (o *MountOrchestrator) unmountVolume(md *dto.MountPointData, force bool) errors.E {
	return o.mounter.Unmount(md, force)
}

// AllowAutomountAttempt reports whether a new mount attempt for the given
// path is currently permitted. It returns (false, true) when the attempt
// budget is exhausted (terminal state) and (false, false) while backing off.
func (o *MountOrchestrator) AllowAutomountAttempt(path string) (allowed bool, exhausted bool) {
	o.automountRetryMu.Lock()
	defer o.automountRetryMu.Unlock()

	st, ok := o.automountRetries[path]
	if !ok {
		return true, false
	}
	// Release the budget after the cooldown so a transient failure burst does
	// not disable automount for this path until the process restarts. This also
	// deletes the entry, bounding the retry map's growth.
	if o.automountResetAfter > 0 && time.Since(st.nextRetryAt) >= o.automountResetAfter {
		delete(o.automountRetries, path)
		return true, false
	}
	if st.attempts >= o.maxAutomountAttempts {
		return false, true
	}
	if time.Now().Before(st.nextRetryAt) {
		return false, false
	}
	return true, false
}

// RecordAutomountFailure registers one failed mount attempt for the given
// path and schedules the next allowed attempt with exponential backoff.
func (o *MountOrchestrator) RecordAutomountFailure(path string) {
	o.automountRetryMu.Lock()
	defer o.automountRetryMu.Unlock()

	if o.automountRetries == nil {
		o.automountRetries = map[string]automountRetryState{}
	}
	st := o.automountRetries[path]
	st.attempts++
	backoff := o.automountBackoffBase << (st.attempts - 1)
	if backoff <= 0 {
		backoff = o.automountBackoffBase
	}
	st.nextRetryAt = time.Now().Add(backoff)
	o.automountRetries[path] = st
}

// ClearAutomountRetry removes the retry state for the given path after a
// successful mount attempt.
func (o *MountOrchestrator) ClearAutomountRetry(path string) {
	o.automountRetryMu.Lock()
	defer o.automountRetryMu.Unlock()
	delete(o.automountRetries, path)
}

// SetAutomountTuning overrides the retry guard parameters (used by tests).
func (o *MountOrchestrator) SetAutomountTuning(backoffBase time.Duration, maxAttempts int) {
	o.automountRetryMu.Lock()
	defer o.automountRetryMu.Unlock()
	o.automountBackoffBase = backoffBase
	o.maxAutomountAttempts = maxAttempts
}

// InferMountPointType defaults a missing mount point type from its path.
func InferMountPointType(mountPoint *dto.MountPointData) string {
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

// PatchMountPointSettings applies a partial mount configuration update. The
// row write goes through the repository; the refreshed DTO is then merged
// into the cache and broadcast. Cache mutation precedes the emit (#971).
func (o *MountOrchestrator) PatchMountPointSettings(root string, path string, patchData dto.MountPointData) (*dto.MountPointData, errors.E) {
	dbMountData, err := o.repo.Patch(o.ctx, root, path, patchData)
	if err != nil {
		return nil, err
	}

	currentDto := dto.MountPointData{}
	// The converter uses the volume cache only to resolve the partition from
	// the DeviceId; on load failure pass nil so the fallback cache-update path
	// below still keeps the persisted state consistent.
	var disksContext []*dto.Disk
	if o.volumes != nil {
		refreshed, errVolumes := o.volumes()
		if errVolumes != nil {
			slog.WarnContext(o.ctx, "PatchMountPointSettings: failed to load volumes data for partition resolution", "err", errVolumes)
		} else {
			disksContext = refreshed
		}
	}
	if convErr := o.conv.MountPointPathToMountPointData(dbMountData, &currentDto, disksContext); convErr != nil {
		return nil, errors.WithStack(convErr)
	}
	// Update cached mount point data
	if currentDto.Partition != nil && currentDto.Partition.DiskId != nil && currentDto.Partition.Id != nil {
		err := o.disks.AddOrUpdateMountPoint(*currentDto.Partition.DiskId, *currentDto.Partition.Id, currentDto)
		if err != nil {
			slog.WarnContext(o.ctx, "Failed to update mount point in cache", "err", err)
		}
	} else {
		// Fallback: partition could not be resolved.
		updated := false
		if existing, ok := o.disks.GetMountPointByPath(path); ok {
			if existing.Partition != nil && existing.Partition.DiskId != nil && existing.Partition.Id != nil {
				existing.IsToMountAtStartup = currentDto.IsToMountAtStartup
				err := o.disks.AddOrUpdateMountPoint(*existing.Partition.DiskId, *existing.Partition.Id, *existing)
				if err != nil {
					slog.WarnContext(o.ctx, "Failed to update mount point in fallback cache update", "err", err)
				}
				updated = true
			}
		}
		if !updated {
			for dk, d := range o.disks.Snapshot() {
				if d.Partitions == nil {
					continue
				}
				for pid, part := range *d.Partitions {
					if part.MountPointData == nil {
						continue
					}
					if existing, ok := (*part.MountPointData)[path]; ok {
						existing.IsToMountAtStartup = currentDto.IsToMountAtStartup
						err := o.disks.AddOrUpdateMountPoint(dk, pid, existing)
						if err != nil {
							slog.WarnContext(o.ctx, "Failed to update mount point in fallback cache update", "err", err)
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
	_ = o.eventBus.EmitMountPoint(events.MountPointEvent{
		Type:       events.EventTypes.UPDATE,
		MountPoint: &currentDto,
	})
	return &currentDto, nil
}

// GetDevicePathByDeviceID resolves a partition device id to its OS device
// path through the partition map; disk IDs never match here.
func (o *MountOrchestrator) GetDevicePathByDeviceID(deviceID string) (string, errors.E) {
	part, _, ok := o.disks.GetPartitionByID(deviceID)
	if !ok {
		return "", errors.WithDetails(dto.ErrorNotFound,
			"Message", "partition not found",
			"DeviceId", deviceID,
		)
	}
	path := o.disks.GetPartitionDevicePath(part)
	if path == "" {
		return "", errors.WithDetails(dto.ErrorDeviceNotFound,
			"Message", "device path not available",
			"DeviceId", deviceID,
		)
	}
	return path, nil
}

// createAutomountFailureNotification creates a persistent notification for failed automount operations
func (o *MountOrchestrator) createAutomountFailureNotification(mountPath, device string, err errors.E) {
	if o.ha == nil {
		slog.DebugContext(o.ctx, "Home Assistant service not available, skipping automount failure notification")
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

	notifyErr := o.ha.CreatePersistentNotification(notificationID, title, message)
	if notifyErr != nil {
		slog.ErrorContext(o.ctx, "Failed to create automount failure notification", "mount_path", mountPath, "device", device, "err", notifyErr)
	} else {
		slog.InfoContext(o.ctx, "Created automount failure notification", "mount_path", mountPath, "device", device, "notification_id", notificationID)
	}
}

// dismissAutomountNotification dismisses an automount-related notification
func (o *MountOrchestrator) dismissAutomountNotification(deviceID string, notificationType string) {
	if o.ha == nil {
		return
	}

	notificationID := fmt.Sprintf("srat_%s_%s", notificationType, xhashes.SHA1(deviceID))

	notifyErr := o.ha.DismissPersistentNotification(notificationID)
	if notifyErr != nil {
		slog.WarnContext(o.ctx, "Failed to dismiss automount notification", "mount_path", deviceID, "notification_type", notificationType, "err", notifyErr)
	} else {
		slog.DebugContext(o.ctx, "Dismissed automount notification", "mount_path", deviceID, "notification_type", notificationType, "notification_id", notificationID)
	}
}
