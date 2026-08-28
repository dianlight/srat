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
	"gitlab.com/tozd/go/errors"
)

func TestBroadcasterDirtyDataDedupe_BroadcastsOnlyOnChange(t *testing.T) {
	ctx := t.Context()

	eventBus := events.NewEventBus(ctx)
	b := &BroadcasterService{
		ctx:      ctx,
		relay:    broadcast.NewRelay[broadcastEvent](),
		eventBus: eventBus,
		disks:    dto.NewDiskMap(),
		state:    &dto.ContextState{},
	}

	unsub := b.setupEventListeners()
	defer func() {
		for _, fn := range unsub {
			fn()
		}
	}()

	listener := b.relay.Listener(10)
	defer listener.Close()

	tracker := dto.DataDirtyTracker{Users: true, Shares: false, Settings: true}
	eventBus.EmitDirtyData(events.DirtyDataEvent{Type: events.EventTypes.UPDATE, DataDirtyTracker: tracker})
	eventBus.EmitDirtyData(events.DirtyDataEvent{Type: events.EventTypes.UPDATE, DataDirtyTracker: tracker})
	eventBus.EmitDirtyData(events.DirtyDataEvent{Type: events.EventTypes.UPDATE, DataDirtyTracker: dto.DataDirtyTracker{Users: true, Shares: true, Settings: true}})

	received := 0
	timeout := time.After(250 * time.Millisecond)
	for {
		select {
		case <-listener.Ch():
			received++
		case <-timeout:
			assert.Equal(t, 2, received)
			return
		}
	}
}

