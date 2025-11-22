package events

import (
	"context"

	"fmt"
	"sync/atomic"

	"github.com/dianlight/tlog"
	"github.com/maniartech/signals"
	"gitlab.com/tozd/go/errors"
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
	OnDisk(handler func(context.Context, DiskEvent) errors.E) func()

	// Partition events
	EmitPartition(event PartitionEvent)
	OnPartition(handler func(context.Context, PartitionEvent) errors.E) func()

	// Share events
	EmitShare(event ShareEvent)
	OnShare(handler func(context.Context, ShareEvent) errors.E) func()
	// Mount point events
	EmitMountPoint(event MountPointEvent)
	OnMountPoint(handler func(context.Context, MountPointEvent) errors.E) func()

	// User events
	EmitUser(event UserEvent)
	OnUser(handler func(context.Context, UserEvent) errors.E) func()

	// Setting events
	EmitSetting(event SettingEvent)
	OnSetting(handler func(context.Context, SettingEvent) errors.E) func()
	// Samba events
	EmitSamba(event SambaEvent)
	OnSamba(handler func(context.Context, SambaEvent) errors.E) func()

	// Volume events
	EmitVolume(event VolumeEvent)
	OnVolume(handler func(context.Context, VolumeEvent) errors.E) func()
	// Dirty data events
	EmitDirtyData(event DirtyDataEvent)
	OnDirtyData(handler func(context.Context, DirtyDataEvent) errors.E) func()

	// Home Assistant events
	EmitHomeAssistant(event HomeAssistantEvent)
	OnHomeAssistant(handler func(context.Context, HomeAssistantEvent) errors.E) func()
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
func onEvent[T any](signal signals.SyncSignal[T], eventName string, handler func(context.Context, T) errors.E) func() {
	tlog.Trace("Registering event handler", append([]any{"event", eventName}, tlog.WithCaller(0)...)...)
	key := generateKey()
	caller := tlog.WithCaller(1)
	count := signal.AddListenerWithErr(func(ctx context.Context, event T) error {
		// Panic/exception safety
		defer func() {
			if r := recover(); r != nil {
				tlog.ErrorContext(ctx, "Event handler panic", append([]any{"event", eventName, "panic", r}, caller...)...)
			}
		}()
		tlog.DebugContext(ctx, "<-- Receiving events ", append([]any{"type", fmt.Sprintf("%T", event), "event", fmt.Sprintf("%#v", event)}, caller...)...)
		return handler(ctx, event)
	}, key)
	tlog.Debug("Event handler registered", append([]any{"event", eventName, "listener_count", count}, caller...)...)
	return func() {
		signal.RemoveListener(key)
		tlog.Trace("Event handler unregistered", append([]any{"event", eventName, "key", key}, caller...)...)
	}
}

func emitEvent[T any](signal signals.SyncSignal[T], ctx context.Context, event T) {
	// Add UUID to context if not already present
	ctx = ContextWithEventUUID(ctx)

	tlog.DebugContext(ctx, "--> Emitting event", append([]any{"type", fmt.Sprintf("%T", event), "event", fmt.Sprintf("%#v", event)}, tlog.WithCaller(1)...)...)
	// Emit synchronously; recover panic inside signal dispatch and log emission errors
	defer func() {
		if r := recover(); r != nil {
			tlog.ErrorContext(ctx, "Panic emitting event", append([]any{"event", fmt.Sprintf("%#v", event), "panic", r}, tlog.WithCaller(1)...)...)
		}
	}()
	if err := signal.TryEmit(ctx, event); err != nil {
		// We log at warn level to avoid noisy error logs for expected cancellations
		tlog.WarnContext(ctx, "Event emission error", append([]any{"event", fmt.Sprintf("%#v", event), "error", err}, tlog.WithCaller(1)...)...)
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

func (eb *EventBus) OnDisk(handler func(context.Context, DiskEvent) errors.E) func() {
	return onEvent(eb.disk, "Disk", handler)
}

// Partition event methods
func (eb *EventBus) EmitPartition(event PartitionEvent) {
	emitEvent(eb.partition, eb.ctx, event)
}

func (eb *EventBus) OnPartition(handler func(context.Context, PartitionEvent) errors.E) func() {
	return onEvent(eb.partition, "Partition", handler)
}

// Share event methods
func (eb *EventBus) EmitShare(event ShareEvent) {
	emitEvent(eb.share, eb.ctx, event)
}

func (eb *EventBus) OnShare(handler func(context.Context, ShareEvent) errors.E) func() {
	return onEvent(eb.share, "Share", handler)
}

// Mount point event methods
func (eb *EventBus) EmitMountPoint(event MountPointEvent) {
	emitEvent(eb.mountPoint, eb.ctx, event)
}

func (eb *EventBus) OnMountPoint(handler func(context.Context, MountPointEvent) errors.E) func() {
	return onEvent(eb.mountPoint, "MountPoint", handler)
}

func (eb *EventBus) OnMountPointUnmounted(handler func(context.Context, MountPointEvent) errors.E) func() {
	return onEvent(eb.mountPoint, "MountPointUnmounted", handler)
}

// User event methods
func (eb *EventBus) EmitUser(event UserEvent) {
	emitEvent(eb.user, eb.ctx, event)
}

func (eb *EventBus) OnUser(handler func(context.Context, UserEvent) errors.E) func() {
	return onEvent(eb.user, "User", handler)
}

// Setting event methods
func (eb *EventBus) EmitSetting(event SettingEvent) {
	emitEvent(eb.setting, eb.ctx, event)
}

func (eb *EventBus) OnSetting(handler func(context.Context, SettingEvent) errors.E) func() {
	return onEvent(eb.setting, "Setting", handler)
}

// Samba event methods
func (eb *EventBus) EmitSamba(event SambaEvent) {
	emitEvent(eb.samba, eb.ctx, event)
}

func (eb *EventBus) OnSamba(handler func(context.Context, SambaEvent) errors.E) func() {
	return onEvent(eb.samba, "Samba", handler)
}

// Volume event methods
func (eb *EventBus) EmitVolume(event VolumeEvent) {
	emitEvent(eb.volume, eb.ctx, event)
}

func (eb *EventBus) OnVolume(handler func(context.Context, VolumeEvent) errors.E) func() {
	return onEvent(eb.volume, "Volume", handler)
}

// Dirty data event methods
func (eb *EventBus) EmitDirtyData(event DirtyDataEvent) {
	emitEvent(eb.dirtyData, eb.ctx, event)
}

func (eb *EventBus) OnDirtyData(handler func(context.Context, DirtyDataEvent) errors.E) func() {
	return onEvent(eb.dirtyData, "DirtyData", handler)
}

// Home Assistant event methods
func (eb *EventBus) EmitHomeAssistant(event HomeAssistantEvent) {
	emitEvent(eb.homeAssistant, eb.ctx, event)
}

func (eb *EventBus) OnHomeAssistant(handler func(context.Context, HomeAssistantEvent) errors.E) func() {
	return onEvent(eb.homeAssistant, "HomeAssistant", handler)
}
