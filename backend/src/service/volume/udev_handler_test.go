// Package volume_test: handler suites (uevent dispatch, add/remove paths,
// mount point persistence with automount, event ordering) with fakes.
package volume_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/service/volume"
	"github.com/pilebones/go-udev/netlink"
	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

type fakeHandlerHardware struct {
	mu          sync.Mutex
	invalidates int
}

func (f *fakeHandlerHardware) InvalidateHardwareInfo() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.invalidates++
}

func (f *fakeHandlerHardware) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.invalidates
}

// UdevHandlerTestSuite exercises uevent dispatch and the add/remove paths:
// ADD of an unknown partition refreshes and schedules a retry, REMOVE of an
// untracked partition stays silent, and disk removal refreshes.
type UdevHandlerTestSuite struct {
	volumeTestFixture
	handler   *volume.UdevHandler
	hardware  *fakeHandlerHardware
	refreshes int
	refreshMu sync.Mutex
	resets    int
}

func TestUdevHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(UdevHandlerTestSuite))
}

func (s *UdevHandlerTestSuite) SetupTest() {
	s.volumeTestFixture.SetupTest()
	s.hardware = &fakeHandlerHardware{}
	s.refreshes = 0
	s.resets = 0
	orchestrator := volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx:        s.ctx,
		Disks:      s.disks,
		Filesystem: &fakeOrchestratorFS{},
		Mounter:    &fakeOrchestratorMounter{},
		EventBus:   s.eventBus,
		Repo:       s.mountRepoIf,
		Volumes: func() ([]*dto.Disk, errors.E) {
			return s.disks.All(), nil
		},
	})
	s.handler = volume.NewUdevHandler(volume.HandlerParams{
		Ctx:          s.ctx,
		Disks:        s.disks,
		Hardware:     s.hardware,
		EventBus:     s.eventBus,
		Repo:         s.repo,
		Orchestrator: orchestrator,
		Filesystem:   &fakeOrchestratorFS{},
		Refresh: func() errors.E {
			s.refreshMu.Lock()
			defer s.refreshMu.Unlock()
			s.refreshes++
			return nil
		},
		ResetRecheck: func() {
			s.refreshMu.Lock()
			defer s.refreshMu.Unlock()
			s.resets++
		},
		ProcfsMounts: func() ([]*procfs.MountInfo, error) { return nil, nil },
	})
	s.handler.SetHardware(s.hardware)
	s.handler.SetContext(s.ctx)
}

func (s *UdevHandlerTestSuite) TearDownTest() {
	s.volumeTestFixture.TearDownTest()
}

func (s *UdevHandlerTestSuite) refreshCount() int {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	return s.refreshes
}

func (s *UdevHandlerTestSuite) TestProcessUdevEvent_NonBlockIgnored() {
	observed := 0
	s.handler.SetProbe(func(netlink.UEvent) { observed++ })
	s.handler.ProcessUdevEvent(netlink.UEvent{
		Action: netlink.ADD,
		Env:    map[string]string{"SUBSYSTEM": "usb", "DEVTYPE": "disk", "DEVNAME": "/dev/sda"},
	})
	s.Equal(1, observed, "probe runs before dispatch")
	s.Equal(0, s.refreshCount(), "non-block events must not refresh")
	s.Equal(0, s.hardware.count())
}

