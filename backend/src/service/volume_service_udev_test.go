package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/darwinstubs/mount/loop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type fakeVolumeMounter struct {
	mountCalls   int
	unmountCalls int
}

func (f *fakeVolumeMounter) Mount(md *dto.MountPointData, flags uintptr, data, mountFsType string) errors.E {
	f.mountCalls++
	md.IsMounted = true
	return nil
}

func (f *fakeVolumeMounter) Unmount(md *dto.MountPointData, force bool) errors.E {
	f.unmountCalls++
	if md != nil {
		md.IsMounted = false
	}
	return nil
}

// failingCacheWriteMounter simulates the volumeMountManager.Mount error path
// where the OS-level mount succeeds but writing the mount point into the
// in-memory cache fails (volume_mount_manager.go cache-write branch). The
// mount point data it returns is unchanged, so the emitted event carries the
// stale IsMounted=false that re-triggers handleMountPointEvent.
type failingCacheWriteMounter struct {
	mountCalls int
}

func (f *failingCacheWriteMounter) Mount(md *dto.MountPointData, flags uintptr, data, mountFsType string) errors.E {
	f.mountCalls++
	return errors.WithDetails(dto.ErrorMountFail,
		"Detail", "simulated cache write failure after successful mount",
		"MountPath", md.Path,
	)
}

func (f *failingCacheWriteMounter) Unmount(md *dto.MountPointData, force bool) errors.E {
	return nil
}

func newTestVolumeService(t testing.TB, disks *dto.DiskMap, mounter VolumeMountManagerInterface) *VolumeService {
	t.Helper()

	ctx := context.Background()
	eventBus := events.NewEventBus(ctx)
	fsService := NewFilesystemService(ctx, func() {}, eventBus)

	return &VolumeService{
		ctx:        ctx,
		state:      &dto.ContextState{},
		fs_service: fsService,
		mounter:    mounter,
		eventBus:   eventBus,
		disks:      disks,

		automountBackoffBase: 2 * time.Second,
		maxAutomountAttempts: 5,
		automountRetries:     map[string]automountRetryState{},
	}
}

// newAutomountRetryVolumeService wires a VolumeService with a real database
// (persistMountPoint requires it) and subscribes handleMountPointEvent to the
// event bus, mirroring the NewVolumeService wiring. The database is a unique
// temp-file SQLite DB per test so rows persisted by one test never leak into
// another test's connection (the package's shared-memory DSN is process-wide).
func newAutomountRetryVolumeService(t testing.TB, disks *dto.DiskMap, mounter VolumeMountManagerInterface) *VolumeService {
	t.Helper()
	svc := newTestVolumeService(t, disks, mounter)

	dbPath := filepath.Join(t.TempDir(), "test.db")

	app := fxtest.New(t,
		fx.Provide(
			func() *dto.ContextState {
				return &dto.ContextState{DatabasePath: dbPath}
			},
			dbom.NewDB,
		),
		fx.Populate(&svc.db),
	)
	app.RequireStart()
	t.Cleanup(func() { app.RequireStop() })

	unsub := svc.eventBus.OnMountPoint(svc.handleMountPointEvent)
	t.Cleanup(unsub)

	return svc
}

func TestHandlePartitionUdevAddEvent_RetriesStartupMount(t *testing.T) {
	tmpDir := t.TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	require.NoError(t, os.WriteFile(deviceFile, []byte("test"), 0o600))

	diskID := "disk-1"
	partitionID := "part-1"
	devName := "sda1"
	mountPath := filepath.Join(tmpDir, "mnt", "share")
	startup := true

	partition := dto.Partition{
		Id:               &partitionID,
		DiskId:           &diskID,
		LegacyDeviceName: &devName,
		LegacyDevicePath: new("/dev/sda1"),
		DevicePath:       &deviceFile,
		MountPointData: &map[string]dto.MountPointData{
			mountPath: {
				Path:               mountPath,
				Root:               "/",
				DeviceId:           partitionID,
				Flags:              &dto.MountFlags{},
				IsToMountAtStartup: &startup,
				IsMounted:          false,
			},
		},
	}

	disk := dto.Disk{
		Id:         &diskID,
		Partitions: &map[string]dto.Partition{partitionID: partition},
	}
	disks := dto.NewDiskMapFrom(&disk)

	mounter := &fakeVolumeMounter{}
	svc := newTestVolumeService(t, disks, mounter)

	handled := svc.handlePartitionUdevAddEvent(devName)

	assert.True(t, handled)
	assert.Equal(t, 1, mounter.mountCalls)
}

