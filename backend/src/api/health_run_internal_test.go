package api

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/addons"
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
func (f *fakeBroadcaster) ProcessHttpChannel(send sse.Sender)     {}
func (f *fakeBroadcaster) ProcessWebSocketChannel(send ws.Sender) {}

// minimal fakes for other services
type fakeSamba struct{}

func (f *fakeSamba) CreateSambaConfigStream() (data *[]byte, err errors.E) { return nil, nil }
func (f *fakeSamba) GetServerProcesses() (*dto.ServerProcessStatus, errors.E) {
	return &dto.ServerProcessStatus{}, nil
}
func (f *fakeSamba) GetSambaStatus() (*dto.SambaStatus, errors.E)                 { return &dto.SambaStatus{}, nil }
func (f *fakeSamba) WriteSambaConfig(ctx context.Context) errors.E                { return nil }
func (f *fakeSamba) RestartSambaService(ctx context.Context) errors.E             { return nil }
func (f *fakeSamba) TestSambaConfig(ctx context.Context) errors.E                 { return nil }
func (f *fakeSamba) WriteConfigsAndRestartProcesses(ctx context.Context) errors.E { return nil }

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

func (f *fakeAddons) GetStats() (*addons.AddonStatsData, errors.E) {
	return &addons.AddonStatsData{}, nil
}

func (f *fakeAddons) GetLatestLogs(ctx context.Context) (string, errors.E) {
	return "fake addon logs", nil
}

func (f *fakeAddons) GetInfo(ctx context.Context) (*addons.AddonInfoData, errors.E) {
	return &addons.AddonInfoData{}, nil
}
func (f *fakeAddons) SetOptions(ctx context.Context, options *addons.AddonOptionsRequest) errors.E {
	return nil
}

type fakeDiskStats struct{}

func (f *fakeDiskStats) GetDiskStats() (*dto.DiskHealth, errors.E) { return &dto.DiskHealth{}, nil }

type fakeNetStats struct{}

func (f *fakeNetStats) GetNetworkStats() (*dto.NetworkStats, errors.E) {
	return &dto.NetworkStats{}, nil
}

func TestHealthRunLoopInternal(t *testing.T) {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "wg", wg)

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
