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

func TestHealthRunLoopInternal(t *testing.T) {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, ctxkeys.WaitGroup, wg)

	state := &dto.ContextState{StartTime: time.Now().Add(-1 * time.Second), Heartbeat: 1}
	b := &fakeBroadcaster{}

	h := &HealthHanler{
		HealthPing:             dto.HealthPing{Alive: true},
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		// run returns when context is canceled
		_ = h.run()
	}()

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