func (s *UdevHandlerTestSuite) TestProcessUdevEvent_PartitionAddUnknown_RefreshesAndRetries() {
	s.handler.ProcessUdevEvent(netlink.UEvent{
		Action: netlink.ADD,
		Env:    map[string]string{"SUBSYSTEM": "block", "DEVTYPE": "partition", "DEVNAME": "/dev/sdz9"},
	})
	s.Equal(1, s.refreshCount(), "unknown partition ADD must refresh immediately")
	s.Equal(1, s.hardware.count())

	deadline := time.Now().Add(3 * time.Second)
	for {
		if s.refreshCount() >= 2 {
			break
		}
		if time.Now().After(deadline) {
			s.Fail("delayed retry must refresh a second time")
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (s *UdevHandlerTestSuite) TestProcessUdevEvent_PartitionRemoveUntracked_Silent() {
	s.handler.ProcessUdevEvent(netlink.UEvent{
		Action: netlink.REMOVE,
		Env:    map[string]string{"SUBSYSTEM": "block", "DEVTYPE": "partition", "DEVNAME": "/dev/sdz9"},
	})
	s.Equal(0, s.refreshCount(), "untracked REMOVE must not refresh")
	s.Equal(0, s.hardware.count())
}

func (s *UdevHandlerTestSuite) TestProcessUdevEvent_DiskRemove_Refreshes() {
	s.handler.ProcessUdevEvent(netlink.UEvent{
		Action: netlink.REMOVE,
		Env:    map[string]string{"SUBSYSTEM": "block", "DEVTYPE": "disk", "DEVNAME": "/dev/sdz"},
	})
	s.Equal(1, s.refreshCount())
	s.Equal(1, s.hardware.count())
}

func (s *UdevHandlerTestSuite) TestHandlePartitionEvent_SingleMode_UsesProcfsHook() {
	diskID := "disk-handler-2"
	partID := "part-handler-2"
	devPath := "/dev/sdz2"
	diskIDCopy, partIDCopy, devPathCopy := diskID, partID, devPath
	parts := map[string]dto.Partition{partID: {Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devPathCopy}}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))

	parsed := 0
	s.handler.SetProcfsMounts(func() ([]*procfs.MountInfo, error) {
		parsed++
		return []*procfs.MountInfo{}, nil
	})

	part := parts[partID]
	s.Require().NoError(s.handler.HandlePartitionEvent(s.ctx, events.PartitionEvent{
		Event:     events.Event{Type: events.EventTypes.ADD},
		Disk:      &disk,
		Partition: &part,
	}))
	s.Equal(1, parsed, "missing MountInfos must be parsed once via the procfs hook")
}

func (s *UdevHandlerTestSuite) TestConsumeUdevChannels_ClosedQueue() {
	queue := make(chan netlink.UEvent, 1)
	errCh := make(chan error, 1)
	close(queue)

	done := make(chan error, 1)
	go func() { done <- s.handler.ConsumeUdevChannels(queue, errCh) }()

	select {
	case err := <-done:
		s.Require().Error(err)
		s.ErrorIs(err, volume.ErrUdevQueueClosed)
	case <-time.After(time.Second):
		s.Fail("closed queue must surface an error")
	}
}

func (s *UdevHandlerTestSuite) TestHandleMountPointEvent_PersistsWithoutAutomount() {
	md := &dto.MountPointData{
		Path: "/mnt/handler-persist", Root: "/", DeviceId: "part-handler-1",
		Type: "ADDON", IsMounted: true,
		Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}
	s.Require().NoError(s.handler.HandleMountPointEvent(s.ctx, events.MountPointEvent{
		Type:       events.EventTypes.ADD,
		MountPoint: md,
	}))

	got, found, err := s.repo.LoadByPath("/mnt/handler-persist", "/")
	s.Require().NoError(err)
	s.Require().True(found)
	s.Equal("part-handler-1", got.DeviceId)
}

func (s *UdevHandlerTestSuite) TestHandleMountPointEvent_StartupAutomountMounts() {
	tmpDir := s.T().TempDir()
	mountPath := filepath.Join(tmpDir, "mnt", "share")
	s.Require().NoError(os.MkdirAll(mountPath, 0o755))
	devicePath := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(devicePath, []byte("test"), 0o600))

	diskID := "disk-handler-1"
	partID := "part-handler-1"
	startup := true
	mounter := &fakeOrchestratorMounter{}
	orchestrator := volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx:        s.ctx,
		Disks:      s.disks,
		Filesystem: &fakeOrchestratorFS{},
		Mounter:    mounter,
		EventBus:   s.eventBus,
		Repo:       s.mountRepoIf,
		Volumes: func() ([]*dto.Disk, errors.E) {
			return s.disks.All(), nil
		},
	})
	handler := volume.NewUdevHandler(volume.HandlerParams{
		Ctx:          s.ctx,
		Disks:        s.disks,
		EventBus:     s.eventBus,
		Repo:         s.repo,
		Orchestrator: orchestrator,
		Filesystem:   &fakeOrchestratorFS{},
		Refresh:      func() errors.E { return nil },
		ProcfsMounts: func() ([]*procfs.MountInfo, error) { return nil, nil },
	})

	diskIDCopy, partIDCopy := diskID, partID
	parts := map[string]dto.Partition{partID: {Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devicePath}}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))

	md := &dto.MountPointData{
		Path: mountPath, Root: "/", DeviceId: partID,
		Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
		IsToMountAtStartup: &startup, IsMounted: false,
		Partition: &dto.Partition{Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devicePath},
	}
	s.Require().NoError(handler.HandleMountPointEvent(s.ctx, events.MountPointEvent{
		Type:       events.EventTypes.ADD,
		MountPoint: md,
	}))
	s.Equal(1, mounter.mountCalls, "startup-marked unmounted mount point must automount")
}

