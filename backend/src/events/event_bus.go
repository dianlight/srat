package events

import (
	"context"
	"fmt"
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
	EmitDiskAndPartition(event DiskEvent)
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

	// Volume events
	EmitVolume(event VolumeEvent)
	OnVolume(handler func(VolumeEvent)) func()

	// Dirty data events
	EmitDirtyData(event DirtyDataEvent)
	OnDirtyData(handler func(DirtyDataEvent)) func()

	// Home Assistant events
	EmitHomeAssistant(event HomeAssistantEvent)
	OnHomeAssistant(handler func(HomeAssistantEvent)) func()
}

// EventBus implements EventBusInterface using maniartech/signals SyncSignal
type EventBus struct {
	ctx context.Context

	// Synchronous signals (no goroutine dispatch) for deterministic ordering & error management
	disk          signals.SyncSignal[DiskEvent]
	partition     signals.SyncSignal[PartitionEvent]
	share         signals.SyncSignal[ShareEvent]
	mountPoint    signals.SyncSignal[MountPointEvent]
	user          signals.SyncSignal[UserEvent]
	setting       signals.SyncSignal[SettingEvent]
	samba         signals.SyncSignal[SambaEvent]
	volume        signals.SyncSignal[VolumeEvent]
	dirtyData     signals.SyncSignal[DirtyDataEvent]
	homeAssistant signals.SyncSignal[HomeAssistantEvent]
}

// NewEventBus creates a new EventBus instance
func NewEventBus(ctx context.Context) EventBusInterface {
	return &EventBus{
		ctx:           ctx,
		disk:          *signals.NewSync[DiskEvent](),
		partition:     *signals.NewSync[PartitionEvent](),
		share:         *signals.NewSync[ShareEvent](),
		mountPoint:    *signals.NewSync[MountPointEvent](),
		user:          *signals.NewSync[UserEvent](),
		setting:       *signals.NewSync[SettingEvent](),
		samba:         *signals.NewSync[SambaEvent](),
		volume:        *signals.NewSync[VolumeEvent](),
		dirtyData:     *signals.NewSync[DirtyDataEvent](),
		homeAssistant: *signals.NewSync[HomeAssistantEvent](),
	}
}

// Generic internal methods for event handling
func onEvent[T any](signal signals.SyncSignal[T], eventName string, handler func(T)) func() {
	tlog.Debug("Registering event handler", "event", eventName)
	key := generateKey()
	count := signal.AddListener(func(ctx context.Context, event T) {
		// Panic/exception safety
		defer func() {
			if r := recover(); r != nil {
				tlog.Error("Event handler panic", "event", eventName, "panic", r)
			}
		}()
		handler(event)
	}, key)
	tlog.Debug("Event handler registered", "event", eventName, "listener_count", count)
	return func() {
		signal.RemoveListener(key)
		tlog.Debug("Event handler unregistered", "event", eventName, "key", key)
	}
}

func emitEvent[T any](signal signals.SyncSignal[T], ctx context.Context, eventName string, event T, logFields ...any) {
	tlog.Debug("Emitting event", append([]any{"event", eventName}, logFields...)...)
	// Emit synchronously; recover panic inside signal dispatch and log emission errors
	defer func() {
		if r := recover(); r != nil {
			tlog.Error("Panic emitting event", "event", eventName, "panic", r)
		}
	}()
	if err := signal.TryEmit(ctx, event); err != nil {
		// We log at warn level to avoid noisy error logs for expected cancellations
		tlog.Warn("Event emission error", "event", eventName, "error", err)
	}
}

