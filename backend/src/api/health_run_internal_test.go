package api

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/apps"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/server/ws"
	"gitlab.com/tozd/go/errors"
)

// fake broadcaster implements BroadcasterServiceInterface; we only collect messages
type fakeBroadcaster struct {
	msgs []any
	mu   sync.Mutex
}

func (f *fakeBroadcaster) BroadcastMessage(msg any) any {
	f.mu.Lock()
	f.msgs = append(f.msgs, msg)
	f.mu.Unlock()
	return msg
}
func (f *fakeBroadcaster) BroadcastGuaranteedMessage(msg any) any {
	f.mu.Lock()
	f.msgs = append(f.msgs, msg)
	f.mu.Unlock()
	return msg
}
func (f *fakeBroadcaster) ProcessWebSocketChannel(send ws.Sender) {}

// minimal fakes for other services
type fakeSamba struct{}

func (f *fakeSamba) CreateSambaConfigStream() (data *[]byte, err errors.E)   { return nil, nil }
func (f *fakeSamba) CreateSambaUsersMapStream() (data *[]byte, err errors.E) { return nil, nil }
func (f *fakeSamba) GetServerProcesses() (*dto.ServerProcessStatus, errors.E) {
	return &dto.ServerProcessStatus{}, nil
}
func (f *fakeSamba) GetSambaStatus() (*dto.SambaStatus, errors.E)                 { return &dto.SambaStatus{}, nil }
func (f *fakeSamba) WriteSambaConfig(ctx context.Context) errors.E                { return nil }
func (f *fakeSamba) RestartSambaService(ctx context.Context) errors.E             { return nil }
func (f *fakeSamba) TestSambaConfig(ctx context.Context) errors.E                 { return nil }
func (f *fakeSamba) WriteConfigsAndRestartProcesses(ctx context.Context) errors.E { return nil }

func (f *fakeSamba) SetState(state *dto.ContextState) {}

type fakeDirty struct {
	// callbacks []func() errors.E
}

// func (f *fakeDirty) SetDirtyShares()                           {}
// func (f *fakeDirty) SetDirtyVolumes()                          {}
// func (f *fakeDirty) SetDirtyUsers()                            {}
// func (f *fakeDirty) SetDirtySettings()                         {}
func (f *fakeDirty) GetDirtyDataTracker() dto.DataDirtyTracker { return dto.DataDirtyTracker{} }

// func (f *fakeDirty) AddRestartCallback(cb func() errors.E)     { f.callbacks = append(f.callbacks, cb) }
// func (f *fakeDirty) ResetDirtyStatus()                         {}
func (f *fakeDirty) IsTimerRunning() bool   { return false }
func (f *fakeDirty) ResetDirtyDataTracker() {}
func (f *fakeDirty) IsClean() bool          { return true }

type fakeAddons struct{}

func (f *fakeAddons) GetStats() (*apps.AppStatsData, errors.E) {
	return &apps.AppStatsData{}, nil
}

func (f *fakeAddons) GetLatestLogs(ctx context.Context) (string, errors.E) {
	return "fake addon logs", nil
}

func (f *fakeAddons) GetInfo(ctx context.Context) (*apps.AppInfoData, errors.E) {
	return &apps.AppInfoData{}, nil
}

func (f *fakeAddons) GetAppConfig(ctx context.Context) (*dto.AppConfigData, errors.E) {
	return &dto.AppConfigData{Options: map[string]any{}, RuntimeConfig: map[string]any{}, RequiresRestart: true}, nil
}

func (f *fakeAddons) GetAppConfigSchema(ctx context.Context) (*dto.AppConfigSchema, errors.E) {
	return &dto.AppConfigSchema{RequiresRestart: true, Fields: []dto.AppConfigSchemaField{}}, nil
}

func (f *fakeAddons) SetAppConfig(ctx context.Context, options map[string]any) errors.E {
	return nil
}

func (f *fakeAddons) RestartSelfApp(ctx context.Context) errors.E {
	return nil
}

type fakeDiskStats struct{}

func (f *fakeDiskStats) GetDiskStats() (*dto.DiskHealth, errors.E) { return &dto.DiskHealth{}, nil }
func (f *fakeDiskStats) InvalidateSmartCache(diskId string)        {}

type fakeNetStats struct{}

func (f *fakeNetStats) GetNetworkStats() (*dto.NetworkStats, errors.E) {
	return &dto.NetworkStats{}, nil
}

func TestIsExpectedStartupHealthError(t *testing.T) {
	testCases := []struct {
		name      string
		component string
		err       error
		expected  bool
	}{
		{
			name:      "disk stats warmup is expected",
			component: "disk stats",
			err:       errors.New("disk stats not initialized"),
			expected:  true,
		},
		{
			name:      "samba non json warmup is expected",
			component: "samba status",
			err:       errors.New("smbstatus returned non-JSON output: /var/cache/samba/locking.tdb not initialised"),
			expected:  true,
		},
		{
			name:      "real addon stats failure still warns",
			component: "addon stats",
			err:       errors.New("permission denied"),
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isExpectedStartupHealthError(tc.component, tc.err); got != tc.expected {
				t.Fatalf("isExpectedStartupHealthError(%q) = %v, want %v", tc.component, got, tc.expected)
			}
		})
	}
}

// blockingAddons blocks in GetStats until release is closed, simulating a
// service call that hangs forever (issue #1252).
type blockingAddons struct {
	fakeAddons
	release chan struct{}
}

func (f *blockingAddons) GetStats() (*apps.AppStatsData, errors.E) {
	<-f.release
	return &apps.AppStatsData{}, nil
}

