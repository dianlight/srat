package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/websocket"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// TestNewHaWsService_NoWebSocketClient tests service creation without a websocket client
func TestNewHaWsService_NoWebSocketClient(t *testing.T) {
	ctx := context.Background()
	state := &dto.ContextState{}
	eventBus := events.NewEventBus(ctx)

	app := fxtest.New(t,
		fx.Provide(func() (context.Context, context.CancelFunc) { return context.WithCancel(ctx) }),
		fx.Supply(state),
		fx.Supply(fx.Annotate(eventBus, fx.As(new(events.EventBusInterface)))),
		fx.Provide(NewHaWsService),
		fx.Invoke(func(HaWsServiceInterface) {}),
	)

	app.RequireStart()
	app.RequireStop()
}

// TestNewHaWsService_WithWebSocketClient tests service creation with a websocket client
func TestNewHaWsService_WithWebSocketClient(t *testing.T) {
	ctrl := mock.NewMockController(t)
	mockClient := mock.Mock[websocket.ClientInterface](ctrl)

	// Setup mock expectations
	connEventHandler := make(chan func(websocket.ConnectionEvent), 1)
	mock.When(mockClient.SubscribeConnectionEvents(mock.Any[func(websocket.ConnectionEvent)]())).
		ThenAnswer(func(args []any) []any {
			handler := args[0].(func(websocket.ConnectionEvent))
			connEventHandler <- handler
			unsub := func() {}
			return []any{unsub, nil}
		})

	mock.When(mockClient.Connect(mock.Any[context.Context]())).ThenReturn(nil)

	ctx := context.Background()
	state := &dto.ContextState{}
	eventBus := events.NewEventBus(ctx)

	app := fxtest.New(t,
		fx.Provide(func() (context.Context, context.CancelFunc) { return context.WithCancel(ctx) }),
		fx.Supply(state),
		fx.Supply(fx.Annotate(eventBus, fx.As(new(events.EventBusInterface)))),
		fx.Supply(fx.Annotate(mockClient, fx.As(new(websocket.ClientInterface)))),
		fx.Provide(NewHaWsService),
		fx.Invoke(func(HaWsServiceInterface) {}),
	)

	app.RequireStart()

	// Wait for subscription to be set up
	select {
	case <-connEventHandler:
		// Handler was registered
	case <-time.After(2 * time.Second):
		t.Fatal("Connection event handler was not registered")
	}

	app.RequireStop()
}

