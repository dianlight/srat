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
	OnDisk(handler func(context.Context, DiskEvent)) func()

	// Partition events
	EmitPartition(event PartitionEvent)
	OnPartition(handler func(context.Context, PartitionEvent)) func()

	// Share events
	EmitShare(event ShareEvent)
	OnShare(handler func(context.Context, ShareEvent)) func()

	// Mount point events
	EmitMountPoint(event MountPointEvent)
	OnMountPoint(handler func(context.Context, MountPointEvent)) func()

	// User events
	EmitUser(event UserEvent)
	OnUser(handler func(context.Context, UserEvent)) func()

	// Setting events
	EmitSetting(event SettingEvent)
	OnSetting(handler func(context.Context, SettingEvent)) func()

	// Samba events
	EmitSamba(event SambaEvent)
	OnSamba(handler func(context.Context, SambaEvent)) func()

	// Volume events
	EmitVolume(event VolumeEvent)
	OnVolume(handler func(context.Context, VolumeEvent)) func()

	// Dirty data events
	EmitDirtyData(event DirtyDataEvent)
	OnDirtyData(handler func(context.Context, DirtyDataEvent)) func()

	// Home Assistant events
	EmitHomeAssistant(event HomeAssistantEvent)
	OnHomeAssistant(handler func(context.Context, HomeAssistantEvent)) func()
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

// EventConstraint ensures that T embeds Event
type EventConstraint interface {
	GetEvent() *Event
}

// Generic internal methods for event handling
func onEvent[T any](signal signals.SyncSignal[T], eventName string, handler func(context.Context, T)) func() {
	tlog.Trace("Registering event handler", append([]any{"event", eventName}, tlog.WithCaller()...)...)
	key := generateKey()
	count := signal.AddListener(func(ctx context.Context, event T) {
		// Panic/exception safety
		defer func() {
			if r := recover(); r != nil {
				tlog.Error("Event handler panic", append([]any{"event", eventName, "panic", r}, tlog.WithCaller()...)...)
			}
		}()
		tlog.Debug("<-- Receiving events ", append([]any{"event", event}, tlog.WithCaller()...)...)
		handler(ctx, event)
	}, key)
	tlog.Debug("Event handler registered", append([]any{"event", eventName, "listener_count", count}, tlog.WithCaller()...)...)
	return func() {
		signal.RemoveListener(key)
		tlog.Trace("Event handler unregistered", append([]any{"event", eventName, "key", key}, tlog.WithCaller()...)...)
	}
}

func emitEvent[T any](signal signals.SyncSignal[T], ctx context.Context, event T) {
	// Add UUID to context if not already present
	ctx = ContextWithEventUUID(ctx)

	tlog.Debug("--> Emitting event", append([]any{"event", event}, tlog.WithCaller()...)...)
	// Emit synchronously; recover panic inside signal dispatch and log emission errors
	defer func() {
		if r := recover(); r != nil {
			tlog.Error("Panic emitting event", append([]any{"event", event, "panic", r}, tlog.WithCaller()...)...)
		}
	}()
	if err := signal.TryEmit(ctx, event); err != nil {
		// We log at warn level to avoid noisy error logs for expected cancellations
		tlog.Warn("Event emission error", append([]any{"event", event, "error", err}, tlog.WithCaller()...)...)
	}
}

// Disk event methods
func (eb *EventBus) EmitDiskAndPartition(event DiskEvent) {
	diskID := "unknown"
	if event.Disk.Id != nil {
		diskID = *event.Disk.Id
	}
	emitEvent(eb.disk, eb.ctx, event)
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

func (eb *EventBus) OnDisk(handler func(context.Context, DiskEvent)) func() {
	return onEvent(eb.disk, "Disk", handler)
}

// Partition event methods
func (eb *EventBus) EmitPartition(event PartitionEvent) {
	emitEvent(eb.partition, eb.ctx, event)
}

func (eb *EventBus) OnPartition(handler func(context.Context, PartitionEvent)) func() {
	return onEvent(eb.partition, "Partition", handler)
}

// Share event methods
func (eb *EventBus) EmitShare(event ShareEvent) {
	emitEvent(eb.share, eb.ctx, event)
}

func (eb *EventBus) OnShare(handler func(context.Context, ShareEvent)) func() {
	return onEvent(eb.share, "Share", handler)
}

// Mount point event methods
func (eb *EventBus) EmitMountPoint(event MountPointEvent) {
	emitEvent(eb.mountPoint, eb.ctx, event)
}

func (eb *EventBus) OnMountPoint(handler func(context.Context, MountPointEvent)) func() {
	return onEvent(eb.mountPoint, "MountPoint", handler)
}

func (eb *EventBus) OnMountPointUnmounted(handler func(context.Context, MountPointEvent)) func() {
	return onEvent(eb.mountPoint, "MountPointUnmounted", handler)
}

// User event methods
func (eb *EventBus) EmitUser(event UserEvent) {
	emitEvent(eb.user, eb.ctx, event)
}

func (eb *EventBus) OnUser(handler func(context.Context, UserEvent)) func() {
	return onEvent(eb.user, "User", handler)
}

// Setting event methods
func (eb *EventBus) EmitSetting(event SettingEvent) {
	emitEvent(eb.setting, eb.ctx, event)
}

func (eb *EventBus) OnSetting(handler func(context.Context, SettingEvent)) func() {
	return onEvent(eb.setting, "Setting", handler)
}

// Samba event methods
func (eb *EventBus) EmitSamba(event SambaEvent) {
	emitEvent(eb.samba, eb.ctx, event)
}

func (eb *EventBus) OnSamba(handler func(context.Context, SambaEvent)) func() {
	return onEvent(eb.samba, "Samba", handler)
}

// Volume event methods
func (eb *EventBus) EmitVolume(event VolumeEvent) {
	emitEvent(eb.volume, eb.ctx, event)
}

func (eb *EventBus) OnVolume(handler func(context.Context, VolumeEvent)) func() {
	return onEvent(eb.volume, "Volume", handler)
}

// Dirty data event methods
func (eb *EventBus) EmitDirtyData(event DirtyDataEvent) {
	emitEvent(eb.dirtyData, eb.ctx, event)
}

func (eb *EventBus) OnDirtyData(handler func(context.Context, DirtyDataEvent)) func() {
	return onEvent(eb.dirtyData, "DirtyData", handler)
}

// Home Assistant event methods
func (eb *EventBus) EmitHomeAssistant(event HomeAssistantEvent) {
	emitEvent(eb.homeAssistant, eb.ctx, event)
}

func (eb *EventBus) OnHomeAssistant(handler func(context.Context, HomeAssistantEvent)) func() {
	return onEvent(eb.homeAssistant, "HomeAssistant", handler)
}
