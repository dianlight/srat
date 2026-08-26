// Package service contains internal tests for the service layer.
package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/stretchr/testify/require"
)

// This file holds focused coverage tests for VolumeService.MountVolume
// (volume_service.go). The suite-level success test (TestMountUnmountVolume_Success
// in volume_service_test.go) is skipped on darwin because it needs a loop
// device, so the internal package tests below exercise the validation,
// partition-lookup, OS-state and mounter branches that the suite cannot reach.
//
// The osutil mount-info override is package-global: every test that depends on
// IsMounted behavior installs its own override and restores it via t.Cleanup.

// TestMountVolume_NilMountPointData covers the md == nil guard.
func TestMountVolume_NilMountPointData(t *testing.T) {
	svc := newTestVolumeService(t, dto.NewDiskMap(), &fakeVolumeMounter{})

	errE := svc.MountVolume(nil)

	require.Error(t, errE)
	require.ErrorIs(t, errE, dto.ErrorInvalidParameter)
	require.Equal(t, "MountPointData is nil", errE.Details()["Message"])
}

// TestMountVolume_EmptyDeviceId covers the DeviceId == "" guard. The suite's
// TestMountVolume_DeviceEmpty only exercises the Root == "" guard because it
// leaves Root empty, so the DeviceId guard needs its own test.
func TestMountVolume_EmptyDeviceId(t *testing.T) {
	svc := newTestVolumeService(t, dto.NewDiskMap(), &fakeVolumeMounter{})

	md := dto.MountPointData{Path: "/mnt/test1", Root: "/", DeviceId: ""}
	errE := svc.MountVolume(&md)

	require.Error(t, errE)
	require.ErrorIs(t, errE, dto.ErrorInvalidParameter)
	require.Equal(t, "Source device name is empty in request", errE.Details()["Message"])
}

// TestMountVolume_DiskWithoutPartitions covers the `disk.Partitions == nil`
// continue branch and the partition-ID non-match branch of the lookup loop,
// ending in the "device does not exist" error.
func TestMountVolume_DiskWithoutPartitions(t *testing.T) {
	disks := dto.NewDiskMapFrom(
		&dto.Disk{Id: new("SD-NO-PART")}, // Partitions nil -> continue
		&dto.Disk{
			Id: new("SD"),
			Partitions: &map[string]dto.Partition{
				"other": {Id: new("other"), DevicePath: new("/dev/other")},
			},
		},
	)
	svc := newTestVolumeService(t, disks, &fakeVolumeMounter{})

	md := dto.MountPointData{Path: "/mnt/test1", Root: "/", DeviceId: "pippo"}
	errE := svc.MountVolume(&md)

	require.Error(t, errE)
	require.ErrorIs(t, errE, dto.ErrorDeviceNotFound)
	require.Equal(t, "Source device does not exist on the system", errE.Details()["Message"])
}

// TestMountVolume_PartitionLookupSuccess covers the partition lookup loop
// finding a match by DeviceId, the Flags nil-init branch, the os.Stat
// existence check against a real file, the mounter success path and the
// dismiss-notification calls (haService is nil, so they early-return).
func TestMountVolume_PartitionLookupSuccess(t *testing.T) {
	devicePath := filepath.Join(t.TempDir(), "dev")
	require.NoError(t, os.WriteFile(devicePath, []byte("x"), 0o600))
	disks := dto.NewDiskMapFrom(
		&dto.Disk{
			Id: new("SD"),
			Partitions: &map[string]dto.Partition{
				"pippo": {Id: new("pippo"), DevicePath: new(devicePath)},
			},
		},
	)
	restore := osutil.MockMountInfo("") // IsMounted -> (false, nil)
	t.Cleanup(restore)

	mounter := &fakeVolumeMounter{}
	svc := newTestVolumeService(t, disks, mounter)

	md := dto.MountPointData{Path: "/mnt/test1", Root: "/", DeviceId: "pippo"}
	errE := svc.MountVolume(&md)

	require.NoError(t, errE)
	require.NotNil(t, md.Partition, "lookup loop should populate md.Partition")
	require.Equal(t, "pippo", *md.Partition.Id)
	require.Equal(t, 1, mounter.mountCalls, "mounter.Mount should be called once")
	require.NotNil(t, md.Flags, "nil Flags should be initialized to empty MountFlags")
}

// TestMountVolume_PartitionIdEmptyTriggersLookup covers the lookup loop entry
// when md.Partition is present but its Id is nil/empty (the `||` guards in the
// loop condition), while still resolving the partition from the disks map.
func TestMountVolume_PartitionIdEmptyTriggersLookup(t *testing.T) {
	devicePath := filepath.Join(t.TempDir(), "dev")
	require.NoError(t, os.WriteFile(devicePath, []byte("x"), 0o600))
	disks := dto.NewDiskMapFrom(
		&dto.Disk{
			Id: new("SD"),
			Partitions: &map[string]dto.Partition{
				"pippo": {Id: new("pippo"), DevicePath: new(devicePath)},
			},
		},
	)
	restore := osutil.MockMountInfo("")
	t.Cleanup(restore)

	mounter := &fakeVolumeMounter{}
	svc := newTestVolumeService(t, disks, mounter)

	md := dto.MountPointData{
		Path:      "/mnt/test1",
		Root:      "/",
		DeviceId:  "pippo",
		Partition: &dto.Partition{}, // Id nil -> lookup still runs
	}
	errE := svc.MountVolume(&md)

	require.NoError(t, errE)
	require.Equal(t, "pippo", *md.Partition.Id)
}

