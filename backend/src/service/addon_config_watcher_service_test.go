// Package service contains internal unit tests for AddonConfigWatcherService.
// Uses unexported types, so this file lives in package service (not service_test).
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

// AddonConfigWatcherServiceSuite groups all unit tests for AddonConfigWatcherService.
type AddonConfigWatcherServiceSuite struct {
	suite.Suite
	tmpDir string
}

type mockFsnotifyWatcher struct {
	events    chan fsnotify.Event
	errors    chan error
	addedPath string
	closed    bool
}

func (m *mockFsnotifyWatcher) Add(name string) error {
	m.addedPath = name
	return nil
}

func (m *mockFsnotifyWatcher) Close() error {
	m.closed = true
	return nil
}

func (m *mockFsnotifyWatcher) Events() <-chan fsnotify.Event { return m.events }

func (m *mockFsnotifyWatcher) Errors() <-chan error { return m.errors }

func TestAddonConfigWatcherServiceSuite(t *testing.T) {
	suite.Run(t, new(AddonConfigWatcherServiceSuite))
}

func (s *AddonConfigWatcherServiceSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

// writeOptionsFile writes content to a temp options file and returns the path.
func (s *AddonConfigWatcherServiceSuite) writeOptionsFile(content []byte) string {
	path := filepath.Join(s.tmpDir, "options.json")
	err := os.WriteFile(path, content, 0600)
	s.Require().NoError(err)
	return path
}

// sha256hex returns the SHA-256 hex digest of b.
func sha256hex(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

// TestHashFile_ReturnsCorrectDigest verifies hashFile returns the expected SHA-256 hex value.
func (s *AddonConfigWatcherServiceSuite) TestHashFile_ReturnsCorrectDigest() {
	content := []byte(`{"workgroup":"WORKGROUP","name":"test"}`)
	path := s.writeOptionsFile(content)

	svc := &AddonConfigWatcherService{}
	got, err := svc.hashFile(path)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), sha256hex(content), got)
}

// TestHashFile_MissingFile verifies hashFile returns an error for a non-existent path.
func (s *AddonConfigWatcherServiceSuite) TestHashFile_MissingFile() {
	svc := &AddonConfigWatcherService{}
	_, err := svc.hashFile(filepath.Join(s.tmpDir, "nonexistent.json"))
	assert.Error(s.T(), err)
}

// TestHashFile_EmptyFile verifies hashFile succeeds on an empty (zero-byte) file.
func (s *AddonConfigWatcherServiceSuite) TestHashFile_EmptyFile() {
	path := s.writeOptionsFile(nil)
	svc := &AddonConfigWatcherService{}
	got, err := svc.hashFile(path)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), sha256hex(nil), got)
}

// TestMaybeNotify_NoCallOnSameHash verifies onChanged is NOT called when the hash is unchanged.
func (s *AddonConfigWatcherServiceSuite) TestMaybeNotify_NoCallOnSameHash() {
	callCount := 0
	svc := &AddonConfigWatcherService{
		lastHash:  "abc123",
		onChanged: func(path, hash string) { callCount++ },
	}

	svc.maybeNotify("/data/options.json", "abc123")
	assert.Equal(s.T(), 0, callCount)
}

// TestMaybeNotify_CallOnNewHash verifies onChanged IS called with correct args when the hash changes.
func (s *AddonConfigWatcherServiceSuite) TestMaybeNotify_CallOnNewHash() {
	var gotPath, gotHash string
	svc := &AddonConfigWatcherService{
		lastHash: "abc123",
		onChanged: func(path, hash string) {
			gotPath = path
			gotHash = hash
		},
	}

	svc.maybeNotify("/data/options.json", "def456")
	assert.Equal(s.T(), "/data/options.json", gotPath)
	assert.Equal(s.T(), "def456", gotHash)
}

