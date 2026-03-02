package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	gorilla "github.com/gorilla/websocket"
)

// ClientInterface defines the methods for a Home Assistant websocket client.
type ClientInterface interface {
	Connect(ctx context.Context) error
	Send(messageType int, data []byte) error
	CallService(ctx context.Context, domain, service string, serviceData map[string]any) error
	GetStates(ctx context.Context) ([]map[string]any, error)
	SubscribeEvents(ctx context.Context, eventType string, handler func(json.RawMessage)) (func() error, error)
	GetConfig(ctx context.Context) (map[string]any, error)
	Receive() <-chan []byte
	Close() error
	SubscribeConnectionEvents(handler func(ConnectionEvent)) (func(), error)
}

// Client is a minimal Home Assistant websocket client.
// It intentionally ignores authentication and accepts a configurable endpoint URL.
// It uses github.com/gorilla/websocket under the hood.
type Client struct {
	endpoint string
	// supervisor token to present as Bearer Authorization header during handshake
	supervisorToken string

	connMu sync.RWMutex
	conn   *gorilla.Conn

	// read channel for incoming messages
	recvCh chan []byte

	// request/response handling
	idMu    sync.Mutex
	nextID  int
	pending map[int]chan json.RawMessage
	pendMu  sync.Mutex

	// event subscribers keyed by subscription request id
	subs   map[int]func(json.RawMessage)
	subsMu sync.RWMutex

	// connection event subscribers (id -> handler)
	connEventMu     sync.RWMutex
	connEvents      map[int]func(event ConnectionEvent)
	connNextEventID int

	// close signaling
	closed    chan struct{}
	closeOnce sync.Once
	// reconnect control
	reconnectCancel func()
	reconnectMu     sync.Mutex
}

// ConnectionEventType indicates the lifecycle event for the websocket connection.
type ConnectionEventType int

const (
	ConnEventConnected ConnectionEventType = iota
	ConnEventDisconnected
	ConnEventRetrying
)

// ConnectionEvent carries information about a connection lifecycle event.
type ConnectionEvent struct {
	Type    ConnectionEventType
	Message string        // optional human readable message (e.g. error or retry info)
	Attempt int           // retry attempt (1-based)
	Backoff time.Duration // current backoff duration
}

// NewClient creates a client that will connect to the given websocket endpoint URL.
// The URL should be a full websocket URL (ws:// or wss://).
func NewClient(endpoint, supervisorToken string) ClientInterface {
	return &Client{
		endpoint:        endpoint + "core/websocket",
		supervisorToken: supervisorToken,
		recvCh:          make(chan []byte, 16),
		closed:          make(chan struct{}),
		pending:         make(map[int]chan json.RawMessage),
		subs:            make(map[int]func(json.RawMessage)),
		connEvents:      make(map[int]func(event ConnectionEvent)),
	}
}

// Connect dials the websocket endpoint and starts the reader loop.
func (c *Client) Connect(ctx context.Context) error {
	c.reconnectMu.Lock()
	if c.reconnectCancel != nil {
		// already started
		c.reconnectMu.Unlock()
		return nil
	}
	connectCtx, cancel := context.WithCancel(context.Background())
	c.reconnectCancel = cancel
	c.reconnectMu.Unlock()

	// channel to notify first successful connection
	connected := make(chan struct{})

	go c.connectLoop(connectCtx, connected)

	// wait for first successful connection or ctx done
	select {
	case <-connected:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closed:
		return errors.New("client closed")
	}
}