func TestMatchPartitionWithDevName(t *testing.T) {
	devPath := "/dev/sda1"
	legacyPath := "/dev/disk/by-id/ata-test"
	legacyName := "sda1"
	id := "part-xyz"
	part := &dto.Partition{
		Id: &id, DevicePath: &devPath, LegacyDevicePath: &legacyPath, LegacyDeviceName: &legacyName,
	}
	for _, dev := range []string{"sda1", "/dev/sda1", "ata-test", "/dev/disk/by-id/ata-test", "part-xyz"} {
		if !volume.MatchPartitionWithDevName(part, dev) {
			t.Errorf("MatchPartitionWithDevName(%q) = false, want true", dev)
		}
	}
	if volume.MatchPartitionWithDevName(part, "sdb1") {
		t.Error("MatchPartitionWithDevName(sdb1) = true, want false")
	}
	if volume.MatchPartitionWithDevName(nil, "sda1") {
		t.Error("nil partition must not match")
	}
}

func (s *UdevHandlerTestSuite) seedHandlerDisk(diskID, partID, devName, devicePath string) {
	diskIDCopy, partIDCopy := diskID, partID
	devNameCopy := devName
	devPathCopy := devicePath
	parts := map[string]dto.Partition{partID: {
		Id: &partIDCopy, DiskId: &diskIDCopy,
		LegacyDeviceName: &devNameCopy, LegacyDevicePath: &devPathCopy, DevicePath: &devPathCopy,
	}}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))
}

func (s *UdevHandlerTestSuite) TestFindPartitionByDevName_FoundAndMissing() {
	s.seedHandlerDisk("disk-h-1", "part-h-1", "sdz1", "/dev/sdz1")

	part, diskID, found := s.handler.FindPartitionByDevName("sdz1")
	s.True(found)
	s.Equal("disk-h-1", diskID)
	s.Require().NotNil(part)
	s.Equal("part-h-1", *part.Id)

	_, _, found = s.handler.FindPartitionByDevName("sdz9")
	s.False(found)
}

func (s *UdevHandlerTestSuite) TestHandlePartitionUdevAddEvent_Handled() {
	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))
	mountPath := filepath.Join(tmpDir, "mnt", "share")
	startup := true

	diskID, partID, devName := "disk-h-2", "part-h-2", "sdz2"
	diskIDCopy, partIDCopy := diskID, partID
	devNameCopy := devName
	devFileCopy := deviceFile
	mps := map[string]dto.MountPointData{mountPath: {
		Path: mountPath, Root: "/", DeviceId: partID,
		Flags: &dto.MountFlags{}, IsToMountAtStartup: &startup, IsMounted: false,
	}}
	parts := map[string]dto.Partition{partID: {
		Id: &partIDCopy, DiskId: &diskIDCopy,
		LegacyDeviceName: &devNameCopy, DevicePath: &devFileCopy, MountPointData: &mps,
	}}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))

	// Point the suite handler at a succeeding mounter.
	mounter := &fakeOrchestratorMounter{}
	orchestrator := volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: &fakeOrchestratorFS{},
		Mounter: mounter, EventBus: s.eventBus, Repo: s.mountRepoIf,
		Volumes: func() ([]*dto.Disk, errors.E) { return s.disks.All(), nil },
	})
	handler := volume.NewUdevHandler(volume.HandlerParams{
		Ctx: s.ctx, Disks: s.disks, EventBus: s.eventBus,
		Repo: s.repo, Orchestrator: orchestrator,
		Refresh: func() errors.E { return nil },
	})

	s.True(handler.HandlePartitionUdevAddEvent(devName))
	s.Equal(1, mounter.mountCalls)
}

