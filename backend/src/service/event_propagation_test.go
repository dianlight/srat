package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/repository"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/xorcare/pointer"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gorm.io/gorm"
)

// EventPropagationTestSuite tests event propagation between services
type EventPropagationTestSuite struct {
	suite.Suite
	app              *fxtest.App
	ctx              context.Context
	cancel           context.CancelFunc
	eventBus         events.EventBusInterface
	shareService     ShareServiceInterface
	userService      UserServiceInterface
	dirtyDataService DirtyDataServiceInterface
	mockShareRepo    repository.ExportedShareRepositoryInterface
	mockUserRepo     repository.SambaUserRepositoryInterface
	mockMountRepo    repository.MountPointPathRepositoryInterface
}

func TestEventPropagationTestSuite(t *testing.T) {
	suite.Run(t, new(EventPropagationTestSuite))
}

func (suite *EventPropagationTestSuite) SetupTest() {
	// Create context with WaitGroup
	wg := &sync.WaitGroup{}

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), "wg", wg)
				return context.WithCancel(ctx)
			},
			events.NewEventBus,
			NewDirtyDataService,
			NewShareService,
			NewUserService,
			mock.Mock[repository.ExportedShareRepositoryInterface],
			mock.Mock[repository.SambaUserRepositoryInterface],
			mock.Mock[repository.MountPointPathRepositoryInterface],
		),
		fx.Populate(&suite.eventBus),
		fx.Populate(&suite.shareService),
		fx.Populate(&suite.userService),
		fx.Populate(&suite.dirtyDataService),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
		fx.Populate(&suite.mockShareRepo),
		fx.Populate(&suite.mockUserRepo),
		fx.Populate(&suite.mockMountRepo),
	)
	suite.app.RequireStart()
}

func (suite *EventPropagationTestSuite) TearDownTest() {
	suite.cancel()
	suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
	suite.app.RequireStop()
}

// TestShareServiceToDirtyDataService tests event flow from ShareService to DirtyDataService
func (suite *EventPropagationTestSuite) TestShareServiceToDirtyDataService() {
	// Setup: Track dirty data changes
	var dirtyTracker dto.DataDirtyTracker
	var wg sync.WaitGroup
	wg.Add(1)

	unsubscribe := suite.eventBus.OnDirtyData(func(event events.DirtyDataEvent) {
		dirtyTracker = event.DataDirtyTracker
		wg.Done()
	})
	defer unsubscribe()

	mock.When(suite.mockShareRepo.FindByName("test-share")).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))
	mock.When(suite.mockShareRepo.Save(mock.Any[*dbom.ExportedShare]())).ThenReturn(nil)
	mock.When(suite.mockUserRepo.GetAdmin()).ThenReturn(dbom.SambaUser{
		Username: "admin",
		Password: "admin",
	}, nil)

	// Action: Create a share (which should emit ShareEvent)
	share := dto.SharedResource{
		Name: "test-share",
		MountPointData: &dto.MountPointData{
			Path: "/mnt/test",
		},
	}
	_, err := suite.shareService.CreateShare(share)
	require.NoError(suite.T(), err)

	// Wait for DirtyData event with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// DirtyDataService should have marked shares as dirty
		assert.True(suite.T(), dirtyTracker.Shares, "Shares should be marked dirty")
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for DirtyData event")
	}
}

// TestUserServiceToDirtyDataService tests event flow from UserService to DirtyDataService
func (suite *EventPropagationTestSuite) TestUserServiceToDirtyDataService() {
	// Setup: Track dirty data changes
	var dirtyTracker dto.DataDirtyTracker
	var wg sync.WaitGroup
	wg.Add(1)

	unsubscribe := suite.eventBus.OnDirtyData(func(event events.DirtyDataEvent) {
		dirtyTracker = event.DataDirtyTracker
		wg.Done()
	})
	defer unsubscribe()

	// Mock repository
	mock.When(suite.mockUserRepo.Create(mock.Any[*dbom.SambaUser]())).ThenReturn(nil)

	// Action: Create a user (which should emit UserEvent)
	user := dto.User{
		Username: "testuser",
		Password: "testpass",
	}
	_, err := suite.userService.CreateUser(user)
	require.NoError(suite.T(), err)

	// Wait for DirtyData event with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// DirtyDataService should have marked users as dirty
		assert.True(suite.T(), dirtyTracker.Users, "Users should be marked dirty")
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for DirtyData event")
	}
}

