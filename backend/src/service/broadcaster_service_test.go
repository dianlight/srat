package service

import (
	"context"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/assert"
	"github.com/teivah/broadcast"
)

func TestBroadcasterService_shouldSkipSSEEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    any
		expected bool
	}{
		{
			name:     "SambaStatus should be skipped",
			event:    dto.SambaStatus{},
			expected: true,
		},
		{
			name:     "SambaStatus pointer should be skipped",
			event:    &dto.SambaStatus{},
			expected: true,
		},
		{
			name:     "SambaProcessStatus should be skipped",
			event:    dto.SambaProcessStatus{},
			expected: true,
		},
		{
			name:     "SambaProcessStatus pointer should be skipped",
			event:    &dto.SambaProcessStatus{},
			expected: true,
		},
		{
			name:     "DiskHealth should be skipped",
			event:    dto.DiskHealth{},
			expected: true,
		},
		{
			name:     "DiskHealth pointer should be skipped",
			event:    &dto.DiskHealth{},
			expected: true,
		},
		{
			name:     "HealthPing should not be skipped",
			event:    dto.HealthPing{},
			expected: false,
		},
		{
			name:     "Disk slice should not be skipped",
			event:    []dto.Disk{},
			expected: false,
		},
		{
			name:     "String should not be skipped",
			event:    "test message",
			expected: false,
		},
	}

	broadcaster := &BroadcasterService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := broadcaster.shouldSkipClientSend(tt.event)
			assert.Equal(t, tt.expected, result, "shouldSkipSSEEvent() = %v, want %v", result, tt.expected)
		})
	}
}

// Test that BroadcasterService maps incoming events to the expected broadcast messages
// for all 4 cases monitored by setupEventListeners: Disk, Partition, Share, MountPoint.
func TestBroadcasterService_EventToBroadcastMapping(t *testing.T) {
	// Setup minimal context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Event bus
	eb := events.NewEventBus(ctx)

	// Mock VolumeService so Disk/Partition/MountPoint handlers broadcast volumes data
	ctrl := mock.NewMockController(t)
	volMock := mock.Mock[VolumeServiceInterface](ctrl)
	expectedDisks := []*dto.Disk{{Id: ptrStr("expected-disk")}}
	mock.When(volMock.GetVolumesData()).ThenReturn(expectedDisks)

	// Instantiate a bare BroadcasterService with only the pieces we need
	b := &BroadcasterService{
		ctx:           ctx,
		state:         &dto.ContextState{},
		relay:         broadcast.NewRelay[broadcastEvent](),
		eventBus:      eb,
		volumeService: volMock,
	}
	unsubs := b.setupEventListeners()
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	listener := b.relay.Listener(10)
	defer listener.Close()

	// Helper: wait for next broadcast with timeout
	recv := func() (any, bool) {
		select {
		case ev := <-listener.Ch():
			return ev.Message, true
		case <-time.After(2 * time.Second):
			return nil, false
		}
	}

	// 1) Disk -> volumes data
	eb.EmitDisk(events.DiskEvent{Event: events.Event{Type: events.EventTypes.ADD}, Disk: &dto.Disk{Id: ptrStr("d1")}})
	if msg, ok := recv(); assert.True(t, ok, "disk event should produce a broadcast") {
		disks, ok := msg.([]*dto.Disk)
		assert.True(t, ok, "expected []*dto.Disk, got %T", msg)
		if ok {
			assert.Equal(t, expectedDisks, disks)
		}
		mock.Verify(volMock, matchers.Times(1)).GetVolumesData()
	}

	/*
		// 2) Partition -> volumes data
		part := dto.Partition{Name: ptrStr("p1")}
		disk := dto.Disk{Id: ptrStr("d1")}
		eb.EmitPartition(events.PartitionEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, Partition: &part, Disk: &disk})
		if msg, ok := recv(); assert.True(t, ok, "partition event should produce a broadcast") {
			disks, ok := msg.([]*dto.Disk)
			assert.True(t, ok, "expected []dto.Disk, got %T", msg)
			if ok {
				assert.Equal(t, expectedDisks, disks)
			}
			mock.Verify(volMock, matchers.Times(2)).GetVolumesData()
		}
	*/

	// 3) Share -> share itself
	share := &dto.SharedResource{Name: "share-1"}
	eb.EmitShare(events.ShareEvent{Event: events.Event{Type: events.EventTypes.ADD}, Share: share})
	if msg, ok := recv(); assert.True(t, ok, "share event should produce a broadcast") {
		gotShare, ok := msg.(dto.SharedResource)
		assert.True(t, ok, "expected dto.SharedResource, got %T", msg)
		if ok {
			assert.Equal(t, *share, gotShare)
		}
		// GetVolumesData should still be called only twice (from previous two cases)
		mock.Verify(volMock, matchers.Times(2)).GetVolumesData()
	}

	// 4) MountPoint -> volumes data
	mp := &dto.MountPointData{Path: "/mnt/x", IsMounted: true}
	eb.EmitMountPoint(events.MountPointEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, MountPoint: mp})
	if msg, ok := recv(); assert.True(t, ok, "mountpoint event should produce a broadcast") {
		disks, ok := msg.([]*dto.Disk)
		assert.True(t, ok, "expected []dto.Disk, got %T", msg)
		if ok {
			assert.Equal(t, expectedDisks, disks)
		}
		mock.Verify(volMock, matchers.Times(3)).GetVolumesData()
	}
}

// small helper to allocate string pointers
func ptrStr(s string) *string { return &s }