// TestHaWsService_OnConnected tests the websocket connected event
func TestHaWsService_OnConnected(t *testing.T) {
	ctrl := mock.NewMockController(t)
	mockClient := mock.Mock[websocket.ClientInterface](ctrl)

	ctx := context.Background()
	state := &dto.ContextState{HACoreReady: false}
	eventBus := events.NewEventBus(ctx)

	// Track connection event handler
	var connectionHandler func(websocket.ConnectionEvent)
	mock.When(mockClient.SubscribeConnectionEvents(mock.Any[func(websocket.ConnectionEvent)]())).
		ThenAnswer(func(args []any) []any {
			connectionHandler = args[0].(func(websocket.ConnectionEvent))
			unsub := func() {}
			return []any{unsub, nil}
		})

	// Track event subscriptions
	var startedHandler func(json.RawMessage)
	var stoppedHandler func(json.RawMessage)
	mock.When(mockClient.SubscribeEvents(mock.Any[context.Context](), mock.AnyString(), mock.Any[func(json.RawMessage)]())).
		ThenAnswer(func(args []any) []any {
			eventType := args[1].(string)
			handler := args[2].(func(json.RawMessage))
			switch eventType {
			case "homeassistant_started":
				startedHandler = handler
			case "homeassistant_stop":
				stoppedHandler = handler
			}
			unsub := func() error { return nil }
			return []any{unsub, nil}
		})

	mock.When(mockClient.Connect(mock.Any[context.Context]())).ThenReturn(nil)

	// Subscribe to HomeAssistant events
	eventReceived := make(chan events.HomeAssistantEvent, 1)
	eventBus.OnHomeAssistant(func(ctx context.Context, event events.HomeAssistantEvent) {
		eventReceived <- event
	})

	app := fxtest.New(t,
		fx.Provide(func() (context.Context, context.CancelFunc) { return context.WithCancel(ctx) }),
		fx.Supply(state),
		fx.Supply(fx.Annotate(eventBus, fx.As(new(events.EventBusInterface)))),
		fx.Supply(fx.Annotate(mockClient, fx.As(new(websocket.ClientInterface)))),
		fx.Provide(NewHaWsService),
		fx.Invoke(func(HaWsServiceInterface) {}),
	)

	app.RequireStart()

	// Wait for handlers to be set up
	time.Sleep(100 * time.Millisecond)
	// First trigger connected event to set up subscriptions
	if connectionHandler != nil {
		connectionHandler(websocket.ConnectionEvent{Type: websocket.ConnEventConnected})
		// Drain the START event from connection
		select {
		case <-eventReceived:
		case <-time.After(100 * time.Millisecond):
		}
	}

	// Trigger connected event
	if connectionHandler != nil {
		connectionHandler(websocket.ConnectionEvent{Type: websocket.ConnEventConnected})
	}

	// Wait for event to be emitted
	select {
	case ev := <-eventReceived:
		assert.Equal(t, events.EventTypes.START, ev.Type)
		assert.True(t, state.HACoreReady)
	case <-time.After(2 * time.Second):
		t.Fatal("HomeAssistant START event was not emitted")
	}

	// Verify subscriptions were made
	assert.NotNil(t, startedHandler)
	assert.NotNil(t, stoppedHandler)

	app.RequireStop()
}

// TestHaWsService_OnDisconnected tests the websocket disconnected event
func TestHaWsService_OnDisconnected(t *testing.T) {
	ctrl := mock.NewMockController(t)
	mockClient := mock.Mock[websocket.ClientInterface](ctrl)

	ctx := context.Background()
	state := &dto.ContextState{HACoreReady: true}
	eventBus := events.NewEventBus(ctx)

	// Track connection event handler
	var connectionHandler func(websocket.ConnectionEvent)
	mock.When(mockClient.SubscribeConnectionEvents(mock.Any[func(websocket.ConnectionEvent)]())).
		ThenAnswer(func(args []any) []any {
			connectionHandler = args[0].(func(websocket.ConnectionEvent))
			unsub := func() {}
			return []any{unsub, nil}
		})

	// Setup subscription mocks
	mock.When(mockClient.SubscribeEvents(mock.Any[context.Context](), mock.AnyString(), mock.Any[func(json.RawMessage)]())).
		ThenAnswer(func(args []any) []any {
			unsub := func() error { return nil }
			return []any{unsub, nil}
		})

	mock.When(mockClient.Connect(mock.Any[context.Context]())).ThenReturn(nil)

	// Subscribe to HomeAssistant events
	eventReceived := make(chan events.HomeAssistantEvent, 1)
	eventBus.OnHomeAssistant(func(ctx context.Context, event events.HomeAssistantEvent) {
		eventReceived <- event
	})

	app := fxtest.New(t,
		fx.Provide(func() (context.Context, context.CancelFunc) { return context.WithCancel(ctx) }),
		fx.Supply(state),
		fx.Supply(fx.Annotate(eventBus, fx.As(new(events.EventBusInterface)))),
		fx.Supply(fx.Annotate(mockClient, fx.As(new(websocket.ClientInterface)))),
		fx.Provide(NewHaWsService),
		fx.Invoke(func(HaWsServiceInterface) {}),
	)

	app.RequireStart()

	// Wait for handlers to be set up
	time.Sleep(100 * time.Millisecond)

	// Trigger disconnected event
	if connectionHandler != nil {
		connectionHandler(websocket.ConnectionEvent{Type: websocket.ConnEventDisconnected})
	}

	// Wait for event to be emitted
	select {
	case ev := <-eventReceived:
		assert.Equal(t, events.EventTypes.STOP, ev.Type)
		assert.False(t, state.HACoreReady)
	case <-time.After(2 * time.Second):
		t.Fatal("HomeAssistant STOP event was not emitted")
	}

	app.RequireStop()
}

