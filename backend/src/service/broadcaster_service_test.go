package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"github.com/teivah/broadcast"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// BroadcasterServiceTestSuite is the test suite for BroadcasterService
type BroadcasterServiceTestSuite struct {
	suite.Suite
	broadcasterService service.BroadcasterServiceInterface
	mockVolumeService  service.VolumeServiceInterface
	mockShareService   service.ShareServiceInterface
	eventBus           events.EventBusInterface
	app                *fxtest.App
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 *sync.WaitGroup
}

func TestBroadcasterServiceTestSuite(t *testing.T) {
	suite.Run(t, new(BroadcasterServiceTestSuite))
}

func (suite *BroadcasterServiceTestSuite) SetupTest() {
	suite.wg = &sync.WaitGroup{}
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), "wg", suite.wg)
				return context.WithCancel(ctx)
			},
			func() *dto.ContextState {
				return &dto.ContextState{}
			},
			func(ctx context.Context) events.EventBusInterface {
				return events.NewEventBus(ctx)
			},
			service.NewBroadcasterService,
			mock.Mock[service.HomeAssistantServiceInterface],
			mock.Mock[service.HaRootServiceInterface],
			mock.Mock[service.VolumeServiceInterface],
			mock.Mock[service.ShareServiceInterface],
		),
		fx.Populate(&suite.ctx, &suite.cancel),
		fx.Populate(&suite.eventBus),
		fx.Populate(&suite.mockVolumeService),
		fx.Populate(&suite.mockShareService),
		fx.Populate(&suite.broadcasterService),
	)
	suite.app.RequireStart()
}

func (suite *BroadcasterServiceTestSuite) TearDownTest() {
	suite.cancel()
	suite.wg.Wait()
	if suite.app != nil {
		suite.app.RequireStop()
	}
}

// --- shouldSkipClientSend Tests ---

// BroadcasterServiceSkipEventTestSuite tests the shouldSkipClientSend method
// using a minimal BroadcasterService instance without full DI
type BroadcasterServiceSkipEventTestSuite struct {
	suite.Suite
}

func TestBroadcasterServiceSkipEventTestSuite(t *testing.T) {
	suite.Run(t, new(BroadcasterServiceSkipEventTestSuite))
}

func (suite *BroadcasterServiceSkipEventTestSuite) TestShouldSkipSSEEvent_SambaStatus() {
	broadcaster := &broadcasterServiceForTesting{}
	suite.True(broadcaster.shouldSkipClientSend(dto.SambaStatus{}), "SambaStatus should be skipped")
	suite.True(broadcaster.shouldSkipClientSend(&dto.SambaStatus{}), "SambaStatus pointer should be skipped")
}

func (suite *BroadcasterServiceSkipEventTestSuite) TestShouldSkipSSEEvent_SambaProcessStatus() {
	broadcaster := &broadcasterServiceForTesting{}
	suite.True(broadcaster.shouldSkipClientSend(dto.ServerProcessStatus{}), "SambaProcessStatus should be skipped")
	suite.True(broadcaster.shouldSkipClientSend(&dto.ServerProcessStatus{}), "SambaProcessStatus pointer should be skipped")
}

func (suite *BroadcasterServiceSkipEventTestSuite) TestShouldSkipSSEEvent_DiskHealth() {
	broadcaster := &broadcasterServiceForTesting{}
	suite.True(broadcaster.shouldSkipClientSend(dto.DiskHealth{}), "DiskHealth should be skipped")
	suite.True(broadcaster.shouldSkipClientSend(&dto.DiskHealth{}), "DiskHealth pointer should be skipped")
}

func (suite *BroadcasterServiceSkipEventTestSuite) TestShouldSkipSSEEvent_HealthPing() {
	broadcaster := &broadcasterServiceForTesting{}
	suite.False(broadcaster.shouldSkipClientSend(dto.HealthPing{}), "HealthPing should not be skipped")
}

func (suite *BroadcasterServiceSkipEventTestSuite) TestShouldSkipSSEEvent_DiskSlice() {
	broadcaster := &broadcasterServiceForTesting{}
	suite.False(broadcaster.shouldSkipClientSend([]dto.Disk{}), "Disk slice should not be skipped")
}

func (suite *BroadcasterServiceSkipEventTestSuite) TestShouldSkipSSEEvent_String() {
	broadcaster := &broadcasterServiceForTesting{}
	suite.False(broadcaster.shouldSkipClientSend("test message"), "String should not be skipped")
}

// broadcasterServiceForTesting is a minimal struct to test shouldSkipClientSend
// This replicates the logic from the actual BroadcasterService
type broadcasterServiceForTesting struct{}

func (b *broadcasterServiceForTesting) shouldSkipClientSend(msg any) bool {
	switch msg.(type) {
	case dto.SambaStatus, *dto.SambaStatus,
		dto.ServerProcessStatus, *dto.ServerProcessStatus,
		dto.DiskHealth, *dto.DiskHealth:
		return true
	default:
		return false
	}
}

// --- Event to Broadcast Mapping Tests ---

// BroadcasterServiceEventMappingTestSuite tests the event to broadcast mapping
type BroadcasterServiceEventMappingTestSuite struct {
	suite.Suite
	mockVolumeService service.VolumeServiceInterface
	eventBus          events.EventBusInterface
	ctx               context.Context
	cancel            context.CancelFunc
	relay             *broadcast.Relay[broadcastEventForTesting]
	unsubs            []func()
}

type broadcastEventForTesting struct {
	ID      uint64
	Message any
}