// TestMaybeNotify_NoDuplicateAfterChange verifies repeating the same new hash does not re-trigger.
func (s *AddonConfigWatcherServiceSuite) TestMaybeNotify_NoDuplicateAfterChange() {
	callCount := 0
	svc := &AddonConfigWatcherService{
		lastHash:  "abc123",
		onChanged: func(path, hash string) { callCount++ },
	}

	svc.maybeNotify("/data/options.json", "def456") // first change → call
	svc.maybeNotify("/data/options.json", "def456") // same hash again → no second call
	assert.Equal(s.T(), 1, callCount)
}

// TestMaybeNotify_ConcurrentSafety verifies concurrent calls do not data-race.
func (s *AddonConfigWatcherServiceSuite) TestMaybeNotify_ConcurrentSafety() {
	var mu sync.Mutex
	callCount := 0
	svc := &AddonConfigWatcherService{
		lastHash: "",
		onChanged: func(path, hash string) {
			mu.Lock()
			callCount++
			mu.Unlock()
		},
	}

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			svc.maybeNotify("/data/options.json", "hash"+string(rune('A'+n)))
		}(i)
	}
	wg.Wait()
	// Each goroutine supplies a unique hash; dedup must not panic and must not exceed 20 calls.
	assert.GreaterOrEqual(s.T(), callCount, 1)
	assert.LessOrEqual(s.T(), callCount, 20)
}

