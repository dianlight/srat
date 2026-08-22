//go:build linux

package service

import (
	"log/slog"
	"time"

	"github.com/dianlight/tlog"
	"github.com/pilebones/go-udev/netlink"
)

func (self *VolumeService) udevEventHandler() {
	tlog.TraceContext(self.ctx, "Starting Udev event handler...")

	const (
		initialBackoff = 1 * time.Second
		maxBackoff     = 30 * time.Second
	)

	backoff := initialBackoff

	for {
		if self.ctx.Err() != nil {
			tlog.InfoContext(self.ctx, "Udev event handler exiting: context cancelled.")
			return
		}

		err := self.runUdevMonitorOnce()
		if self.ctx.Err() != nil {
			return
		}

		// Monitor exited unexpectedly (e.g. ENOBUFS from udev burst).
		// Log and reconnect with exponential backoff.
		slog.WarnContext(self.ctx, "Udev monitor exited, will reconnect",
			"err", err, "backoff", backoff)

		select {
		case <-self.ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// runUdevMonitorOnce connects to the netlink socket, starts the monitor, and
// runs consumeUdevChannels until it returns (either due to context cancellation
// or a closed channel from the monitor goroutine). On non-nil return the caller
// should reconnect after a backoff.
func (self *VolumeService) runUdevMonitorOnce() error {
	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		slog.ErrorContext(self.ctx, "Unable to connect to Netlink Kobject UEvent socket", "err", err)
		return err
	}
	defer conn.Close()

	queue := make(chan netlink.UEvent, 64)
	errorChan := make(chan error, 1)
	quit := conn.Monitor(queue, errorChan, nil)

	tlog.TraceContext(self.ctx, "Udev monitor started successfully.")

	// Consume events. Returns nil on ctx cancellation (clean shutdown) or a
	// sentinel error when the monitor goroutine closes the channels (e.g.
	// ENOBUFS from a flapping USB device).
	err := self.consumeUdevChannels(queue, errorChan)

	// Signal the monitor goroutine to stop and drain any in-flight sends so
	// it does not leak.
	if quit != nil {
		close(quit)
	}
	drainUdevChannels(queue, errorChan, 100*time.Millisecond)

	return err
}