// TestHaWsService_OnHaStarted tests the homeassistant_started event
func TestHaWsService_OnHaStarted(t *testing.T) {
	// Track connection event handler
	var connectionHandler func(websocket.ConnectionEvent)
	ctrl := mock.NewMockController(t)
	mockClient := mock.Mock[websocket.ClientInterface](ctrl)

	ctx := context.Background()
	state := &dto.ContextState{HACoreReady: false}
	eventBus := events.NewEventBus(ctx)

	// Track event handlers
	var startedHandler func(json.RawMessage)
	mock.When(mockClient.SubscribeConnectionEvents(mock.Any[func(websocket.ConnectionEvent)]())).
		ThenAnswer(func(args []any) []any {
			connectionHandler = args[0].(func(websocket.ConnectionEvent))
			unsub := func() {}
			return []any{unsub, nil}
		})

	mock.When(mockClient.SubscribeEvents(mock.Any[context.Context](), mock.AnyString(), mock.Any[func(json.RawMessage)]())).
		ThenAnswer(func(args []any) []any {
			eventType := args[1].(string)
			if eventType == "homeassistant_started" {
				startedHandler = args[2].(func(json.RawMessage))
			}
			unsub := func() error { return nil }
			return []any{unsub, nil}
		})

	mock.When(mockClient.Connect(mock.Any[context.Context]())).ThenReturn(nil)

	// Subscribe to HomeAssistant events
	eventReceived := make(chan events.HomeAssistantEvent, 1)
	eventBus.OnHomeAssistant(func(ctx context.Context, event events.HomeAssistantEvent) {
		eventReceived <- event
	})

	app := fxtest.New(t,
		fx.Provide(func() (context.Context, context.CancelFunc) { return context.WithCancel(ctx) }),
		fx.Supply(state),
		fx.Supply(fx.Annotate(eventBus, fx.As(new(events.EventBusInterface)))),
		fx.Supply(fx.Annotate(mockClient, fx.As(new(websocket.ClientInterface)))),
		fx.Provide(NewHaWsService),
		fx.Invoke(func(HaWsServiceInterface) {}),
	)

	app.RequireStart()

	// Wait for handlers to be set up
	time.Sleep(100 * time.Millisecond)

	// Ensure connection is established so that event subscriptions are registered
	if connectionHandler != nil {
		connectionHandler(websocket.ConnectionEvent{Type: websocket.ConnEventConnected})
		// Drain the initial START event emitted on connection
		select {
		case <-eventReceived:
		case <-time.After(200 * time.Millisecond):
		}
	}

	// Trigger homeassistant_started event
	if assert.NotNil(t, startedHandler, "startedHandler should be registered after connection") {
		startedHandler(json.RawMessage(`{}`))
	}

	// Wait for event to be emitted
	select {
	case ev := <-eventReceived:
		assert.Equal(t, events.EventTypes.START, ev.Type)
		assert.True(t, state.HACoreReady)
	case <-time.After(2 * time.Second):
		t.Fatal("HomeAssistant START event was not emitted")
	}

	app.RequireStop()
}