func TestHandlePartitionUdevRemoveEvent_UnmountsAndEvictsPartition(t *testing.T) {
	diskID := "disk-1"
	partitionID := "part-1"
	devName := "sda1"
	mountPath := "/mnt/share"

	partition := dto.Partition{
		Id:               &partitionID,
		DiskId:           &diskID,
		LegacyDeviceName: &devName,
		LegacyDevicePath: new("/dev/sda1"),
		DevicePath:       new("/dev/disk/by-id/test-part-1"),
		MountPointData: &map[string]dto.MountPointData{
			mountPath: {
				Path:      mountPath,
				DeviceId:  partitionID,
				IsMounted: true,
			},
		},
	}

	disk := dto.Disk{
		Id:         &diskID,
		Partitions: &map[string]dto.Partition{partitionID: partition},
	}
	disks := dto.NewDiskMapFrom(&disk)

	mounter := &fakeVolumeMounter{}
	svc := newTestVolumeService(t, disks, mounter)

	svc.handlePartitionUdevRemoveEvent(devName)

	assert.Equal(t, 1, mounter.unmountCalls)
	_, found := disks.GetPartition(diskID, partitionID)
	assert.False(t, found)
}

func TestHandlePartitionUdevRemoveEvent_LoopbackExt4EvictsCache(t *testing.T) {
	device, err := loop.FindDevice()
	if err != nil {
		t.Skipf("no loop device available: %v", err)
	}

	imagePath := filepath.Clean("../../test/data/image.dmg")
	require.NoError(t, loop.SetFile(device, imagePath))
	t.Cleanup(func() {
		_ = loop.ClearFile(device)
	})

	diskID := "loop-disk"
	partitionID := "loop-partition"
	devName := filepath.Base(device)
	mountPath := filepath.Join(t.TempDir(), "loop-mount")
	fsType := "ext4"

	partition := dto.Partition{
		Id:               &partitionID,
		DiskId:           &diskID,
		LegacyDeviceName: &devName,
		LegacyDevicePath: &device,
		DevicePath:       &device,
		FsType:           &fsType,
		MountPointData:   &map[string]dto.MountPointData{},
	}

	disk := dto.Disk{
		Id:         &diskID,
		Partitions: &map[string]dto.Partition{partitionID: partition},
	}
	disks := dto.NewDiskMapFrom(&disk)

	ctx := context.Background()
	eventBus := events.NewEventBus(ctx)
	fsService := NewFilesystemService(ctx, func() {}, eventBus)
	mounter := NewVolumeMountManager(VolumeMountManagerParams{
		Ctx:       ctx,
		FsService: fsService,
		Disks:     disks,
		EventBus:  eventBus,
	})

	svc := &VolumeService{
		ctx:        ctx,
		state:      &dto.ContextState{},
		fs_service: fsService,
		mounter:    mounter,
		eventBus:   eventBus,
		disks:      disks,
	}

	mountData := dto.MountPointData{
		Path:        mountPath,
		Root:        "/",
		DeviceId:    partitionID,
		FSType:      &fsType,
		Flags:       &dto.MountFlags{},
		CustomFlags: &dto.MountFlags{},
		Partition:   &partition,
	}

	require.NoError(t, svc.MountVolume(&mountData))
	t.Cleanup(func() {
		_ = svc.unmountVolume(&mountData, true)
	})

	cachedMount, ok := disks.GetMountPoint(diskID, partitionID, mountPath)
	require.True(t, ok)
	require.NotNil(t, cachedMount)
	require.True(t, cachedMount.IsMounted)

	svc.handlePartitionUdevRemoveEvent(devName)

	_, found := disks.GetPartition(diskID, partitionID)
	assert.False(t, found)

	_, statErr := os.Stat(mountPath)
	assert.Error(t, statErr)
	assert.True(t, os.IsNotExist(statErr))
}

// TestHandleMountPointEvent_AutomountRetryBounded verifies that a mount-point
// event loop (ADD/UPDATE events carrying a stale IsMounted=false for a
// startup-mount path) cannot spin mount attempts forever: the retry budget
// caps the number of Mount calls and backoff pauses further attempts.
func TestHandleMountPointEvent_AutomountRetryBounded(t *testing.T) {
	tmpDir := t.TempDir()
	mountPath := filepath.Join(tmpDir, "mnt", "share")
	require.NoError(t, os.MkdirAll(mountPath, 0o755))
	devicePath := filepath.Join(tmpDir, "device.img")
	require.NoError(t, os.WriteFile(devicePath, []byte("test"), 0o600))

	diskID := "disk-1"
	partitionID := "part-1"
	startup := true
	md := dto.MountPointData{
		Path:               mountPath,
		Root:               "/",
		DeviceId:           partitionID,
		Flags:              &dto.MountFlags{},
		IsToMountAtStartup: &startup,
		IsMounted:          false,
		Partition: &dto.Partition{
			Id:         &partitionID,
			DiskId:     &diskID,
			DevicePath: &devicePath,
		},
	}
	disk := dto.Disk{
		Id: &diskID,
		Partitions: &map[string]dto.Partition{
			partitionID: {
				Id:         &partitionID,
				DiskId:     &diskID,
				DevicePath: &devicePath,
			},
		},
	}
	disks := dto.NewDiskMapFrom(&disk)

	mounter := &failingCacheWriteMounter{}
	svc := newAutomountRetryVolumeService(t, disks, mounter)

	// Zero the backoff so every emitted event may attempt a mount; the
	// attempt budget alone must bound the number of Mount calls.
	svc.automountBackoffBase = 0

	// Emit the same stale ADD event repeatedly; the event bus emits
	// synchronously, so this mirrors the re-entrant event loop.
	for i := 0; i < 20; i++ {
		require.NoError(t, svc.eventBus.EmitMountPoint(events.MountPointEvent{
			Event:      events.Event{Type: events.EventTypes.ADD},
			MountPoint: &md,
		}))
	}

	assert.Equal(t, 5, mounter.mountCalls, "mount attempts must be bounded by the retry budget")
}

