package events

import (
	"context"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/google/uuid"
	"github.com/prometheus/procfs"
)

// Event is the base event struct without UUID (UUID is now stored in context)
type Event struct {
	Type EventType
}

// GetEventUUID retrieves the UUID from context
func GetEventUUID(ctx context.Context) string {
	if val := ctx.Value(ctxkeys.EventUUID); val != nil {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// ContextWithEventUUID returns a new context with the event UUID set
func ContextWithEventUUID(ctx context.Context) context.Context {
	if GetEventUUID(ctx) == "" {
		return context.WithValue(ctx, ctxkeys.EventUUID, uuid.New().String())
	}
	return ctx
}

// DiskEvent represents a disk-related event
type DiskEvent struct {
	Event
	Disk *dto.Disk
}

// PartitionEvent represents a partition-related event.
//
// Two payload shapes are supported:
//   - Single mode (Partition set): one partition to sync, emitted by legacy
//     emitters (udev events, tests, external callers).
//   - Batch mode (Partitions set): all changed partitions of one disk,
//     emitted by VolumeService.getVolumesData. MountInfos carries a
//     pre-parsed procfs snapshot shared by the whole batch so the handler
//     parses procfs once per refresh cycle instead of once per partition.
//     When MountInfos is nil the handler falls back to parsing procfs
//     itself, keeping the single-partition path and external emitters
//     working unchanged.
type PartitionEvent struct {
	Event
	Partition  *dto.Partition
	Disk       *dto.Disk
	Partitions []*dto.Partition
	MountInfos []*procfs.MountInfo
}

// ShareEvent represents a share-related event
type ShareEvent struct {
	Event
	Share *dto.SharedResource
}

// MountPointEvent represents a mount point event
type MountPointEvent struct {
	Event
	MountPoint *dto.MountPointData
}

// UserEvent represents a user-related event
type UserEvent struct {
	Event
	User *dto.User
}

// SettingEvent represents a setting-related event
type SettingEvent struct {
	Event
	Setting *dto.Settings
}

// AppConfigEvent represents an app configuration-related event.
type AppConfigEvent struct {
	Event
	Config *dto.AppConfigUpdateRequest
	Path   string
	Hash   string
}

// ServerProcessEvent represents a Samba-related event
type ServerProcessEvent struct {
	Event
	DataDirtyTracker dto.DataDirtyTracker
}

// VolumeEvent represents a volume operation event (mount/unmount)
type VolumeEvent struct {
	Event
	MountPoint *dto.MountPointData
	Operation  string // "mount" or "unmount"
}

// DirtyDataEvent represents a dirty data event
type DirtyDataEvent struct {
	Event
	DataDirtyTracker dto.DataDirtyTracker
}

// HomeAssistantEvent represents a Home Assistant-related event
type HomeAssistantEvent struct {
	Event
	Error *dto.ErrorDataModel
}

type SmartEvent struct {
	Event
	SmartInfo       dto.SmartInfo
	SmartTestStatus dto.SmartTestStatus
}

// PowerEventKind discriminates the two payload variants of PowerEvent.
//
// Subscribers should branch on Kind rather than comparing zero-values of
// PowerInfo/PowerStatus, which used to be ambiguous because both fields
// were always present (each defaulted to its struct zero value).
type PowerEventKind string

const (
	// PowerEventKindConfig signals that PowerInfo carries a per-disk
	// configuration update (Save/PUT); PowerStatus is unset.
	PowerEventKindConfig PowerEventKind = "config"
	// PowerEventKindStatus signals that PowerStatus carries a spin-state
	// transition (spun_up / spun_down); PowerInfo is unset.
	PowerEventKindStatus PowerEventKind = "status"
)

type PowerEvent struct {
	Event
	Kind        PowerEventKind
	PowerInfo   dto.HDIdleDevice
	PowerStatus dto.HDIdleDeviceStatus
}

// FilesystemTaskEvent represents a filesystem operation event (format, check)
type FilesystemTaskEvent struct {
	Event
	Task *dto.FilesystemTask
}

// CommandExecutionEvent represents a command lifecycle notification emitted by
// the shared command executor. Message is always one of the typed command
// notification DTOs.
type CommandExecutionEvent struct {
	Event
	Message dto.CommandExecutionNotification
}

// ProblemEvent represents a unified problem lifecycle event.
type ProblemEvent struct {
	Event
	Problem *dto.Problem
}
