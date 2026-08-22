package service

import (
	"log/slog"
	"strings"
	"time"

	"github.com/pilebones/go-udev/netlink"
	"gitlab.com/tozd/go/errors"
)

// Sentinel errors reported by consumeUdevChannels when the udev monitor
// goroutine has exited and its output channels have been closed. Callers use
// these to decide whether to reconnect (any non-nil return) or shut down
// cleanly (nil return on context cancellation).
var (
	errUdevQueueClosed     = errors.New("udev event queue closed: monitor exited")
	errUdevErrorChanClosed = errors.New("udev error channel closed: monitor exited")
)

// processUdevEvent dispatches one block-device uevent to the matching
// add/remove handler. Non-block subsystems and non-disk/partition devices are
// dropped here so the platform-specific monitor loop stays minimal.
func (self *VolumeService) processUdevEvent(uevent netlink.UEvent) {
	if self.udevEventProbe != nil {
		self.udevEventProbe(uevent)
	}

	subsystem, ok := uevent.Env["SUBSYSTEM"]
	if !ok || subsystem != "block" {
		return
	}
	action := uevent.Action
	devName := uevent.Env["DEVNAME"]
	devType := uevent.Env["DEVTYPE"]

	slog.DebugContext(self.ctx, "Received Udev block event", "action", action, "devname", devName, "devtype", devType, "env", uevent.Env)

	if devType != "disk" && devType != "partition" {
		slog.DebugContext(self.ctx, "Ignoring Udev event for non-disk/partition block device", "devname", devName, "devtype", devType)
		return
	}

	switch {
	case devType == "disk" && action == netlink.REMOVE:
		// Removal is handled by invalidating the hardware cache and
		// re-synchronizing the volume map; reconciliation evicts the
		// disk because it is absent from the new snapshot.
		self.handleDiskUdevRemoveEvent(devName)
	case devType == "disk" && action == netlink.ADD:
		slog.InfoContext(self.ctx, "Processing block device event", "action", action, "devname", devName)
		if self.hardwareClient != nil {
			self.hardwareClient.InvalidateHardwareInfo()
		}
		if err := self.getVolumesData(); err != nil {
			slog.ErrorContext(self.ctx, "Failed to get volumes data after udev event", "err", err)
		}
	case devType == "disk" && action == netlink.CHANGE:
		slog.InfoContext(self.ctx, "Ignore: Processing block device change event", "action", action, "devname", devName)
	case devType == "partition" && action == netlink.ADD:
		slog.InfoContext(self.ctx, "Processing partition addition event", "action", action, "devname", devName)
		if self.handlePartitionUdevAddEvent(devName) {
			return
		}
		if self.hardwareClient != nil {
			self.hardwareClient.InvalidateHardwareInfo()
		}
		if err := self.getVolumesData(); err != nil {
			slog.ErrorContext(self.ctx, "Failed to refresh volume cache after partition add event", "devname", devName, "err", err)
		}
		// The Supervisor API may not have processed its own udev events yet
		// when a flapping device's partition ADD event arrives, causing the
		// initial refresh to return stale data with a synthetic whole-disk
		// entry. Schedule a delayed retry so the Supervisor has time to
		// pick up the real partitions, and reset the provisional recheck
		// budget so the bounded recheck loop gets another chance.
		self.schedulePartitionAddRetry(devName)
	case devType == "partition" && action == netlink.REMOVE:
		slog.InfoContext(self.ctx, "Processing partition removal event", "action", action, "devname", devName)
		self.handlePartitionUdevRemoveEvent(devName)
	}
}

// schedulePartitionAddRetry schedules a delayed retry of getVolumesData after
// a partition ADD event that was not found in the DiskMap. The Supervisor API
// may not have processed its own udev events yet when a flapping device's
// partition ADD event arrives, so the initial refresh can return stale data
// containing only a synthetic whole-disk entry. The 500ms delay gives the
// Supervisor time to pick up the real partitions; the retry also resets the
// provisional recheck budget so the bounded recheck loop gets another chance
// to settle the layout.
func (self *VolumeService) schedulePartitionAddRetry(devName string) {
	time.AfterFunc(500*time.Millisecond, func() {
		if self.ctx.Err() != nil {
			return
		}
		slog.DebugContext(self.ctx, "Delayed retry: refreshing volume cache for partition", "devname", devName)
		self.resetProvisionalRecheckBudget()
		if self.hardwareClient != nil {
			self.hardwareClient.InvalidateHardwareInfo()
		}
		if err := self.getVolumesData(); err != nil {
			slog.ErrorContext(self.ctx, "Delayed retry: failed to refresh volume cache for partition", "devname", devName, "err", err)
		}
	})
}

// logUdevMonitorError reports a non-fatal monitor error. Malformed uevents
// (e.g. non-standard kernel formatting) are logged at debug so a single bad
// message does not spam the addon log; other errors stay at error level.
func (self *VolumeService) logUdevMonitorError(err error) {
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "unable to parse uevent") {
		if strings.Contains(err.Error(), "invalid env data") {
			slog.DebugContext(self.ctx, "Ignoring malformed uevent with invalid env data",
				"err", err,
				"detail", "This can occur when kernel sends events with non-standard formatting")
			return
		}
		slog.DebugContext(self.ctx, "Failed to parse uevent, skipping",
			"err", err,
			"detail", "Event format not recognized or incompatible")
		return
	}
	slog.ErrorContext(self.ctx, "Error received from Udev monitor", "err", err)
}

// consumeUdevChannels reads from the monitor's queue and error channels until
// one of them closes or the service context is cancelled.
//
// The go-udev monitor goroutine exits permanently on fatal receive errors
// (e.g. ENOBUFS when the kernel netlink socket buffer fills up during a udev
// burst from a flapping USB device) and then closes both channels. Reading
// from a closed channel returns the zero value immediately, so without the
// ok-idiom checks below the loop would spin forever on empty UEvents and
// never observe another hardware event. Returning a sentinel error instead
// lets the caller (the platform-specific udevEventHandler) reconnect with
// backoff, which is what eventually re-populates the disk map once the
// hardware has settled.
//
// A nil return means the service context was cancelled (clean shutdown).
func (self *VolumeService) consumeUdevChannels(queue <-chan netlink.UEvent, errorChan <-chan error) error {
	for {
		select {
		case <-self.ctx.Done():
			return nil
		case uevent, ok := <-queue:
			if !ok {
				return errors.WithStack(errUdevQueueClosed)
			}
			self.processUdevEvent(uevent)
		case err, ok := <-errorChan:
			if !ok {
				return errors.WithStack(errUdevErrorChanClosed)
			}
			self.logUdevMonitorError(err)
		}
	}
}