// Disk event methods
func (eb *EventBus) EmitDiskAndPartition(event DiskEvent) {
	diskID := "unknown"
	if event.Disk.Id != nil {
		diskID = *event.Disk.Id
	}
	emitEvent(eb.disk, eb.ctx, "Disk", event, "disk", diskID)
	if event.Disk.Partitions != nil {
		for _, partition := range *event.Disk.Partitions {
			tlog.Trace("Emitting Partition event for disk", "partition", partition, "disk", diskID)
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
	return onEvent(eb.disk, "Disk", handler)
}

// Partition event methods
func (eb *EventBus) EmitPartition(event PartitionEvent) {
	partName := "unknown"
	if event.Partition != nil && event.Partition.Name != nil && *event.Partition.Name != "" {
		partName = *event.Partition.Name
	}
	diskID := "unknown"
	if event.Disk != nil && event.Disk.Id != nil {
		diskID = *event.Disk.Id
	}
	emitEvent(eb.partition, eb.ctx, "Partition", event, "partition", partName, "disk", diskID)
}

func (eb *EventBus) OnPartition(handler func(PartitionEvent)) func() {
	return onEvent(eb.partition, "Partition", handler)
}

// Share event methods
func (eb *EventBus) EmitShare(event ShareEvent) {
	emitEvent(eb.share, eb.ctx, "Share", event, "share", event.Share.Name)
}

func (eb *EventBus) OnShare(handler func(ShareEvent)) func() {
	return onEvent(eb.share, "Share", handler)
}

// Mount point event methods
func (eb *EventBus) EmitMountPoint(event MountPointEvent) {
	emitEvent(eb.mountPoint, eb.ctx, "MountPoint", event, "mount_point", event.MountPoint.Path)
}

func (eb *EventBus) OnMountPoint(handler func(MountPointEvent)) func() {
	return onEvent(eb.mountPoint, "MountPoint", handler)
}

func (eb *EventBus) OnMountPointUnmounted(handler func(MountPointEvent)) func() {
	return onEvent(eb.mountPoint, "MountPointUnmounted", handler)
}

// User event methods
func (eb *EventBus) EmitUser(event UserEvent) {
	emitEvent(eb.user, eb.ctx, "User", event, "user", event.User.Username)
}

func (eb *EventBus) OnUser(handler func(UserEvent)) func() {
	return onEvent(eb.user, "User", handler)
}

// Setting event methods
func (eb *EventBus) EmitSetting(event SettingEvent) {
	emitEvent(eb.setting, eb.ctx, "Setting", event, "setting", event.Setting)
}

func (eb *EventBus) OnSetting(handler func(SettingEvent)) func() {
	return onEvent(eb.setting, "Setting", handler)
}

// Samba event methods
func (eb *EventBus) EmitSamba(event SambaEvent) {
	emitEvent(eb.samba, eb.ctx, "Samba", event)
}

func (eb *EventBus) OnSamba(handler func(SambaEvent)) func() {
	return onEvent(eb.samba, "Samba", handler)
}

// Volume event methods
func (eb *EventBus) EmitVolume(event VolumeEvent) {
	mp := ""
	if event.MountPoint.Path != "" {
		mp = event.MountPoint.Path
	}
	emitEvent(eb.volume, eb.ctx, "Volume", event, "operation", event.Operation, "mount_point", mp)
}

func (eb *EventBus) OnVolume(handler func(VolumeEvent)) func() {
	return onEvent(eb.volume, "Volume", handler)
}

// Dirty data event methods
func (eb *EventBus) EmitDirtyData(event DirtyDataEvent) {
	emitEvent(eb.dirtyData, eb.ctx, "DirtyData", event, "tracker", event.DataDirtyTracker)
}

func (eb *EventBus) OnDirtyData(handler func(DirtyDataEvent)) func() {
	return onEvent(eb.dirtyData, "DirtyData", handler)
}

// Home Assistant event methods
func (eb *EventBus) EmitHomeAssistant(event HomeAssistantEvent) {
	emitEvent(eb.homeAssistant, eb.ctx, "HomeAssistant", event)
}

func (eb *EventBus) OnHomeAssistant(handler func(HomeAssistantEvent)) func() {
	return onEvent(eb.homeAssistant, "HomeAssistant", handler)
}