func (s *UdevHandlerTestSuite) TestHandlePartitionUdevAddEvent_AlreadyMountedClearsRetry() {
	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))

	devName := "sdz3"
	mountPath := filepath.Join(tmpDir, "mnt", "share")
	startup := true

	diskID, partID := "disk-h-3", "part-h-3"
	diskIDCopy, partIDCopy := diskID, partID
	devNameCopy, devFileCopy := devName, deviceFile
	mps := map[string]dto.MountPointData{mountPath: {
		Path: mountPath, Root: "/", DeviceId: partID,
		Flags: &dto.MountFlags{}, IsToMountAtStartup: &startup, IsMounted: false,
	}}
	parts := map[string]dto.Partition{partID: {
		Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devFileCopy,
		LegacyDeviceName: &devNameCopy, MountPointData: &mps,
	}}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))

	mounter := &fakeOrchestratorMounter{
		mountErr: errors.WithDetails(dto.ErrorAlreadyMounted, "Message", "already"),
	}
	orchestrator := volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: &fakeOrchestratorFS{},
		Mounter: mounter, EventBus: s.eventBus, Repo: s.mountRepoIf,
		Volumes: func() ([]*dto.Disk, errors.E) { return s.disks.All(), nil },
	})
	// Seed near-cap retry entries with an already-expired backoff so the
	// guard allows one attempt; the already-mounted success must then clear
	// the state (observable: a fresh failure afterwards counts from 1).
	orchestrator.SetAutomountTuning(-time.Second, 5)
	for i := 0; i < 4; i++ {
		orchestrator.RecordAutomountFailure(mountPath)
	}
	handler := volume.NewUdevHandler(volume.HandlerParams{
		Ctx: s.ctx, Disks: s.disks, EventBus: s.eventBus,
		Repo: s.repo, Orchestrator: orchestrator,
		Refresh: func() errors.E { return nil },
	})

	s.True(handler.HandlePartitionUdevAddEvent(devName))
	s.Equal(1, mounter.mountCalls)
	orchestrator.RecordAutomountFailure(mountPath)
	allowed, exhausted := orchestrator.AllowAutomountAttempt(mountPath)
	s.True(allowed, "already-mounted must clear the retry state")
	s.False(exhausted)
}

func (s *UdevHandlerTestSuite) TestHandlePartitionUdevAddEvent_GuardBoundsRetries() {
	devName := "sdz3b"
	mountPath := "/mnt/h-guard"
	startup := true

	diskID, partID := "disk-h-3b", "part-h-3b"
	diskIDCopy, partIDCopy := diskID, partID
	devNameCopy := devName
	mps := map[string]dto.MountPointData{mountPath: {
		Path: mountPath, Root: "/", DeviceId: partID,
		Flags: &dto.MountFlags{}, IsToMountAtStartup: &startup, IsMounted: false,
	}}
	parts := map[string]dto.Partition{partID: {
		Id: &partIDCopy, DiskId: &diskIDCopy,
		LegacyDeviceName: &devNameCopy, MountPointData: &mps,
	}}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))

	mounter := &fakeOrchestratorMounter{
		mountErr: errors.WithDetails(dto.ErrorMountFail, "Detail", "boom"),
	}
	orchestrator := volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: &fakeOrchestratorFS{},
		Mounter: mounter, EventBus: s.eventBus, Repo: s.mountRepoIf,
		Volumes: func() ([]*dto.Disk, errors.E) { return s.disks.All(), nil },
	})
	orchestrator.SetAutomountTuning(0, 1)
	handler := volume.NewUdevHandler(volume.HandlerParams{
		Ctx: s.ctx, Disks: s.disks, EventBus: s.eventBus,
		Repo: s.repo, Orchestrator: orchestrator,
		Refresh: func() errors.E { return nil },
	})

	// No DevicePath on the partition, so MountVolume fails device-not-found
	// and arms the guard; the second add hits the exhausted branch.
	s.False(handler.HandlePartitionUdevAddEvent(devName))
	s.False(handler.HandlePartitionUdevAddEvent(devName))
	allowed, exhausted := orchestrator.AllowAutomountAttempt(mountPath)
	s.False(allowed)
	s.True(exhausted)
}