// connectLoop runs in background and attempts to maintain a connection.
// It signals on connectedCh when the first successful connection is established.
func (c *Client) connectLoop(ctx context.Context, connectedCh chan<- struct{}) {
	defer func() {
		// clear reconnectCancel
		c.reconnectMu.Lock()
		c.reconnectCancel = nil
		c.reconnectMu.Unlock()
	}()

	first := true
	backoff := time.Second
	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closed:
			return
		default:
		}

		u, err := url.Parse(c.endpoint)
		if err != nil {
			// invalid endpoint, nothing to do
			return
		}

		dialer := gorilla.DefaultDialer
		var reqHeader http.Header
		if c.supervisorToken != "" {
			reqHeader = http.Header{}
			reqHeader.Set("Authorization", "Bearer "+c.supervisorToken)
		}
		conn, _, err := dialer.DialContext(ctx, u.String(), reqHeader)
		if err != nil {
			slog.Warn("websocket connect failed", "url", u, "error", err)
			// wait with backoff and retry
			attempt++
			// notify retrying handlers with attempt and backoff
			c.connEventMu.RLock()
			for _, h := range c.connEvents {
				go h(ConnectionEvent{Type: ConnEventRetrying, Message: err.Error(), Attempt: attempt, Backoff: backoff})
			}
			c.connEventMu.RUnlock()

			select {
			case <-time.After(backoff):
				// exponential backoff up to 60s
				backoff *= 2
				if backoff > 60*time.Second {
					backoff = 60 * time.Second
				}
				continue
			case <-ctx.Done():
				return
			case <-c.closed:
				return
			}
		}

		// reset backoff/attempt after success
		backoff = time.Second
		attempt = 0

		c.connMu.Lock()
		c.conn = conn
		c.connMu.Unlock()

		if c.supervisorToken != "" {
			// perform Home Assistant websocket auth handshake
			// read the initial message which should be an auth/hello message
			var helloMsg map[string]json.RawMessage
			// set a short deadline for the handshake
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			_, msg, err := conn.ReadMessage()
			if err != nil {
				_ = conn.Close()
				slog.Warn("failed to read hello from websocket", "error", err)
				// continue to retry
				continue
			}
			if err := json.Unmarshal(msg, &helloMsg); err != nil {
				_ = conn.Close()
				slog.Warn("invalid hello message from websocket", "error", err)
				continue
			}

			// default: proceed if no auth required
			needAuth := false
			if tRaw, ok := helloMsg["type"]; ok {
				var t string
				if err := json.Unmarshal(tRaw, &t); err == nil && t == "auth_required" {
					needAuth = true
				}
			}

			if needAuth {
				// send auth payload. If supervisorToken available, use it as access_token
				authPayload := map[string]any{"type": "auth"}
				if c.supervisorToken != "" {
					authPayload["access_token"] = c.supervisorToken
				}
				// write auth
				if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err == nil {
					// ignore SetWriteDeadline error; do the write
				}
				if err := conn.WriteJSON(authPayload); err != nil {
					_ = conn.Close()
					slog.Warn("failed to send auth payload", "error", err)
					continue
				}

				// wait for auth_ok or auth_invalid
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				_, msg, err := conn.ReadMessage()
				if err != nil {
					_ = conn.Close()
					slog.Warn("failed to read auth response", "error", err)
					continue
				}
				var authResp map[string]json.RawMessage
				if err := json.Unmarshal(msg, &authResp); err != nil {
					_ = conn.Close()
					slog.Warn("invalid auth response", "error", err)
					continue
				}
				if tRaw, ok := authResp["type"]; ok {
					var t string
					if err := json.Unmarshal(tRaw, &t); err == nil {
						if t == "auth_ok" {
							// success, proceed
						} else {
							// auth_invalid or other
							_ = conn.Close()
							slog.Warn("authentication failed", "type", t)
							continue
						}
					}
				} else {
					_ = conn.Close()
					slog.Warn("auth response missing type")
					continue
				}
			}
		}
		// signal first connect
		if first {
			first = false
			close(connectedCh)
		}

		// notify connected handlers (Attempt and Backoff may be zero)
		c.connEventMu.RLock()
		for _, h := range c.connEvents {
			go h(ConnectionEvent{Type: ConnEventConnected, Message: "connected", Attempt: attempt, Backoff: backoff})
		}
		c.connEventMu.RUnlock()

		// run readLoop and wait until it returns (connection closed)
		done := make(chan struct{})
		go func() {
			c.readLoop()
			close(done)
		}()

		// wait for either readLoop to finish, or ctx/closed
		select {
		case <-done:
			// connection lost; cleanup and notify pending requests of failure
			// inform disconnect handlers
			c.connEventMu.RLock()
			for _, h := range c.connEvents {
				go h(ConnectionEvent{Type: ConnEventDisconnected, Message: "connection lost", Attempt: attempt, Backoff: backoff})
			}
			c.connEventMu.RUnlock()
			c.connMu.Lock()
			if c.conn != nil {
				_ = c.conn.Close()
				c.conn = nil
			}
			c.connMu.Unlock()

			// notify pending requests with failure result
			c.pendMu.Lock()
			for id, ch := range c.pending {
				// best-effort: send a failure result JSON
				msg := json.RawMessage([]byte(fmt.Sprintf(`{"id":%d,"type":"result","success":false}`, id)))
				select {
				case ch <- msg:
				default:
				}
			}
			c.pendMu.Unlock()

			// loop to reconnect
			continue
		case <-ctx.Done():
			return
		case <-c.closed:
			return
		}
	}
}