// TestHaWsService_OnHaStopped tests the homeassistant_stop event
func TestHaWsService_OnHaStopped(t *testing.T) {
	// Track connection event handler
	var connectionHandler func(websocket.ConnectionEvent)
	ctrl := mock.NewMockController(t)
	mockClient := mock.Mock[websocket.ClientInterface](ctrl)

	ctx := context.Background()
	state := &dto.ContextState{HACoreReady: true}
	eventBus := events.NewEventBus(ctx)

	// Track event handlers
	var stoppedHandler func(json.RawMessage)
	mock.When(mockClient.SubscribeConnectionEvents(mock.Any[func(websocket.ConnectionEvent)]())).
		ThenAnswer(func(args []any) []any {
			connectionHandler = args[0].(func(websocket.ConnectionEvent))
			unsub := func() {}
			return []any{unsub, nil}
		})

	mock.When(mockClient.SubscribeEvents(mock.Any[context.Context](), mock.AnyString(), mock.Any[func(json.RawMessage)]())).
		ThenAnswer(func(args []any) []any {
			eventType := args[1].(string)
			if eventType == "homeassistant_stop" {
				stoppedHandler = args[2].(func(json.RawMessage))
			}
			unsub := func() error { return nil }
			return []any{unsub, nil}
		})

	mock.When(mockClient.Connect(mock.Any[context.Context]())).ThenReturn(nil)

	// Subscribe to HomeAssistant events
	eventReceived := make(chan events.HomeAssistantEvent, 1)
	eventBus.OnHomeAssistant(func(ctx context.Context, event events.HomeAssistantEvent) {
		eventReceived <- event
	})

	app := fxtest.New(t,
		fx.Provide(func() (context.Context, context.CancelFunc) { return context.WithCancel(ctx) }),
		fx.Supply(state),
		fx.Supply(fx.Annotate(eventBus, fx.As(new(events.EventBusInterface)))),
		fx.Supply(fx.Annotate(mockClient, fx.As(new(websocket.ClientInterface)))),
		fx.Provide(NewHaWsService),
		fx.Invoke(func(HaWsServiceInterface) {}),
	)

	app.RequireStart()

	// Wait for handlers to be set up
	time.Sleep(100 * time.Millisecond)

	// Ensure connection is established so that event subscriptions are registered
	if connectionHandler != nil {
		connectionHandler(websocket.ConnectionEvent{Type: websocket.ConnEventConnected})
		// Drain the initial START event emitted on connection
		select {
		case <-eventReceived:
		case <-time.After(200 * time.Millisecond):
		}
	}

	// Trigger homeassistant_stop event
	if assert.NotNil(t, stoppedHandler, "stoppedHandler should be registered after connection") {
		stoppedHandler(json.RawMessage(`{}`))
	}

	// Wait for event to be emitted
	select {
	case ev := <-eventReceived:
		assert.Equal(t, events.EventTypes.STOP, ev.Type)
		assert.False(t, state.HACoreReady)
	case <-time.After(2 * time.Second):
		t.Fatal("HomeAssistant STOP event was not emitted")
	}

	app.RequireStop()
}

// TestHaWsService_ConnectAndDisconnectSequence tests a full connect/disconnect sequence
func TestHaWsService_ConnectAndDisconnectSequence(t *testing.T) {
	ctrl := mock.NewMockController(t)
	mockClient := mock.Mock[websocket.ClientInterface](ctrl)

	ctx := context.Background()
	state := &dto.ContextState{HACoreReady: false}
	eventBus := events.NewEventBus(ctx)

	// Track connection event handler
	var connectionHandler func(websocket.ConnectionEvent)
	mock.When(mockClient.SubscribeConnectionEvents(mock.Any[func(websocket.ConnectionEvent)]())).
		ThenAnswer(func(args []any) []any {
			connectionHandler = args[0].(func(websocket.ConnectionEvent))
			unsub := func() {}
			return []any{unsub, nil}
		})

	// Setup subscription mocks
	mock.When(mockClient.SubscribeEvents(mock.Any[context.Context](), mock.AnyString(), mock.Any[func(json.RawMessage)]())).
		ThenAnswer(func(args []any) []any {
			unsub := func() error { return nil }
			return []any{unsub, nil}
		})

	mock.When(mockClient.Connect(mock.Any[context.Context]())).ThenReturn(nil)

	// Subscribe to HomeAssistant events
	eventReceived := make(chan events.HomeAssistantEvent, 10)
	eventBus.OnHomeAssistant(func(ctx context.Context, event events.HomeAssistantEvent) {
		eventReceived <- event
	})

	app := fxtest.New(t,
		fx.Provide(func() (context.Context, context.CancelFunc) { return context.WithCancel(ctx) }),
		fx.Supply(state),
		fx.Supply(fx.Annotate(eventBus, fx.As(new(events.EventBusInterface)))),
		fx.Supply(fx.Annotate(mockClient, fx.As(new(websocket.ClientInterface)))),
		fx.Provide(NewHaWsService),
		fx.Invoke(func(HaWsServiceInterface) {}),
	)

	app.RequireStart()

	// Wait for handlers to be set up
	time.Sleep(100 * time.Millisecond)

	// Sequence: Connect -> Disconnect -> Reconnect
	if connectionHandler != nil {
		// 1. Connected
		connectionHandler(websocket.ConnectionEvent{Type: websocket.ConnEventConnected})
		ev1 := <-eventReceived
		assert.Equal(t, events.EventTypes.START, ev1.Type)
		assert.True(t, state.HACoreReady)

		// 2. Disconnected
		connectionHandler(websocket.ConnectionEvent{Type: websocket.ConnEventDisconnected})
		ev2 := <-eventReceived
		assert.Equal(t, events.EventTypes.STOP, ev2.Type)
		assert.False(t, state.HACoreReady)

		// 3. Reconnected
		connectionHandler(websocket.ConnectionEvent{Type: websocket.ConnEventConnected})
		ev3 := <-eventReceived
		assert.Equal(t, events.EventTypes.START, ev3.Type)
		assert.True(t, state.HACoreReady)
	}

	app.RequireStop()
}

