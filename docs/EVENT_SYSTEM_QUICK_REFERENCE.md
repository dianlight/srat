# Event System Quick Reference

## Quick Start

### Emit an Event

```go
// Disk events
service.eventBus.EmitDiskAdded(events.DiskEvent{Disk: disk})
service.eventBus.EmitDiskRemoved(events.DiskEvent{Disk: disk})

// Partition events
service.eventBus.EmitPartitionAdded(events.PartitionEvent{
    Partition: partition,
    Disk: disk,
})
service.eventBus.EmitPartitionRemoved(events.PartitionEvent{
    Partition: partition,
    Disk: disk,
})

// Share events
service.eventBus.EmitShareCreated(events.ShareEvent{Share: share})
service.eventBus.EmitShareUpdated(events.ShareEvent{Share: share})
service.eventBus.EmitShareDeleted(events.ShareEvent{Share: share})
service.eventBus.EmitShareEnabled(events.ShareEvent{Share: share})
service.eventBus.EmitShareDisabled(events.ShareEvent{Share: share})

// Mount point events
service.eventBus.EmitMountPointMounted(events.MountPointEvent{MountPoint: mp})
service.eventBus.EmitMountPointUnmounted(events.MountPointEvent{MountPoint: mp})
```

### Subscribe to an Event (Manual)

```go
// In your service's OnStart hook or initialization
unsubscribe := service.eventBus.OnDiskAdded(func(event events.DiskEvent) {
    slog.Info("Disk added", "id", event.Disk.Id)
    // Handle event
})

// Always unsubscribe when done (usually not needed due to context cancellation)
defer unsubscribe()
```

## Service Integration Checklist

### Adding EventBus to a Service

1. **Import the events package**
   ```go
   import "github.com/dianlight/srat/events"
   ```

2. **Add field to struct**
   ```go
   type MyService struct {
       eventBus events.EventBusInterface
       // other fields...
   }
   ```

3. **Add to FX params struct**
   ```go
   type MyServiceParams struct {
       fx.In
       EventBus events.EventBusInterface
       // other fields...
   }
   ```

4. **Update constructor**
   ```go
   func NewMyService(in MyServiceParams) MyServiceInterface {
       return &MyService{
           eventBus: in.EventBus,
           // initialize other fields...
       }
   }
   ```

5. **Emit events at change points**
   ```go
   // When something changes
   service.eventBus.EmitShareCreated(events.ShareEvent{Share: newShare})
   ```

## Available Events

### Disk Events
| Event | Method | When |
|-------|--------|------|
| DiskAdded | `EmitDiskAdded()` | New disk detected |
| DiskRemoved | `EmitDiskRemoved()` | Disk removed |

### Partition Events
| Event | Method | When |
|-------|--------|------|
| PartitionAdded | `EmitPartitionAdded()` | New partition found |
| PartitionRemoved | `EmitPartitionRemoved()` | Partition removed |

### Share Events
| Event | Method | When |
|-------|--------|------|
| ShareCreated | `EmitShareCreated()` | Share created |
| ShareUpdated | `EmitShareUpdated()` | Share modified |
| ShareDeleted | `EmitShareDeleted()` | Share deleted |
| ShareEnabled | `EmitShareEnabled()` | Share enabled |
| ShareDisabled | `EmitShareDisabled()` | Share disabled |

### Mount Point Events
| Event | Method | When |
|-------|--------|------|
| MountPointMounted | `EmitMountPointMounted()` | Mount succeeds |
| MountPointUnmounted | `EmitMountPointUnmounted()` | Unmount succeeds |

## Common Patterns

### Service Emitting Event
```go
func (s *Service) CreateShare(share dto.SharedResource) (*dto.SharedResource, errors.E) {
    // ... validation and creation logic ...
    
    // Emit event
    s.eventBus.EmitShareCreated(events.ShareEvent{Share: &share})
    
    return &share, nil
}
```

### Service Listening to Events
```go
func (s *Service) setupEventListeners() {
    s.eventBus.OnDiskAdded(func(event events.DiskEvent) {
        slog.Debug("Disk added", "id", event.Disk.Id)
        // Handle disk addition
    })
}
```

### Testing Events
```go
func TestDiskAddedEvent(t *testing.T) {
    bus := events.NewEventBus(context.Background())
    
    var received *events.DiskEvent
    unsubscribe := bus.OnDiskAdded(func(event events.DiskEvent) {
        received = &event
    })
    defer unsubscribe()
    
    disk := &dto.Disk{Id: pointer.String("test")}
    bus.EmitDiskAdded(events.DiskEvent{Disk: disk})
    
    // Give goroutine time to execute
    time.Sleep(10 * time.Millisecond)
    
    assert.NotNil(t, received)
}
```

## Event Flow

```
Service (VolumeService, ShareService)
    │
    ├─ Detects change
    ├─ Emits event
    │
    ▼
EventBus
    │
    ├─ Calls all registered handlers
    │
    ▼
BroadcasterService
    │
    ├─ Relays through broadcast relay
    │
    ▼
Connected Clients (SSE, WebSocket)
    │
    ├─ Receive events in real-time
    │
    ▼
Home Assistant (if enabled)
```

## Files Reference

| File | Purpose |
|------|---------|
| `backend/src/events/events.go` | Event type definitions |
| `backend/src/events/event_bus.go` | EventBus implementation |
| `backend/src/events/event_bus_test.go` | Tests |
| `backend/src/service/broadcaster_service.go` | Event relay |
| `backend/src/internal/appsetup/appsetup.go` | FX registration |

## Logging

Enable debug logging to see events:
```bash
srat-server -loglevel debug
```

Look for:
- `"Emitting [Type] event"` - Event emitted
- `"Registering [Type] event handler"` - Handler registered
- `"BroadcasterService received [Type] event"` - Event received
- `"Queued Message"` - Event queued for broadcast

## Performance Notes

- ✅ Events are handled asynchronously in goroutines
- ✅ Thread-safe using sync.RWMutex
- ✅ No memory leaks (listeners cleaned up on unsubscribe)
- ✅ Minimal overhead (~microseconds per event)

## Troubleshooting

**Event not received?**
- Check logs for "Emitting" message
- Verify subscriber is connected before emit
- Check for panics in event handler

**Multiple events?**
- Multiple listeners will all be called
- This is expected behavior
- Add deduplication logic if needed

**Memory leaks?**
- Always defer unsubscribe (though context cancellation handles it)
- Check goroutine count with `runtime.NumGoroutine()`

## Adding New Events

1. Define event type in `backend/src/events/events.go`:
   ```go
   type UserEvent struct {
       User *dto.SambaUser
   }
   ```

2. Add methods to `EventBusInterface`:
   ```go
   EmitUserCreated(event UserEvent)
   OnUserCreated(handler func(UserEvent)) func()
   ```

3. Add signal to `EventBus` struct and implement methods

4. Register listener in `BroadcasterService.setupEventListeners()`

5. Add tests in `event_bus_test.go`

6. Update this reference guide

That's it! The rest is automatic through FX dependency injection.