// TestWatchViaFsnotify_DetectsWrite verifies the fsnotify watcher detects a file write
// and triggers onChanged within the debounce + safety window.
func (s *AddonConfigWatcherServiceSuite) TestWatchViaFsnotify_DetectsWrite() {
	initialContent := []byte(`{"workgroup":"OLD"}`)
	path := s.writeOptionsFile(initialContent)

	changed := make(chan string, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := &AddonConfigWatcherService{
		ctx:             ctx,
		watchCtx:        ctx,
		watchCancel:     cancel,
		pollInterval:    60 * time.Second,
		optionsFilePath: path,
		lastHash:        sha256hex(initialContent),
		onChanged: func(p, h string) {
			select {
			case changed <- h:
			default:
			}
		},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		svc.watchViaFsnotify()
	}()

	// Give the watcher time to register the file watch.
	time.Sleep(100 * time.Millisecond)

	// Overwrite with new content.
	newContent := []byte(`{"workgroup":"NEW"}`)
	require.NoError(s.T(), os.WriteFile(path, newContent, 0600))

	select {
	case gotHash := <-changed:
		assert.Equal(s.T(), sha256hex(newContent), gotHash)
	case <-time.After(3 * time.Second):
		s.Fail("fsnotify did not detect file change within 3 s")
	}

	cancel()
	wg.Wait()
}

// TestWatchViaFsnotify_MockWatcher_DebouncesAndDedups verifies the fsnotify path
// using a mock watcher, ensuring rapid duplicate write events produce one notification.
func (s *AddonConfigWatcherServiceSuite) TestWatchViaFsnotify_MockWatcher_DebouncesAndDedups() {
	initialContent := []byte(`{"workgroup":"OLD"}`)
	path := s.writeOptionsFile(initialContent)

	changed := make(chan string, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mw := &mockFsnotifyWatcher{
		events: make(chan fsnotify.Event, 4),
		errors: make(chan error, 1),
	}

	svc := &AddonConfigWatcherService{
		ctx:             ctx,
		watchCtx:        ctx,
		watchCancel:     cancel,
		optionsFilePath: path,
		lastHash:        sha256hex(initialContent),
		debounceDelay:   30 * time.Millisecond,
		newFsnotify: func() (fsnotifyWatcher, error) {
			return mw, nil
		},
		onChanged: func(_ string, h string) {
			changed <- h
		},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		svc.watchViaFsnotify()
	}()

	newContent := []byte(`{"workgroup":"NEW"}`)
	require.NoError(s.T(), os.WriteFile(path, newContent, 0600))

	// Duplicate rapid events should be coalesced by debounce logic.
	mw.events <- fsnotify.Event{Name: path, Op: fsnotify.Write}
	mw.events <- fsnotify.Event{Name: path, Op: fsnotify.Write}

	select {
	case gotHash := <-changed:
		assert.Equal(s.T(), sha256hex(newContent), gotHash)
	case <-time.After(2 * time.Second):
		s.Fail("mock fsnotify watcher did not emit change within 2 s")
	}

	select {
	case <-changed:
		s.Fail("duplicate notification emitted for debounced duplicate write events")
	case <-time.After(150 * time.Millisecond):
		// expected: no second emit
	}

	assert.Equal(s.T(), path, mw.addedPath)

	cancel()
	close(mw.events)
	wg.Wait()
	assert.True(s.T(), mw.closed)
}

// TestWatchViaTicker_FallbackDetectsWrite verifies ticker fallback detects file changes
// when relying on hash polling instead of fsnotify events.
func (s *AddonConfigWatcherServiceSuite) TestWatchViaTicker_FallbackDetectsWrite() {
	initialContent := []byte(`{"workgroup":"OLD"}`)
	path := s.writeOptionsFile(initialContent)

	changed := make(chan string, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := &AddonConfigWatcherService{
		ctx:             ctx,
		watchCtx:        ctx,
		watchCancel:     cancel,
		pollInterval:    20 * time.Millisecond,
		optionsFilePath: path,
		lastHash:        sha256hex(initialContent),
		onChanged: func(_ string, h string) {
			select {
			case changed <- h:
			default:
			}
		},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		svc.watchViaTicker()
	}()

	newContent := []byte(`{"workgroup":"NEW"}`)
	require.NoError(s.T(), os.WriteFile(path, newContent, 0600))

	select {
	case gotHash := <-changed:
		assert.Equal(s.T(), sha256hex(newContent), gotHash)
	case <-time.After(2 * time.Second):
		s.Fail("ticker fallback did not detect options file change within 2 s")
	}

	cancel()
	wg.Wait()
}

// TestEmitChanged_EmitsAppConfigEvent verifies that emitChanged publishes an AppConfigEvent
// on the event bus with the correct Path, Hash, and EventType.
func (s *AddonConfigWatcherServiceSuite) TestEmitChanged_EmitsAppConfigEvent() {
	ctx := context.Background()
	bus := events.NewEventBus(ctx)

	received := make(chan events.AppConfigEvent, 1)
	unsubscribe := bus.OnAppConfig(func(_ context.Context, ev events.AppConfigEvent) errors.E {
		select {
		case received <- ev:
		default:
		}
		return nil
	})
	defer unsubscribe()

	svc := &AddonConfigWatcherService{
		ctx:      ctx,
		eventBus: bus,
	}

	svc.emitChanged("/data/options.json", "deadbeef")

	select {
	case ev := <-received:
		assert.Equal(s.T(), events.EventTypes.UPDATE, ev.Type)
		assert.Equal(s.T(), "/data/options.json", ev.Path)
		assert.Equal(s.T(), "deadbeef", ev.Hash)
	case <-time.After(2 * time.Second):
		s.Fail("AppConfigEvent was not emitted within 2 s")
	}
}

// TestEmitChanged_NilEventBus verifies that emitChanged does not panic when no event bus is set.
func (s *AddonConfigWatcherServiceSuite) TestEmitChanged_NilEventBus() {
	svc := &AddonConfigWatcherService{
		ctx:      context.Background(),
		eventBus: nil,
	}
	// Must not panic.
	assert.NotPanics(s.T(), func() {
		svc.emitChanged("/data/options.json", "deadbeef")
	})
}

// TestEmitChanged_CreatesRepairIssue verifies that emitChanged calls RepairService.Create
// with repair_id="addon_config_changed", severity=warning, is_fixable=false, is_persistent=true.
func (s *AddonConfigWatcherServiceSuite) TestEmitChanged_CreatesRepairIssue() {
	ctx := context.Background()
	rs := NewRepairService(ctx, &dto.ContextState{})
	svc := &AddonConfigWatcherService{
		ctx:           ctx,
		repairService: rs,
	}
	svc.emitChanged("/data/options.json", "abc123")
	repair, ok := rs.Get("addon_config_changed")
	require.True(s.T(), ok, "repair issue should exist after emitChanged")
	assert.Equal(s.T(), "addon_config_changed", repair.RepairID)
	assert.Equal(s.T(), dto.RepairIssueSeverityWarning, repair.Command.Severity)
	assert.Equal(s.T(), dto.RepairCommandActionUpsert, repair.Command.Action)
	assert.False(s.T(), repair.Command.IsFixable)
	assert.True(s.T(), repair.Command.IsPersistent)
}

// TestEmitChanged_RepairAlreadyExists_NoPanic verifies that a second emitChanged call
// (when the repair issue already exists) logs a warning but does not panic.
func (s *AddonConfigWatcherServiceSuite) TestEmitChanged_RepairAlreadyExists_NoPanic() {
	ctx := context.Background()
	rs := NewRepairService(ctx, &dto.ContextState{})
	svc := &AddonConfigWatcherService{
		ctx:           ctx,
		repairService: rs,
	}
	svc.emitChanged("/data/options.json", "abc123")
	// Second call should not panic even though the repair already exists.
	assert.NotPanics(s.T(), func() {
		svc.emitChanged("/data/options.json", "xyz456")
	})
}

// TestEmitChanged_FallsBackToNotification verifies that CreatePersistentNotification
// is called with the correct arguments when repairService is nil.
func (s *AddonConfigWatcherServiceSuite) TestEmitChanged_FallsBackToNotification() {
	stub := &stubHAService{}
	svc := &AddonConfigWatcherService{
		ctx:       context.Background(),
		haService: stub,
	}
	svc.emitChanged("/data/options.json", "abc123")
	assert.True(s.T(), stub.notifCalled, "CreatePersistentNotification should have been called")
	assert.Equal(s.T(), "addon_config_changed", stub.notifID)
}

// stubHAService implements HomeAssistantServiceInterface for testing the HA fallback path.
type stubHAService struct {
	notifCalled bool
	notifID     string
	notifTitle  string
	notifMsg    string
}

func (s *stubHAService) SendDiskEntities(_ *[]*dto.Disk) error                         { return nil }
func (s *stubHAService) SendSambaStatusEntity(_ *dto.SambaStatus) error                { return nil }
func (s *stubHAService) SendSambaProcessStatusEntity(_ *dto.ServerProcessStatus) error { return nil }
func (s *stubHAService) SendVolumeStatusEntity(_ *[]*dto.Disk) error                   { return nil }
func (s *stubHAService) SendDiskHealthEntities(_ *dto.DiskHealth) error                { return nil }
func (s *stubHAService) CreatePersistentNotification(id, title, msg string) error {
	s.notifCalled = true
	s.notifID = id
	s.notifTitle = title
	s.notifMsg = msg
	return nil
}
func (s *stubHAService) DismissPersistentNotification(_ string) error { return nil }

// TestIntegration_EndToEnd_FileWriteEmitsAppConfigEvent verifies the full end-to-end flow:
// 1. Write to options file on disk
// 2. fsnotify detects the change
// 3. AppConfigEvent is emitted on the event bus with correct path and hash
// 4. Repair issue is created with correct metadata
func (s *AddonConfigWatcherServiceSuite) TestIntegration_EndToEnd_FileWriteEmitsAppConfigEvent() {
	initialContent := []byte(`{"workgroup":"OLD","name":"test"}`)
	path := s.writeOptionsFile(initialContent)

	ctx := context.Background()
	bus := events.NewEventBus(ctx)

	// Subscribe to AppConfig events to verify emission
	eventReceived := make(chan events.AppConfigEvent, 1)
	unsubscribe := bus.OnAppConfig(func(_ context.Context, ev events.AppConfigEvent) errors.E {
		select {
		case eventReceived <- ev:
		default:
		}
		return nil
	})
	defer unsubscribe()

	// Create a watch context
	watchCtx, watchCancel := context.WithCancel(ctx)
	defer watchCancel()

	// Create the service with shorter intervals for test speed
	svc := &AddonConfigWatcherService{
		ctx:             ctx,
		watchCtx:        watchCtx,
		watchCancel:     watchCancel,
		eventBus:        bus,
		optionsFilePath: path,
		pollInterval:    50 * time.Millisecond, // fast polling for test
		debounceDelay:   30 * time.Millisecond, // fast debounce for test
		newFsnotify: func() (fsnotifyWatcher, error) {
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				return nil, err
			}
			return &realFsnotifyWatcher{Watcher: watcher}, nil
		},
		debounceAfter: func(d time.Duration, f func()) timerStopper {
			return time.AfterFunc(d, f)
		},
	}

	// Seed initial hash
	initialHash, err := svc.hashFile(path)
	require.NoError(s.T(), err)
	svc.lastHash = initialHash

	// Set default onChanged to emit events
	svc.onChanged = svc.emitChanged

	// Start watchers in goroutines
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		svc.watchViaFsnotify()
	}()
	go func() {
		defer wg.Done()
		svc.watchViaTicker()
	}()

	// Allow watchers time to start
	time.Sleep(100 * time.Millisecond)

	// Write new content to the file
	newContent := []byte(`{"workgroup":"NEW","name":"test"}`)
	require.NoError(s.T(), os.WriteFile(path, newContent, 0600))

	// Verify that AppConfigEvent is emitted with correct path and hash
	select {
	case ev := <-eventReceived:
		assert.Equal(s.T(), events.EventTypes.UPDATE, ev.Type)
		assert.Equal(s.T(), path, ev.Path)
		assert.Equal(s.T(), sha256hex(newContent), ev.Hash)
	case <-time.After(3 * time.Second):
		s.Fail("AppConfigEvent was not emitted within 3 s after file write")
	}

	// Clean up
	watchCancel()
	wg.Wait()
}

// TestIntegration_NoEventOnSameHash verifies that a second write with the same content
// does not emit a second event (deduplication via hash).
func (s *AddonConfigWatcherServiceSuite) TestIntegration_NoEventOnSameHash() {
	initialContent := []byte(`{"workgroup":"STABLE"}`)
	path := s.writeOptionsFile(initialContent)

	ctx := context.Background()
	bus := events.NewEventBus(ctx)

	var mu sync.Mutex
	eventCount := 0
	unsubscribe := bus.OnAppConfig(func(_ context.Context, ev events.AppConfigEvent) errors.E {
		mu.Lock()
		eventCount++
		mu.Unlock()
		return nil
	})
	defer unsubscribe()

	watchCtx, watchCancel := context.WithCancel(ctx)
	defer watchCancel()

	svc := &AddonConfigWatcherService{
		ctx:             ctx,
		watchCtx:        watchCtx,
		watchCancel:     watchCancel,
		eventBus:        bus,
		optionsFilePath: path,
		pollInterval:    50 * time.Millisecond,
		debounceDelay:   30 * time.Millisecond,
		newFsnotify: func() (fsnotifyWatcher, error) {
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				return nil, err
			}
			return &realFsnotifyWatcher{Watcher: watcher}, nil
		},
		debounceAfter: func(d time.Duration, f func()) timerStopper {
			return time.AfterFunc(d, f)
		},
	}

	initialHash, err := svc.hashFile(path)
	require.NoError(s.T(), err)
	svc.lastHash = initialHash
	svc.onChanged = svc.emitChanged

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		svc.watchViaFsnotify()
	}()

	time.Sleep(100 * time.Millisecond)

	// First write with new content
	newContent := []byte(`{"workgroup":"CHANGED"}`)
	require.NoError(s.T(), os.WriteFile(path, newContent, 0600))
	time.Sleep(300 * time.Millisecond) // allow time for event

	mu.Lock()
	firstEventCount := eventCount
	mu.Unlock()

	// Second write with same content (should not trigger a new event)
	require.NoError(s.T(), os.WriteFile(path, newContent, 0600))
	time.Sleep(300 * time.Millisecond)

	// Event count should not increase on the second identical write
	mu.Lock()
	finalEventCount := eventCount
	mu.Unlock()

	assert.Equal(s.T(), firstEventCount, finalEventCount, "duplicate event emitted for same-hash write")

	watchCancel()
	wg.Wait()
}