func (s *UdevHandlerTestSuite) TestHandlePartitionUdevRemoveEvent_Found_UnmountsAndRefreshes() {
	diskID, partID := "disk-h-4", "part-h-4"
	s.seedHandlerDisk(diskID, partID, "sdz4", "/dev/sdz4")
	s.Require().NoError(s.disks.AddOrUpdateMountPoint(diskID, partID, dto.MountPointData{
		Path: "/mnt/h-remove", DeviceId: partID, IsMounted: true,
		Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}))

	mounter := &fakeOrchestratorMounter{}
	orchestrator := volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: &fakeOrchestratorFS{},
		Mounter: mounter, EventBus: s.eventBus, Repo: s.mountRepoIf,
		Volumes: func() ([]*dto.Disk, errors.E) { return s.disks.All(), nil },
	})
	refreshed := 0
	handler := volume.NewUdevHandler(volume.HandlerParams{
		Ctx: s.ctx, Disks: s.disks, Hardware: s.hardware, EventBus: s.eventBus,
		Repo: s.repo, Orchestrator: orchestrator,
		Refresh: func() errors.E { refreshed++; return nil },
	})

	handler.HandlePartitionUdevRemoveEvent("sdz4")
	s.Equal(1, mounter.unmountCalls, "mounted path must be force-unmounted")
	s.Equal(1, refreshed)
	s.Equal(1, s.hardware.count())
	_, found := s.disks.GetPartition(diskID, partID)
	s.False(found, "partition must be evicted from the cache")
}

func (s *UdevHandlerTestSuite) TestHandleFilesystemTaskEvent_NonFormatIgnored() {
	s.Require().NoError(s.handler.HandleFilesystemTaskEvent(s.ctx, events.FilesystemTaskEvent{}))
	s.Require().NoError(s.handler.HandleFilesystemTaskEvent(s.ctx, events.FilesystemTaskEvent{
		Task: &dto.FilesystemTask{Operation: "check", Status: "success", Device: "/dev/sdz1"},
	}))
	s.Equal(0, s.refreshCount())
}

func (s *UdevHandlerTestSuite) TestHandlePartitionEvent_ProcfsErrors() {
	diskIDCopy, partIDCopy := "disk-h-9", "part-h-9"
	parts := map[string]dto.Partition{partIDCopy: {Id: &partIDCopy, DiskId: &diskIDCopy}}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))
	part := parts[partIDCopy]

	s.handler.SetProcfsMounts(func() ([]*procfs.MountInfo, error) {
		return nil, errors.New("procfs unavailable")
	})

	// Batch mode with no carried snapshot must surface the parse error.
	err := s.handler.HandlePartitionEvent(s.ctx, events.PartitionEvent{
		Event:      events.Event{Type: events.EventTypes.ADD},
		Disk:       &disk,
		Partitions: []*dto.Partition{&part},
	})
	s.Require().Error(err)

	// Single mode likewise.
	err = s.handler.HandlePartitionEvent(s.ctx, events.PartitionEvent{
		Event:     events.Event{Type: events.EventTypes.ADD},
		Disk:      &disk,
		Partition: &part,
	})
	s.Require().Error(err)
}

func (s *UdevHandlerTestSuite) TestConsumeUdevChannels_ContextCancelled() {
	ctx, cancel := context.WithCancel(context.Background())
	s.handler.SetContext(ctx)
	queue := make(chan netlink.UEvent)
	errCh := make(chan error)

	done := make(chan error, 1)
	go func() { done <- s.handler.ConsumeUdevChannels(queue, errCh) }()
	cancel()

	select {
	case err := <-done:
		s.NoError(err, "cancellation is a clean shutdown")
	case <-time.After(2 * time.Second):
		s.Fail("cancelled context must stop the loop")
	}
}

