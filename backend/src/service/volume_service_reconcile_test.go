package service

import (
	"context"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

// fakeReconcileHardware is a scripted HardwareServiceInterface used to drive
// the volume reconciliation paths deterministically. Each
// InvalidateHardwareInfo advances the phase; GetHardwareInfo serves the
// response of the current phase (clamped to the last one), so a later phase
// can model "the disk is gone" or "the disk settled into its real partition
// layout".
type fakeReconcileHardware struct {
	phase     int
	calls     int
	responses []map[string]dto.Disk
}

func (f *fakeReconcileHardware) GetHardwareInfo() (map[string]dto.Disk, errors.E) {
	f.calls++
	idx := f.phase
	if idx >= len(f.responses) {
		idx = len(f.responses) - 1
	}
	return f.responses[idx], nil
}

func (f *fakeReconcileHardware) InvalidateHardwareInfo() { f.phase++ }

func (f *fakeReconcileHardware) MockSetFSProbeFunc(func(string) (string, uintptr, error)) {}

// reconcileDisk builds a dto.Disk keyed by its by-id style id.
func reconcileDisk(id, name string, partitions ...dto.Partition) dto.Disk {
	d := dto.Disk{Id: &id, LegacyDeviceName: &name}
	if len(partitions) > 0 {
		m := make(map[string]dto.Partition)
		for _, p := range partitions {
			m[*p.Id] = p
		}
		d.Partitions = &m
	}
	return d
}

// wholeDiskPartition models the synthesized whole-disk filesystem partition:
// its legacy device name equals the disk's own name (e.g. "sda" on "sda").
func wholeDiskPartition(diskID, name string) dto.Partition {
	pid := "by-id-" + name
	return dto.Partition{
		Id:               &pid,
		DiskId:           &diskID,
		LegacyDeviceName: &name,
	}
}

// realPartition models a real partition child, whose name carries a number
// suffix (e.g. "sda1") and therefore differs from the disk name.
func realPartition(diskID, name string) dto.Partition {
	pid := "by-id-" + name
	return dto.Partition{
		Id:               &pid,
		DiskId:           &diskID,
		LegacyDeviceName: &name,
	}
}

func newReconcileVolumeService(t *testing.T, hw HardwareServiceInterface, recheckInterval time.Duration, maxRechecks int) *VolumeService {
	t.Helper()
	svc := newTestVolumeService(t, dto.NewDiskMap(), &fakeVolumeMounter{})
	svc.hardwareClient = hw
	svc.recheckInterval = recheckInterval
	svc.maxProvisionalRechecks = maxRechecks
	return svc
}

func TestGetVolumesData_PrunesDisksMissingFromSnapshot(t *testing.T) {
	diskA := reconcileDisk("by-id-ata-disk-a", "sda", realPartition("by-id-ata-disk-a", "sda1"))
	diskB := reconcileDisk("by-id-ata-disk-b", "sdb", realPartition("by-id-ata-disk-b", "sdb1"))

	hw := &fakeReconcileHardware{
		responses: []map[string]dto.Disk{
			{"by-id-ata-disk-a": diskA},
			{"by-id-ata-disk-b": diskB},
		},
	}
	svc := newReconcileVolumeService(t, hw, 0, 0)

	var removed []string
	unsub := svc.eventBus.OnDisk(func(ctx context.Context, e events.DiskEvent) errors.E {
		if e.Type == events.EventTypes.REMOVE {
			removed = append(removed, *e.Disk.Id)
		}
		return nil
	})
	defer unsub()

	require.NoError(t, svc.getVolumesData())
	_, okA := svc.disks.Get("by-id-ata-disk-a")
	require.True(t, okA, "disk A should be present after the first snapshot")

	// A fresh snapshot that no longer contains disk A must evict it.
	hw.InvalidateHardwareInfo()
	require.NoError(t, svc.getVolumesData())

	_, okA = svc.disks.Get("by-id-ata-disk-a")
	assert.False(t, okA, "disk A must be pruned when absent from the snapshot")
	_, okB := svc.disks.Get("by-id-ata-disk-b")
	assert.True(t, okB, "disk B from the new snapshot must be present")
	assert.Equal(t, []string{"by-id-ata-disk-a"}, removed, "pruned disks must be broadcast as REMOVE events")
}

func TestHandleDiskUdevRemoveEvent_PrunesRemovedDisk(t *testing.T) {
	diskA := reconcileDisk("by-id-ata-disk-a", "sda", realPartition("by-id-ata-disk-a", "sda1"))
	hw := &fakeReconcileHardware{
		responses: []map[string]dto.Disk{
			{"by-id-ata-disk-a": diskA},
			{},
		},
	}
	svc := newReconcileVolumeService(t, hw, 0, 0)

	require.NoError(t, svc.getVolumesData())
	_, ok := svc.disks.Get("by-id-ata-disk-a")
	require.True(t, ok, "disk should be present before the removal event")

	svc.handleDiskUdevRemoveEvent("sda")

	assert.Equal(t, 1, hw.phase, "disk removal must invalidate the hardware cache")
	_, ok = svc.disks.Get("by-id-ata-disk-a")
	assert.False(t, ok, "removed disk must be evicted from the map")
}

func TestGetVolumesData_SettlesWholeDiskSynthesizedEntryViaRecheck(t *testing.T) {
	diskID := "by-id-ata-disk-a"
	synthesized := reconcileDisk(diskID, "sda", wholeDiskPartition(diskID, "sda"))
	settled := reconcileDisk(diskID, "sda", realPartition(diskID, "sda1"))

	hw := &fakeReconcileHardware{
		responses: []map[string]dto.Disk{
			{diskID: synthesized},
			{diskID: settled},
		},
	}
	svc := newReconcileVolumeService(t, hw, 10*time.Millisecond, 5)

	require.NoError(t, svc.getVolumesData())
	_, ok := svc.disks.GetPartition(diskID, "by-id-sda")
	require.True(t, ok, "whole-disk synthesized partition should be present initially")

	require.Eventually(t, func() bool {
		_, realOK := svc.disks.GetPartition(diskID, "by-id-sda1")
		_, synthOK := svc.disks.GetPartition(diskID, "by-id-sda")
		return realOK && !synthOK
	}, 3*time.Second, 10*time.Millisecond,
		"recheck should replace the synthesized partition with the real layout")
}

func TestGetVolumesData_RecheckGivesUpAfterMaxAttempts(t *testing.T) {
	diskID := "by-id-ata-disk-a"
	synthesized := reconcileDisk(diskID, "sda", wholeDiskPartition(diskID, "sda"))
	hw := &fakeReconcileHardware{
		responses: []map[string]dto.Disk{{diskID: synthesized}},
	}
	const maxRechecks = 3
	svc := newReconcileVolumeService(t, hw, 10*time.Millisecond, maxRechecks)

	require.NoError(t, svc.getVolumesData())

	// Initial fetch + maxRechecks rechecks, then the chain must stop.
	require.Eventually(t, func() bool {
		return hw.calls >= maxRechecks+1
	}, 2*time.Second, 10*time.Millisecond)

	time.Sleep(150 * time.Millisecond) // allow any straggler timers to fire
	assert.Equal(t, maxRechecks+1, hw.calls, "recheck chain must be bounded")
	_, ok := svc.disks.GetPartition(diskID, "by-id-sda")
	assert.True(t, ok, "entry stays visible when the hardware never settles")
}

func TestFindDiskForDevicePath_ReturnsNilWhenNoPartitionMatches(t *testing.T) {
	diskID := "by-id-ata-disk-a"
	devPath := "/dev/sda1"
	disk := reconcileDisk(diskID, "sda", dto.Partition{
		Id:               new("by-id-sda1"),
		DiskId:           &diskID,
		LegacyDeviceName: new("sda1"),
		LegacyDevicePath: &devPath,
	})
	svc := newTestVolumeService(t, dto.NewDiskMapFrom(&disk), &fakeVolumeMounter{})

	got := svc.findDiskForDevicePath("/dev/sda1")
	require.NotNil(t, got)
	assert.Equal(t, diskID, *got.Id)

	assert.Nil(t, svc.findDiskForDevicePath("/dev/sdb9"),
		"no match must return nil, not an arbitrary disk")
}