// TestHandleMountPointEvent_AutomountRetryBackoff asserts that a second
// failure within the backoff window is not attempted again immediately.
func TestHandleMountPointEvent_AutomountRetryBackoff(t *testing.T) {
	tmpDir := t.TempDir()
	mountPath := filepath.Join(tmpDir, "mnt", "share")
	require.NoError(t, os.MkdirAll(mountPath, 0o755))
	devicePath := filepath.Join(tmpDir, "device.img")
	require.NoError(t, os.WriteFile(devicePath, []byte("test"), 0o600))

	diskID := "disk-1"
	partitionID := "part-1"
	startup := true
	md := dto.MountPointData{
		Path:               mountPath,
		Root:               "/",
		DeviceId:           partitionID,
		Flags:              &dto.MountFlags{},
		IsToMountAtStartup: &startup,
		IsMounted:          false,
		Partition: &dto.Partition{
			Id:         &partitionID,
			DiskId:     &diskID,
			DevicePath: &devicePath,
		},
	}
	disks := dto.NewDiskMapFrom(&dto.Disk{
		Id: &diskID,
		Partitions: &map[string]dto.Partition{
			partitionID: {
				Id:         &partitionID,
				DiskId:     &diskID,
				DevicePath: &devicePath,
			},
		},
	})

	mounter := &failingCacheWriteMounter{}
	svc := newAutomountRetryVolumeService(t, disks, mounter)

	// First event fails and schedules the next attempt after the backoff.
	require.NoError(t, svc.eventBus.EmitMountPoint(events.MountPointEvent{
		Event:      events.Event{Type: events.EventTypes.ADD},
		MountPoint: &md,
	}))
	assert.Equal(t, 1, mounter.mountCalls)

	// A second event immediately after must be skipped (backoff window).
	require.NoError(t, svc.eventBus.EmitMountPoint(events.MountPointEvent{
		Event:      events.Event{Type: events.EventTypes.UPDATE},
		MountPoint: &md,
	}))
	assert.Equal(t, 1, mounter.mountCalls, "second attempt within backoff window must be skipped")
}

// TestHandlePartitionUdevAddEvent_AutomountRetryBounded verifies the same
// retry budget bounds the udev-add automount retry path.
func TestHandlePartitionUdevAddEvent_AutomountRetryBounded(t *testing.T) {
	tmpDir := t.TempDir()
	deviceFile := filepath.Join(tmpDir, "device.img")
	require.NoError(t, os.WriteFile(deviceFile, []byte("test"), 0o600))

	diskID := "disk-1"
	partitionID := "part-1"
	devName := "sda1"
	mountPath := filepath.Join(tmpDir, "mnt", "share")
	startup := true

	partition := dto.Partition{
		Id:               &partitionID,
		DiskId:           &diskID,
		LegacyDeviceName: &devName,
		LegacyDevicePath: new("/dev/sda1"),
		DevicePath:       &deviceFile,
		MountPointData: &map[string]dto.MountPointData{
			mountPath: {
				Path:               mountPath,
				Root:               "/",
				DeviceId:           partitionID,
				Flags:              &dto.MountFlags{},
				IsToMountAtStartup: &startup,
				IsMounted:          false,
			},
		},
	}
	disk := dto.Disk{
		Id:         &diskID,
		Partitions: &map[string]dto.Partition{partitionID: partition},
	}
	disks := dto.NewDiskMapFrom(&disk)

	mounter := &failingCacheWriteMounter{}
	svc := newTestVolumeService(t, disks, mounter)
	svc.automountBackoffBase = 0
	svc.maxAutomountAttempts = 5

	for i := 0; i < 20; i++ {
		svc.handlePartitionUdevAddEvent(devName)
	}

	assert.Equal(t, 5, mounter.mountCalls, "udev automount retries must be bounded by the retry budget")
}