func (s *UdevHandlerTestSuite) TestHandleMountPointEvent_AlreadyMountedClearsRetry() {
	ha := &fakeNotifier{}
	mounter := &fakeOrchestratorMounter{
		mountErr: errors.WithDetails(dto.ErrorAlreadyMounted, "Message", "already"),
	}
	orchestrator := volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: &fakeOrchestratorFS{},
		Mounter: mounter, HA: ha, EventBus: s.eventBus, Repo: s.mountRepoIf,
		Volumes: func() ([]*dto.Disk, errors.E) { return s.disks.All(), nil },
	})
	handler := volume.NewUdevHandler(volume.HandlerParams{
		Ctx: s.ctx, Disks: s.disks, EventBus: s.eventBus,
		Repo: s.repo, Orchestrator: orchestrator, Filesystem: &fakeOrchestratorFS{},
		Refresh: func() errors.E { return nil },
	})

	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))
	diskID, partID := "disk-h-10", "part-h-10"
	diskIDCopy, partIDCopy, devCopy := diskID, partID, deviceFile
	parts := map[string]dto.Partition{partID: {Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devCopy}}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))

	orchestrator.SetAutomountTuning(-time.Second, 5)
	for i := 0; i < 4; i++ {
		orchestrator.RecordAutomountFailure(filepath.Join(tmpDir, "mnt", "share"))
	}
	startup := true
	md := &dto.MountPointData{
		Path: filepath.Join(tmpDir, "mnt", "share"), Root: "/", DeviceId: partID,
		Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
		IsToMountAtStartup: &startup, IsMounted: false,
		Partition: &dto.Partition{Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devCopy},
	}
	s.Require().NoError(handler.HandleMountPointEvent(s.ctx, events.MountPointEvent{
		Type:       events.EventTypes.ADD,
		MountPoint: md,
	}))
	s.Equal(1, mounter.mountCalls)
	orchestrator.RecordAutomountFailure(md.Path)
	allowed, exhausted := orchestrator.AllowAutomountAttempt(md.Path)
	s.True(allowed, "already-mounted must clear the retry state")
	s.False(exhausted)
}

func (s *UdevHandlerTestSuite) TestHandleMountPointEvent_ExhaustedGuardSkipsMount() {
	mounter := &fakeOrchestratorMounter{
		mountErr: errors.WithDetails(dto.ErrorMountFail, "Detail", "boom"),
	}
	orchestrator := volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: &fakeOrchestratorFS{},
		Mounter: mounter, EventBus: s.eventBus, Repo: s.mountRepoIf,
		Volumes: func() ([]*dto.Disk, errors.E) { return s.disks.All(), nil },
	})
	orchestrator.SetAutomountTuning(0, 1)
	handler := volume.NewUdevHandler(volume.HandlerParams{
		Ctx: s.ctx, Disks: s.disks, EventBus: s.eventBus,
		Repo: s.repo, Orchestrator: orchestrator, Filesystem: &fakeOrchestratorFS{},
		Refresh: func() errors.E { return nil },
	})

	md := &dto.MountPointData{
		Path: "/mnt/h-exhausted", Root: "/", DeviceId: "part-h-11",
		Type: "ADDON", Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
		IsToMountAtStartup: new(true), IsMounted: false,
	}
	// First event arms the guard (persist succeeds, mount fails before the
	// mounter: unknown device), the second hits the exhausted branch.
	s.Require().NoError(handler.HandleMountPointEvent(s.ctx, events.MountPointEvent{
		Type: events.EventTypes.ADD, MountPoint: md,
	}))
	s.Require().NoError(handler.HandleMountPointEvent(s.ctx, events.MountPointEvent{
		Type: events.EventTypes.UPDATE, MountPoint: md,
	}))
	s.Equal(0, mounter.mountCalls)
}

func (s *UdevHandlerTestSuite) TestHandleFilesystemTaskEvent_FormatEmptyLabel_ReadsLiveLabel() {
	diskID, partID := "disk-h-12", "part-h-12"
	fsType := "ext4"
	diskIDCopy, partIDCopy, fsCopy := diskID, partID, fsType
	devPath := "/dev/sdz12"
	devCopy := devPath
	parts := map[string]dto.Partition{partID: {
		Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devCopy, FsType: &fsCopy,
	}}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))

	fs := &fakeOrchestratorFS{label: "live-label"}
	handler := volume.NewUdevHandler(volume.HandlerParams{
		Ctx: s.ctx, Disks: s.disks, Hardware: s.hardware, EventBus: s.eventBus,
		Repo: s.repo,
		Orchestrator: volume.NewMountOrchestrator(volume.OrchestratorParams{
			Ctx: s.ctx, Disks: s.disks, Filesystem: fs,
			Mounter: &fakeOrchestratorMounter{}, EventBus: s.eventBus, Repo: s.mountRepoIf,
			Volumes: func() ([]*dto.Disk, errors.E) { return s.disks.All(), nil },
		}),
		Filesystem: fs,
		Refresh:    func() errors.E { return nil },
	})

	s.Require().NoError(handler.HandleFilesystemTaskEvent(s.ctx, events.FilesystemTaskEvent{
		Task: &dto.FilesystemTask{
			Operation: "format", Status: "success",
			Device: devPath, FilesystemType: fsType, Label: "",
		},
	}))
	part, ok := s.disks.GetPartition(diskID, partID)
	s.Require().True(ok)
	s.Require().NotNil(part.Name)
	s.Equal("live-label", *part.Name, "empty format label must fall back to the live read")
}

