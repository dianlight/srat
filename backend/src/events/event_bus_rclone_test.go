package events

import (
	"context"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	errors "gitlab.com/tozd/go/errors"
)

// TestEmitOnRcloneTask covers the rclone_task convenience wrappers on the
// event bus: a registered handler receives the emitted task synchronously.
func TestEmitOnRcloneTask(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus := NewEventBus(ctx)

	received := make(chan RcloneTaskEvent, 1)
	unsubscribe := bus.OnRcloneTask(func(_ context.Context, ev RcloneTaskEvent) errors.E {
		received <- ev
		return nil
	})
	defer unsubscribe()

	task := &dto.RcloneTask{TargetKind: "volume", TargetID: "/mnt/x", Status: "start"}
	bus.EmitRcloneTask(RcloneTaskEvent{Event: Event{Type: EventTypes.UPDATE}, Task: task})

	select {
	case ev := <-received:
		if ev.Task != task {
			t.Fatalf("received task %+v, want %+v", ev.Task, task)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for rclone_task event")
	}
}
