package service

import (
	"context"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/stretchr/testify/assert"
	"github.com/teivah/broadcast"
)

func TestBroadcasterDirtyDataDedupe_BroadcastsOnlyOnChange(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventBus := events.NewEventBus(ctx)
	b := &BroadcasterService{
		ctx:      ctx,
		relay:    broadcast.NewRelay[broadcastEvent](),
		eventBus: eventBus,
		disks:    &dto.DiskMap{},
		state:    &dto.ContextState{},
	}

	unsub := b.setupEventListeners()
	defer func() {
		for _, fn := range unsub {
			fn()
		}
	}()

	listener := b.relay.Listener(10)
	defer listener.Close()

	tracker := dto.DataDirtyTracker{Users: true, Shares: false, Settings: true}
	eventBus.EmitDirtyData(events.DirtyDataEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, DataDirtyTracker: tracker})
	eventBus.EmitDirtyData(events.DirtyDataEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, DataDirtyTracker: tracker})
	eventBus.EmitDirtyData(events.DirtyDataEvent{Event: events.Event{Type: events.EventTypes.UPDATE}, DataDirtyTracker: dto.DataDirtyTracker{Users: true, Shares: true, Settings: true}})

	received := 0
	timeout := time.After(250 * time.Millisecond)
	for {
		select {
		case <-listener.Ch():
			received++
		case <-timeout:
			assert.Equal(t, 2, received)
			return
		}
	}
}
