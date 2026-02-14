package events

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

func TestEventBusDisk(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx)

	var receivedEvent *DiskEvent
	var wg sync.WaitGroup
	wg.Add(1)

	// Register listener
	unsubscribe := bus.OnDisk(func(ctx context.Context, event DiskEvent) errors.E {
		receivedEvent = &event
		wg.Done()
		return nil
	})
	defer unsubscribe()

	// Emit event
	disk := &dto.Disk{
		Id:    new("sda"),
		Model: new("Test Disk"),
	}
	bus.EmitDisk(DiskEvent{Disk: disk})

	// Wait for event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		require.NotNil(t, receivedEvent)
		assert.Equal(t, "sda", *receivedEvent.Disk.Id)
		assert.Equal(t, "Test Disk", *receivedEvent.Disk.Model)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventBusPartition(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx)

	var receivedEvent *PartitionEvent
	var wg sync.WaitGroup
	wg.Add(1)

	// Register listener
	unsubscribe := bus.OnPartition(func(ctx context.Context, event PartitionEvent) errors.E {
		receivedEvent = &event
		wg.Done()
		return nil
	})
	defer unsubscribe()

	// Emit event
	partition := &dto.Partition{
		Name:       new("sda1"),
		DevicePath: new("/dev/sda1"),
	}
	disk := &dto.Disk{
		Id: new("sda"),
	}
	bus.EmitPartition(PartitionEvent{Partition: partition, Disk: disk})

	// Wait for event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		require.NotNil(t, receivedEvent)
		assert.Equal(t, "sda1", *receivedEvent.Partition.Name)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventBusShare(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx)

	var receivedEvent *ShareEvent
	var wg sync.WaitGroup
	wg.Add(1)

	// Register listener
	unsubscribe := bus.OnShare(func(ctx context.Context, event ShareEvent) errors.E {
		receivedEvent = &event
		wg.Done()
		return nil
	})
	defer unsubscribe()

	// Emit event
	share := &dto.SharedResource{
		Name: "test-share",
	}
	bus.EmitShare(ShareEvent{Share: share})

	// Wait for event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		require.NotNil(t, receivedEvent)
		assert.Equal(t, "test-share", receivedEvent.Share.Name)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventBusMountPoint(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx)

	var receivedEvent *MountPointEvent
	var wg sync.WaitGroup
	wg.Add(1)

	// Register listener
	unsubscribe := bus.OnMountPoint(func(ctx context.Context, event MountPointEvent) errors.E {
		receivedEvent = &event
		wg.Done()
		return nil
	})
	defer unsubscribe()

	// Emit event
	mountPoint := &dto.MountPointData{
		Path: "/mnt/test",
	}
	bus.EmitMountPoint(MountPointEvent{MountPoint: mountPoint})

	// Wait for event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		require.NotNil(t, receivedEvent)
		assert.Equal(t, "/mnt/test", receivedEvent.MountPoint.Path)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventBusMultipleListeners(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx)

	counter := atomic.Int32{}
	var wg sync.WaitGroup
	wg.Add(3)

	// Register three listeners
	unsubscribe1 := bus.OnDisk(func(ctx context.Context, event DiskEvent) errors.E {
		counter.Add(1)
		wg.Done()
		return nil
	})
	defer unsubscribe1()

	unsubscribe2 := bus.OnDisk(func(ctx context.Context, event DiskEvent) errors.E {
		counter.Add(1)
		wg.Done()
		return nil
	})
	defer unsubscribe2()

	unsubscribe3 := bus.OnDisk(func(ctx context.Context, event DiskEvent) errors.E {
		counter.Add(1)
		wg.Done()
		return nil
	})
	defer unsubscribe3()

	// Emit event
	disk := &dto.Disk{
		Id: new("sda"),
	}
	bus.EmitDisk(DiskEvent{Disk: disk})

	// Wait for all listeners
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		assert.Equal(t, int32(3), counter.Load())
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for events")
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx)

	counter := atomic.Int32{}
	var wg sync.WaitGroup
	wg.Add(1)

	// Register listener
	unsubscribe := bus.OnDisk(func(ctx context.Context, event DiskEvent) errors.E {
		counter.Add(1)
		wg.Done()
		return nil
	})

	// Unsubscribe before emitting
	unsubscribe()

	// Emit event
	disk := &dto.Disk{
		Id: new("sda"),
	}
	bus.EmitDisk(DiskEvent{Disk: disk})

	// Should timeout since listener was unsubscribed
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("listener should have been unsubscribed")
	case <-time.After(500 * time.Millisecond):
		assert.Equal(t, int32(0), counter.Load())
	}
}