// TestMountPointEventPropagation tests mount point event handling
func (suite *EventPropagationTestSuite) TestMountPointEventPropagation() {
	// Setup: Track mount point events
	var receivedEvent *events.MountPointEvent
	var wg sync.WaitGroup
	wg.Add(1)

	mock.When(suite.mockShareRepo.FindByMountPath("/mnt/test")).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))

	unsubscribe := suite.eventBus.OnMountPoint(func(event events.MountPointEvent) {
		receivedEvent = &event
		wg.Done()
	})
	defer unsubscribe()

	// Action: Emit a MountPoint event
	mountPoint := &dto.MountPointData{
		Path:      "/mnt/test",
		IsMounted: true,
	}
	suite.eventBus.EmitMountPoint(events.MountPointEvent{
		Event: events.Event{
			Type: events.EventTypes.UPDATE,
		},
		MountPoint: mountPoint,
	})

	// Wait for event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		require.NotNil(suite.T(), receivedEvent, "Should receive mount point event")
		assert.Equal(suite.T(), "/mnt/test", receivedEvent.MountPoint.Path)
		assert.True(suite.T(), receivedEvent.MountPoint.IsMounted)
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for mount point event")
	}
}

// TestDiskEventPropagation tests disk event propagation
func (suite *EventPropagationTestSuite) TestDiskEventPropagation() {
	// Setup: Track disk events
	var receivedDiskEvent atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)

	unsubscribe := suite.eventBus.OnDisk(func(event events.DiskEvent) {
		receivedDiskEvent.Store(true)
		wg.Done()
	})
	defer unsubscribe()

	// Action: Emit a Disk event
	disk := &dto.Disk{
		Id:    pointer.String("sda"),
		Model: pointer.String("Test Disk"),
	}
	suite.eventBus.EmitDisk(events.DiskEvent{
		Event: events.Event{
			Type: events.EventTypes.ADD,
		},
		Disk: disk,
	})

	// Wait for event with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		assert.True(suite.T(), receivedDiskEvent.Load(), "Should receive disk event")
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for disk event")
	}
}

// TestMultipleListenersReceiveSameEvent tests that multiple services receive the same event
func (suite *EventPropagationTestSuite) TestMultipleListenersReceiveSameEvent() {
	// Setup: Multiple listeners for Share events
	listener1Received := atomic.Bool{}
	listener2Received := atomic.Bool{}
	listener3Received := atomic.Bool{}

	var wg sync.WaitGroup
	wg.Add(3)

	unsubscribe1 := suite.eventBus.OnShare(func(event events.ShareEvent) {
		listener1Received.Store(true)
		wg.Done()
	})
	defer unsubscribe1()

	unsubscribe2 := suite.eventBus.OnShare(func(event events.ShareEvent) {
		listener2Received.Store(true)
		wg.Done()
	})
	defer unsubscribe2()

	unsubscribe3 := suite.eventBus.OnShare(func(event events.ShareEvent) {
		listener3Received.Store(true)
		wg.Done()
	})
	defer unsubscribe3()

	// Action: Emit one Share event
	share := &dto.SharedResource{
		Name: "broadcast-test",
	}
	suite.eventBus.EmitShare(events.ShareEvent{
		Event: events.Event{
			Type: events.EventTypes.ADD,
		},
		Share: share,
	})

	// Wait for all listeners
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		assert.True(suite.T(), listener1Received.Load(), "listener1 should receive event")
		assert.True(suite.T(), listener2Received.Load(), "listener2 should receive event")
		assert.True(suite.T(), listener3Received.Load(), "listener3 should receive event")
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for all listeners to receive event")
	}
}

