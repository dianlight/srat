package service

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// fakeH2HardwareClient is a minimal HardwareServiceInterface that returns a
// fixed snapshot of disks/partitions without touching real hardware.
type fakeH2HardwareClient struct {
	disks map[string]dto.Disk
}

func (f *fakeH2HardwareClient) GetHardwareInfo() (map[string]dto.Disk, errors.E) {
	return f.disks, nil
}

func (f *fakeH2HardwareClient) InvalidateHardwareInfo() {}

func (f *fakeH2HardwareClient) MockSetFSProbeFunc(func(string) (string, uintptr, error)) {}

// buildH2Hardware returns diskCount disks each with partCount partitions.
// Every partition carries a distinct device path and the id of its parent
// disk, mirroring what the hardware client produces in production.
func buildH2Hardware(diskCount, partCount int) map[string]dto.Disk {
	hw := make(map[string]dto.Disk, diskCount)
	for d := 0; d < diskCount; d++ {
		diskID := fmt.Sprintf("disk-%02d", d)
		parts := make(map[string]dto.Partition, partCount)
		for p := 0; p < partCount; p++ {
			partID := fmt.Sprintf("part-%02d-%02d", d, p)
			devPath := fmt.Sprintf("/dev/sd%c%d", 'a'+d, p+1)
			fsType := "ext4"
			parts[partID] = dto.Partition{
				Id:         &partID,
				DiskId:     &diskID,
				DevicePath: &devPath,
				FsType:     &fsType,
			}
		}
		hw[diskID] = dto.Disk{Id: &diskID, Partitions: &parts}
	}
	return hw
}

// newH2VolumeService wires a VolumeService with a fake hardware client, a
// real temp-file SQLite DB (syncPartitionMountData queries it) and a counting
// procfs mock. The returned counters let tests assert how many times procfs
// was parsed and how many batched partition events were emitted.
func newH2VolumeService(t testing.TB, disks *dto.DiskMap, hw map[string]dto.Disk) (*VolumeService, *int, *int) {
	t.Helper()

	svc := newTestVolumeService(t, disks, &fakeVolumeMounter{})
	svc.hardwareClient = &fakeH2HardwareClient{disks: hw}

	parseCount := 0
	svc.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) {
		parseCount++
		// Two mounts matching the first partitions of the first disk.
		src1 := "/dev/sda1"
		src2 := "/dev/sda2"
		return []*procfs.MountInfo{
			{MountID: 5001, ParentID: 1, MajorMinorVer: "0:96", Root: "/", Source: src1, MountPoint: "/mnt/sda1", FSType: "ext4", Options: map[string]string{"rw": ""}, SuperOptions: map[string]string{}},
			{MountID: 5002, ParentID: 1, MajorMinorVer: "0:96", Root: "/", Source: src2, MountPoint: "/mnt/sda2", FSType: "ext4", Options: map[string]string{"rw": ""}, SuperOptions: map[string]string{}},
		}, nil
	})

	// Real DB so loadMountPointFromDB inside syncPartitionMountData works.
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

	// Wire the same subscription production uses (NewVolumeService).
	unsub := svc.eventBus.OnPartition(svc.handlePartitionEvent)
	t.Cleanup(unsub)

	// Count emitted partition events on the bus.
	emitCount := 0
	unsubEmit := svc.eventBus.OnPartition(func(_ context.Context, _ events.PartitionEvent) errors.E {
		emitCount++
		return nil
	})
	t.Cleanup(unsubEmit)

	return svc, &parseCount, &emitCount
}

// TestGetVolumesData_ParsesProcfsOncePerCycle is the H2 regression test: with
// 10 disks x 5 partitions, the refresh cycle must parse procfs exactly once
// and emit one batched event per disk (10 events, not 50).
func TestGetVolumesData_ParsesProcfsOncePerCycle(t *testing.T) {
	hw := buildH2Hardware(10, 5)
	svc, parseCount, emitCount := newH2VolumeService(t, dto.NewDiskMap(), hw)

	err := svc.getVolumesData()
	require.NoError(t, err)

	// The procfs snapshot is parsed once per cycle and shared via the event
	// payload; the handler must not re-parse for any partition.
	assert.Equal(t, 1, *parseCount, "procfs must be parsed exactly once per refresh cycle")
	// One batched event per disk, not one per partition.
	assert.Equal(t, 10, *emitCount, "one batched partition event per disk expected")
}