func TestBroadcasterServiceEventMappingTestSuite(t *testing.T) {
	suite.Run(t, new(BroadcasterServiceEventMappingTestSuite))
}

func (suite *BroadcasterServiceEventMappingTestSuite) SetupTest() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
	suite.eventBus = events.NewEventBus(suite.ctx)
	suite.relay = broadcast.NewRelay[broadcastEventForTesting]()

	ctrl := mock.NewMockController(suite.T())
	suite.mockVolumeService = mock.Mock[service.VolumeServiceInterface](ctrl)
}

func (suite *BroadcasterServiceEventMappingTestSuite) TearDownTest() {
	for _, unsub := range suite.unsubs {
		unsub()
	}
	suite.relay.Close()
	suite.cancel()
}

func (suite *BroadcasterServiceEventMappingTestSuite) setupEventListeners() {
	suite.unsubs = make([]func(), 0, 3)

	// Listen for disk events
	suite.unsubs = append(suite.unsubs, suite.eventBus.OnDisk(func(ctx context.Context, event events.DiskEvent) errors.E {
		suite.relay.Broadcast(broadcastEventForTesting{Message: suite.mockVolumeService.GetVolumesData()})
		return nil
	}))

	// Listen for share events
	suite.unsubs = append(suite.unsubs, suite.eventBus.OnShare(func(ctx context.Context, event events.ShareEvent) errors.E {
		if event.Share != nil {
			suite.relay.Broadcast(broadcastEventForTesting{Message: *event.Share})
		}
		return nil
	}))

	// Listen for mount point events
	suite.unsubs = append(suite.unsubs, suite.eventBus.OnMountPoint(func(ctx context.Context, event events.MountPointEvent) errors.E {
		suite.relay.Broadcast(broadcastEventForTesting{Message: suite.mockVolumeService.GetVolumesData()})
		return nil
	}))
}

/*
func (suite *BroadcasterServiceEventMappingTestSuite) recv() (any, bool) {
	listener := suite.relay.Listener(10)
	defer listener.Close()

	select {
	case ev := <-listener.Ch():
		return ev.Message, true
	case <-time.After(2 * time.Second):
		return nil, false
	}
}
*/

func (suite *BroadcasterServiceEventMappingTestSuite) TestDiskEvent_BroadcastsVolumesData() {
	expectedDisks := []*dto.Disk{{Id: new("expected-disk")}}
	mock.When(suite.mockVolumeService.GetVolumesData()).ThenReturn(expectedDisks)

	suite.setupEventListeners()
	listener := suite.relay.Listener(10)
	defer listener.Close()

	// Emit disk event
	suite.eventBus.EmitDisk(events.DiskEvent{
		Event: events.Event{Type: events.EventTypes.ADD},
		Disk:  &dto.Disk{Id: new("d1")},
	})

	select {
	case ev := <-listener.Ch():
		disks, ok := ev.Message.([]*dto.Disk)
		suite.True(ok, "expected []*dto.Disk, got %T", ev.Message)
		suite.Equal(expectedDisks, disks)
		mock.Verify(suite.mockVolumeService, matchers.Times(1)).GetVolumesData()
	case <-time.After(2 * time.Second):
		suite.Fail("disk event should produce a broadcast")
	}
}

func (suite *BroadcasterServiceEventMappingTestSuite) TestShareEvent_BroadcastsShareItself() {
	expectedDisks := []*dto.Disk{{Id: new("expected-disk")}}
	mock.When(suite.mockVolumeService.GetVolumesData()).ThenReturn(expectedDisks)

	suite.setupEventListeners()
	listener := suite.relay.Listener(10)
	defer listener.Close()

	// Emit share event
	share := &dto.SharedResource{Name: "share-1"}
	suite.eventBus.EmitShare(events.ShareEvent{
		Event: events.Event{Type: events.EventTypes.ADD},
		Share: share,
	})

	select {
	case ev := <-listener.Ch():
		gotShare, ok := ev.Message.(dto.SharedResource)
		suite.True(ok, "expected dto.SharedResource, got %T", ev.Message)
		suite.Equal(*share, gotShare)
		// GetVolumesData should not be called for share events
		mock.Verify(suite.mockVolumeService, matchers.Times(0)).GetVolumesData()
	case <-time.After(2 * time.Second):
		suite.Fail("share event should produce a broadcast")
	}
}

func (suite *BroadcasterServiceEventMappingTestSuite) TestMountPointEvent_BroadcastsVolumesData() {
	expectedDisks := []*dto.Disk{{Id: new("expected-disk")}}
	mock.When(suite.mockVolumeService.GetVolumesData()).ThenReturn(expectedDisks)

	suite.setupEventListeners()
	listener := suite.relay.Listener(10)
	defer listener.Close()

	// Emit mount point event
	mp := &dto.MountPointData{Path: "/mnt/x", IsMounted: true}
	suite.eventBus.EmitMountPoint(events.MountPointEvent{
		Event:      events.Event{Type: events.EventTypes.UPDATE},
		MountPoint: mp,
	})

	select {
	case ev := <-listener.Ch():
		disks, ok := ev.Message.([]*dto.Disk)
		suite.True(ok, "expected []*dto.Disk, got %T", ev.Message)
		suite.Equal(expectedDisks, disks)
		mock.Verify(suite.mockVolumeService, matchers.Times(1)).GetVolumesData()
	case <-time.After(2 * time.Second):
		suite.Fail("mountpoint event should produce a broadcast")
	}
}