// TestEventPropagationChain tests a full chain: Share -> DirtyData -> Samba
func (suite *EventPropagationTestSuite) TestEventPropagationChain() {
	// Setup: Track all events in the chain
	shareEventReceived := atomic.Bool{}
	dirtyDataEventReceived := atomic.Bool{}
	sambaEventReceived := atomic.Bool{}

	var wg sync.WaitGroup
	wg.Add(3)

	unsubscribe1 := suite.eventBus.OnShare(func(event events.ShareEvent) {
		shareEventReceived.Store(true)
		wg.Done()
	})
	defer unsubscribe1()

	unsubscribe2 := suite.eventBus.OnDirtyData(func(event events.DirtyDataEvent) {
		if event.DataDirtyTracker.Shares {
			dirtyDataEventReceived.Store(true)
			wg.Done()
		}
	})
	defer unsubscribe2()

	unsubscribe3 := suite.eventBus.OnSamba(func(event events.SambaEvent) {
		sambaEventReceived.Store(true)
		wg.Done()
	})
	defer unsubscribe3()

	// Mock repository
	//dbShare := &dbom.ExportedShare{
	//	Name:               "chain-test",
	//	MountPointDataPath: "/mnt/chain",
	//	Disabled:           pointer.Bool(false),
	//}
	mock.When(suite.mockShareRepo.FindByName("chain-test")).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))
	mock.When(suite.mockShareRepo.Save(mock.Any[*dbom.ExportedShare]())).ThenReturn(nil)
	//mock.When(suite.mockShareRepo.FindByName("chain-test")).ThenReturn(dbShare, nil)

	// Action: Create a share (triggers the chain)
	share := dto.SharedResource{
		Name: "chain-test",
		MountPointData: &dto.MountPointData{
			Path: "/mnt/chain",
		},
	}
	_, err := suite.shareService.CreateShare(share)
	require.NoError(suite.T(), err)

	// Wait for all events
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		assert.True(suite.T(), shareEventReceived.Load(), "Share event should be received")
		assert.True(suite.T(), dirtyDataEventReceived.Load(), "DirtyData event should be received")
		assert.True(suite.T(), sambaEventReceived.Load(), "Samba event should be received")
	case <-time.After((5 + 1) * time.Second):
		assert.True(suite.T(), shareEventReceived.Load(), "Share event should be received")
		assert.True(suite.T(), dirtyDataEventReceived.Load(), "DirtyData event should be received")
		assert.True(suite.T(), sambaEventReceived.Load(), "Samba event should be received")
		suite.T().Fatal("timeout waiting for event propagation chain")
	}
}

// TestConcurrentEventPropagation tests concurrent event emissions and handling
func (suite *EventPropagationTestSuite) TestConcurrentEventPropagation() {
	// Setup: Count events from multiple concurrent emissions
	shareCounter := atomic.Int32{}
	userCounter := atomic.Int32{}
	diskCounter := atomic.Int32{}

	const numEmissions = 10
	var wg sync.WaitGroup
	wg.Add(numEmissions * 3) // 10 emissions * 3 event types

	unsubscribe1 := suite.eventBus.OnShare(func(event events.ShareEvent) {
		shareCounter.Add(1)
		wg.Done()
	})
	defer unsubscribe1()

	unsubscribe2 := suite.eventBus.OnUser(func(event events.UserEvent) {
		userCounter.Add(1)
		wg.Done()
	})
	defer unsubscribe2()

	unsubscribe3 := suite.eventBus.OnDisk(func(event events.DiskEvent) {
		diskCounter.Add(1)
		wg.Done()
	})
	defer unsubscribe3()

	// Action: Emit multiple events concurrently
	for i := 0; i < numEmissions; i++ {
		go func(idx int) {
			suite.eventBus.EmitShare(events.ShareEvent{
				Share: &dto.SharedResource{Name: "share-" + string(rune(idx))},
			})
		}(i)
		go func(idx int) {
			suite.eventBus.EmitUser(events.UserEvent{
				User: &dto.User{Username: "user-" + string(rune(idx))},
			})
		}(i)
		go func(idx int) {
			suite.eventBus.EmitDisk(events.DiskEvent{
				Disk: &dto.Disk{Id: pointer.String("disk-" + string(rune(idx)))},
			})
		}(i)
	}

	// Wait for all events
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		assert.Equal(suite.T(), int32(numEmissions), shareCounter.Load(), "All share events should be received")
		assert.Equal(suite.T(), int32(numEmissions), userCounter.Load(), "All user events should be received")
		assert.Equal(suite.T(), int32(numEmissions), diskCounter.Load(), "All disk events should be received")
	case <-time.After(5 * time.Second):
		suite.T().Fatal("timeout waiting for concurrent events")
	}
}

