package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/internal/darwinstubs/mount/loop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
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

func newTestVolumeService(t *testing.T, disks *dto.DiskMap, mounter VolumeMountManagerInterface) *VolumeService {
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
	}
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

	disks := dto.DiskMap{
		diskID: {
			Id:         &diskID,
			Partitions: &map[string]dto.Partition{partitionID: partition},
		},
	}

	mounter := &fakeVolumeMounter{}
	svc := newTestVolumeService(t, &disks, mounter)

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

	disks := dto.DiskMap{
		diskID: {
			Id:         &diskID,
			Partitions: &map[string]dto.Partition{partitionID: partition},
		},
	}

	mounter := &fakeVolumeMounter{}
	svc := newTestVolumeService(t, &disks, mounter)

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

	disks := dto.DiskMap{
		diskID: {
			Id:         &diskID,
			Partitions: &map[string]dto.Partition{partitionID: partition},
		},
	}

	ctx := context.Background()
	eventBus := events.NewEventBus(ctx)
	fsService := NewFilesystemService(ctx, func() {}, eventBus)
	mounter := NewVolumeMountManager(VolumeMountManagerParams{
		Ctx:       ctx,
		FsService: fsService,
		Disks:     &disks,
		EventBus:  eventBus,
	})

	svc := &VolumeService{
		ctx:        ctx,
		state:      &dto.ContextState{},
		fs_service: fsService,
		mounter:    mounter,
		eventBus:   eventBus,
		disks:      &disks,
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
