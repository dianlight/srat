package events

import (
	"context"
	"database/sql/driver"

	"fmt"
	"reflect"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
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

// formatWithoutPointerAddresses returns a human-readable string of v where:
// - Pointers are dereferenced to print their underlying values (no memory addresses)
// - Cycles are detected and annotated to avoid infinite recursion
// - Output is bounded by a reasonable max depth to prevent excessive logs
func formatWithoutPointerAddresses(v any) string {
	const maxDepth = 5
	var b strings.Builder
	seen := map[uintptr]bool{}
	writeValue(&b, reflect.ValueOf(v), seen, 0, maxDepth)
	return b.String()
}

func writeValue(b *strings.Builder, rv reflect.Value, seen map[uintptr]bool, depth, maxDepth int) {
	if !rv.IsValid() {
		b.WriteString("nil")
		return
	}
	if depth > maxDepth {
		b.WriteString("<max-depth>")
		return
	}

	// Check if rv implements database/sql/driver.Valuer interface before unwrapping
	// But skip nil pointers to avoid panics when calling Value() on nil receivers
	if rv.Kind() != reflect.Pointer || !rv.IsNil() {
		if rv.CanInterface() {
			if v, ok := rv.Interface().(driver.Valuer); ok {
				val, err := v.Value()
				if err == nil {
					writeValue(b, reflect.ValueOf(val), seen, depth+1, maxDepth)
					return
				}
			}
		}
	}

	// Unwrap interfaces
	if rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			b.WriteString("nil")
			return
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Pointer:
		if rv.IsNil() {
			b.WriteString("nil")
			return
		}
		// Cycle detection for pointers
		ptr := rv.Pointer()
		if ptr != 0 {
			if seen[ptr] {
				b.WriteString("<cycle>")
				return
			}
			seen[ptr] = true
			defer delete(seen, ptr)
		}
		writeValue(b, rv.Elem(), seen, depth+1, maxDepth)
		return

	case reflect.Struct:
		b.WriteString(rv.Type().String())
		b.WriteString("{")
		n := rv.NumField()
		first := true
		for i := 0; i < n; i++ {
			tf := rv.Type().Field(i)
			// Skip unexported fields we can't safely interface
			if tf.PkgPath != "" { // unexported
				continue
			}
			if !first {
				b.WriteString(", ")
			}
			first = false
			b.WriteString(tf.Name)
			b.WriteString(": ")
			writeValue(b, rv.Field(i), seen, depth+1, maxDepth)
		}
		b.WriteString("}")
		return

	case reflect.Slice, reflect.Array:
		b.WriteString(rv.Type().String())
		b.WriteString("{")
		l := rv.Len()
		for i := 0; i < l; i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			writeValue(b, rv.Index(i), seen, depth+1, maxDepth)
		}
		b.WriteString("}")
		return

	case reflect.Map:
		b.WriteString(rv.Type().String())
		b.WriteString("{")
		keys := rv.MapKeys()
		// Try to sort keys for determinism where possible
		sort.SliceStable(keys, func(i, j int) bool {
			return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j])
		})
		for i, k := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			writeValue(b, k, seen, depth+1, maxDepth)
			b.WriteString(": ")
			writeValue(b, rv.MapIndex(k), seen, depth+1, maxDepth)
		}
		b.WriteString("}")
		return

	case reflect.String:
		b.WriteString(strconv.Quote(rv.String()))
		return

	case reflect.Bool:
		if rv.Bool() {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
		return

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		b.WriteString(strconv.FormatInt(rv.Int(), 10))
		return
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		b.WriteString(strconv.FormatUint(rv.Uint(), 10))
		return
	case reflect.Float32, reflect.Float64:
		b.WriteString(strconv.FormatFloat(rv.Float(), 'f', -1, rv.Type().Bits()))
		return
	case reflect.Complex64, reflect.Complex128:
		c := rv.Complex()
		b.WriteString("(")
		b.WriteString(strconv.FormatFloat(real(c), 'f', -1, rv.Type().Bits()/2))
		b.WriteString("+")
		b.WriteString(strconv.FormatFloat(imag(c), 'f', -1, rv.Type().Bits()/2))
		b.WriteString("i)")
		return

	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		// Avoid printing addresses for these kinds; just print the type
		b.WriteString("<")
		b.WriteString(rv.Type().String())
		b.WriteString(">")
		return
	}

	// Fallback: try to use Stringer if available
	if rv.CanInterface() {
		if s, ok := rv.Interface().(fmt.Stringer); ok {
			b.WriteString(s.String())
			return
		}
		b.WriteString(fmt.Sprint(rv.Interface()))
		return
	}
	// Last resort: type name
	b.WriteString(rv.Type().String())
}