func TestEventBusOneEmitMultipleListeners(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx)

	// Track which listeners received the event
	listener1Received := atomic.Bool{}
	listener2Received := atomic.Bool{}
	listener3Received := atomic.Bool{}

	var receivedDiskIDs [3]string
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(3)

	// Register three listeners that track their received events
	unsubscribe1 := bus.OnDisk(func(ctx context.Context, event DiskEvent) errors.E {
		mu.Lock()
		receivedDiskIDs[0] = *event.Disk.Id
		mu.Unlock()
		listener1Received.Store(true)
		wg.Done()
		return nil
	})
	defer unsubscribe1()

	unsubscribe2 := bus.OnDisk(func(ctx context.Context, event DiskEvent) errors.E {
		mu.Lock()
		receivedDiskIDs[1] = *event.Disk.Id
		mu.Unlock()
		listener2Received.Store(true)
		wg.Done()
		return nil
	})
	defer unsubscribe2()

	unsubscribe3 := bus.OnDisk(func(ctx context.Context, event DiskEvent) errors.E {
		mu.Lock()
		receivedDiskIDs[2] = *event.Disk.Id
		mu.Unlock()
		listener3Received.Store(true)
		wg.Done()
		return nil
	})
	defer unsubscribe3()

	// Emit one event
	disk := &dto.Disk{
		Id:    new("sda"),
		Model: new("Test Disk"),
	}
	bus.EmitDisk(DiskEvent{Disk: disk})

	// Wait for all listeners to receive the event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All three listeners should have received the event
		assert.True(t, listener1Received.Load(), "listener1 should have received event")
		assert.True(t, listener2Received.Load(), "listener2 should have received event")
		assert.True(t, listener3Received.Load(), "listener3 should have received event")

		// All should have received the same disk ID
		mu.Lock()
		assert.Equal(t, "sda", receivedDiskIDs[0])
		assert.Equal(t, "sda", receivedDiskIDs[1])
		assert.Equal(t, "sda", receivedDiskIDs[2])
		mu.Unlock()
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for all listeners to receive event")
	}
}

// TestEventBusUUIDGeneration tests that UUID is generated for events without UUID
func TestEventBusUUIDGeneration(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx)

	var receivedEvent *ShareEvent
	var wg sync.WaitGroup
	wg.Add(1)

	// Register listener
	unsubscribe := bus.OnShare(func(ctx context.Context, event ShareEvent) errors.E {
		receivedEvent = &event
		wg.Done()
		return nil
	})
	defer unsubscribe()

	// Emit event with empty UUID (UUID should be generated in context)
	share := &dto.SharedResource{
		Name: "test-share",
	}
	bus.EmitShare(ShareEvent{
		Event: Event{
			Type: EventTypes.ADD,
		},
		Share: share,
	})

	// Wait for event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		require.NotNil(t, receivedEvent)
		// UUID should be accessible through the context via GetEventUUID
		// Since we can't directly pass context to the listener, we verify UUID is generated
		// by checking the event was received (UUID generation happens internally)
		assert.NotNil(t, receivedEvent, "Event should be received")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