// TestHaWsService_ConnectError tests service behavior when connection fails
func TestHaWsService_ConnectError(t *testing.T) {
	ctrl := mock.NewMockController(t)
	mockClient := mock.Mock[websocket.ClientInterface](ctrl)

	ctx := context.Background()
	state := &dto.ContextState{}
	eventBus := events.NewEventBus(ctx)

	// Mock connect to fail
	mock.When(mockClient.SubscribeConnectionEvents(mock.Any[func(websocket.ConnectionEvent)]())).
		ThenAnswer(func(args []any) []any {
			unsub := func() {}
			return []any{unsub, nil}
		})
	mock.When(mockClient.Connect(mock.Any[context.Context]())).ThenReturn(assert.AnError)

	lc := fxtest.NewLifecycle(t)
	params := HaWsServiceParams{
		Ctx:      ctx,
		State:    state,
		WsClient: mockClient,
		EventBus: eventBus,
	}

	_, err := NewHaWsService(lc, params)
	assert.NoError(t, err)

	// Should fail to start due to connection error
	err = lc.Start(context.Background())
	assert.Error(t, err)
}

// TestHaWsService_UnsubscribeOnStop tests that event subscriptions are cleaned up on stop
func TestHaWsService_UnsubscribeOnStop(t *testing.T) {
	ctrl := mock.NewMockController(t)
	mockClient := mock.Mock[websocket.ClientInterface](ctrl)

	ctx := context.Background()
	state := &dto.ContextState{HACoreReady: false}
	eventBus := events.NewEventBus(ctx)

	// Track connection event handler
	var connectionHandler func(websocket.ConnectionEvent)
	unsubConnCalled := false
	mock.When(mockClient.SubscribeConnectionEvents(mock.Any[func(websocket.ConnectionEvent)]())).
		ThenAnswer(func(args []any) []any {
			connectionHandler = args[0].(func(websocket.ConnectionEvent))
			unsub := func() {
				unsubConnCalled = true
			}
			return []any{unsub, nil}
		})

	// Track unsubscribe calls
	unsubStartedCalled := false
	unsubStoppedCalled := false
	mock.When(mockClient.SubscribeEvents(mock.Any[context.Context](), mock.AnyString(), mock.Any[func(json.RawMessage)]())).
		ThenAnswer(func(args []any) []any {
			eventType := args[1].(string)
			unsub := func() error {
				switch eventType {
				case "homeassistant_started":
					unsubStartedCalled = true
				case "homeassistant_stop":
					unsubStoppedCalled = true
				}
				return nil
			}
			return []any{unsub, nil}
		})

	mock.When(mockClient.Connect(mock.Any[context.Context]())).ThenReturn(nil)

	app := fxtest.New(t,
		fx.Provide(func() (context.Context, context.CancelFunc) { return context.WithCancel(ctx) }),
		fx.Supply(state),
		fx.Supply(fx.Annotate(eventBus, fx.As(new(events.EventBusInterface)))),
		fx.Supply(fx.Annotate(mockClient, fx.As(new(websocket.ClientInterface)))),
		fx.Provide(NewHaWsService),
		fx.Invoke(func(HaWsServiceInterface) {}),
	)

	app.RequireStart()

	// Wait for handlers to be set up
	time.Sleep(100 * time.Millisecond)

	// Trigger connected to set up subscriptions
	if connectionHandler != nil {
		connectionHandler(websocket.ConnectionEvent{Type: websocket.ConnEventConnected})
	}

	// Wait for subscriptions
	time.Sleep(100 * time.Millisecond)

	// Stop the app
	app.RequireStop()

	// Verify all unsubscribe functions were called
	assert.True(t, unsubStartedCalled, "homeassistant_started unsubscribe should be called")
	assert.True(t, unsubStoppedCalled, "homeassistant_stop unsubscribe should be called")
	assert.True(t, unsubConnCalled, "connection events unsubscribe should be called")
}