// EventBusInterface defines the interface for the event bus
type EventBusInterface interface {
	// Disk events
	EmitDisk(event DiskEvent)
	OnDisk(handler func(context.Context, DiskEvent) errors.E) func()

	// Partition events
	EmitPartition(event PartitionEvent)
	OnPartition(handler func(context.Context, PartitionEvent) errors.E) func()

	// Share events
	EmitShare(event ShareEvent) errors.E
	OnShare(handler func(context.Context, ShareEvent) errors.E) func()
	// Mount point events
	EmitMountPoint(event MountPointEvent) errors.E
	OnMountPoint(handler func(context.Context, MountPointEvent) errors.E) func()

	// User events
	EmitUser(event UserEvent)
	OnUser(handler func(context.Context, UserEvent) errors.E) func()

	// Setting events
	EmitSetting(event SettingEvent)
	OnSetting(handler func(context.Context, SettingEvent) errors.E) func()
	// Samba events
	EmitServerProcess(event ServerProcessEvent)
	OnServerProccess(handler func(context.Context, ServerProcessEvent) errors.E) func()

	// Volume events
	EmitVolume(event VolumeEvent)
	OnVolume(handler func(context.Context, VolumeEvent) errors.E) func()
	// Dirty data events
	EmitDirtyData(event DirtyDataEvent)
	OnDirtyData(handler func(context.Context, DirtyDataEvent) errors.E) func()

	// Home Assistant events
	EmitHomeAssistant(event HomeAssistantEvent)
	OnHomeAssistant(handler func(context.Context, HomeAssistantEvent) errors.E) func()

	// Smart events
	EmitSmart(event SmartEvent)
	OnSmart(handler func(context.Context, SmartEvent) errors.E) func()

	// Power events
	EmitPower(event PowerEvent)
	OnPower(handler func(context.Context, PowerEvent) errors.E) func()

	// Filesystem task events
	EmitFilesystemTask(event FilesystemTaskEvent)
	OnFilesystemTask(handler func(context.Context, FilesystemTaskEvent) errors.E) func()
}

// EventBus implements EventBusInterface using maniartech/signals SyncSignal
type EventBus struct {
	ctx context.Context

	// Synchronous signals (no goroutine dispatch) for deterministic ordering & error management
	disk           signals.SyncSignal[DiskEvent]
	partition      signals.SyncSignal[PartitionEvent]
	share          signals.SyncSignal[ShareEvent]
	mountPoint     signals.SyncSignal[MountPointEvent]
	user           signals.SyncSignal[UserEvent]
	setting        signals.SyncSignal[SettingEvent]
	samba          signals.SyncSignal[ServerProcessEvent]
	volume         signals.SyncSignal[VolumeEvent]
	dirtyData      signals.SyncSignal[DirtyDataEvent]
	homeAssistant  signals.SyncSignal[HomeAssistantEvent]
	smart          signals.SyncSignal[SmartEvent]
	power          signals.SyncSignal[PowerEvent]
	filesystemTask signals.SyncSignal[FilesystemTaskEvent]
}