func (s *UdevHandlerTestSuite) TestHandleFilesystemTaskEvent_FormatSuccess_PatchesAndBroadcasts() {
	diskID, partID := "disk-h-5", "part-h-5"
	s.seedHandlerDisk(diskID, partID, "sdz5", "/dev/sdz5")

	var emitted []events.DiskEvent
	unsub := s.eventBus.OnDisk(func(_ context.Context, e events.DiskEvent) errors.E {
		emitted = append(emitted, e)
		return nil
	})
	defer unsub()

	s.Require().NoError(s.handler.HandleFilesystemTaskEvent(s.ctx, events.FilesystemTaskEvent{
		Task: &dto.FilesystemTask{
			Operation: "format", Status: "success",
			Device: "/dev/sdz5", FilesystemType: "ext4", Label: "formatted-label",
		},
	}))
	s.Equal(1, s.refreshCount())
	s.Require().NotEmpty(emitted, "format refresh must broadcast the patched disk")
	s.Equal(events.EventTypes.UPDATE, emitted[len(emitted)-1].Type)

	part, ok := s.disks.GetPartition(diskID, partID)
	s.Require().True(ok)
	s.Require().NotNil(part.Name)
	s.Equal("formatted-label", *part.Name)
}

func (s *UdevHandlerTestSuite) TestHandleMountPointEvent_Failure_NotifiesAndBounds() {
	ha := &fakeNotifier{}
	mounter := &fakeOrchestratorMounter{
		mountErr: errors.WithDetails(dto.ErrorMountFail, "Detail", "boom"),
	}
	orchestrator := volume.NewMountOrchestrator(volume.OrchestratorParams{
		Ctx: s.ctx, Disks: s.disks, Filesystem: &fakeOrchestratorFS{},
		Mounter: mounter, HA: ha, EventBus: s.eventBus, Repo: s.mountRepoIf,
		Volumes: func() ([]*dto.Disk, errors.E) { return s.disks.All(), nil },
	})
	handler := volume.NewUdevHandler(volume.HandlerParams{
		Ctx: s.ctx, Disks: s.disks, EventBus: s.eventBus,
		Repo: s.repo, Orchestrator: orchestrator, Filesystem: &fakeOrchestratorFS{},
		Refresh: func() errors.E { return nil },
	})

	tmpDir := s.T().TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	s.Require().NoError(os.WriteFile(deviceFile, []byte("test"), 0o600))
	diskID, partID := "disk-h-6", "part-h-6"
	diskIDCopy, partIDCopy, devCopy := diskID, partID, deviceFile
	parts := map[string]dto.Partition{partID: {Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devCopy}}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))

	startup := true
	md := &dto.MountPointData{
		Path: filepath.Join(tmpDir, "mnt", "share"), Root: "/", DeviceId: partID,
		Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
		IsToMountAtStartup: &startup, IsMounted: false,
		Partition: &dto.Partition{Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devCopy},
	}
	s.Require().NoError(handler.HandleMountPointEvent(s.ctx, events.MountPointEvent{
		Type:       events.EventTypes.ADD,
		MountPoint: md,
	}))
	s.Equal(1, mounter.mountCalls)
	s.NotEmpty(ha.created, "failed automount must raise a notification")

	allowed, _ := orchestrator.AllowAutomountAttempt(md.Path)
	s.False(allowed, "failure must arm the backoff guard")
}

