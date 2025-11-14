package events

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/dianlight/srat/tlog"
	"github.com/maniartech/signals"
)

var keyCounter uint64

// generateKey generates a unique key for event listeners
func generateKey() string {
	return fmt.Sprintf("listener_%d", atomic.AddUint64(&keyCounter, 1))
}

// EventBusInterface defines the interface for the event bus
type EventBusInterface interface {

	// Disk events
	EmitDisk(event DiskEvent)
	OnDisk(handler func(DiskEvent)) func()

	// Partition events
	EmitPartition(event PartitionEvent)
	OnPartition(handler func(PartitionEvent)) func()

	// Share events
	EmitShare(event ShareEvent)
	OnShare(handler func(ShareEvent)) func()

	// Mount point events
	EmitMountPoint(event MountPointEvent)
	OnMountPoint(handler func(MountPointEvent)) func()

	// User events
	EmitUser(event UserEvent)
	OnUser(handler func(UserEvent)) func()

	// Setting events
	EmitSetting(event SettingEvent)
	OnSetting(handler func(SettingEvent)) func()

	// Samba events
	EmitSamba(event SambaEvent)
	OnSamba(handler func(SambaEvent)) func()

	// Dirty data events
	EmitDirtyData(event DirtyDataEvent)
	OnDirtyData(handler func(DirtyDataEvent)) func()
}

// EventBus implements EventBusInterface using maniartech/signals
type EventBus struct {
	ctx context.Context

	// Disk event signals
	disk signals.Signal[DiskEvent]

	// Partition event signals
	partition signals.Signal[PartitionEvent]

	// Share event signals
	share signals.Signal[ShareEvent]

	// Mount point event signals
	mountPoint signals.Signal[MountPointEvent]

	// User event signals
	user signals.Signal[UserEvent]

	// Setting event signals
	setting signals.Signal[SettingEvent]

	// Samba event signals
	samba signals.Signal[SambaEvent]

	// Dirty data event signals
	dirtyData signals.Signal[DirtyDataEvent]
}

// NewEventBus creates a new EventBus instance
func NewEventBus(ctx context.Context) EventBusInterface {
	return &EventBus{
		ctx:        ctx,
		disk:       signals.New[DiskEvent](),
		partition:  signals.New[PartitionEvent](),
		share:      signals.New[ShareEvent](),
		mountPoint: signals.New[MountPointEvent](),
		user:       signals.New[UserEvent](),
		setting:    signals.New[SettingEvent](),
		samba:      signals.New[SambaEvent](),
		dirtyData:  signals.New[DirtyDataEvent](),
	}
}

// Disk event methods
func (eb *EventBus) EmitDisk(event DiskEvent) {
	diskID := "unknown"
	if event.Disk.Id != nil {
		diskID = *event.Disk.Id
	}
	tlog.Trace("Emitting Disk event", "disk", diskID)
	eb.disk.Emit(eb.ctx, event)
	if event.Disk.Partitions != nil {
		for _, partition := range *event.Disk.Partitions {
			tlog.Trace("Emitting Partition event for  disk", "partition", partition, "disk", diskID)
			eb.EmitPartition(PartitionEvent{
				Event: Event{
					Type: event.Type,
				},
				Partition: &partition,
				Disk:      event.Disk,
			})
		}
	}

}

func (eb *EventBus) OnDisk(handler func(DiskEvent)) func() {
	tlog.Debug("Registering DiskAdded event handler")
	key := generateKey()
	eb.disk.AddListener(func(ctx context.Context, event DiskEvent) {
		handler(event)
	}, key)
	return func() {
		eb.disk.RemoveListener(key)
	}
}

// Partition event methods
func (eb *EventBus) EmitPartition(event PartitionEvent) {
	partName := "unknown"
	if event.Partition.Name != nil && *event.Partition.Name != "" {
		partName = *event.Partition.Name
	}
	diskID := "unknown"
	if event.Disk.Id != nil {
		diskID = *event.Disk.Id
	}
	tlog.Debug("Emitting PartitionAdded event", "partition", partName, "disk", diskID)
	eb.partition.Emit(eb.ctx, event)

}

