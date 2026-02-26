package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// VolumeMountManagerInterface defines the contract for OS-level mount/unmount operations.
// Validation and higher-level coordination are handled by the caller (VolumeService).
type VolumeMountManagerInterface interface {
	Mount(md *dto.MountPointData, flags uintptr, data, mountFsType string) errors.E
	Unmount(md *dto.MountPointData, force bool) errors.E
}

// volumeMountManager handles low-level mount and unmount OS operations.
// It is responsible for creating/removing mount directories, calling the
// filesystem service for actual mount/unmount, updating the in-memory disk
// cache, and emitting MountPoint events.
// Validation and cache lookups are performed by the caller (VolumeService).
type volumeMountManager struct {
	ctx       context.Context
	fsService FilesystemServiceInterface
	disks     *dto.DiskMap
	convMDto  converter.MountToDtoImpl
	eventBus  events.EventBusInterface
}

// VolumeMountManagerParams holds the fx-injectable dependencies for VolumeMountManager.
type VolumeMountManagerParams struct {
	fx.In
	Ctx       context.Context
	FsService FilesystemServiceInterface
	Disks     *dto.DiskMap
	EventBus  events.EventBusInterface
}

// NewVolumeMountManager creates a new VolumeMountManager via fx dependency injection.
func NewVolumeMountManager(in VolumeMountManagerParams) VolumeMountManagerInterface {
	return &volumeMountManager{
		ctx:       in.Ctx,
		fsService: in.FsService,
		disks:     in.Disks,
		convMDto:  converter.MountToDtoImpl{},
		eventBus:  in.EventBus,
	}
}

// Mount performs the actual OS-level mount operation: creates the mount
// directory, delegates to the filesystem adapter, updates the in-memory
// cache, and emits a MountPointEvent on success.
// Pre-conditions: md.Partition.DevicePath must be non-nil and non-empty.
func (m *volumeMountManager) Mount(md *dto.MountPointData, flags uintptr, data, mountFsType string) errors.E {
	slog.DebugContext(m.ctx, "Attempting to mount volume",
		"device", md.DeviceId, "path", md.Path,
		"fstype", md.FSType, "mount_fstype", mountFsType,
		"flags", flags, "data", data)

	mountFunc := func() error { return os.MkdirAll(md.Path, 0o750) }

	mp, errMount := m.fsService.MountPartition(m.ctx, *md.Partition.DevicePath, md.Path, mountFsType, data, flags, mountFunc)
	if errMount != nil {
		fsTypeStr := "auto"
		if md.FSType != nil {
			fsTypeStr = *md.FSType
		}
		slog.ErrorContext(m.ctx, "Failed to mount volume",
			"device_id", md.DeviceId,
			"device_path", *md.Partition.DevicePath,
			"fstype", fsTypeStr,
			"mount_fstype", mountFsType,
			"mount_path", md.Path,
			"flags", flags,
			"data", data,
			"mount_error", errMount,
			"mountpoint_details", mp)

		// Attempt to clean up the directory we created on failure.
		if _, statErr := os.Stat(md.Path); statErr == nil {
			if removeErr := os.Remove(md.Path); removeErr != nil {
				slog.WarnContext(m.ctx, "Failed to cleanup mount directory after mount failure",
					"path", md.Path, "cleanup_error", removeErr)
			}
		}

		return errors.WithDetails(dto.ErrorMountFail,
			"Detail", fmt.Sprintf("Mount command failed: %v", errMount),
			"DeviceId", md.DeviceId,
			"DevicePath", *md.Partition.DevicePath,
			"MountPath", md.Path,
			"FSType", fsTypeStr,
			"MountFSType", mountFsType,
			"Flags", flags,
			"Data", data,
			"Error", errMount.Error(),
		)
	}

	slog.InfoContext(m.ctx, "Successfully mounted volume",
		"device", md.DeviceId, "path", md.Path,
		"fstype", md.FSType, "mount_fstype", mountFsType,
		"flags", mp.Flags, "data", mp.Data)

	if errConv := m.convMDto.MountToMountPointData(mp, md, m.disks); errConv != nil {
		return errors.WithDetails(dto.ErrorMountFail,
			"Detail", "Failed to convert mount details back to DTO after successful mount",
			"DeviceId", md.DeviceId,
			"MountPath", md.Path,
			"Error", errConv.Error(),
		)
	}

	if errCache := m.disks.AddOrUpdateMountPoint(*md.Partition.DiskId, *md.Partition.Id, *md); errCache != nil {
		slog.ErrorContext(m.ctx, "Failed to add mount point to in-memory cache after successful mount",
			"device", md.DeviceId, "path", md.Path, "err", errCache)
		return errors.WithDetails(dto.ErrorMountFail,
			"Detail", "Failed to add mount point to in-memory cache after successful mount",
			"DeviceId", md.DeviceId,
			"MountPath", md.Path,
			"Error", errCache.Error(),
		)
	}

	m.eventBus.EmitMountPoint(events.MountPointEvent{
		Event:      events.Event{Type: events.EventTypes.UPDATE},
		MountPoint: md,
	})

	return nil
}

// Unmount performs the actual OS-level unmount operation: calls the filesystem
// adapter, removes the mount directory, updates the in-memory cache, and
// emits a MountPointEvent.
// Validation and cache lookups are performed by the caller (VolumeService).
func (m *volumeMountManager) Unmount(md *dto.MountPointData, force bool) errors.E {
	slog.DebugContext(m.ctx, "Attempting to unmount volume", "path", md.Path, "force", force)
	fsType := ""
	if md != nil && md.FSType != nil {
		fsType = *md.FSType
	}

	if unmountErr := m.fsService.UnmountPartition(m.ctx, md.Path, fsType, force, !force); unmountErr != nil {
		slog.ErrorContext(m.ctx, "Failed to unmount volume", "path", md.Path, "err", unmountErr)
		return errors.WithDetails(dto.ErrorUnmountFail,
			"Detail", unmountErr.Error(), "Path", md.Path, "Error", unmountErr)
	}

	slog.InfoContext(m.ctx, "Successfully unmounted volume", "path", md.Path)

	if err := os.Remove(md.Path); err != nil {
		slog.WarnContext(m.ctx, "Failed to remove mount point directory", "path", md.Path, "err", err)
	} else {
		slog.DebugContext(m.ctx, "Removed mount point directory", "path", md.Path)
	}

	if md != nil && md.Partition != nil && md.Partition.DiskId != nil && md.Partition.Id != nil {
		md.IsMounted = false
		m.eventBus.EmitMountPoint(events.MountPointEvent{
			Event:      events.Event{Type: events.EventTypes.UPDATE},
			MountPoint: md,
		})
		m.disks.AddOrUpdateMountPoint(*md.Partition.DiskId, *md.Partition.Id, *md)
	}

	return nil
}
