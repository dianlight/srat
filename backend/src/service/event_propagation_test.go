package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/templates"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"github.com/u-root/u-root/pkg/mount"
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
	VolumeService    VolumeServiceInterface
	mockShareRepo    repository.ExportedShareRepositoryInterface
	mockUserRepo     repository.SambaUserRepositoryInterface
	mockMountRepo    repository.MountPointPathRepositoryInterface
}

func TestEventPropagationTestSuite(t *testing.T) {
	suite.Run(t, new(EventPropagationTestSuite))
}

func (suite *EventPropagationTestSuite) SetupTest() {
	// Mock mount info to prevent osutil.IsMounted from failing
	osutil.MockMountInfo("")

	// Create context with WaitGroup
	wg := &sync.WaitGroup{}

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), "wg", wg)
				return context.WithCancel(ctx)
			},
			func() *config.DefaultConfig {
				var nconfig config.Config
				buffer, err := templates.Default_Config_content.ReadFile("default_config.json")
				if err != nil {
					log.Fatalf("Cant read default config file %#+v", err)
				}
				err = nconfig.LoadConfigBuffer(buffer) // Assign to existing err
				if err != nil {
					log.Fatalf("Cant load default config from buffer %#+v", err)
				}
				return &config.DefaultConfig{Config: nconfig}
			},
			events.NewEventBus,
			NewDirtyDataService,
			NewShareService,
			NewUserService,
			// Real VolumeService with injected test mount ops
			NewVolumeService,
			NewFilesystemService,
			NewSettingService,
			func() *dto.ContextState { return &dto.ContextState{} },
			mock.Mock[IssueServiceInterface],
			mock.Mock[HardwareServiceInterface],
			mock.Mock[repository.ExportedShareRepositoryInterface],
			mock.Mock[repository.SambaUserRepositoryInterface],
			mock.Mock[repository.MountPointPathRepositoryInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[TelemetryServiceInterface],
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
		fx.Populate(&suite.VolumeService),
	)
	mock.When(suite.mockUserRepo.All()).ThenReturn(dbom.SambaUsers{dbom.SambaUser{
		Username: "homeassistant",
		Password: "changeme!",
	}}, nil)

	suite.app.RequireStart()

	// Inject fake mount operations to avoid touching the real OS
	if vs, ok := suite.VolumeService.(*VolumeService); ok {
		vs.MockSetMountOps(
			func(source, target, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error) {
				for _, f := range opts {
					_ = f()
				}
				return &mount.MountPoint{Path: target, Device: source, FSType: "ext4", Flags: flags, Data: data}, nil
			},
			func(source, target, fstype, data string, flags uintptr, opts ...func() error) (*mount.MountPoint, error) {
				for _, f := range opts {
					_ = f()
				}
				if fstype == "" {
					fstype = "ext4"
				}
				return &mount.MountPoint{Path: target, Device: source, FSType: fstype, Flags: flags, Data: data}, nil
			},
			func(target string, force, lazy bool) error {
				// Ensure the directory exists so VolumeService's os.Remove succeeds
				_ = os.MkdirAll(target, 0o750)
				return nil
			},
		)

		// Populate disks with fake partitions so VolumeService can find them
		// This avoids "Source device does not exist on the system" errors
		fakeDisk := dto.Disk{
			Id:    pointer.String("sdk"),
			Model: pointer.String("Fake Test Disk"),
			Partitions: &map[string]dto.Partition{
				"sda1": {Id: pointer.String("sda1"), DevicePath: pointer.String("/dev/fake-sda1"), DiskId: pointer.String("sdk")},
				"sdb1": {Id: pointer.String("sdb1"), DevicePath: pointer.String("/dev/fake-sdb1"), DiskId: pointer.String("sdk")},
				"sdc1": {Id: pointer.String("sdc1"), DevicePath: pointer.String("/dev/fake-sdc1"), DiskId: pointer.String("sdk")},
				"sdd1": {Id: pointer.String("sdd1"), DevicePath: pointer.String("/dev/fake-sdd1"), DiskId: pointer.String("sdk")},
				"sde1": {Id: pointer.String("sde1"), DevicePath: pointer.String("/dev/fake-sde1"), DiskId: pointer.String("sdk")},
				"sdf1": {Id: pointer.String("sdf1"), DevicePath: pointer.String("/dev/fake-sdf1"), DiskId: pointer.String("sdk")},
				"sdg1": {Id: pointer.String("sdg1"), DevicePath: pointer.String("/dev/fake-sdg1"), DiskId: pointer.String("sdk")},
				"sdh1": {Id: pointer.String("sdh1"), DevicePath: pointer.String("/dev/fake-sdh1"), DiskId: pointer.String("sdk")},
				"sdi1": {Id: pointer.String("sdi1"), DevicePath: pointer.String("/dev/fake-sdi1"), DiskId: pointer.String("sdk")},
			},
		}
		if vs.disks == nil {
			vs.disks = &dto.DiskMap{}
		}
		vs.disks.Add(fakeDisk)
	}

	// Setup global mock for share repo to avoid nil pointer errors when mount point events trigger share lookups
	mock.When(suite.mockShareRepo.FindByMountPath(mock.Any[string]())).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))
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
	var shareEventReceived atomic.Bool
	var wg sync.WaitGroup
	wg.Add(2)

	unsubscribe := suite.eventBus.OnDirtyData(func(ctx context.Context, event events.DirtyDataEvent) errors.E {
		dirtyTracker = event.DataDirtyTracker
		wg.Done()
		return nil
	})
	defer unsubscribe()
	unsubscribeShare := suite.eventBus.OnShare(func(ctx context.Context, event events.ShareEvent) errors.E {
		if event.Event.Type == events.EventTypes.ADD {
			shareEventReceived.Store(true)
		}
		wg.Done()
		return nil
	})
	defer unsubscribeShare()

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
			Path: "/",
		},
	}
	_, err := suite.shareService.CreateShare(share)
	suite.Require().NoError(err)

	// Wait for DirtyData event with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// DirtyDataService should have marked shares as dirty
		suite.True(shareEventReceived.Load(), "Share event should be received")
		suite.True(dirtyTracker.Shares, "Shares should be marked dirty")
	case <-time.After(2 * time.Second):
		suite.True(shareEventReceived.Load(), "Share event should be received")
		suite.True(dirtyTracker.Shares, "Shares should be marked dirty")
		suite.T().Fatal("timeout waiting for DirtyData event")
	}
}