func (eb *EventBus) OnPartition(handler func(PartitionEvent)) func() {
	tlog.Debug("Registering Partition event handler")
	key := generateKey()
	eb.partition.AddListener(func(ctx context.Context, event PartitionEvent) {
		handler(event)
	}, key)
	return func() {
		eb.partition.RemoveListener(key)
	}
}

// Share event methods
func (eb *EventBus) EmitShare(event ShareEvent) {
	slog.Debug("Emitting Share event", "share", event.Share.Name)
	eb.share.Emit(eb.ctx, event)
}

func (eb *EventBus) OnShare(handler func(ShareEvent)) func() {
	slog.Debug("Registering Share event handler")
	key := generateKey()
	eb.share.AddListener(func(ctx context.Context, event ShareEvent) {
		handler(event)
	}, key)
	return func() {
		eb.share.RemoveListener(key)
	}
}

// Mount point event methods
func (eb *EventBus) EmitMountPoint(event MountPointEvent) {
	slog.Debug("Emitting MountPoint event", "mount_point", event.MountPoint.Path)
	eb.mountPoint.Emit(eb.ctx, event)
}

func (eb *EventBus) OnMountPoint(handler func(MountPointEvent)) func() {
	slog.Debug("Registering MountPoint event handler")
	key := generateKey()
	eb.mountPoint.AddListener(func(ctx context.Context, event MountPointEvent) {
		handler(event)
	}, key)
	return func() {
		eb.mountPoint.RemoveListener(key)
	}
}

func (eb *EventBus) OnMountPointUnmounted(handler func(MountPointEvent)) func() {
	slog.Debug("Registering MountPointUnmounted event handler")
	key := generateKey()
	eb.mountPoint.AddListener(func(ctx context.Context, event MountPointEvent) {
		handler(event)
	}, key)
	return func() {
		eb.mountPoint.RemoveListener(key)
	}
}

// User event methods
func (eb *EventBus) EmitUser(event UserEvent) {
	slog.Debug("Emitting User event", "user", event.User.Username)
	eb.user.Emit(eb.ctx, event)
}

func (eb *EventBus) OnUser(handler func(UserEvent)) func() {
	slog.Debug("Registering User event handler")
	key := generateKey()
	eb.user.AddListener(func(ctx context.Context, event UserEvent) {
		handler(event)
	}, key)
	return func() {
		eb.user.RemoveListener(key)
	}
}

// Setting event methods
func (eb *EventBus) EmitSetting(event SettingEvent) {
	slog.Debug("Emitting Setting event", "setting", event.Setting)
	eb.setting.Emit(eb.ctx, event)
}

func (eb *EventBus) OnSetting(handler func(SettingEvent)) func() {
	slog.Debug("Registering Setting event handler")
	key := generateKey()
	eb.setting.AddListener(func(ctx context.Context, event SettingEvent) {
		handler(event)
	}, key)
	return func() {
		eb.setting.RemoveListener(key)
	}
}

// Samba event methods
func (eb *EventBus) EmitSamba(event SambaEvent) {
	slog.Debug("Emitting Samba event")
	eb.samba.Emit(eb.ctx, event)
}

func (eb *EventBus) OnSamba(handler func(SambaEvent)) func() {
	slog.Debug("Registering Samba event handler")
	key := generateKey()
	eb.samba.AddListener(func(ctx context.Context, event SambaEvent) {
		handler(event)
	}, key)
	return func() {
		eb.samba.RemoveListener(key)
	}
}

// Dirty data event methods
func (eb *EventBus) EmitDirtyData(event DirtyDataEvent) {
	slog.Debug("Emitting DirtyData event", "tracker", event.DataDirtyTracker)
	eb.dirtyData.Emit(eb.ctx, event)
}

func (eb *EventBus) OnDirtyData(handler func(DirtyDataEvent)) func() {
	slog.Debug("Registering DirtyData event handler")
	key := generateKey()
	eb.dirtyData.AddListener(func(ctx context.Context, event DirtyDataEvent) {
		handler(event)
	}, key)
	return func() {
		eb.dirtyData.RemoveListener(key)
	}
}
