package events

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"

	"github.com/dianlight/srat/tlog"
	"github.com/google/uuid"
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

// EventConstraint ensures that T embeds Event
type EventConstraint interface {
	GetEvent() *Event
}

// Helper function to set UUID on Event if it's embedded in T
// Note: This modifies the event in place through reflection
func setEventUUID[T any](event *T) {
	if event == nil {
		return
	}
	val := reflect.ValueOf(event).Elem()
	eventField := val.FieldByName("Event")
	if eventField.IsValid() && eventField.Kind() == reflect.Struct {
		uuidField := eventField.FieldByName("UUID")
		if uuidField.IsValid() && uuidField.CanSet() && uuidField.Kind() == reflect.String {
			if uuidField.String() == "" {
				uuidField.SetString(uuid.New().String())
			}
		}
	}
}

// Helper function to get Event from any type that embeds it
func getEvent[T any](event T) *Event {
	// Use reflection to get the Event field
	val := reflect.ValueOf(event)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	eventField := val.FieldByName("Event")
	if !eventField.IsValid() {
		return nil
	}
	if e, ok := eventField.Interface().(Event); ok {
		return &e
	}
	return nil
}

// Generic internal methods for event handling
func onEvent[T any](signal signals.SyncSignal[T], eventName string, handler func(T)) func() {
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
		handler(event)
	}, key)
	tlog.Debug("Event handler registered", append([]any{"event", eventName, "listener_count", count}, tlog.WithCaller()...)...)
	return func() {
		signal.RemoveListener(key)
		tlog.Trace("Event handler unregistered", append([]any{"event", eventName, "key", key}, tlog.WithCaller()...)...)
	}
}

func emitEvent[T any](signal signals.SyncSignal[T], ctx context.Context, event T) {
	// Generate UUID if Event is embedded in T
	if e := getEvent(event); e != nil && e.UUID == "" {
		setEventUUID(&event)
	}

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

func (eb *EventBus) OnDisk(handler func(DiskEvent)) func() {
	return onEvent(eb.disk, "Disk", handler)
}

// Partition event methods
func (eb *EventBus) EmitPartition(event PartitionEvent) {
	emitEvent(eb.partition, eb.ctx, event)
}

func (eb *EventBus) OnPartition(handler func(PartitionEvent)) func() {
	return onEvent(eb.partition, "Partition", handler)
}

// Share event methods
func (eb *EventBus) EmitShare(event ShareEvent) {
	emitEvent(eb.share, eb.ctx, event)
}

func (eb *EventBus) OnShare(handler func(ShareEvent)) func() {
	return onEvent(eb.share, "Share", handler)
}

// Mount point event methods
func (eb *EventBus) EmitMountPoint(event MountPointEvent) {
	emitEvent(eb.mountPoint, eb.ctx, event)
}

func (eb *EventBus) OnMountPoint(handler func(MountPointEvent)) func() {
	return onEvent(eb.mountPoint, "MountPoint", handler)
}

func (eb *EventBus) OnMountPointUnmounted(handler func(MountPointEvent)) func() {
	return onEvent(eb.mountPoint, "MountPointUnmounted", handler)
}

// User event methods
func (eb *EventBus) EmitUser(event UserEvent) {
	emitEvent(eb.user, eb.ctx, event)
}

func (eb *EventBus) OnUser(handler func(UserEvent)) func() {
	return onEvent(eb.user, "User", handler)
}

// Setting event methods
func (eb *EventBus) EmitSetting(event SettingEvent) {
	emitEvent(eb.setting, eb.ctx, event)
}

func (eb *EventBus) OnSetting(handler func(SettingEvent)) func() {
	return onEvent(eb.setting, "Setting", handler)
}

// Samba event methods
func (eb *EventBus) EmitSamba(event SambaEvent) {
	emitEvent(eb.samba, eb.ctx, event)
}

func (eb *EventBus) OnSamba(handler func(SambaEvent)) func() {
	return onEvent(eb.samba, "Samba", handler)
}

// Volume event methods
func (eb *EventBus) EmitVolume(event VolumeEvent) {
	emitEvent(eb.volume, eb.ctx, event)
}

func (eb *EventBus) OnVolume(handler func(VolumeEvent)) func() {
	return onEvent(eb.volume, "Volume", handler)
}

// Dirty data event methods
func (eb *EventBus) EmitDirtyData(event DirtyDataEvent) {
	emitEvent(eb.dirtyData, eb.ctx, event)
}

func (eb *EventBus) OnDirtyData(handler func(DirtyDataEvent)) func() {
	return onEvent(eb.dirtyData, "DirtyData", handler)
}

// Home Assistant event methods
func (eb *EventBus) EmitHomeAssistant(event HomeAssistantEvent) {
	emitEvent(eb.homeAssistant, eb.ctx, event)
}

func (eb *EventBus) OnHomeAssistant(handler func(HomeAssistantEvent)) func() {
	return onEvent(eb.homeAssistant, "HomeAssistant", handler)
}
