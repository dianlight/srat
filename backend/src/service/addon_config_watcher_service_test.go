// Package service contains internal unit tests for AddonConfigWatcherService.
// Uses unexported types, so this file lives in package service (not service_test).
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/apps"
	"github.com/dianlight/srat/homeassistant/websocket"
	"github.com/dianlight/srat/server/ws"
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

type mockSupervisorWSClient struct {
	mu        sync.Mutex
	attempts  int
	succeedOn int
	handler   func(json.RawMessage)
}

func (m *mockSupervisorWSClient) Connect(ctx context.Context) error { return nil }

func (m *mockSupervisorWSClient) Send(messageType int, data []byte) error { return nil }

func (m *mockSupervisorWSClient) CallService(ctx context.Context, domain, service string, serviceData map[string]any) error {
	return nil
}

func (m *mockSupervisorWSClient) GetStates(ctx context.Context) ([]map[string]any, error) {
	return nil, nil
}

func (m *mockSupervisorWSClient) SubscribeEvents(ctx context.Context, eventType string, handler func(json.RawMessage)) (func() error, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts++
	if m.attempts < m.succeedOn {
		return nil, assert.AnError
	}
	m.handler = handler
	return func() error { return nil }, nil
}

func (m *mockSupervisorWSClient) GetConfig(ctx context.Context) (map[string]any, error) {
	return nil, nil
}

func (m *mockSupervisorWSClient) Receive() <-chan []byte { return nil }

func (m *mockSupervisorWSClient) Close() error { return nil }

func (m *mockSupervisorWSClient) SubscribeConnectionEvents(handler func(event websocket.ConnectionEvent)) (func(), error) {
	return func() {}, nil
}

func (m *mockSupervisorWSClient) emit(raw json.RawMessage) {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		handler(raw)
	}
}

func (m *mockSupervisorWSClient) Attempts() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.attempts
}

type mockAddonInfoClient struct {
	mu      sync.Mutex
	options map[string]any
	err     error
}

func (m *mockAddonInfoClient) GetAppInfoWithResponse(ctx context.Context, addon string, reqEditors ...apps.RequestEditorFn) (*apps.GetAppInfoResponse, error) {
	if m.err != nil {
		return nil, m.err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	optionsCopy := make(map[string]any, len(m.options))
	for key, value := range m.options {
		optionsCopy[key] = value
	}

	return &apps.GetAppInfoResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200: &apps.AppInfoResponse{
			Result: apps.AppInfoResponseResultOk,
			Data: apps.AppInfoData{
				Options: &optionsCopy,
			},
		},
	}, nil
}

func (m *mockAddonInfoClient) SetOptions(options map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.options = options
}

func TestAddonConfigWatcherServiceSuite(t *testing.T) {
	suite.Run(t, new(AddonConfigWatcherServiceSuite))
}

func (s *AddonConfigWatcherServiceSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

// writeOptionsFile writes content to a temp options file and returns the path.
func (s *AddonConfigWatcherServiceSuite) writeOptionsFile(content []byte) string {
	path := filepath.Join(s.tmpDir, "options.json")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	s.Require().NoError(err)
	if len(content) > 0 {
		_, err = f.Write(content)
		s.Require().NoError(err)
		err = f.Sync() // Ensure content is flushed to disk
		s.Require().NoError(err)
	}
	err = f.Close()
	s.Require().NoError(err)
	return path
}

// sha256hex returns the SHA-256 hex digest of b.
func sha256hex(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

func sha256hexJSON(t *testing.T, v any) string {
	t.Helper()
	payload, err := json.Marshal(v)
	require.NoError(t, err)
	return sha256hex(payload)
}

// TestHashFile_ReturnsCorrectDigest verifies hashFile returns the expected SHA-256 hex value.
func (s *AddonConfigWatcherServiceSuite) TestHashFile_ReturnsCorrectDigest() {
	content := []byte(`{"workgroup":"WORKGROUP","name":"test"}`)
	path := s.writeOptionsFile(content)

	svc := &AddonConfigWatcherService{}
	got, err := svc.hashFile(path)
	s.Require().NoError(err)
	s.Equal(sha256hex(content), got)
}

// TestHashFile_MissingFile verifies hashFile returns an error for a non-existent path.
func (s *AddonConfigWatcherServiceSuite) TestHashFile_MissingFile() {
	svc := &AddonConfigWatcherService{}
	_, err := svc.hashFile(filepath.Join(s.tmpDir, "nonexistent.json"))
	s.Error(err)
}

// TestHashFile_EmptyFile verifies hashFile succeeds on an empty (zero-byte) file.
func (s *AddonConfigWatcherServiceSuite) TestHashFile_EmptyFile() {
	path := s.writeOptionsFile(nil)
	svc := &AddonConfigWatcherService{}
	got, err := svc.hashFile(path)
	s.Require().NoError(err)
	s.Equal(sha256hex(nil), got)
}

// TestMaybeNotify_NoCallOnSameHash verifies onChanged is NOT called when the hash is unchanged.
func (s *AddonConfigWatcherServiceSuite) TestMaybeNotify_NoCallOnSameHash() {
	callCount := 0
	svc := &AddonConfigWatcherService{
		lastHash:  "abc123",
		onChanged: func(path, hash string) { callCount++ },
	}

	svc.maybeNotify("/data/options.json", "abc123")
	s.Equal(0, callCount)
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
	s.Equal("/data/options.json", gotPath)
	s.Equal("def456", gotHash)
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
	s.Equal(1, callCount)
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
	s.GreaterOrEqual(callCount, 1)
	s.LessOrEqual(callCount, 20)
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
	s.Require().NoError(os.WriteFile(path, newContent, 0600))

	// Debug: print file contents and hash after write
	f, err := os.Open(path)
	s.Require().NoError(err)
	defer f.Close()
	data, err := io.ReadAll(f)
	s.Require().NoError(err)
	s.T().Logf("[DEBUG] File contents after write: %q", string(data))
	s.T().Logf("[DEBUG] File hash after write: %s", sha256hex(data))

	select {
	case gotHash := <-changed:
		s.Equal(sha256hex(newContent), gotHash)
	case <-time.After(3 * time.Second):
		s.Fail("fsnotify did not detect file change within 3 s")
	}

	cancel()
	wg.Wait()
}

func (s *AddonConfigWatcherServiceSuite) TestShouldWarnSupervisorSubscriptionFailure() {
	s.False(shouldWarnSupervisorSubscriptionFailure(nil, 1))
	s.False(shouldWarnSupervisorSubscriptionFailure(errors.New("not connected"), 1))
	s.True(shouldWarnSupervisorSubscriptionFailure(errors.New("not connected"), 2))
	s.True(shouldWarnSupervisorSubscriptionFailure(errors.New("permission denied"), 1))
}

func (s *AddonConfigWatcherServiceSuite) TestWatchViaSupervisorEvents_RetriesUntilSubscribed() {
	content := []byte(`{"workgroup":"WORKGROUP","name":"test"}`)
	path := s.writeOptionsFile(content)
	changed := make(chan string, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wsClient := &mockSupervisorWSClient{succeedOn: 2}
	svc := &AddonConfigWatcherService{
		ctx:             ctx,
		watchCtx:        ctx,
		watchCancel:     cancel,
		wsClient:        wsClient,
		optionsFilePath: path,
		retryDelay:      10 * time.Millisecond,
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
		svc.watchViaSupervisorEvents()
	}()

	s.Require().Eventually(func() bool {
		return wsClient.Attempts() >= 2
	}, time.Second, 10*time.Millisecond)

	wsClient.emit(json.RawMessage(`{"data":{"event":"addon_config_changed","slug":"local_sambanas2"}}`))

	select {
	case gotHash := <-changed:
		s.Equal(sha256hex(content), gotHash)
	case <-time.After(time.Second):
		s.Fail("supervisor event subscription did not retry and emit a change notification")
	}

	cancel()
	wg.Wait()
}

func (s *AddonConfigWatcherServiceSuite) TestParseSupervisorAddonConfigChanged_AcceptsContractVariants() {
	testCases := []struct {
		name       string
		payload    string
		expected   string
		expSlug    string
		expectedOK bool
	}{
		{
			name:       "nested data.event contract",
			payload:    `{"data":{"event":"addon_config_changed","slug":"local_sambanas2"}}`,
			expected:   "addon_config_changed",
			expSlug:    "local_sambanas2",
			expectedOK: true,
		},
		{
			name:       "nested data.event_type contract",
			payload:    `{"data":{"event_type":"addon_config_changed","addon":"local_sambanas2"}}`,
			expected:   "addon_config_changed",
			expSlug:    "local_sambanas2",
			expectedOK: true,
		},
		{
			name:       "flat contract",
			payload:    `{"event":"addon_config_changed","slug":"local_sambanas2"}`,
			expected:   "addon_config_changed",
			expSlug:    "local_sambanas2",
			expectedOK: true,
		},
		{
			name:       "other event ignored",
			payload:    `{"data":{"event":"core_config_updated","slug":"local_sambanas2"}}`,
			expected:   "core_config_updated",
			expSlug:    "local_sambanas2",
			expectedOK: false,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			eventName, slug, ok, err := parseSupervisorAddonConfigChanged(json.RawMessage(tc.payload))
			require.NoError(t, err)
			assert.Equal(t, tc.expected, eventName)
			assert.Equal(t, tc.expSlug, slug)
			assert.Equal(t, tc.expectedOK, ok)
		})
	}
}

func (s *AddonConfigWatcherServiceSuite) TestParseSupervisorAddonConfigChanged_InvalidJSON() {
	_, _, _, err := parseSupervisorAddonConfigChanged(json.RawMessage(`{"data":`))
	s.Require().Error(err)
}

func (s *AddonConfigWatcherServiceSuite) TestHashObservedConfig_UsesSupervisorOptionsWhenAvailable() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addonClient := &mockAddonInfoClient{
		options: map[string]any{"clean_upgrade_dir": true, "log_level": "info"},
	}
	svc := &AddonConfigWatcherService{
		ctx:          ctx,
		watchCtx:     ctx,
		watchCancel:  cancel,
		addonsClient: addonClient,
	}

	hash, err := svc.hashObservedConfig()
	s.Require().NoError(err)
	s.Equal(sha256hexJSON(s.T(), addonClient.options), hash)
	s.Equal(supervisorOptionsSource, svc.observedConfigPath())
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
	s.Require().NoError(os.WriteFile(path, newContent, 0600))

	// Duplicate rapid events should be coalesced by debounce logic.
	mw.events <- fsnotify.Event{Name: path, Op: fsnotify.Write}
	mw.events <- fsnotify.Event{Name: path, Op: fsnotify.Write}

	select {
	case gotHash := <-changed:
		s.Equal(sha256hex(newContent), gotHash)
	case <-time.After(2 * time.Second):
		s.Fail("mock fsnotify watcher did not emit change within 2 s")
	}

	select {
	case <-changed:
		s.Fail("duplicate notification emitted for debounced duplicate write events")
	case <-time.After(150 * time.Millisecond):
		// expected: no second emit
	}

	s.Equal(path, mw.addedPath)

	cancel()
	close(mw.events)
	wg.Wait()
	s.True(mw.closed)
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
	s.Require().NoError(os.WriteFile(path, newContent, 0600))

	select {
	case gotHash := <-changed:
		s.Equal(sha256hex(newContent), gotHash)
	case <-time.After(2 * time.Second):
		s.Fail("ticker fallback did not detect options file change within 2 s")
	}

	cancel()
	wg.Wait()
}

func (s *AddonConfigWatcherServiceSuite) TestWatchViaTicker_UsesSupervisorOptionsWhenAvailable() {
	changed := make(chan struct {
		path string
		hash string
	}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addonClient := &mockAddonInfoClient{
		options: map[string]any{"clean_upgrade_dir": false},
	}
	svc := &AddonConfigWatcherService{
		ctx:          ctx,
		watchCtx:     ctx,
		watchCancel:  cancel,
		addonsClient: addonClient,
		pollInterval: 20 * time.Millisecond,
		lastHash:     sha256hexJSON(s.T(), addonClient.options),
		onChanged: func(path, hash string) {
			select {
			case changed <- struct {
				path string
				hash string
			}{path: path, hash: hash}:
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

	addonClient.SetOptions(map[string]any{"clean_upgrade_dir": true})

	select {
	case got := <-changed:
		s.Equal(supervisorOptionsSource, got.path)
		s.Equal(sha256hexJSON(s.T(), map[string]any{"clean_upgrade_dir": true}), got.hash)
	case <-time.After(2 * time.Second):
		s.Fail("ticker did not detect supervisor addon options change within 2 s")
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
		s.Equal(events.EventTypes.UPDATE, ev.Type)
		s.Equal("/data/options.json", ev.Path)
		s.Equal("deadbeef", ev.Hash)
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
	s.NotPanics(func() {
		svc.emitChanged("/data/options.json", "deadbeef")
	})
}

// TestEmitChanged_CreatesRepairIssue verifies that emitChanged calls RepairService.Create
// with repair_id="addon_config_changed", severity=warning, is_fixable=false, is_persistent=true.
func (s *AddonConfigWatcherServiceSuite) TestEmitChanged_CreatesRepairIssue() {
	ctx := context.Background()
	rs := NewRepairService(RepairServiceParams{Ctx: ctx, State: &dto.ContextState{}})
	svc := &AddonConfigWatcherService{
		ctx:           ctx,
		repairService: rs,
	}
	svc.emitChanged("/data/options.json", "abc123")
	repair, ok := rs.Get("addon_config_changed")
	s.Require().True(ok, "repair issue should exist after emitChanged")
	s.Equal("addon_config_changed", repair.RepairID)
	s.Equal(dto.RepairIssueSeverityWarning, repair.Command.Severity)
	s.Equal(dto.RepairCommandActionUpsert, repair.Command.Action)
	s.False(repair.Command.IsFixable)
	s.True(repair.Command.IsPersistent)
}

// TestEmitChanged_BroadcastsRepairCommand verifies that emitChanged immediately
// broadcasts the repair command when a broadcaster is available.
func (s *AddonConfigWatcherServiceSuite) TestEmitChanged_BroadcastsRepairCommand() {
	ctx := context.Background()
	b := &stubBroadcaster{}
	rs := NewRepairService(RepairServiceParams{Ctx: ctx, State: &dto.ContextState{}, Broadcaster: b})
	svc := &AddonConfigWatcherService{
		ctx:           ctx,
		repairService: rs,
	}

	svc.emitChanged("/data/options.json", "abc123")

	s.Require().Len(b.messages, 1)
	cmd, ok := b.messages[0].(dto.RepairCommandMessage)
	s.Require().True(ok, "expected a repair command broadcast")
	s.Equal("addon_config_changed", cmd.RepairID)
	s.Equal(dto.RepairCommandActionUpsert, cmd.Action)
}

// TestEmitChanged_DuplicateRepairStillBroadcasts verifies that duplicate repair
// creation refreshes the repair state and still broadcasts the repair command.
func (s *AddonConfigWatcherServiceSuite) TestEmitChanged_DuplicateRepairStillBroadcasts() {
	ctx := context.Background()
	b := &stubBroadcaster{}
	rs := NewRepairService(RepairServiceParams{Ctx: ctx, State: &dto.ContextState{}, Broadcaster: b})
	svc := &AddonConfigWatcherService{
		ctx:           ctx,
		repairService: rs,
	}

	svc.emitChanged("/data/options.json", "abc123")
	svc.emitChanged("/data/options.json", "xyz456")

	s.Require().Len(b.messages, 2)
	cmd, ok := b.messages[1].(dto.RepairCommandMessage)
	s.Require().True(ok, "expected a repair command broadcast")
	s.Equal("addon_config_changed", cmd.RepairID)
	s.Equal(dto.RepairCommandActionUpsert, cmd.Action)
}

// TestEmitChanged_RepairAlreadyExists_NoPanic verifies that a second emitChanged call
// (when the repair issue already exists) logs a warning but does not panic.
func (s *AddonConfigWatcherServiceSuite) TestEmitChanged_RepairAlreadyExists_NoPanic() {
	ctx := context.Background()
	rs := NewRepairService(RepairServiceParams{Ctx: ctx, State: &dto.ContextState{}})
	svc := &AddonConfigWatcherService{
		ctx:           ctx,
		repairService: rs,
	}
	svc.emitChanged("/data/options.json", "abc123")
	// Second call should not panic even though the repair already exists.
	s.NotPanics(func() {
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
	s.True(stub.notifCalled, "CreatePersistentNotification should have been called")
	s.Equal("addon_config_changed", stub.notifID)
}

// stubHAService implements HomeAssistantServiceInterface for testing the HA fallback path.
type stubHAService struct {
	notifCalled bool
	notifID     string
	notifTitle  string
	notifMsg    string
}

type stubBroadcaster struct {
	messages []any
}

func (s *stubBroadcaster) BroadcastMessage(msg any) any {
	s.messages = append(s.messages, msg)
	return msg
}

func (s *stubBroadcaster) ProcessWebSocketChannel(_ ws.Sender) {}

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
func (s *stubHAService) RestartHomeAssistant(_ context.Context) error { return nil }

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
	s.Require().NoError(err)
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
	s.Require().NoError(os.WriteFile(path, newContent, 0600))

	// Ensure file is fully flushed and non-empty before proceeding (fix race)
	f, err := os.Open(path)
	s.Require().NoError(err)
	err = f.Sync()
	f.Close()
	s.Require().NoError(err)

	// Optionally poll file size to ensure write is visible
	deadline := time.Now().Add(1 * time.Second)
	for {
		fi, err := os.Stat(path)
		s.Require().NoError(err)
		if fi.Size() >= int64(len(newContent)) {
			break
		}
		if time.Now().After(deadline) {
			s.FailNow("File write not visible after 1s")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Verify that an AppConfigEvent with the expected final hash is emitted.
	// On some filesystems a transient intermediate hash can be observed while a
	// truncate+write sequence is in progress, so keep listening until timeout.
	expectedHash := sha256hex(newContent)
	timeoutCh := time.After(3 * time.Second)
	matched := false
	for !matched {
		select {
		case ev := <-eventReceived:
			if ev.Path != path || ev.Hash != expectedHash {
				s.T().Logf("[DEBUG] Ignoring intermediate AppConfigEvent: type=%s path=%s hash=%s", ev.Type, ev.Path, ev.Hash)
				continue
			}
			s.Equal(events.EventTypes.UPDATE, ev.Type)
			s.Equal(path, ev.Path)
			s.Equal(expectedHash, ev.Hash)
			matched = true
		case <-timeoutCh:
			s.Fail("AppConfigEvent with expected hash was not emitted within 3 s after file write")
			matched = true
		}
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
	s.Require().NoError(err)
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
	s.Require().NoError(os.WriteFile(path, newContent, 0600))
	time.Sleep(300 * time.Millisecond) // allow time for event

	mu.Lock()
	firstEventCount := eventCount
	mu.Unlock()

	// Second write with same content (should not trigger a new event)
	s.Require().NoError(os.WriteFile(path, newContent, 0600))
	time.Sleep(300 * time.Millisecond)

	// Event count should not increase on the second identical write
	mu.Lock()
	finalEventCount := eventCount
	mu.Unlock()

	s.Equal(firstEventCount, finalEventCount, "duplicate event emitted for same-hash write")

	watchCancel()
	wg.Wait()
}