// TestHealthRunLoopSurvivesBlockingService verifies that the health run loop
// keeps broadcasting health pings even when a service call blocks indefinitely.
// Regression test for issue #1252: the loop must never wedge on a single call.
func TestHealthRunLoopSurvivesBlockingService(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	ctx = context.WithValue(ctx, ctxkeys.WaitGroup, wg)

	b := &fakeBroadcaster{}
	h := &HealthHanler{
		Alive:                  true,
		state:                  &dto.ContextState{StartTime: time.Now().Add(-time.Second), Heartbeat: 1},
		ctx:                    ctx,
		OutputEventsInterleave: 20 * time.Millisecond,
		callTimeouts:           healthCallTimeouts{heavy: 50 * time.Millisecond, light: 50 * time.Millisecond},
		broadcaster:            b,
		dirtyService:           &fakeDirty{},
		addonsService:          &blockingAddons{release: release},
		diskStatsService:       &fakeDiskStats{},
		networkStatsService:    &fakeNetStats{},
		sambaService:           &fakeSamba{},
	}

	wg.Go(func() {
		_ = h.run()
	})

	// Several tick cycles (each blocked call costs at most 50ms + 20ms wait).
	time.Sleep(300 * time.Millisecond)

	cancel()
	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		t.Error("health run loop did not exit after context cancel (still wedged)")
	}

	b.mu.Lock()
	count := len(b.msgs)
	b.mu.Unlock()
	if count < 2 {
		t.Fatalf("health run loop wedged by blocking service call: got %d broadcasts in 300ms, want >= 2", count)
	}
}

// TestCallWithTimeout verifies the helper reports completion vs timeout.
func TestCallWithTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := &HealthHanler{ctx: ctx}

	t.Run("completes within timeout", func(t *testing.T) {
		if !h.callWithTimeout(time.Second, "fast", func() {}) {
			t.Fatal("expected callWithTimeout to report completion")
		}
	})

	t.Run("reports timeout for blocked call", func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		if h.callWithTimeout(20*time.Millisecond, "blocked", func() { <-release }) {
			t.Fatal("expected callWithTimeout to report timeout")
		}
	})
}

// errSamba fails GetServerProcesses with a nil result, which must not panic
// the health loop (issue #1252 hardening).
type errSamba struct{ fakeSamba }

func (f *errSamba) GetServerProcesses() (*dto.ServerProcessStatus, errors.E) {
	return nil, errors.New("process scan failed")
}

// blockingSamba blocks in GetServerProcesses until release is closed.
type blockingSamba struct {
	fakeSamba
	release chan struct{}
}

func (f *blockingSamba) GetServerProcesses() (*dto.ServerProcessStatus, errors.E) {
	<-f.release
	return &dto.ServerProcessStatus{}, nil
}

// TestCheckSambaKeepsPreviousOnError verifies a failing process scan logs the
// error and keeps the previous SambaProcessStatus instead of panicking on nil.
func TestCheckSambaKeepsPreviousOnError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := &HealthHanler{ctx: ctx, sambaService: &errSamba{}}
	h.SambaProcessStatus = dto.ServerProcessStatus{"smbd": &dto.ProcessStatus{Pid: 42}}

	h.checkSamba(time.Second)

	if pid := h.SambaProcessStatus["smbd"].Pid; pid != 42 {
		t.Fatalf("expected previous status to be kept, got pid %d", pid)
	}
}

// TestCheckSambaTimeoutKeepsPrevious verifies a hung process scan is bounded
// by the timeout and the previous status is kept (issue #1252).
func TestCheckSambaTimeoutKeepsPrevious(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := &HealthHanler{ctx: ctx, sambaService: &blockingSamba{release: release}}
	h.SambaProcessStatus = dto.ServerProcessStatus{"smbd": &dto.ProcessStatus{Pid: 42}}

	start := time.Now()
	h.checkSamba(20 * time.Millisecond)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("checkSamba did not respect timeout, took %v", elapsed)
	}
	if pid := h.SambaProcessStatus["smbd"].Pid; pid != 42 {
		t.Fatalf("expected previous status to be kept on timeout, got pid %d", pid)
	}
}

// TestLogHealthFetchError exercises both the warmup and warning paths.
func TestLogHealthFetchError(t *testing.T) {
	ctx := context.Background()
	logHealthFetchError(ctx, "disk stats", errors.New("disk stats not initialized")) // warmup path
	logHealthFetchError(ctx, "disk stats", errors.New("permission denied"))          // warning path
	logHealthFetchError(ctx, "disk stats", nil)                                      // no error
}

func TestHealthRunLoopInternal(t *testing.T) {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, ctxkeys.WaitGroup, wg)

	state := &dto.ContextState{StartTime: time.Now().Add(-1 * time.Second), Heartbeat: 1}
	b := &fakeBroadcaster{}

	h := &HealthHanler{
		Alive:                  true,
		state:                  state,
		ctx:                    ctx,
		OutputEventsInterleave: 20 * time.Millisecond,
		broadcaster:            b,
		dirtyService:           &fakeDirty{},
		addonsService:          &fakeAddons{},
		diskStatsService:       &fakeDiskStats{},
		networkStatsService:    &fakeNetStats{},
		sambaService:           &fakeSamba{},
	}

	wg.Go(func() {
		// run returns when context is canceled
		_ = h.run()
	})

	// give it a few cycles
	time.Sleep(120 * time.Millisecond)

	b.mu.Lock()
	count := len(b.msgs)
	b.mu.Unlock()
	if count == 0 {
		cancel()
		wg.Wait()
		t.Fatalf("expected at least one broadcast message, got %d", count)
	}

	// stop and wait
	cancel()
	wg.Wait()
}
