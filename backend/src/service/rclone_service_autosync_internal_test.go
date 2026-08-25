package service

// White-box tests for the rclone auto-sync scheduler internals. These live in
// the same package because startAutoSync/autoSyncLoop/autoSyncInterval/
// autoSyncTick/reconfigureAutoSync are unexported and the default idle tick
// (5 minutes) makes end-to-end timer waits impractical.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	sr "github.com/dianlight/srat/service/rclone"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// stubRPC implements sr.RcloneRPC, records called methods and fails fast on
// data-transfer calls so spawned sync jobs terminate promptly and
// deterministically (failure path writes LastSync bookkeeping).
type stubRPC struct {
	mu    sync.Mutex
	calls []string
}

func (s *stubRPC) Available() bool { return true }

func (s *stubRPC) called(methods ...string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, want := range methods {
		found := false
		for _, c := range s.calls {
			if c == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return len(methods) > 0
}

func (s *stubRPC) RPC(_ context.Context, method string, _ any, _ any) error {
	s.mu.Lock()
	s.calls = append(s.calls, method)
	s.mu.Unlock()
	switch method {
	case "sync/sync", "sync/copy":
		return assert.AnError
	default:
		return nil
	}
}

var _ sr.RcloneRPC = (*stubRPC)(nil)

// newAutoSyncTestService builds a minimal RcloneService wired to an in-memory
// database, a real event bus and the failing-fast stubRPC.
func newAutoSyncTestService(t *testing.T) (*RcloneService, *stubRPC, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// Single pooled connection: :memory: schemas are per-connection and the
	// sync job goroutine queries the DB concurrently with the test.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&dbom.RcloneLink{}))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	rc := &stubRPC{}
	svc := &RcloneService{
		db:       db,
		ctx:      ctx,
		state:    &dto.ContextState{},
		eventBus: events.NewEventBus(ctx),
		rc:       rc,
		running:  map[string]*rcloneRunningJob{},
		pending:  map[string]rclonePendingAuth{},
	}
	return svc, rc, db
}

func seedLink(t *testing.T, db *gorm.DB, mutate func(*dbom.RcloneLink)) dbom.RcloneLink {
	t.Helper()
	row := dbom.RcloneLink{TargetKind: dto.RcloneTargetKindVolume, TargetID: "/mnt/data", Provider: "dropbox"}
	if mutate != nil {
		mutate(&row)
	}
	require.NoError(t, db.Create(&row).Error)
	return row
}

func TestAutoSyncInterval_DefaultWithoutEligibleLinks(t *testing.T) {
	svc, _, _ := newAutoSyncTestService(t)

	assert.Equal(t, 5*time.Minute, svc.autoSyncInterval())
}

func TestAutoSyncInterval_PicksShortestEligibleSchedule(t *testing.T) {
	svc, _, db := newAutoSyncTestService(t)
	seedLink(t, db, func(r *dbom.RcloneLink) {
		r.TargetID = "/mnt/a"
		r.Status = dto.RcloneStatusAuthorized
		r.AutoSync = true
		r.ScheduleMinutes = 30
	})
	seedLink(t, db, func(r *dbom.RcloneLink) {
		r.TargetID = "/mnt/b"
		r.Status = dto.RcloneStatusAuthorized
		r.AutoSync = true
		r.ScheduleMinutes = 10
	})
	// Ineligible rows must be excluded from the minimum.
	seedLink(t, db, func(r *dbom.RcloneLink) {
		r.TargetID = "/mnt/c"
		r.Status = dto.RcloneStatusAuthorized
		r.AutoSync = false
		r.ScheduleMinutes = 1
	})
	seedLink(t, db, func(r *dbom.RcloneLink) {
		r.TargetID = "/mnt/d"
		r.Status = dto.RcloneStatusError
		r.AutoSync = true
		r.ScheduleMinutes = 2
	})

	assert.Equal(t, 10*time.Minute, svc.autoSyncInterval())
}

func TestAutoSyncInterval_QueryErrorFallsBackToDefault(t *testing.T) {
	svc, _, db := newAutoSyncTestService(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	assert.Equal(t, 5*time.Minute, svc.autoSyncInterval())
}

func TestAutoSyncTick_StartsDueSyncAndRecordsBookkeeping(t *testing.T) {
	svc, rc, db := newAutoSyncTestService(t)
	seedLink(t, db, func(r *dbom.RcloneLink) {
		r.Status = dto.RcloneStatusAuthorized
		r.AutoSync = true
		r.ScheduleMinutes = 15
	})

	svc.autoSyncTick()

	key := rcloneJobKey(dto.RcloneTargetKindVolume, "/mnt/data")
	require.Eventually(t, func() bool { return rc.called("sync/sync") }, 3*time.Second, 10*time.Millisecond,
		"auto-sync tick should have started a push sync")
	require.Eventually(t, func() bool {
		var row dbom.RcloneLink
		if err := db.Where("target_kind = ? AND target_id = ?", dto.RcloneTargetKindVolume, "/mnt/data").First(&row).Error; err != nil {
			return false
		}
		return row.LastSyncAt != nil && row.LastSyncResult == "failure"
	}, 3*time.Second, 10*time.Millisecond,
		"completed job goroutine should record last-sync bookkeeping")
	assert.NotContains(t, svc.running, key, "job should deregister itself when done")
}

func TestAutoSyncTick_SkipsNotYetDueLink(t *testing.T) {
	svc, rc, db := newAutoSyncTestService(t)
	recent := time.Now().Add(-time.Minute)
	seedLink(t, db, func(r *dbom.RcloneLink) {
		r.Status = dto.RcloneStatusAuthorized
		r.AutoSync = true
		r.ScheduleMinutes = 30
		r.LastSyncAt = &recent
	})

	svc.autoSyncTick()

	time.Sleep(150 * time.Millisecond)
	rc.mu.Lock()
	calls := append([]string(nil), rc.calls...)
	rc.mu.Unlock()
	assert.NotContains(t, calls, "sync/sync", "link synced a minute ago must not resync within its 30 minute window")
}

func TestAutoSyncTick_SkipsBusyTarget(t *testing.T) {
	svc, rc, db := newAutoSyncTestService(t)
	stale := time.Now().Add(-2 * time.Hour)
	seedLink(t, db, func(r *dbom.RcloneLink) {
		r.Status = dto.RcloneStatusAuthorized
		r.AutoSync = true
		r.ScheduleMinutes = 15
		r.LastSyncAt = &stale
	})

	key := rcloneJobKey(dto.RcloneTargetKindVolume, "/mnt/data")
	svc.running[key] = &rcloneRunningJob{cancel: func() {}}
	defer delete(svc.running, key)

	svc.autoSyncTick()

	time.Sleep(150 * time.Millisecond)
	rc.mu.Lock()
	calls := append([]string(nil), rc.calls...)
	rc.mu.Unlock()
	assert.NotContains(t, calls, "sync/sync", "tick must not stack a second job onto a busy target")
	assert.Contains(t, svc.running, key)
}

func TestAutoSyncTick_IgnoresManualOnlyLinks(t *testing.T) {
	svc, rc, db := newAutoSyncTestService(t)
	seedLink(t, db, func(r *dbom.RcloneLink) {
		r.Status = dto.RcloneStatusAuthorized
		r.AutoSync = true
		r.ScheduleMinutes = 0
	})

	svc.autoSyncTick()

	time.Sleep(150 * time.Millisecond)
	rc.mu.Lock()
	calls := append([]string(nil), rc.calls...)
	rc.mu.Unlock()
	assert.Empty(t, calls, "ScheduleMinutes=0 means manual-only; nothing may be dispatched")
}

func TestStartStopAutoSync_Idempotent(t *testing.T) {
	svc, _, _ := newAutoSyncTestService(t)

	svc.startAutoSync()
	first := svc.autoSyncStop
	require.NotNil(t, first)
	svc.startAutoSync()
	assert.Equal(t, first, svc.autoSyncStop, "second start must be a no-op")

	svc.stopAutoSync()
	assert.Nil(t, svc.autoSyncStop)
	svc.stopAutoSync()
	assert.Nil(t, svc.autoSyncStop, "second stop must be a no-op")
}

func TestReconfigureAutoSync_KeepsLoopAliveAcrossRestarts(t *testing.T) {
	svc, _, _ := newAutoSyncTestService(t)

	svc.startAutoSync()
	require.NotNil(t, svc.autoSyncStop)

	// Simulate a SaveLink-triggered reconfigure while running.
	svc.reconfigureAutoSync()
	require.NotNil(t, svc.autoSyncStop, "loop must be restarted, not stopped")

	svc.stopAutoSync()
	assert.Nil(t, svc.autoSyncStop)
}

func TestAutoSyncLoop_TicksAndExitsOnStop(t *testing.T) {
	svc, rc, db := newAutoSyncTestService(t)
	stale := time.Now().Add(-2 * time.Hour)
	seedLink(t, db, func(r *dbom.RcloneLink) {
		r.Status = dto.RcloneStatusAuthorized
		r.AutoSync = true
		r.ScheduleMinutes = 15
		r.LastSyncAt = &stale
	})

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.autoSyncLoop(stop, 20*time.Millisecond)
	}()

	require.Eventually(t, func() bool { return rc.called("sync/sync") }, 3*time.Second, 10*time.Millisecond,
		"loop tick must dispatch a due sync")

	close(stop)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("auto-sync loop did not exit after stop signal")
	}
}

func TestSaveAndDeleteLink_ReconfigureWhileLoopRunning(t *testing.T) {
	svc, _, db := newAutoSyncTestService(t)
	svc.startAutoSync()
	defer svc.stopAutoSync()

	link := dto.RcloneLink{TargetKind: dto.RcloneTargetKindVolume, TargetID: "/mnt/live", Provider: "dropbox"}
	require.NoError(t, svc.SaveLink(link))

	var count int64
	require.NoError(t, db.Model(&dbom.RcloneLink{}).
		Where("target_kind = ? AND target_id = ?", dto.RcloneTargetKindVolume, "/mnt/live").
		Count(&count).Error)
	require.EqualValues(t, 1, count)

	require.NoError(t, svc.DeleteLink(svc.ctx, dto.RcloneTargetKindVolume, "/mnt/live"))
	require.NoError(t, db.Model(&dbom.RcloneLink{}).
		Where("target_kind = ? AND target_id = ?", dto.RcloneTargetKindVolume, "/mnt/live").
		Count(&count).Error)
	assert.EqualValues(t, 0, count, "soft-deleted link should disappear from default scope")
}
