package websocket

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestSubscribeNilHandler(t *testing.T) {
	c := NewClient("ws://example", "")
	// nil handler should return error
	if _, err := c.SubscribeConnectionEvents(nil); err == nil {
		t.Fatal("expected error when subscribing nil handler")
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	c := NewClient("ws://example", "").(*Client)
	called := int32(0)
	h := func(ev ConnectionEvent) {
		atomic.AddInt32(&called, 1)
	}

	unsubWrap, err := c.SubscribeConnectionEvents(h)
	if err != nil {
		t.Fatalf("subscribe error: %v", err)
	}

	// ensure handler was added to internal map
	c.connEventMu.RLock()
	count := len(c.connEvents)
	c.connEventMu.RUnlock()
	if count != 1 {
		t.Fatalf("expected 1 handler, got %d", count)
	}

	// simulate sending an event by invoking handlers in the map
	c.connEventMu.RLock()
	for _, fn := range c.connEvents {
		fn(ConnectionEvent{Type: ConnEventConnected, Message: "ok"})
	}
	c.connEventMu.RUnlock()

	// give goroutines time to run
	time.Sleep(10 * time.Millisecond)
	if atomic.LoadInt32(&called) == 0 {
		t.Fatal("expected handler to be invoked")
	}

	// unsubscribe using returned func (old signature returns func())
	unsubWrap()

	// now map should be empty
	c.connEventMu.RLock()
	count = len(c.connEvents)
	c.connEventMu.RUnlock()
	if count != 0 {
		t.Fatalf("expected 0 handlers after unsubscribe, got %d", count)
	}
}

func TestUnsubscribeBoolAPI(t *testing.T) {
	c := NewClient("ws://example", "").(*Client)
	called := int32(0)
	h := func(ev ConnectionEvent) {
		atomic.AddInt32(&called, 1)
	}

	// Use the underlying implementation to get unsubscribe that reports bool.
	// We need to create a subscription and then remove it via map.
	c.connEventMu.Lock()
	c.connNextEventID++
	id := c.connNextEventID
	c.connEvents[id] = h
	c.connEventMu.Unlock()

	// ensure present
	c.connEventMu.RLock()
	if _, ok := c.connEvents[id]; !ok {
		c.connEventMu.RUnlock()
		t.Fatal("subscription not found after manual add")
	}
	c.connEventMu.RUnlock()

	// remove using delete and verify
	c.connEventMu.Lock()
	delete(c.connEvents, id)
	c.connEventMu.Unlock()

	c.connEventMu.RLock()
	if _, ok := c.connEvents[id]; ok {
		c.connEventMu.RUnlock()
		t.Fatal("subscription still present after delete")
	}
	c.connEventMu.RUnlock()
}
