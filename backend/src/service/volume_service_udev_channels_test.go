package service

import (
	"context"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/pilebones/go-udev/netlink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newUdevChannelTestService builds a minimal VolumeService with a cancellable
// context so consumeUdevChannels can be exercised without touching real
// netlink sockets. Only the fields read by consumeUdevChannels are wired.
func newUdevChannelTestService(t *testing.T) (*VolumeService, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	svc := newTestVolumeService(t, dto.NewDiskMap(), &fakeVolumeMounter{})
	svc.ctx = ctx
	return svc, cancel
}

// TestConsumeUdevChannels_QueueClosedReturnsError reproduces the regression
// observed on a flapping USB disk: the go-udev monitor goroutine exits on
// ENOBUFS and closes the queue channel. The handler must notice the closed
// channel and return; the previous implementation re-read the zero UEvent
// from the closed channel forever, burning 100% CPU and never processing
// further events (so the disk map went stale and the UI kept showing the
// disk as a whole-disk/raw device even after the hardware had settled).
func TestConsumeUdevChannels_QueueClosedReturnsError(t *testing.T) {
	svc, _ := newUdevChannelTestService(t)

	queue := make(chan netlink.UEvent, 1)
	errCh := make(chan error, 1)
	close(queue) // simulate the go-udev monitor exiting

	done := make(chan error, 1)
	go func() { done <- svc.consumeUdevChannels(queue, errCh) }()

	select {
	case err := <-done:
		require.Error(t, err, "closed queue must surface an error so the caller can reconnect")
		assert.ErrorIs(t, err, errUdevQueueClosed, "expected errUdevQueueClosed, got %v", err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("consumeUdevChannels did not return after queue close; it is stuck in a busy loop")
	}
}

// TestConsumeUdevChannels_ErrorChanClosedReturnsError covers the symmetric
// case: if the monitor closes the error channel (which happens together with
// the queue close), the handler must also exit instead of spinning.
func TestConsumeUdevChannels_ErrorChanClosedReturnsError(t *testing.T) {
	svc, _ := newUdevChannelTestService(t)

	queue := make(chan netlink.UEvent, 1)
	errCh := make(chan error, 1)
	close(errCh)

	done := make(chan error, 1)
	go func() { done <- svc.consumeUdevChannels(queue, errCh) }()

	select {
	case err := <-done:
		require.Error(t, err, "closed error channel must surface an error so the caller can reconnect")
		assert.ErrorIs(t, err, errUdevErrorChanClosed, "expected errUdevErrorChanClosed, got %v", err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("consumeUdevChannels did not return after error channel close")
	}
}

// TestConsumeUdevChannels_ProcessesEventsUntilQueueCloses verifies that
// events delivered before the close are still processed in order, and the
// loop only exits once the channel is drained.
func TestConsumeUdevChannels_ProcessesEventsUntilQueueCloses(t *testing.T) {
	svc, _ := newUdevChannelTestService(t)

	queue := make(chan netlink.UEvent, 4)
	errCh := make(chan error, 1)

	processed := make(chan string, 4)
	svc.udevEventProbe = func(ue netlink.UEvent) {
		processed <- ue.Env["DEVNAME"]
	}

	queue <- netlink.UEvent{Action: netlink.ADD, Env: map[string]string{
		"SUBSYSTEM": "block", "DEVTYPE": "disk", "DEVNAME": "/dev/sdb",
	}}
	queue <- netlink.UEvent{Action: netlink.ADD, Env: map[string]string{
		"SUBSYSTEM": "block", "DEVTYPE": "partition", "DEVNAME": "/dev/sdb1",
	}}
	close(queue)

	done := make(chan error, 1)
	go func() { done <- svc.consumeUdevChannels(queue, errCh) }()

	select {
	case err := <-done:
		require.Error(t, err)
		assert.ErrorIs(t, err, errUdevQueueClosed)
	case <-time.After(time.Second):
		t.Fatal("consumeUdevChannels did not return after queue close")
	}

	got := []string{}
	for len(processed) > 0 {
		got = append(got, <-processed)
	}
	assert.Equal(t, []string{"/dev/sdb", "/dev/sdb1"}, got,
		"events queued before the close must be processed in order")
}

// TestConsumeUdevChannels_ContextCancelledReturnsNil ensures a clean shutdown
// is reported as nil so the caller does not schedule a reconnect.
func TestConsumeUdevChannels_ContextCancelledReturnsNil(t *testing.T) {
	svc, cancel := newUdevChannelTestService(t)

	queue := make(chan netlink.UEvent)
	errCh := make(chan error)

	done := make(chan error, 1)
	go func() { done <- svc.consumeUdevChannels(queue, errCh) }()

	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err, "context cancellation is a clean shutdown, not an error")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("consumeUdevChannels did not observe context cancellation")
	}
}

// TestConsumeUdevChannels_NilErrorValueIgnored checks that a nil error read
// from errorChan (which the previous implementation would have logged as
// "Error received from Udev monitor err=nil") is skipped.
func TestConsumeUdevChannels_NilErrorValueIgnored(t *testing.T) {
	svc, _ := newUdevChannelTestService(t)

	queue := make(chan netlink.UEvent, 1)
	errCh := make(chan error, 2)

	errCh <- nil // spurious nil — must be ignored
	close(queue)

	done := make(chan error, 1)
	go func() { done <- svc.consumeUdevChannels(queue, errCh) }()

	select {
	case err := <-done:
		require.Error(t, err)
		assert.ErrorIs(t, err, errUdevQueueClosed)
	case <-time.After(time.Second):
		t.Fatal("consumeUdevChannels did not return after queue close")
	}
}

// ---------------------------------------------------------------------------
// Tests for delayed retry on partition ADD (Bug A) and recheck budget reset
// (Bug B)
// ---------------------------------------------------------------------------

// TestResetProvisionalRecheckBudget_ClearsPendingState verifies that
// resetProvisionalRecheckBudget stops any running timer and sets
// pendingRecheck to nil, allowing a fresh recheck chain to start.
func TestResetProvisionalRecheckBudget_ClearsPendingState(t *testing.T) {
	diskID := "by-id-usb-disk"
	synthesized := reconcileDisk(diskID, "sdb", wholeDiskPartition(diskID, "sdb"))

	hw := &fakeReconcileHardware{
		responses: []map[string]dto.Disk{{diskID: synthesized}},
	}
	svc := newReconcileVolumeService(t, hw, 10*time.Millisecond, 3)

	// Seed the DiskMap with a synthesized entry so
	// manageProvisionalRechecks creates a pending state.
	require.NoError(t, svc.getVolumesData())
	svc.recheckMu.Lock()
	require.NotNil(t, svc.pendingRecheck, "pendingRecheck should be set after getVolumesData with synthesized entry")
	svc.recheckMu.Unlock()

	// Reset the budget.
	svc.resetProvisionalRecheckBudget()

	svc.recheckMu.Lock()
	require.Nil(t, svc.pendingRecheck, "pendingRecheck must be nil after reset")
	svc.recheckMu.Unlock()
}

// TestProcessUdevEvent_PartitionAdd_SchedulesRetry verifies that a partition
// ADD event whose device is NOT found in the DiskMap (the partition is unknown
// because the Supervisor hasn't reported it yet) triggers a delayed retry
// goroutine. The retry fires after 500ms, invalidates the hardware cache,
// resets the recheck budget, and calls getVolumesData again. We verify the
// retry by observing that a subsequent getVolumesData reflects the settled
// layout that the delayed retry would have fetched.
func TestProcessUdevEvent_PartitionAdd_SchedulesRetry(t *testing.T) {
	diskID := "by-id-usb-disk"
	synthesized := reconcileDisk(diskID, "sdb", wholeDiskPartition(diskID, "sdb"))
	settled := reconcileDisk(diskID, "sdb", realPartition(diskID, "sdb1"))

	// Phase 0: initial + immediate retry return synthesized.
	// Phase 1+: delayed retry returns settled layout.
	hw := &fakeReconcileHardware{
		responses: []map[string]dto.Disk{
			{diskID: synthesized},
			{diskID: settled},
		},
	}

	svc := newReconcileVolumeService(t, hw, 10*time.Millisecond, 5)
	require.NoError(t, svc.getVolumesData())

	// Exhaust the recheck budget so we can prove the retry resets it.
	svc.recheckMu.Lock()
	svc.pendingRecheck = &provisionalRecheckState{attempts: 0}
	svc.recheckMu.Unlock()

	// Confirm synthesized partition is present.
	_, ok := svc.disks.GetPartition(diskID, "by-id-sdb")
	require.True(t, ok, "synthesized whole-disk partition should be present initially")

	// Send a partition ADD for /dev/sdb1 which is NOT in the DiskMap.
	// handlePartitionUdevAddEvent returns false (unknown partition),
	// so processUdevEvent invalidates + refreshes AND schedules the
	// delayed retry.
	svc.processUdevEvent(netlink.UEvent{
		Action: netlink.ADD,
		Env: map[string]string{
			"SUBSYSTEM": "block",
			"DEVTYPE":   "partition",
			"DEVNAME":   "/dev/sdb1",
		},
	})

	// The delayed retry fires after 500ms and calls getVolumesData which
	// returns the settled layout (sdb1). Verify the synthesized entry is
	// gone and the real partition is present.
	require.Eventually(t, func() bool {
		_, synthOK := svc.disks.GetPartition(diskID, "by-id-sdb")
		_, realOK := svc.disks.GetPartition(diskID, "by-id-sdb1")
		return !synthOK && realOK
	}, 3*time.Second, 50*time.Millisecond,
		"delayed retry should replace synthesized entry with real partition")

	// Verify the recheck budget was reset by the delayed retry.
	svc.recheckMu.Lock()
	pending := svc.pendingRecheck
	svc.recheckMu.Unlock()
	require.Nil(t, pending, "delayed retry must reset the recheck budget")
}

// TestProcessUdevEvent_PartitionAdd_UnknownPartition_SettlesLayout covers the
// full scenario: a partition ADD event for an unknown partition (not in DiskMap)
// causes an immediate invalidation+refresh AND a delayed retry that
// invalidates again. When the delayed retry sees the settled layout from the
// Supervisor, it replaces the synthesized entry with real partitions.
//
// This is the core fix for the flapping USB disk bug where the Supervisor API
// returns stale data on the first call but may have settled by the delayed
// retry.
func TestProcessUdevEvent_PartitionAdd_UnknownPartition_SettlesLayout(t *testing.T) {
	diskID := "by-id-usb-flash"
	synthesized := reconcileDisk(diskID, "sdb", wholeDiskPartition(diskID, "sdb"))
	settled := reconcileDisk(diskID, "sdb", realPartition(diskID, "sdb1"), realPartition(diskID, "sdb2"))

	hw := &fakeReconcileHardware{
		responses: []map[string]dto.Disk{
			{diskID: synthesized}, // phase 0: initial fetch
			{diskID: synthesized}, // phase 1: immediate retry (still stale)
			{diskID: settled},     // phase 2: delayed retry sees settled layout
		},
	}

	svc := newReconcileVolumeService(t, hw, 10*time.Millisecond, 5)
	require.NoError(t, svc.getVolumesData())

	// Confirm the synthetic whole-disk partition is present.
	_, ok := svc.disks.GetPartition(diskID, "by-id-sdb")
	require.True(t, ok, "synthesized whole-disk partition should be present initially")

	// Send partition ADD for /dev/sdb1 — not in the DiskMap.
	svc.processUdevEvent(netlink.UEvent{
		Action: netlink.ADD,
		Env: map[string]string{
			"SUBSYSTEM": "block",
			"DEVTYPE":   "partition",
			"DEVNAME":   "/dev/sdb1",
		},
	})

	// The delayed retry fires after 500ms and calls getVolumesData which
	// returns the settled layout (sdb1 + sdb2).
	require.Eventually(t, func() bool {
		_, synthOK := svc.disks.GetPartition(diskID, "by-id-sdb")
		_, realOK := svc.disks.GetPartition(diskID, "by-id-sdb1")
		return !synthOK && realOK
	}, 3*time.Second, 50*time.Millisecond,
		"delayed retry should replace synthesized entry with real partitions")
}

// TestProcessUdevEvent_PartitionRemove_Untracked_NoInvalidation covers the
// flapping USB disk scenario: a partition REMOVE event for a device that was
// never tracked in the DiskMap (e.g. sdb1 remove while only the synthesized
// whole-disk "sdb" entry exists) must NOT trigger InvalidateHardwareInfo or
// getVolumesData.  If it did, every remove in a constant remove/add cycle
// would cause perpetual cache churn and the disk could settle on stale data.
func TestProcessUdevEvent_PartitionRemove_Untracked_NoInvalidation(t *testing.T) {
	diskID := "by-id-usb-flash"
	synthesized := reconcileDisk(diskID, "sdb", wholeDiskPartition(diskID, "sdb"))

	hw := &fakeReconcileHardware{
		responses: []map[string]dto.Disk{{diskID: synthesized}},
	}
	svc := newReconcileVolumeService(t, hw, 10*time.Millisecond, 3)
	require.NoError(t, svc.getVolumesData())

	// Confirm synthesized partition is present.
	_, ok := svc.disks.GetPartition(diskID, "by-id-sdb")
	require.True(t, ok, "synthesized whole-disk partition should be present")

	phaseBefore := hw.phaseValue()

	// Simulate flapping: rapid REMOVE events for partitions that are NOT
	// in the DiskMap (sdb1, sdb2 have names different from the synthesized
	// whole-disk entry "sdb").
	for _, dev := range []string{"/dev/sdb1", "/dev/sdb2", "/dev/sdb1"} {
		svc.processUdevEvent(netlink.UEvent{
			Action: netlink.REMOVE,
			Env: map[string]string{
				"SUBSYSTEM": "block",
				"DEVTYPE":   "partition",
				"DEVNAME":   dev,
			},
		})
	}

	// InvalidateHardwareInfo must NOT have been called — the partitions
	// were never tracked, so their removal doesn't change our state.
	assert.Equal(t, phaseBefore, hw.phaseValue(),
		"InvalidateHardwareInfo must not be called for untracked partition REMOVE events")

	// The disk must still be present with its synthesized partition.
	_, ok = svc.disks.GetPartition(diskID, "by-id-sdb")
	assert.True(t, ok, "synthesized entry must survive untracked partition remove events")
}