// TestMountVolume_DevicePathNil covers the DevicePath nil/empty guard when the
// partition was resolved but carries no device path.
func TestMountVolume_DevicePathNil(t *testing.T) {
	disks := dto.NewDiskMapFrom(
		&dto.Disk{
			Id: new("SD"),
			Partitions: &map[string]dto.Partition{
				"pippo": {Id: new("pippo")}, // DevicePath nil
			},
		},
	)
	svc := newTestVolumeService(t, disks, &fakeVolumeMounter{})

	md := dto.MountPointData{Path: "/mnt/test1", Root: "/", DeviceId: "pippo"}
	errE := svc.MountVolume(&md)

	require.Error(t, errE)
	require.ErrorIs(t, errE, dto.ErrorDeviceNotFound)
	require.Equal(t, "Source device does not exist on the system", errE.Details()["Message"])
}

// TestMountVolume_DevicePathNotExist covers the os.Stat failure branch that
// reports a missing device path (statErr != nil, not a permission error).
func TestMountVolume_DevicePathNotExist(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	disks := dto.NewDiskMapFrom(
		&dto.Disk{
			Id: new("SD"),
			Partitions: &map[string]dto.Partition{
				"pippo": {Id: new("pippo"), DevicePath: new(missing)},
			},
		},
	)
	restore := osutil.MockMountInfo("")
	t.Cleanup(restore)

	svc := newTestVolumeService(t, disks, &fakeVolumeMounter{})

	md := dto.MountPointData{Path: "/mnt/test1", Root: "/", DeviceId: "pippo"}
	errE := svc.MountVolume(&md)

	require.Error(t, errE)
	require.ErrorIs(t, errE, dto.ErrorDeviceNotFound)
	require.Equal(t, "Device path does not exist", errE.Details()["Message"])
}

// TestMountVolume_AlreadyMounted covers the ok == true branch reported by
// osutil.IsMounted when the target path is present in the mount table.
func TestMountVolume_AlreadyMounted(t *testing.T) {
	restore := osutil.MockMountInfo("674 610 0:46 / /mnt/test1 rw,relatime - overlay overlay rw")
	t.Cleanup(restore)

	disks := dto.NewDiskMapFrom(
		&dto.Disk{
			Id: new("SD"),
			Partitions: &map[string]dto.Partition{
				"pippo": {Id: new("pippo"), DevicePath: new("/dev/pippo")},
			},
		},
	)
	svc := newTestVolumeService(t, disks, &fakeVolumeMounter{})

	md := dto.MountPointData{Path: "/mnt/test1", Root: "/", DeviceId: "pippo"}
	errE := svc.MountVolume(&md)

	require.Error(t, errE)
	require.ErrorIs(t, errE, dto.ErrorAlreadyMounted)
	require.Equal(t, "Volume is already mounted", errE.Details()["Message"])
}

// TestMountVolume_InvalidFlags covers the MountFlagsToSyscallFlagAndData error
// branch: a boolean/switch flag provided with a value is rejected.
func TestMountVolume_InvalidFlags(t *testing.T) {
	devicePath := filepath.Join(t.TempDir(), "dev")
	require.NoError(t, os.WriteFile(devicePath, []byte("x"), 0o600))
	disks := dto.NewDiskMapFrom(
		&dto.Disk{
			Id: new("SD"),
			Partitions: &map[string]dto.Partition{
				"pippo": {Id: new("pippo"), DevicePath: new(devicePath)},
			},
		},
	)
	restore := osutil.MockMountInfo("")
	t.Cleanup(restore)

	svc := newTestVolumeService(t, disks, &fakeVolumeMounter{})

	md := dto.MountPointData{
		Path:     "/mnt/test1",
		Root:     "/",
		DeviceId: "pippo",
		Flags: &dto.MountFlags{
			{Name: "ro", NeedsValue: false, FlagValue: "unexpected"},
		},
	}
	errE := svc.MountVolume(&md)

	require.Error(t, errE)
	require.ErrorIs(t, errE, dto.ErrorInvalidParameter)
	require.Equal(t, "Invalid Flags", errE.Details()["Message"])
}

// TestMountVolume_MounterError covers the mounter.Mount error propagation
// branch (the error returned by the mount manager is passed through as-is).
func TestMountVolume_MounterError(t *testing.T) {
	devicePath := filepath.Join(t.TempDir(), "dev")
	require.NoError(t, os.WriteFile(devicePath, []byte("x"), 0o600))
	disks := dto.NewDiskMapFrom(
		&dto.Disk{
			Id: new("SD"),
			Partitions: &map[string]dto.Partition{
				"pippo": {Id: new("pippo"), DevicePath: new(devicePath)},
			},
		},
	)
	restore := osutil.MockMountInfo("")
	t.Cleanup(restore)

	svc := newTestVolumeService(t, disks, &failingCacheWriteMounter{})

	md := dto.MountPointData{Path: "/mnt/test1", Root: "/", DeviceId: "pippo"}
	errE := svc.MountVolume(&md)

	require.Error(t, errE)
	require.ErrorIs(t, errE, dto.ErrorMountFail)
}

// TestMountVolume_ProtectedMode covers the ProtectedMode guard at the top of
// MountVolume.
func TestMountVolume_ProtectedMode(t *testing.T) {
	svc := newTestVolumeService(t, dto.NewDiskMap(), &fakeVolumeMounter{})
	svc.state.ProtectedMode = true

	md := dto.MountPointData{Path: "/mnt/test1", Root: "/", DeviceId: "pippo"}
	errE := svc.MountVolume(&md)

	require.Error(t, errE)
	require.ErrorIs(t, errE, dto.ErrorOperationNotPermittedInProtectedMode)
}