// Send sends a raw message (text or binary) to the websocket.
func (c *Client) Send(messageType int, data []byte) error {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	if c.conn == nil {
		return errors.New("not connected")
	}

	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.conn.WriteMessage(messageType, data)
}

// writeJSON writes v as JSON to the websocket connection.
func (c *Client) writeJSON(v any) error {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	if c.conn == nil {
		return errors.New("not connected")
	}
	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.conn.WriteJSON(v)
}

// doRequest sends a request and waits for the response (by id) or context done.
func (c *Client) doRequest(ctx context.Context, payload map[string]any) (json.RawMessage, error) {
	// allocate id
	c.idMu.Lock()
	c.nextID++
	id := c.nextID
	c.idMu.Unlock()

	payload["id"] = id

	respCh := make(chan json.RawMessage, 1)
	c.pendMu.Lock()
	c.pending[id] = respCh
	c.pendMu.Unlock()

	// ensure pending is cleaned up
	defer func() {
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
	}()

	if err := c.writeJSON(payload); err != nil {
		return nil, err
	}

	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closed:
		return nil, errors.New("client closed")
	}
}

// CallService calls a service on Home Assistant. It waits for a result and returns an error when the call failed.
func (c *Client) CallService(ctx context.Context, domain, service string, serviceData map[string]any) error {
	payload := map[string]any{
		"type":         "call_service",
		"domain":       domain,
		"service":      service,
		"service_data": serviceData,
	}

	resp, err := c.doRequest(ctx, payload)
	if err != nil {
		return err
	}

	var r struct {
		Success bool            `json:"success"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		return err
	}
	if !r.Success {
		return errors.New("service call failed")
	}
	return nil
}

// GetStates requests all entity states and returns raw JSON-decoded slice.
func (c *Client) GetStates(ctx context.Context) ([]map[string]any, error) {
	payload := map[string]any{"type": "get_states"}
	resp, err := c.doRequest(ctx, payload)
	if err != nil {
		return nil, err
	}
	var r struct {
		Success bool             `json:"success"`
		Result  []map[string]any `json:"result"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, err
	}
	if !r.Success {
		return nil, errors.New("get_states failed")
	}
	return r.Result, nil
}