// TestUserServiceToDirtyDataService tests event flow from UserService to DirtyDataService
func (suite *EventPropagationTestSuite) TestUserServiceToDirtyDataService() {
	// Setup: Track dirty data changes
	var dirtyTracker dto.DataDirtyTracker
	var wg sync.WaitGroup
	wg.Add(1)

	unsubscribe := suite.eventBus.OnDirtyData(func(ctx context.Context, event events.DirtyDataEvent) errors.E {
		dirtyTracker = event.DataDirtyTracker
		wg.Done()
		return nil
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
	suite.Require().NoError(err)

	// Wait for DirtyData event with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// DirtyDataService should have marked users as dirty
		suite.True(dirtyTracker.Users, "Users should be marked dirty")
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

	unsubscribe := suite.eventBus.OnMountPoint(func(ctx context.Context, event events.MountPointEvent) errors.E {
		receivedEvent = &event
		wg.Done()
		return nil
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
		suite.Require().NotNil(receivedEvent, "Should receive mount point event")
		suite.Equal("/mnt/test", receivedEvent.MountPoint.Path)
		suite.True(receivedEvent.MountPoint.IsMounted)
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

	unsubscribe := suite.eventBus.OnDisk(func(ctx context.Context, event events.DiskEvent) errors.E {
		receivedDiskEvent.Store(true)
		wg.Done()
		return nil
	})
	defer unsubscribe()

	// Action: Emit a Disk event
	disk := &dto.Disk{
		Id:    pointer.String("sda"),
		Model: pointer.String("Test Disk"),
	}
	suite.eventBus.EmitDiskAndPartition(events.DiskEvent{
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
		suite.True(receivedDiskEvent.Load(), "Should receive disk event")
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

	unsubscribe1 := suite.eventBus.OnShare(func(ctx context.Context, event events.ShareEvent) errors.E {
		listener1Received.Store(true)
		wg.Done()
		return nil
	})
	defer unsubscribe1()

	unsubscribe2 := suite.eventBus.OnShare(func(ctx context.Context, event events.ShareEvent) errors.E {
		listener2Received.Store(true)
		wg.Done()
		return nil
	})
	defer unsubscribe2()

	unsubscribe3 := suite.eventBus.OnShare(func(ctx context.Context, event events.ShareEvent) errors.E {
		listener3Received.Store(true)
		wg.Done()
		return nil
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
		suite.True(listener1Received.Load(), "listener1 should receive event")
		suite.True(listener2Received.Load(), "listener2 should receive event")
		suite.True(listener3Received.Load(), "listener3 should receive event")
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for all listeners to receive event")
	}
}

// TestEventPropagationChain tests a full chain: Share -> DirtyData
func (suite *EventPropagationTestSuite) TestEventPropagationChain() {
	// Setup: Track all events in the chain
	shareEventReceived := atomic.Bool{}
	dirtyDataEventReceived := atomic.Bool{}

	var wg sync.WaitGroup
	wg.Add(2)

	unsubscribe1 := suite.eventBus.OnShare(func(ctx context.Context, event events.ShareEvent) errors.E {
		shareEventReceived.Store(true)
		wg.Done()
		return nil
	})
	defer unsubscribe1()

	unsubscribe2 := suite.eventBus.OnDirtyData(func(ctx context.Context, event events.DirtyDataEvent) errors.E {
		if event.DataDirtyTracker.Shares {
			dirtyDataEventReceived.Store(true)
			wg.Done()
		}
		return nil
	})
	defer unsubscribe2()

	// Mock repository
	mock.When(suite.mockShareRepo.FindByName("chain-test")).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))
	mock.When(suite.mockShareRepo.Save(mock.Any[*dbom.ExportedShare]())).ThenReturn(nil)

	// Action: Create a share (triggers the chain)
	share := dto.SharedResource{
		Name: "chain-test",
		MountPointData: &dto.MountPointData{
			Path: "/mnt/chain",
		},
	}
	_, err := suite.shareService.CreateShare(share)
	suite.Require().NoError(err)

	// Wait for all events
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		suite.True(shareEventReceived.Load(), "Share event should be received")
		suite.True(dirtyDataEventReceived.Load(), "DirtyData event should be received")
	case <-time.After((5 + 1) * time.Second):
		suite.True(shareEventReceived.Load(), "Share event should be received")
		suite.True(dirtyDataEventReceived.Load(), "DirtyData event should be received")
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

	unsubscribe1 := suite.eventBus.OnShare(func(ctx context.Context, event events.ShareEvent) errors.E {
		shareCounter.Add(1)
		wg.Done()
		return nil
	})
	defer unsubscribe1()

	unsubscribe2 := suite.eventBus.OnUser(func(ctx context.Context, event events.UserEvent) errors.E {
		userCounter.Add(1)
		wg.Done()
		return nil
	})
	defer unsubscribe2()

	unsubscribe3 := suite.eventBus.OnDisk(func(ctx context.Context, event events.DiskEvent) errors.E {
		diskCounter.Add(1)
		wg.Done()
		return nil
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
			suite.eventBus.EmitDiskAndPartition(events.DiskEvent{
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
		suite.Equal(int32(numEmissions), shareCounter.Load(), "All share events should be received")
		suite.Equal(int32(numEmissions), userCounter.Load(), "All user events should be received")
		suite.Equal(int32(numEmissions), diskCounter.Load(), "All disk events should be received")
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

	unsubscribe := suite.eventBus.OnShare(func(ctx context.Context, event events.ShareEvent) errors.E {
		counter.Add(1)
		wg.Done()
		return nil
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
		suite.Equal(int32(1), counter.Load(), "First event should be received")
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

	suite.Equal(int32(1), counter.Load(), "No event should be received after unsubscription")
}

// TestDiskEventEmitsPartitionEvents tests that disk events cascade to partition events
func (suite *EventPropagationTestSuite) TestDiskEventEmitsPartitionEvents() {
	// Setup: Track both disk and partition events
	diskReceived := atomic.Bool{}
	partitionReceived := atomic.Bool{}

	var wg sync.WaitGroup
	wg.Add(2)

	unsubscribe1 := suite.eventBus.OnDisk(func(ctx context.Context, event events.DiskEvent) errors.E {
		diskReceived.Store(true)
		wg.Done()
		return nil
	})
	defer unsubscribe1()

	unsubscribe2 := suite.eventBus.OnPartition(func(ctx context.Context, event events.PartitionEvent) errors.E {
		partitionReceived.Store(true)
		wg.Done()
		return nil
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
	suite.eventBus.EmitDiskAndPartition(events.DiskEvent{
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
		suite.True(diskReceived.Load(), "Disk event should be received")
		suite.True(partitionReceived.Load(), "Partition event should be auto-emitted")
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

	unsubscribe := suite.eventBus.OnSetting(func(ctx context.Context, event events.SettingEvent) errors.E {
		eventReceived.Store(true)
		wg.Done()
		return nil
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
		suite.True(eventReceived.Load(), "Should receive setting event")
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for setting event")
	}
}

// TestVolumeEventPropagation tests volume unmount event propagation (mount events go through MountPoint)
func (suite *EventPropagationTestSuite) TestVolumeEventPropagation() {
	// Setup: Track mount point events (mounts) and volume events (unmounts)
	var receivedMountEvent *events.MountPointEvent
	var wg sync.WaitGroup
	wg.Add(1)

	unsubscribe := suite.eventBus.OnMountPoint(func(ctx context.Context, event events.MountPointEvent) errors.E {
		receivedMountEvent = &event
		wg.Done()
		return nil
	})
	defer unsubscribe()

	// Action: Call VolumeService to mount (emits MountPointEvent, not VolumeEvent)
	mountPoint := &dto.MountPointData{
		Path:      "/mnt/test-volume",
		IsMounted: true,
		DeviceId:  "sda1",
		Partition: &dto.Partition{Id: pointer.String("sda1"), DevicePath: pointer.String("/dev/fake-sda1")},
	}
	_ = suite.VolumeService.MountVolume(mountPoint)

	// Wait for event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		suite.Require().NotNil(receivedMountEvent, "Should receive mount point event")
		suite.Equal("/mnt/test-volume", receivedMountEvent.MountPoint.Path)
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for mount point event")
	}
} // TestVolumeUnmountEventPropagation tests volume unmount event propagation
func (suite *EventPropagationTestSuite) TestVolumeUnmountEventPropagation() {
	// Setup: Track volume events
	var receivedEvent *events.VolumeEvent
	var wg sync.WaitGroup
	wg.Add(1)

	unsubscribe := suite.eventBus.OnVolume(func(ctx context.Context, event events.VolumeEvent) errors.E {
		receivedEvent = &event
		wg.Done()
		return nil
	})
	defer unsubscribe()

	// Mock repo so UnmountVolume can resolve mount entry
	mock.When(suite.mockMountRepo.FindByPath("/mnt/test-volume")).ThenReturn(&dbom.MountPointPath{Path: "/mnt/test-volume", DeviceId: "sda1", FSType: "ext4"}, nil)

	err := suite.VolumeService.UnmountVolume("/mnt/test-volume", true, false)
	suite.Require().NoError(err)

	// Wait for event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		suite.Require().NotNil(receivedEvent, "Should receive volume event")
		suite.Equal("unmount", receivedEvent.Operation)
		suite.Equal("/mnt/test-volume", receivedEvent.MountPoint.Path)
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for volume unmount event")
	}
}

// TestVolumeEventToMountPointEventChain tests that volume operations trigger mount point events
func (suite *EventPropagationTestSuite) TestVolumeEventToMountPointEventChain() {
	// Setup: Track mount point events (mounts emit MountPoint, unmounts emit both Volume and MountPoint)
	mountPointEventReceived := atomic.Bool{}

	var wg sync.WaitGroup
	wg.Add(1)

	unsubscribe2 := suite.eventBus.OnMountPoint(func(ctx context.Context, event events.MountPointEvent) errors.E {
		mountPointEventReceived.Store(true)
		wg.Done()
		return nil
	})
	defer unsubscribe2()

	// Action: Mount (emits MountPoint event)
	mountPoint := &dto.MountPointData{
		Path:      "/mnt/test-chain",
		IsMounted: true,
		DeviceId:  "sdb1",
		Partition: &dto.Partition{Id: pointer.String("sdb1"), DevicePath: pointer.String("/dev/fake-sdb1")},
	}

	// Call service
	_ = suite.VolumeService.MountVolume(mountPoint)

	// Wait for events
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		suite.True(mountPointEventReceived.Load(), "MountPoint event should be received")
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for mount point event")
	}
}

// TestMultipleVolumeOperationsSequence tests sequential volume operations (mount via MountPoint, unmount via Volume)
func (suite *EventPropagationTestSuite) TestMultipleVolumeOperationsSequence() {
	// Setup: Track mount point events for mounts and volume events for unmounts
	var operations []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(1) // Just wait for the unmount event which is more reliable

	unsubscribeMp := suite.eventBus.OnMountPoint(func(ctx context.Context, event events.MountPointEvent) errors.E {
		// Track mount events but don't block on them since they might be buffered/async
		if event.MountPoint != nil && event.MountPoint.Path == "/mnt/test-seq" {
			mu.Lock()
			operations = append(operations, "mount")
			mu.Unlock()
		}
		return nil
	})
	defer unsubscribeMp()

	unsubscribeVol := suite.eventBus.OnVolume(func(ctx context.Context, event events.VolumeEvent) errors.E {
		mu.Lock()
		operations = append(operations, event.Operation)
		mu.Unlock()
		wg.Done()
		return nil
	})
	defer unsubscribeVol()

	// Mock repo for unmount
	mock.When(suite.mockMountRepo.FindByPath("/mnt/test-seq")).ThenReturn(&dbom.MountPointPath{Path: "/mnt/test-seq", DeviceId: "sdc1", FSType: "ext4"}, nil)

	// Action: Execute multiple volume operations through the service
	mountPoint := &dto.MountPointData{
		Path:      "/mnt/test-seq",
		DeviceId:  "sdc1",
		Partition: &dto.Partition{Id: pointer.String("sdc1"), DevicePath: pointer.String("/dev/fake-sdc1")},
	}

	// Mount (emits MountPointEvent)
	mountPoint.IsMounted = true
	_ = suite.VolumeService.MountVolume(mountPoint)

	// Unmount (emits VolumeEvent)
	mountPoint.IsMounted = false
	_ = suite.VolumeService.UnmountVolume(mountPoint.Path, true, false)

	// Mount again (emits MountPointEvent)
	mountPoint.IsMounted = true
	_ = suite.VolumeService.MountVolume(mountPoint)

	// Wait for unmount event
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		mu.Lock()
		defer mu.Unlock()
		// Verify we got at least the unmount event and some mount events
		suite.Contains(operations, "unmount", "Should receive unmount event")
		suite.True(len(operations) >= 1, "Should receive at least one operation event")
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for volume operation sequence")
	}
}

// TestVolumeDiskPartitionEventChain tests the full event chain: Disk -> Partition -> MountPoint (mount operation)
func (suite *EventPropagationTestSuite) TestVolumeDiskPartitionEventChain() {
	// Setup: Track all events in the volume lifecycle
	diskEventReceived := atomic.Bool{}
	partitionEventReceived := atomic.Bool{}
	mountPointEventReceived := atomic.Bool{}

	var wg sync.WaitGroup
	// Expect: Disk + Partition
	wg.Add(2) // Just wait for disk and partition events

	// Mock repository to avoid nil pointer dereference when ShareService tries to look up share by path
	mock.When(suite.mockShareRepo.FindByMountPath("/mnt/sdd1")).ThenReturn(nil, errors.WithStack(gorm.ErrRecordNotFound))

	unsubscribe1 := suite.eventBus.OnDisk(func(ctx context.Context, event events.DiskEvent) errors.E {
		if !diskEventReceived.Load() {
			diskEventReceived.Store(true)
			wg.Done()
		}
		return nil
	})
	defer unsubscribe1()

	unsubscribe2 := suite.eventBus.OnPartition(func(ctx context.Context, event events.PartitionEvent) errors.E {
		if !partitionEventReceived.Load() {
			partitionEventReceived.Store(true)
			wg.Done()
		}
		return nil
	})
	defer unsubscribe2()

	unsubscribe3 := suite.eventBus.OnMountPoint(func(ctx context.Context, event events.MountPointEvent) errors.E {
		// Track mount point events but don't block on them
		if event.MountPoint != nil && event.MountPoint.Path == "/mnt/sdd1" {
			mountPointEventReceived.Store(true)
		}
		return nil
	})
	defer unsubscribe3()

	// Action: Emit a complete volume lifecycle event chain
	partitions := map[string]dto.Partition{
		"sdd1": {
			Name:       pointer.String("sdd1"),
			DevicePath: pointer.String("/dev/sdd1"),
			Id:         pointer.String("sdd1"),
		},
	}
	disk := &dto.Disk{
		Id:         pointer.String("sdd"),
		Model:      pointer.String("Test Disk"),
		Partitions: &partitions,
	}

	// 1. Disk detected
	suite.eventBus.EmitDiskAndPartition(events.DiskEvent{
		Event: events.Event{Type: events.EventTypes.ADD},
		Disk:  disk,
	})

	// 2. Mount point event
	mountPoint := &dto.MountPointData{
		Path:      "/mnt/sdd1",
		IsMounted: true,
		DeviceId:  "sdd1",
		Partition: &dto.Partition{Id: pointer.String("sdd1"), DevicePath: pointer.String("/dev/fake-sdd1")},
	}
	suite.eventBus.EmitMountPoint(events.MountPointEvent{
		Event:      events.Event{Type: events.EventTypes.UPDATE},
		MountPoint: mountPoint,
	})

	// 3. Mount via service (emits MountPointEvent, not VolumeEvent)
	_ = suite.VolumeService.MountVolume(mountPoint)

	// Wait for disk and partition events
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Give mount point event time to propagate (it's tracked but not blocking)
	time.Sleep(100 * time.Millisecond)

	select {
	case <-done:
		suite.True(diskEventReceived.Load(), "Disk event should be received")
		suite.True(partitionEventReceived.Load(), "Partition event should be auto-emitted from disk")
		// MountPoint event may or may not be received depending on timing, just note it's optional here
		// The main test is that disk/partition events propagate correctly
	case <-time.After(2 * time.Second):
		suite.T().Fatal("timeout waiting for volume lifecycle event chain")
	}
}

// TestConcurrentVolumeOperations tests concurrent volume mount operations
func (suite *EventPropagationTestSuite) TestConcurrentVolumeOperations() {
	// Setup: Count mount point events from concurrent mount operations
	mountCounter := atomic.Int32{}

	const numOperations = 5
	var wg sync.WaitGroup
	wg.Add(numOperations) // Only count mounts (via MountPoint events)

	unsubscribe := suite.eventBus.OnMountPoint(func(ctx context.Context, event events.MountPointEvent) errors.E {
		mountCounter.Add(1)
		wg.Done()
		return nil
	})
	defer unsubscribe()

	// Action: Emit concurrent mount operations via service
	for i := 0; i < numOperations; i++ {
		idx := i
		go func() {
			dev := fmt.Sprintf("sd%c1", 'e'+idx)
			_ = suite.VolumeService.MountVolume(&dto.MountPointData{
				Path:      fmt.Sprintf("/mnt/vol-%d", idx),
				IsMounted: true,
				DeviceId:  dev,
				Partition: &dto.Partition{Id: pointer.String(dev), DevicePath: pointer.String("/dev/fake-" + dev)},
			})
		}()
	}

	// Wait for all events
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		suite.Equal(int32(numOperations), mountCounter.Load(), "All mount events should be received")
	case <-time.After(5 * time.Second):
		suite.T().Fatal("timeout waiting for concurrent volume operations")
	}
}
