package events

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xorcare/pointer"
)

func TestEventBusDisk(t *testing.T) {
	ctx := context.Background()
	bus := NewEventBus(ctx)

	var receivedEvent *DiskEvent
	var wg sync.WaitGroup
	wg.Add(1)

	// Register listener
	unsubscribe := bus.OnDisk(func(event DiskEvent) {
		receivedEvent = &event
		wg.Done()
	})
	defer unsubscribe()

	// Emit event
	disk := &dto.Disk{
		Id:    pointer.String("sda"),
		Model: pointer.String("Test Disk"),
	}
	bus.EmitDiskAndPartition(DiskEvent{Disk: disk})

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
	unsubscribe := bus.OnPartition(func(event PartitionEvent) {
		receivedEvent = &event
		wg.Done()
	})
	defer unsubscribe()

	// Emit event
	partition := &dto.Partition{
		Name:       pointer.String("sda1"),
		DevicePath: pointer.String("/dev/sda1"),
	}
	disk := &dto.Disk{
		Id: pointer.String("sda"),
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
	unsubscribe := bus.OnShare(func(event ShareEvent) {
		receivedEvent = &event
		wg.Done()
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
	unsubscribe := bus.OnMountPoint(func(event MountPointEvent) {
		receivedEvent = &event
		wg.Done()
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
	unsubscribe1 := bus.OnDisk(func(event DiskEvent) {
		counter.Add(1)
		wg.Done()
	})
	defer unsubscribe1()

	unsubscribe2 := bus.OnDisk(func(event DiskEvent) {
		counter.Add(1)
		wg.Done()
	})
	defer unsubscribe2()

	unsubscribe3 := bus.OnDisk(func(event DiskEvent) {
		counter.Add(1)
		wg.Done()
	})
	defer unsubscribe3()

	// Emit event
	disk := &dto.Disk{
		Id: pointer.String("sda"),
	}
	bus.EmitDiskAndPartition(DiskEvent{Disk: disk})

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
	unsubscribe := bus.OnDisk(func(event DiskEvent) {
		counter.Add(1)
		wg.Done()
	})

	// Unsubscribe before emitting
	unsubscribe()

	// Emit event
	disk := &dto.Disk{
		Id: pointer.String("sda"),
	}
	bus.EmitDiskAndPartition(DiskEvent{Disk: disk})

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
	unsubscribe1 := bus.OnDisk(func(event DiskEvent) {
		mu.Lock()
		receivedDiskIDs[0] = *event.Disk.Id
		mu.Unlock()
		listener1Received.Store(true)
		wg.Done()
	})
	defer unsubscribe1()

	unsubscribe2 := bus.OnDisk(func(event DiskEvent) {
		mu.Lock()
		receivedDiskIDs[1] = *event.Disk.Id
		mu.Unlock()
		listener2Received.Store(true)
		wg.Done()
	})
	defer unsubscribe2()

	unsubscribe3 := bus.OnDisk(func(event DiskEvent) {
		mu.Lock()
		receivedDiskIDs[2] = *event.Disk.Id
		mu.Unlock()
		listener3Received.Store(true)
		wg.Done()
	})
	defer unsubscribe3()

	// Emit one event
	disk := &dto.Disk{
		Id:    pointer.String("sda"),
		Model: pointer.String("Test Disk"),
	}
	bus.EmitDiskAndPartition(DiskEvent{Disk: disk})

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