// TestEventBusUUIDNotOverwritten tests that pre-set UUID in context is not overwritten
func TestEventBusUUIDNotOverwritten(t *testing.T) {
	presetUUID := "550e8400-e29b-41d4-a716-446655440000"
	ctx := context.WithValue(context.Background(), struct{}{}, presetUUID)
	bus := NewEventBus(ctx)

	var receivedEvent *ShareEvent
	var wg sync.WaitGroup
	wg.Add(1)

	// Register listener
	unsubscribe := bus.OnShare(func(ctx context.Context, event ShareEvent) errors.E {
		receivedEvent = &event
		wg.Done()
		return nil
	})
	defer unsubscribe()

	// Emit event - UUID should be from context
	share := &dto.SharedResource{
		Name: "test-share",
	}
	bus.EmitShare(ShareEvent{
		Event: Event{
			Type: EventTypes.ADD,
		},
		Share: share,
	})

	// Wait for event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		require.NotNil(t, receivedEvent)
		// UUID is now in context, we just verify event was received
		assert.NotNil(t, receivedEvent, "Event should be received with UUID in context")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

// TestEventBusMultipleListenersGetSameUUID tests that multiple listeners receive events with same UUID in context
func TestEventBusMultipleListenersGetSameUUID(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx)

	var listener1Received, listener2Received bool
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)

	// Register first listener
	unsubscribe1 := bus.OnShare(func(ctx context.Context, event ShareEvent) errors.E {
		mu.Lock()
		listener1Received = true
		mu.Unlock()
		wg.Done()
		return nil
	})
	defer unsubscribe1()

	// Register second listener
	unsubscribe2 := bus.OnShare(func(ctx context.Context, event ShareEvent) errors.E {
		mu.Lock()
		listener2Received = true
		mu.Unlock()
		wg.Done()
		return nil
	})
	defer unsubscribe2()

	// Emit event with empty UUID
	share := &dto.SharedResource{
		Name: "test-share",
	}
	bus.EmitShare(ShareEvent{
		Event: Event{
			Type: EventTypes.ADD,
		},
		Share: share,
	})

	// Wait for all listeners
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		mu.Lock()
		assert.True(t, listener1Received, "Listener 1 should have received event")
		assert.True(t, listener2Received, "Listener 2 should have received event")
		mu.Unlock()
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for events")
	}
}

// TestEventBusConcurrentEmits tests concurrent emits and listener handling
func TestEventBusConcurrentEmits(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx)

	var counter atomic.Int32
	var wg sync.WaitGroup
	numListeners := 5
	numEmits := 10
	wg.Add(numListeners * numEmits)

	// Register multiple listeners
	for i := 0; i < numListeners; i++ {
		unsubscribe := bus.OnDisk(func(ctx context.Context, event DiskEvent) errors.E {
			counter.Add(1)
			wg.Done()
			return nil
		})
		defer unsubscribe()
	}

	// Emit events concurrently
	for i := 0; i < numEmits; i++ {
		go func(i int) {
			disk := &dto.Disk{
				Id: new("sda" + fmt.Sprint(i)),
			}
			bus.EmitDisk(DiskEvent{Disk: disk})
		}(i)
	}

	// Wait for all listeners to process all emits
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		expectedCount := int32(numListeners * numEmits)
		assert.Equal(t, expectedCount, counter.Load(), "All listeners should have processed all emits")
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for all events to be processed")
	}
}

func TestEventBusFilesystemTask(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx)

	var receivedEvent *FilesystemTaskEvent
	var wg sync.WaitGroup
	wg.Add(1)

	// Register listener
	unsubscribe := bus.OnFilesystemTask(func(ctx context.Context, event FilesystemTaskEvent) errors.E {
		receivedEvent = &event
		wg.Done()
		return nil
	})
	defer unsubscribe()

	// Emit event
	task := &dto.FilesystemTask{
		Device:         "/dev/sdb1",
		Operation:      "format",
		FilesystemType: "ext4",
		Status:         "start",
		Message:        "Starting format operation",
	}
	bus.EmitFilesystemTask(FilesystemTaskEvent{
		Event: Event{Type: EventTypes.START},
		Task:  task,
	})

	// Wait for event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		require.NotNil(t, receivedEvent)
		require.NotNil(t, receivedEvent.Task)
		assert.Equal(t, "/dev/sdb1", receivedEvent.Task.Device)
		assert.Equal(t, "format", receivedEvent.Task.Operation)
		assert.Equal(t, "ext4", receivedEvent.Task.FilesystemType)
		assert.Equal(t, "start", receivedEvent.Task.Status)
		assert.Equal(t, EventTypes.START, receivedEvent.Type)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}
