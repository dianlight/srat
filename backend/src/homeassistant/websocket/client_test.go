package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gorilla "github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// startTestServer starts a local websocket server that understands a tiny subset of
// Home Assistant websocket protocol for testing helpers: get_states, get_config,
// call_service, subscribe_events.
func startTestServer(t *testing.T) *httptest.Server {
	upgrader := gorilla.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer c.Close()

		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			t.Log("<--" + string(msg))
			// parse incoming JSON
			var m map[string]any
			if err := json.Unmarshal(msg, &m); err != nil {
				// not JSON, echo back
				_ = c.WriteMessage(gorilla.TextMessage, msg)
				continue
			}

			typ, _ := m["type"].(string)
			idf := m["id"]

			var id float64
			if idf != nil {
				id, _ = idf.(float64)
			}

			switch typ {
			case "get_states":
				resp := map[string]any{"id": id, "type": "result", "success": true, "result": []map[string]any{{"entity_id": "light.test", "state": "on"}}}
				t.Logf("<-- %s", string(msg))
				_ = c.WriteJSON(resp)
			case "get_config":
				resp := map[string]any{"id": id, "type": "result", "success": true, "result": map[string]any{"version": "test"}}
				t.Logf("<-- %s", string(msg))
				_ = c.WriteJSON(resp)
			case "call_service":
				resp := map[string]any{"id": id, "type": "result", "success": true, "result": nil}
				t.Logf("<-- %s", string(msg))
				_ = c.WriteJSON(resp)
			case "subscribe_events":
				// send subscribe success
				resp := map[string]any{"id": id, "type": "result", "success": true, "result": nil}
				t.Logf("<-- %s", string(msg))
				_ = c.WriteJSON(resp)
				// then periodically send an event
				go func() {
					time.Sleep(50 * time.Millisecond)
					ev := map[string]any{"type": "event", "event": map[string]any{"event_type": m["event_type"], "data": map[string]any{"hello": "world"}}}
					t.Logf("<-- %s", string(msg))
					_ = c.WriteJSON(ev)
				}()
			default:
				// unknown - send generic result if id present
				if idf != nil {
					resp := map[string]any{"id": id, "type": "result", "success": true, "result": nil}
					t.Logf("<-- %s", string(msg))
					_ = c.WriteJSON(resp)
				}
			}
		}
	}))

	return srv
}

func TestConnectSendReceive(t *testing.T) {
	srv := startTestServer(t)
	defer srv.Close()

	t.Log(srv.URL)
	// convert http://127.0.0.1 -> ws://127.0.0.1
	wsURL := "ws" + srv.URL[len("http"):] + "/" // keep trailing slash

	c := NewClient(wsURL, "")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := c.Connect(ctx)
	require.NoError(t, err)
	defer c.Close()

	// send a text message
	msg := []byte("hello-ha")
	err = c.Send(gorilla.TextMessage, msg)
	require.NoError(t, err)

	select {
	case got := <-c.Receive():
		require.Equal(t, msg, got)
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for message")
	}
}

func TestCloseStopsReceive(t *testing.T) {
	srv := startTestServer(t)
	defer srv.Close()

	wsURL := "ws" + srv.URL[len("http"):] + "/"
	c := NewClient(wsURL, "")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	require.NoError(t, c.Connect(ctx))

	// close and ensure receive channel is closed
	require.NoError(t, c.Close())

	// draining from Receive should not block forever; channel is closed
	select {
	case _, ok := <-c.Receive():
		require.False(t, ok)
	case <-time.After(1 * time.Second):
		t.Fatalf("receive channel not closed")
	}
}

func TestHelpers_CallGetAndSubscribe(t *testing.T) {
	srv := startTestServer(t)
	defer srv.Close()

	wsURL := "ws" + srv.URL[len("http"):] + "/"
	c := NewClient(wsURL, "")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	require.NoError(t, c.Connect(ctx))
	defer c.Close()

	// GetStates
	states, err := c.GetStates(ctx)
	require.NoError(t, err)
	require.Len(t, states, 1)

	// GetConfig
	cfg, err := c.GetConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, "test", cfg["version"])

	// CallService
	err = c.CallService(ctx, "light", "turn_on", map[string]any{"entity_id": "light.test"})
	require.NoError(t, err)

	// SubscribeEvents
	got := make(chan json.RawMessage, 1)
	unsub, err := c.SubscribeEvents(ctx, "state_changed", func(ev json.RawMessage) {
		got <- ev
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case ev := <-got:
		// ensure event contains data
		var m map[string]any
		require.NoError(t, json.Unmarshal(ev, &m))
		require.Equal(t, "state_changed", m["event_type"])
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for event")
	}
}
