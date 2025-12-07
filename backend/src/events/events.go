package events

import (
	"context"

	"github.com/dianlight/srat/dto"
	"github.com/google/uuid"
)

// Event is the base event struct without UUID (UUID is now stored in context)
type Event struct {
	Type EventType
}

// GetEventUUID retrieves the UUID from context
func GetEventUUID(ctx context.Context) string {
	if val := ctx.Value("event_uuid"); val != nil {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// ContextWithEventUUID returns a new context with the event UUID set
func ContextWithEventUUID(ctx context.Context) context.Context {
	if GetEventUUID(ctx) == "" {
		return context.WithValue(ctx, "event_uuid", uuid.New().String())
	}
	return ctx
}

// DiskEvent represents a disk-related event
type DiskEvent struct {
	Event
	Disk *dto.Disk
}

// PartitionEvent represents a partition-related event
type PartitionEvent struct {
	Event
	Partition *dto.Partition
	Disk      *dto.Disk
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

// SambaEvent represents a Samba-related event
type SambaEvent struct {
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
}
