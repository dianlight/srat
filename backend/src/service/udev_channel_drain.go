package service

import "time"

// drainUdevChannels drains the udev monitor's output channels for up to
// timeout after the quit channel has been closed.
//
// The go-udev Monitor producer goroutine performs a blocking `queue <- *uevent`
// send outside its select loop (see netlink/conn.go); if the queue is full when
// the quit channel is closed, that in-flight send blocks forever and leaks the
// producer goroutine. Draining both channels lets the in-flight send complete
// so the producer can observe the closed quit channel and exit. The timeout
// bounds the shutdown wait: the producer may emit a couple of events after
// quit closes (select picks randomly between the closed quit case and a
// buffered read), so we cannot return the instant the channels look empty.
//
// The helper is generic over the event type so it stays compilable on
// non-Linux platforms (the netlink package itself is Linux-only).
func drainUdevChannels[T any](queue <-chan T, errorChan <-chan error, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-queue:
		case <-errorChan:
		case <-timer.C:
			return
		}
	}
}
