package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestDrainUdevChannels_DrainsBuffered verifies that a queue with buffered
// items and a pending error are fully drained.
func TestDrainUdevChannels_DrainsBuffered(t *testing.T) {
	queue := make(chan int, 4)
	queue <- 1
	queue <- 2
	errorChan := make(chan error, 1)
	errorChan <- errors.New("test drain error")

	drainUdevChannels(queue, errorChan, 200*time.Millisecond)

	require.Empty(t, queue)
	require.Empty(t, errorChan)
}

// TestDrainUdevChannels_UnblocksBlockedProducer reproduces the H7 leak: the
// udev monitor producer sends on `queue` with a blocking send outside its
// select loop. If the queue is full when quit closes, that send would block
// forever; draining the channels must let it complete and the producer exit.
func TestDrainUdevChannels_UnblocksBlockedProducer(t *testing.T) {
	queue := make(chan int, 1)
	queue <- 1 // fill the buffer so the next send blocks
	errorChan := make(chan error, 1)

	producerDone := make(chan struct{})
	go func() {
		queue <- 2 // blocks until drain makes room
		close(producerDone)
	}()

	// Give the producer a moment to block on the send.
	select {
	case <-producerDone:
		t.Fatal("producer should be blocked on a full queue before drain")
	case <-time.After(50 * time.Millisecond):
	}

	drainUdevChannels(queue, errorChan, 500*time.Millisecond)

	select {
	case <-producerDone:
		// producer unblocked and exited
	default:
		t.Fatal("producer goroutine still blocked after drain; goroutine leaked")
	}
	require.Empty(t, queue)
	require.Empty(t, errorChan)
}

// TestDrainUdevChannels_BoundedWhenIdle verifies the drain returns promptly
// (within the timeout) when both channels are already empty.
func TestDrainUdevChannels_BoundedWhenIdle(t *testing.T) {
	queue := make(chan int, 2)
	errorChan := make(chan error, 1)

	start := time.Now()
	drainUdevChannels(queue, errorChan, 100*time.Millisecond)
	require.WithinDuration(t, start, time.Now(), 500*time.Millisecond)
}

// TestDrainUdevChannels_ConcurrentProducer verifies the drain keeps consuming
// while a producer is actively sending, and both finish cleanly.
func TestDrainUdevChannels_ConcurrentProducer(t *testing.T) {
	queue := make(chan int, 2)
	errorChan := make(chan error, 1)
	const total = 20

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < total; i++ {
			queue <- i
		}
	}()

	drainUdevChannels(queue, errorChan, 500*time.Millisecond)
	wg.Wait()
	require.Empty(t, queue)
	require.Empty(t, errorChan)
}