// TestEventUnsubscription tests that unsubscribed listeners don't receive events
func (suite *EventPropagationTestSuite) TestEventUnsubscription() {
	// Setup: Subscribe and then unsubscribe
	counter := atomic.Int32{}
	var wg sync.WaitGroup
	wg.Add(1)

	unsubscribe := suite.eventBus.OnShare(func(event events.ShareEvent) {
		counter.Add(1)
		wg.Done()
	})

	// First emission - should be received
	suite.eventBus.EmitShare(events.ShareEvent{
		Share: &dto.SharedResource{Name: "before-unsub"},
	})

	// Wait for first event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		assert.Equal(suite.T(), int32(1), counter.Load(), "First event should be received")
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for first event")
	}

	// Unsubscribe
	unsubscribe()

	// Second emission - should NOT be received
	suite.eventBus.EmitShare(events.ShareEvent{
		Share: &dto.SharedResource{Name: "after-unsub"},
	})

	// Give some time to ensure no event is received
	time.Sleep(500 * time.Millisecond)

	assert.Equal(suite.T(), int32(1), counter.Load(), "No event should be received after unsubscription")
}

// TestDiskEventEmitsPartitionEvents tests that disk events cascade to partition events
func (suite *EventPropagationTestSuite) TestDiskEventEmitsPartitionEvents() {
	// Setup: Track both disk and partition events
	diskReceived := atomic.Bool{}
	partitionReceived := atomic.Bool{}

	var wg sync.WaitGroup
	wg.Add(2)

	unsubscribe1 := suite.eventBus.OnDisk(func(event events.DiskEvent) {
		diskReceived.Store(true)
		wg.Done()
	})
	defer unsubscribe1()

	unsubscribe2 := suite.eventBus.OnPartition(func(event events.PartitionEvent) {
		partitionReceived.Store(true)
		wg.Done()
	})
	defer unsubscribe2()

	// Action: Emit disk with partitions
	partitions := map[string]dto.Partition{
		"sda1": {
			Name:       pointer.String("sda1"),
			DevicePath: pointer.String("/dev/sda1"),
		},
	}
	disk := &dto.Disk{
		Id:         pointer.String("sda"),
		Model:      pointer.String("Test Disk"),
		Partitions: &partitions,
	}
	suite.eventBus.EmitDisk(events.DiskEvent{
		Event: events.Event{
			Type: events.EventTypes.ADD,
		},
		Disk: disk,
	})

	// Wait for events
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		assert.True(suite.T(), diskReceived.Load(), "Disk event should be received")
		assert.True(suite.T(), partitionReceived.Load(), "Partition event should be auto-emitted")
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for disk and partition events")
	}
}

// TestSettingEventPropagation tests setting event propagation
func (suite *EventPropagationTestSuite) TestSettingEventPropagation() {
	// Setup: Track setting events
	eventReceived := atomic.Bool{}
	var wg sync.WaitGroup
	wg.Add(1)

	unsubscribe := suite.eventBus.OnSetting(func(event events.SettingEvent) {
		eventReceived.Store(true)
		wg.Done()
	})
	defer unsubscribe()

	// Action: Emit a setting event
	suite.eventBus.EmitSetting(events.SettingEvent{
		Event: events.Event{
			Type: events.EventTypes.UPDATE,
		},
		Setting: &dto.Settings{},
	})

	// Wait for event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		assert.True(suite.T(), eventReceived.Load(), "Should receive setting event")
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for setting event")
	}
}