// TestHaWsService_MultipleConnectionEvents tests handling multiple connection events
func TestHaWsService_MultipleConnectionEvents(t *testing.T) {
	ctrl := mock.NewMockController(t)
	mockClient := mock.Mock[websocket.ClientInterface](ctrl)

	ctx := context.Background()
	state := &dto.ContextState{HACoreReady: false}
	eventBus := events.NewEventBus(ctx)

	// Track connection event handler
	var connectionHandler func(websocket.ConnectionEvent)
	mock.When(mockClient.SubscribeConnectionEvents(mock.Any[func(websocket.ConnectionEvent)]())).
		ThenAnswer(func(args []any) []any {
			connectionHandler = args[0].(func(websocket.ConnectionEvent))
			unsub := func() {}
			return []any{unsub, nil}
		})

	// Setup subscription mocks
	mock.When(mockClient.SubscribeEvents(mock.Any[context.Context](), mock.AnyString(), mock.Any[func(json.RawMessage)]())).
		ThenAnswer(func(args []any) []any {
			unsub := func() error { return nil }
			return []any{unsub, nil}
		})

	mock.When(mockClient.Connect(mock.Any[context.Context]())).ThenReturn(nil)

	// Subscribe to HomeAssistant events
	eventReceived := make(chan events.HomeAssistantEvent, 10)
	eventBus.OnHomeAssistant(func(ctx context.Context, event events.HomeAssistantEvent) {
		eventReceived <- event
	})

	app := fxtest.New(t,
		fx.Provide(func() (context.Context, context.CancelFunc) { return context.WithCancel(ctx) }),
		fx.Supply(state),
		fx.Supply(fx.Annotate(eventBus, fx.As(new(events.EventBusInterface)))),
		fx.Supply(fx.Annotate(mockClient, fx.As(new(websocket.ClientInterface)))),
		fx.Provide(NewHaWsService),
		fx.Invoke(func(HaWsServiceInterface) {}),
	)

	app.RequireStart()

	// Wait for handlers to be set up
	time.Sleep(100 * time.Millisecond)

	// Send multiple connected events in quick succession
	if connectionHandler != nil {
		connectionHandler(websocket.ConnectionEvent{Type: websocket.ConnEventConnected})
		connectionHandler(websocket.ConnectionEvent{Type: websocket.ConnEventConnected})
		connectionHandler(websocket.ConnectionEvent{Type: websocket.ConnEventConnected})
	}

	// Collect events with timeout
	eventsCollected := 0
	timeout := time.After(2 * time.Second)
collectLoop:
	for {
		select {
		case ev := <-eventReceived:
			assert.Equal(t, events.EventTypes.START, ev.Type)
			eventsCollected++
			if eventsCollected >= 3 {
				break collectLoop
			}
		case <-timeout:
			break collectLoop
		}
	}

	// Should have received all 3 events
	assert.Equal(t, 3, eventsCollected)
	assert.True(t, state.HACoreReady)

	app.RequireStop()
}