// GetConfig requests Home Assistant config.
func (c *Client) GetConfig(ctx context.Context) (map[string]any, error) {
	payload := map[string]any{"type": "get_config"}
	resp, err := c.doRequest(ctx, payload)
	if err != nil {
		return nil, err
	}
	var r struct {
		Success bool           `json:"success"`
		Result  map[string]any `json:"result"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, err
	}
	if !r.Success {
		return nil, errors.New("get_config failed")
	}
	return r.Result, nil
}

// SubscribeEvents subscribes to events of the given type (e.g. "state_changed").
// The handler will be invoked for each event. The returned function unsubscribes.
func (c *Client) SubscribeEvents(ctx context.Context, eventType string, handler func(json.RawMessage)) (func() error, error) {
	payload := map[string]any{"type": "subscribe_events", "event_type": eventType}

	// allocate id and store handler after successful subscribe
	c.idMu.Lock()
	c.nextID++
	id := c.nextID
	c.idMu.Unlock()

	payload["id"] = id

	respCh := make(chan json.RawMessage, 1)
	c.pendMu.Lock()
	c.pending[id] = respCh
	c.pendMu.Unlock()

	// send subscribe
	if err := c.writeJSON(payload); err != nil {
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
		return nil, err
	}

	// wait for subscribe result
	select {
	case resp := <-respCh:
		var r struct {
			Success bool `json:"success"`
		}
		if err := json.Unmarshal(resp, &r); err != nil {
			return nil, err
		}
		if !r.Success {
			return nil, errors.New("subscribe failed")
		}
		// register handler
		c.subsMu.Lock()
		c.subs[id] = handler
		c.subsMu.Unlock()

		// unsubscribe function
		unsub := func() error {
			// remove local handler
			c.subsMu.Lock()
			delete(c.subs, id)
			c.subsMu.Unlock()
			// send unsubscribe request
			payload := map[string]any{"type": "unsubscribe_events", "id": id}
			// best-effort
			_ = c.writeJSON(payload)
			return nil
		}
		return unsub, nil
	case <-ctx.Done():
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
		return nil, ctx.Err()
	case <-c.closed:
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
		return nil, errors.New("client closed")
	}
}

// Receive returns a channel with incoming raw messages (text frames as bytes).
// The channel is closed when the client is closed.
func (c *Client) Receive() <-chan []byte {
	return c.recvCh
}

// Close closes the websocket connection and stops background goroutines.
func (c *Client) Close() error {
	var retErr error
	c.closeOnce.Do(func() {
		close(c.closed)
		// cancel reconnect loop if running
		c.reconnectMu.Lock()
		if c.reconnectCancel != nil {
			c.reconnectCancel()
			c.reconnectCancel = nil
		}
		c.reconnectMu.Unlock()
		c.connMu.Lock()
		if c.conn != nil {
			retErr = c.conn.Close()
			c.conn = nil
		}
		c.connMu.Unlock()
		close(c.recvCh)
	})
	return retErr
}

func (c *Client) readLoop() {
	for {
		select {
		case <-c.closed:
			return
		default:
		}

		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()
		if conn == nil {
			return
		}

		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			// on read error, close and exit
			c.Close()
			return
		}

		// Only forward text or binary frames. Try to parse as JSON and dispatch
		if mt == gorilla.TextMessage || mt == gorilla.BinaryMessage {
			// send raw to recvCh non-blocking
			select {
			case c.recvCh <- msg:
			default:
			}

			var m map[string]json.RawMessage
			if err := json.Unmarshal(msg, &m); err != nil {
				// not JSON, continue
				continue
			}

			// if message contains id, it's a response to a request
			if idRaw, ok := m["id"]; ok {
				var id int
				if err := json.Unmarshal(idRaw, &id); err == nil {
					c.pendMu.Lock()
					ch, ok := c.pending[id]
					c.pendMu.Unlock()
					if ok {
						// send the entire raw message so the requester can unmarshal fields like success/result
						select {
						case ch <- json.RawMessage(msg):
						default:
						}
						continue
					}
				}
			}

			// handle event messages: type = "event" and contains "event" field with "event_type"
			if tRaw, ok := m["type"]; ok {
				var t string
				if err := json.Unmarshal(tRaw, &t); err == nil && t == "event" {
					// call all subscribers with the raw event object
					if evRaw, ok := m["event"]; ok {
						c.subsMu.RLock()
						for _, h := range c.subs {
							// call handler in goroutine to avoid blocking
							go h(evRaw)
						}
						c.subsMu.RUnlock()
					}
				}
			}
		}
	}
}

// SubscribeConnectionEvents registers a handler that will be called for connection lifecycle events.
// It returns an unsubscribe function which removes the handler. This is safe for concurrent use.
func (c *Client) SubscribeConnectionEvents(handler func(ConnectionEvent)) (func(), error) {
	if handler == nil {
		return nil, errors.New("handler cannot be nil")
	}
	c.connEventMu.Lock()
	c.connNextEventID++
	id := c.connNextEventID
	c.connEvents[id] = handler
	c.connEventMu.Unlock()

	unsub := func() bool {
		c.connEventMu.Lock()
		defer c.connEventMu.Unlock()
		if _, ok := c.connEvents[id]; !ok {
			return false
		}
		delete(c.connEvents, id)
		return true
	}
	// wrap to match previous signature returning func()
	return func() { _ = unsub() }, nil
}