// TestBroadcasterSetupEventListeners_CoversAllEventTypes verifies that every
// production listener registered by setupEventListeners broadcasts a message
// when its matching event is emitted on the event bus.
func TestBroadcasterSetupEventListeners_CoversAllEventTypes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctrl := mock.NewMockController(t)
	shareService := mock.Mock[ShareServiceInterface](ctrl)
	shareName := "media"
	shares := []dto.SharedResource{{Name: shareName}}
	mock.When(shareService.ListShares()).ThenReturn(shares, nil)

	eventBus := events.NewEventBus(ctx)
	b := &BroadcasterService{
		ctx:          ctx,
		relay:        broadcast.NewRelay[broadcastEvent](),
		eventBus:     eventBus,
		disks:        dto.NewDiskMap(),
		state:        &dto.ContextState{},
		shareService: shareService,
	}

	unsub := b.setupEventListeners()
	defer func() {
		for _, fn := range unsub {
			fn()
		}
	}()

	listener := b.relay.Listener(64)
	defer listener.Close()

	// Drain helper: wait for exactly one broadcast within the timeout.
	expectBroadcast := func(desc string) {
		t.Helper()
		timeout := time.After(250 * time.Millisecond)
		select {
		case <-listener.Ch():
		case <-timeout:
			t.Fatalf("expected broadcast for %s", desc)
		}
	}
	expectNoBroadcast := func(desc string) {
		t.Helper()
		timeout := time.After(80 * time.Millisecond)
		select {
		case msg := <-listener.Ch():
			t.Fatalf("unexpected broadcast for %s: %#v", desc, msg)
		case <-timeout:
		}
	}

	// 1. Disk event -> disks snapshot
	diskID := "disk-1"
	_ = b.disks.AddOrUpdate(&dto.Disk{Id: &diskID})
	_ = eventBus.EmitDisk(events.DiskEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, Disk: &dto.Disk{Id: &diskID}})
	expectBroadcast("disk")

	// 2. Share event -> share list from shareService
	_ = eventBus.EmitShare(events.ShareEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, Share: &dto.SharedResource{Name: shareName}})
	expectBroadcast("share")

	// 3. MountPoint event -> disks snapshot
	_ = eventBus.EmitMountPoint(events.MountPointEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, MountPoint: &dto.MountPointData{Path: "/mnt/test"}})
	expectBroadcast("mount point")

	// 4. Smart event with DiskId set -> SmartTestStatus broadcast
	eventBus.EmitSmart(events.SmartEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, SmartTestStatus: dto.SmartTestStatus{DiskId: "disk-1", Running: true, Status: "running"}})
	expectBroadcast("smart test status")

	// 5. Smart event with empty DiskId -> no broadcast
	eventBus.EmitSmart(events.SmartEvent{Event: events.Event{Type: events.EventTypes.UPDATE}})
	expectNoBroadcast("smart without disk id")

	// 6. HomeAssistant event of ERROR type -> error broadcast
	eventBus.EmitHomeAssistant(events.HomeAssistantEvent{Event: events.Event{Type: events.EventTypes.ERROR}, Error: &dto.ErrorDataModel{Title: "boom"}})
	expectBroadcast("home assistant error")

	// 7. HomeAssistant event of non-ERROR type -> no broadcast
	eventBus.EmitHomeAssistant(events.HomeAssistantEvent{Event: events.Event{Type: events.EventTypes.UPDATE}})
	expectNoBroadcast("home assistant non-error")

	// 8. AppConfig event -> AppConfigChangedNotification broadcast
	eventBus.EmitAppConfig(events.AppConfigEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, Path: "/data/options.json", Hash: "abc123"})
	expectBroadcast("app config")

	// 9. CommandExecution event -> notification broadcast
	eventBus.EmitCommandExecution(events.CommandExecutionEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, Message: &dto.CommandStartedNotification{ExecutionID: "exec-1", CommandID: "cmd-1", Command: "smbd", StartedAt: 1}})
	expectBroadcast("command execution")

	// 10. FilesystemTask event with Task set -> task broadcast
	eventBus.EmitFilesystemTask(events.FilesystemTaskEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, Task: &dto.FilesystemTask{Device: "sda1", Operation: "check", Status: "running"}})
	expectBroadcast("filesystem task")

	// 11. FilesystemTask event with nil Task -> no broadcast
	eventBus.EmitFilesystemTask(events.FilesystemTaskEvent{Event: events.Event{Type: events.EventTypes.UPDATE}})
	expectNoBroadcast("filesystem task nil")

	// 12. Problem event with Problem set -> problem broadcast
	eventBus.EmitProblem(events.ProblemEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, Problem: &dto.Problem{ProblemKey: "test-problem", Title: "Test"}})
	expectBroadcast("problem")

	// 13. Problem event with nil Problem -> no broadcast
	eventBus.EmitProblem(events.ProblemEvent{Event: events.Event{Type: events.EventTypes.UPDATE}})
	expectNoBroadcast("problem nil")

	// 14. RcloneTask event with Task set -> task broadcast
	eventBus.EmitRcloneTask(events.RcloneTaskEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, Task: &dto.RcloneTask{TargetKind: "volume", TargetID: "/mnt/x", Operation: "sync", Status: "start"}})
	expectBroadcast("rclone task")

	// 15. RcloneTask event with nil Task -> no broadcast
	eventBus.EmitRcloneTask(events.RcloneTaskEvent{Event: events.Event{Type: events.EventTypes.UPDATE}})
	expectNoBroadcast("rclone task nil")

	_, _ = mock.Verify(shareService, matchers.Times(1)).ListShares()
	mock.VerifyNoMoreInteractions(shareService)
}

// TestBroadcasterSetupEventListeners_ShareListError ensures a share listing
// failure is logged but does not crash the listener.
func TestBroadcasterSetupEventListeners_ShareListError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctrl := mock.NewMockController(t)
	shareService := mock.Mock[ShareServiceInterface](ctrl)
	mock.When(shareService.ListShares()).ThenReturn(nil, errors.New("list failed"))

	eventBus := events.NewEventBus(ctx)
	b := &BroadcasterService{
		ctx:          ctx,
		relay:        broadcast.NewRelay[broadcastEvent](),
		eventBus:     eventBus,
		disks:        dto.NewDiskMap(),
		state:        &dto.ContextState{},
		shareService: shareService,
	}

	unsub := b.setupEventListeners()
	defer func() {
		for _, fn := range unsub {
			fn()
		}
	}()

	listener := b.relay.Listener(4)
	defer listener.Close()

	_ = eventBus.EmitShare(events.ShareEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, Share: &dto.SharedResource{Name: "media"}})

	// No broadcast must be emitted when listing shares fails.
	timeout := time.After(80 * time.Millisecond)
	select {
	case msg := <-listener.Ch():
		t.Fatalf("unexpected broadcast on share list error: %#v", msg)
	case <-timeout:
	}
	_, _ = mock.Verify(shareService, matchers.Times(1)).ListShares()
}