// BenchmarkGetVolumesData_10Disks50Partitions measures the cost of a full
// refresh cycle (hardware scan + per-disk batched partition events + mount
// data sync) with 10 disks x 5 partitions. Before the H2 change this path
// parsed procfs 50 times (once per partition event); after the change it
// parses exactly once per cycle.
func BenchmarkGetVolumesData_10Disks50Partitions(b *testing.B) {
	hw := buildH2Hardware(10, 5)
	svc, _, _ := newH2VolumeService(b, dto.NewDiskMap(), hw)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := svc.getVolumesData(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetVolumesData_50Disks250Partitions measures the same refresh
// cycle at a larger scale (50 disks x 5 partitions) to expose per-disk
// scaling behaviour of the batched emit path.
func BenchmarkGetVolumesData_50Disks250Partitions(b *testing.B) {
	hw := buildH2Hardware(50, 5)
	svc, _, _ := newH2VolumeService(b, dto.NewDiskMap(), hw)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := svc.getVolumesData(); err != nil {
			b.Fatal(err)
		}
	}
}

// TestHandlePartitionEvent_BatchModeWithoutSnapshot covers the batch branch
// of handlePartitionEvent when the emitter provides no shared procfs snapshot
// (MountInfos == nil): the handler must parse procfs exactly once itself.
func TestHandlePartitionEvent_BatchModeWithoutSnapshot(t *testing.T) {
	hw := buildH2Hardware(1, 2)
	disk := hw["disk-00"]
	disks := dto.NewDiskMapFrom(&disk)
	svc, parseCount, _ := newH2VolumeService(t, disks, hw)

	parts := make([]*dto.Partition, 0, len(*disk.Partitions))
	for _, p := range *disk.Partitions {
		parts = append(parts, &p)
	}

	err := svc.handlePartitionEvent(context.Background(), events.PartitionEvent{
		Event:      events.Event{Type: events.EventTypes.ADD},
		Disk:       &disk,
		Partitions: parts,
		// MountInfos intentionally nil -> handler must parse once.
	})
	require.NoError(t, err)
	assert.Equal(t, 1, *parseCount, "handler must parse procfs once when no snapshot is provided")
}

// TestHandlePartitionEvent_BatchModeProcfsError covers the error path of the
// batch branch: when the handler must parse procfs itself and parsing fails,
// the event is rejected with the parse error.
func TestHandlePartitionEvent_BatchModeProcfsError(t *testing.T) {
	hw := buildH2Hardware(1, 1)
	disk := hw["disk-00"]
	svc, _, _ := newH2VolumeService(t, dto.NewDiskMapFrom(&disk), hw)
	svc.MockSetProcfsGetMounts(func() ([]*procfs.MountInfo, error) {
		return nil, errors.New("procfs unavailable")
	})

	part := (*disk.Partitions)["part-00-00"]
	err := svc.handlePartitionEvent(context.Background(), events.PartitionEvent{
		Event:      events.Event{Type: events.EventTypes.ADD},
		Disk:       &disk,
		Partitions: []*dto.Partition{&part},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "procfs unavailable")
}

// TestHandlePartitionEvent_SinglePartitionMode covers the legacy single
// partition branch of handlePartitionEvent (udev-style emitters): the handler
// parses procfs once, syncs the partition's mount data and records the mount
// point in the cache.
func TestHandlePartitionEvent_SinglePartitionMode(t *testing.T) {
	hw := buildH2Hardware(1, 1)
	disk := hw["disk-00"]
	disks := dto.NewDiskMapFrom(&disk)
	svc, parseCount, _ := newH2VolumeService(t, disks, hw)

	part := (*disk.Partitions)["part-00-00"]
	err := svc.handlePartitionEvent(context.Background(), events.PartitionEvent{
		Event:     events.Event{Type: events.EventTypes.ADD},
		Disk:      &disk,
		Partition: &part,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, *parseCount, "single-partition mode must parse procfs once")

	mp, ok := disks.GetMountPoint("disk-00", "part-00-00", "/mnt/sda1")
	require.True(t, ok, "mount point from procfs snapshot must be in the cache")
	assert.True(t, mp.IsMounted)
}

// TestSyncPartitionMountData_UpdatesExistingMountPoint covers the branch of
// syncPartitionMountData where a mount point already exists in the cache at a
// path present in the procfs snapshot: it must be flipped to mounted (UPDATE
// semantics) instead of being re-added.
func TestSyncPartitionMountData_UpdatesExistingMountPoint(t *testing.T) {
	hw := buildH2Hardware(1, 1)
	disk := hw["disk-00"]
	disks := dto.NewDiskMapFrom(&disk)
	svc, _, _ := newH2VolumeService(t, disks, hw)

	startup := false
	require.NoError(t, disks.AddOrUpdateMountPoint("disk-00", "part-00-00", dto.MountPointData{
		Path:               "/mnt/sda1",
		DeviceId:           "part-00-00",
		IsMounted:          false,
		IsToMountAtStartup: &startup,
		Flags:              &dto.MountFlags{},
		CustomFlags:        &dto.MountFlags{},
	}))

	err := svc.getVolumesData()
	require.NoError(t, err)

	mp, ok := disks.GetMountPoint("disk-00", "part-00-00", "/mnt/sda1")
	require.True(t, ok)
	assert.True(t, mp.IsMounted, "existing mount point must be marked mounted after refresh")
}

// TestSyncPartitionMountData_MarksStaleUnmounted covers the staleness loop of
// syncPartitionMountData: a cache mount point absent from the procfs snapshot
// (and with an older refresh version) must be marked unmounted.
func TestSyncPartitionMountData_MarksStaleUnmounted(t *testing.T) {
	hw := buildH2Hardware(1, 1)
	disk := hw["disk-00"]
	disks := dto.NewDiskMapFrom(&disk)
	svc, _, _ := newH2VolumeService(t, disks, hw)

	require.NoError(t, disks.AddOrUpdateMountPoint("disk-00", "part-00-00", dto.MountPointData{
		Path:        "/mnt/stale",
		DeviceId:    "part-00-00",
		IsMounted:   true,
		Flags:       &dto.MountFlags{},
		CustomFlags: &dto.MountFlags{},
		// RefreshVersion zero -> older than the refresh cycle's version.
	}))

	err := svc.getVolumesData()
	require.NoError(t, err)

	mp, ok := disks.GetMountPoint("disk-00", "part-00-00", "/mnt/stale")
	require.True(t, ok)
	assert.False(t, mp.IsMounted, "stale mount point must be marked unmounted")
}

// TestSyncPartitionMountData_LoadsDBRows covers the DB-load branch of
// syncPartitionMountData: mount points persisted in the database for a
// partition are re-added to the in-memory cache during the refresh cycle.
func TestSyncPartitionMountData_LoadsDBRows(t *testing.T) {
	hw := buildH2Hardware(1, 1)
	disk := hw["disk-00"]
	disks := dto.NewDiskMapFrom(&disk)
	svc, _, _ := newH2VolumeService(t, disks, hw)

	part := (*disk.Partitions)["part-00-00"]
	mp := dto.MountPointData{
		Path:        "/mnt/db",
		Root:        "/",
		DeviceId:    *part.Id,
		Type:        "ADDON",
		IsMounted:   false,
		Flags:       &dto.MountFlags{},
		CustomFlags: &dto.MountFlags{},
	}
	require.NoError(t, svc.persistMountPoint(&mp))

	err := svc.getVolumesData()
	require.NoError(t, err)

	got, ok := disks.GetMountPoint("disk-00", "part-00-00", "/mnt/db")
	require.True(t, ok, "mount point persisted in DB must be re-added to the cache")
	assert.False(t, got.IsMounted)
}

// TestSyncPartitionMountData_SkipsNilDevicePath covers the early-return guard
// of syncPartitionMountData for partitions without a device path.
func TestSyncPartitionMountData_SkipsNilDevicePath(t *testing.T) {
	hw := buildH2Hardware(1, 1)
	disk := hw["disk-00"]
	disks := dto.NewDiskMapFrom(&disk)
	svc, _, _ := newH2VolumeService(t, disks, hw)

	part := (*disk.Partitions)["part-00-00"]
	part.DevicePath = nil
	require.NoError(t, svc.syncPartitionMountData(context.Background(), &disk, &part, nil))
}

// TestSyncPartitionMountData_FixesNilDiskId covers the DiskId fixup of
// syncPartitionMountData: a partition missing its disk id inherits the disk's
// id before mount point operations run.
func TestSyncPartitionMountData_FixesNilDiskId(t *testing.T) {
	hw := buildH2Hardware(1, 1)
	disk := hw["disk-00"]
	disks := dto.NewDiskMapFrom(&disk)
	svc, _, _ := newH2VolumeService(t, disks, hw)

	part := (*disk.Partitions)["part-00-00"]
	part.DiskId = nil
	require.NoError(t, svc.syncPartitionMountData(context.Background(), &disk, &part, nil))
	assert.Equal(t, "disk-00", *part.DiskId)
}