func (s *UdevHandlerTestSuite) TestConsumeUdevChannels_ErrorPaths() {
	// Non-nil monitor error is logged and consumed; the loop continues until
	// the queue closes.
	queue := make(chan netlink.UEvent, 1)
	errCh := make(chan error, 2)
	errCh <- errors.New("monitor hiccup")
	close(queue)

	done := make(chan error, 1)
	go func() { done <- s.handler.ConsumeUdevChannels(queue, errCh) }()
	select {
	case err := <-done:
		s.Require().Error(err)
		s.ErrorIs(err, volume.ErrUdevQueueClosed)
	case <-time.After(2 * time.Second):
		s.Fail("loop must exit on queue close")
	}

	// Closed error channel surfaces its own sentinel.
	queue2 := make(chan netlink.UEvent, 1)
	errCh2 := make(chan error, 1)
	close(errCh2)
	done2 := make(chan error, 1)
	go func() { done2 <- s.handler.ConsumeUdevChannels(queue2, errCh2) }()
	select {
	case err := <-done2:
		s.Require().Error(err)
		s.ErrorIs(err, volume.ErrUdevErrorChanClosed)
	case <-time.After(2 * time.Second):
		s.Fail("loop must exit on error-channel close")
	}
}

func (s *UdevHandlerTestSuite) TestFindDiskForDevicePath_Found() {
	s.seedHandlerDisk("disk-h-7", "part-h-7", "sdz7", "/dev/sdz7")
	got := s.handler.FindDiskForDevicePath("/dev/sdz7")
	s.Require().NotNil(got)
	s.Equal("disk-h-7", *got.Id)
	s.Nil(s.handler.FindDiskForDevicePath("/dev/sdz9"))
}

func (s *UdevHandlerTestSuite) TestHandlePartitionEvent_Batch_SyncsAllBranches() {
	diskID, partID := "disk-h-8", "part-h-8"
	devPath := "/dev/sdz8"
	diskIDCopy, partIDCopy, devCopy := diskID, partID, devPath
	parts := map[string]dto.Partition{partID: {Id: &partIDCopy, DiskId: &diskIDCopy, DevicePath: &devCopy}}
	disk := dto.Disk{Id: &diskIDCopy, Partitions: &parts}
	s.Require().NoError(s.disks.AddOrUpdate(&disk))

	// Persisted row re-added to the cache.
	s.Require().NoError(s.repo.Persist(&dto.MountPointData{
		Path: "/mnt/h-db", Root: "/", DeviceId: partID,
		Type: "ADDON", Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}))
	// Cached unmounted entry updated by the procfs snapshot.
	s.Require().NoError(s.disks.AddOrUpdateMountPoint(diskID, partID, dto.MountPointData{
		Path: "/mnt/h-cached", DeviceId: partID,
		Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}))
	// Stale mounted entry with an old refresh version.
	s.Require().NoError(s.disks.AddOrUpdateMountPoint(diskID, partID, dto.MountPointData{
		Path: "/mnt/h-stale", DeviceId: partID, IsMounted: true,
		Flags: &dto.MountFlags{}, CustomFlags: &dto.MountFlags{},
	}))
	s.disks.NextRefreshVersion()

	mountInfos := []*procfs.MountInfo{
		{MountID: 1, ParentID: 1, MajorMinorVer: "0:1", Root: "/", Source: devPath, MountPoint: "/mnt/h-cached", FSType: "ext4", Options: map[string]string{}, SuperOptions: map[string]string{}},
		{MountID: 2, ParentID: 1, MajorMinorVer: "0:1", Root: "/", Source: devPath, MountPoint: "/mnt/h-fresh", FSType: "ext4", Options: map[string]string{}, SuperOptions: map[string]string{}},
	}
	part := parts[partID]
	s.Require().NoError(s.handler.HandlePartitionEvent(s.ctx, events.PartitionEvent{
		Event:      events.Event{Type: events.EventTypes.ADD},
		Disk:       &disk,
		Partitions: []*dto.Partition{&part},
		MountInfos: mountInfos,
	}))

	cached, ok := s.disks.GetMountPoint(diskID, partID, "/mnt/h-cached")
	s.Require().True(ok)
	s.True(cached.IsMounted, "procfs match must mark the cached entry mounted")

	_, ok = s.disks.GetMountPoint(diskID, partID, "/mnt/h-fresh")
	s.True(ok, "procfs-only source must be added")

	_, ok = s.disks.GetMountPoint(diskID, partID, "/mnt/h-db")
	s.True(ok, "persisted row must be re-added")

	stale, ok := s.disks.GetMountPoint(diskID, partID, "/mnt/h-stale")
	s.Require().True(ok)
	s.False(stale.IsMounted, "entry missing from procfs must be marked unmounted")
}