// NewEventBus creates a new EventBus instance
func NewEventBus(ctx context.Context) EventBusInterface {
	return &EventBus{
		ctx:            ctx,
		disk:           *signals.NewSync[DiskEvent](),
		partition:      *signals.NewSync[PartitionEvent](),
		share:          *signals.NewSync[ShareEvent](),
		mountPoint:     *signals.NewSync[MountPointEvent](),
		user:           *signals.NewSync[UserEvent](),
		setting:        *signals.NewSync[SettingEvent](),
		samba:          *signals.NewSync[ServerProcessEvent](),
		volume:         *signals.NewSync[VolumeEvent](),
		dirtyData:      *signals.NewSync[DirtyDataEvent](),
		homeAssistant:  *signals.NewSync[HomeAssistantEvent](),
		smart:          *signals.NewSync[SmartEvent](),
		power:          *signals.NewSync[PowerEvent](),
		filesystemTask: *signals.NewSync[FilesystemTaskEvent](),
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
				tlog.ErrorContext(ctx, "Event handler panic", append([]any{"event", eventName, "panic", r, "stack", string(debug.Stack())}, caller...)...)
			}
		}()
		tlog.TraceContext(ctx, "<-- Receiving events ", append([]any{"type", fmt.Sprintf("%T", event), "event", formatWithoutPointerAddresses(event)}, caller...)...)
		return handler(ctx, event)
	}, key)
	tlog.Trace("Event handler registered", append([]any{"event", eventName, "listener_count", count}, caller...)...)
	return func() {
		signal.RemoveListener(key)
		tlog.Trace("Event handler unregistered", append([]any{"event", eventName, "key", key}, caller...)...)
	}
}

func emitEvent[T any](signal signals.SyncSignal[T], ctx context.Context, event T) errors.E {
	// Add UUID to context if not already present
	ctx = ContextWithEventUUID(ctx)

	tlog.TraceContext(ctx, "--> Emitting event", append([]any{"type", fmt.Sprintf("%T", event), "event", formatWithoutPointerAddresses(event)}, tlog.WithCaller(1)...)...)
	// Emit synchronously; recover panic inside signal dispatch and log emission errors
	defer func() {
		if r := recover(); r != nil {
			tlog.ErrorContext(ctx, "Panic emitting event", append([]any{"event", formatWithoutPointerAddresses(event), "panic", r, "stack", string(debug.Stack())}, tlog.WithCaller(1)...)...)
		}
	}()
	if err := signal.TryEmit(ctx, event); err != nil {
		// We log at warn level to avoid noisy error logs for expected cancellations
		tlog.WarnContext(ctx, "Event emission error", append([]any{"event", formatWithoutPointerAddresses(event), "error", err}, tlog.WithCaller(1)...)...)
		return errors.WithStack(err)
	}
	return nil
}

// Disk event methods
func (eb *EventBus) EmitDisk(event DiskEvent) {
	emitEvent(eb.disk, eb.ctx, event)
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
func (eb *EventBus) EmitShare(event ShareEvent) errors.E {
	return emitEvent(eb.share, eb.ctx, event)
}

func (eb *EventBus) OnShare(handler func(context.Context, ShareEvent) errors.E) func() {
	return onEvent(eb.share, "Share", handler)
}

// Mount point event methods
func (eb *EventBus) EmitMountPoint(event MountPointEvent) errors.E {
	return emitEvent(eb.mountPoint, eb.ctx, event)
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
func (eb *EventBus) EmitServerProcess(event ServerProcessEvent) {
	emitEvent(eb.samba, eb.ctx, event)
}

func (eb *EventBus) OnServerProccess(handler func(context.Context, ServerProcessEvent) errors.E) func() {
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

// Smart event methods
func (eb *EventBus) EmitSmart(event SmartEvent) {
	emitEvent(eb.smart, eb.ctx, event)
}

func (eb *EventBus) OnSmart(handler func(context.Context, SmartEvent) errors.E) func() {
	return onEvent(eb.smart, "Smart", handler)
}

// Power event methods
func (eb *EventBus) EmitPower(event PowerEvent) {
	emitEvent(eb.power, eb.ctx, event)
}

func (eb *EventBus) OnPower(handler func(context.Context, PowerEvent) errors.E) func() {
	return onEvent(eb.power, "Power", handler)
}

// Filesystem task event methods
func (eb *EventBus) EmitFilesystemTask(event FilesystemTaskEvent) {
	emitEvent(eb.filesystemTask, eb.ctx, event)
}

func (eb *EventBus) OnFilesystemTask(handler func(context.Context, FilesystemTaskEvent) errors.E) func() {
	return onEvent(eb.filesystemTask, "FilesystemTask", handler)
}
