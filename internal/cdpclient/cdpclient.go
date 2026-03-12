// Package cdpclient provides a Go client for the Chrome DevTools Protocol
// over WebSocket. It handles the JSONRPC-style message framing, concurrent
// command dispatching, and event subscription that CDP requires.
package cdpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// CDPRequest is the JSONRPC-style request sent over the WebSocket.
type CDPRequest struct {
	ID        int64           `json:"id"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
}

// CDPResponse is the JSONRPC-style response received over the WebSocket.
type CDPResponse struct {
	ID        int64           `json:"id,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *CDPError       `json:"error,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
}

// CDPError represents a protocol-level error from Chrome.
type CDPError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *CDPError) Error() string {
	if e.Data != "" {
		return fmt.Sprintf("CDP error %d: %s (%s)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("CDP error %d: %s", e.Code, e.Message)
}

// pendingCommand tracks an in-flight command awaiting a response.
type pendingCommand struct {
	ch chan *CDPResponse
}

// EventHandler is called when a CDP event is received.
// The method is the full event name (e.g. "Page.loadEventFired")
// and params is the raw JSON of the event parameters.
type EventHandler func(method string, params json.RawMessage, sessionID string)

// Client manages a WebSocket connection to a Chrome DevTools Protocol endpoint.
type Client struct {
	wsURL   string // stored for reconnection
	conn    *websocket.Conn
	connMu  sync.RWMutex // protects conn during reconnection
	writeMu sync.Mutex   // gorilla/websocket requires serialized writes
	nextID  atomic.Int64
	mu      sync.Mutex
	pending map[int64]*pendingCommand
	closed  chan struct{} // closed permanently when Close() is called

	// ready is closed when the connection is established and ready for use.
	// Recreated each time a reconnection begins.
	ready   chan struct{}
	readyMu sync.Mutex

	eventMu   sync.RWMutex
	handlers  map[string][]EventHandler // keyed by method name
	wildcard  []EventHandler            // handlers for all events
	sessionID string                    // default session ID for flat mode

	userClosed atomic.Bool // true when Close() is called intentionally

	// OnReconnect is called after a successful reconnection. Use this to
	// re-establish sessions (e.g., re-attach to targets). Called in the
	// reconnection goroutine; errors are logged but do not prevent the
	// client from becoming ready.
	OnReconnect func(ctx context.Context, c *Client) error
}

// Dial connects to a CDP WebSocket endpoint.
func Dial(ctx context.Context, wsURL string) (*Client, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cdpclient: dial %s: %w", wsURL, err)
	}

	ready := make(chan struct{})
	close(ready) // immediately ready

	c := &Client{
		wsURL:    wsURL,
		conn:     conn,
		pending:  make(map[int64]*pendingCommand),
		closed:   make(chan struct{}),
		ready:    ready,
		handlers: make(map[string][]EventHandler),
	}

	go c.readLoop()
	return c, nil
}

// SetSessionID sets the default session ID for commands sent via this client.
func (c *Client) SetSessionID(sid string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = sid
}

// SessionID returns the current default session ID.
func (c *Client) SessionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionID
}

// waitReady blocks until the connection is ready or ctx is cancelled.
func (c *Client) waitReady(ctx context.Context) error {
	c.readyMu.Lock()
	ready := c.ready
	c.readyMu.Unlock()

	select {
	case <-ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closed:
		return fmt.Errorf("cdpclient: connection closed")
	}
}

// Send sends a CDP command and waits for the response.
// The method should be in the form "Domain.method" (e.g. "Page.navigate").
// params should be a struct or map that serializes to the expected JSON params.
// If params is nil, no params field is sent.
func (c *Client) Send(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	if err := c.waitReady(ctx); err != nil {
		return nil, err
	}

	id := c.nextID.Add(1)

	req := CDPRequest{
		ID:     id,
		Method: method,
	}

	// Use default session ID if set.
	c.mu.Lock()
	if c.sessionID != "" {
		req.SessionID = c.sessionID
	}
	c.mu.Unlock()

	if params != nil {
		raw, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("cdpclient: marshal params: %w", err)
		}
		req.Params = raw
	}

	// Register pending command before sending to avoid race.
	pc := &pendingCommand{ch: make(chan *CDPResponse, 1)}
	c.mu.Lock()
	c.pending[id] = pc
	c.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("cdpclient: marshal request: %w", err)
	}

	c.connMu.RLock()
	c.writeMu.Lock()
	writeErr := c.conn.WriteMessage(websocket.TextMessage, data)
	c.writeMu.Unlock()
	c.connMu.RUnlock()
	if writeErr != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("cdpclient: write: %w", writeErr)
	}

	// Wait for response or context cancellation.
	select {
	case resp := <-pc.ch:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case <-c.closed:
		return nil, fmt.Errorf("cdpclient: connection closed")
	}
}

// SendWithSession sends a CDP command with an explicit session ID,
// overriding the default.
func (c *Client) SendWithSession(ctx context.Context, method string, params interface{}, sessionID string) (json.RawMessage, error) {
	if err := c.waitReady(ctx); err != nil {
		return nil, err
	}

	id := c.nextID.Add(1)

	req := CDPRequest{
		ID:        id,
		Method:    method,
		SessionID: sessionID,
	}

	if params != nil {
		raw, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("cdpclient: marshal params: %w", err)
		}
		req.Params = raw
	}

	pc := &pendingCommand{ch: make(chan *CDPResponse, 1)}
	c.mu.Lock()
	c.pending[id] = pc
	c.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("cdpclient: marshal request: %w", err)
	}

	c.connMu.RLock()
	c.writeMu.Lock()
	writeErr := c.conn.WriteMessage(websocket.TextMessage, data)
	c.writeMu.Unlock()
	c.connMu.RUnlock()
	if writeErr != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("cdpclient: write: %w", writeErr)
	}

	select {
	case resp := <-pc.ch:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case <-c.closed:
		return nil, fmt.Errorf("cdpclient: connection closed")
	}
}

// On registers an event handler for a specific CDP event method.
// Returns a function to unregister the handler.
func (c *Client) On(method string, handler EventHandler) func() {
	c.eventMu.Lock()
	defer c.eventMu.Unlock()
	c.handlers[method] = append(c.handlers[method], handler)
	return func() {
		c.eventMu.Lock()
		defer c.eventMu.Unlock()
		handlers := c.handlers[method]
		for i, h := range handlers {
			if &h == &handler {
				c.handlers[method] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}
}

// OnAll registers an event handler that receives all CDP events.
func (c *Client) OnAll(handler EventHandler) {
	c.eventMu.Lock()
	defer c.eventMu.Unlock()
	c.wildcard = append(c.wildcard, handler)
}

// Close closes the WebSocket connection permanently. No reconnection
// will be attempted after Close is called.
func (c *Client) Close() error {
	c.userClosed.Store(true)
	select {
	case <-c.closed:
		return nil // already closed
	default:
	}
	close(c.closed)
	c.connMu.RLock()
	err := c.conn.Close()
	c.connMu.RUnlock()
	return err
}

// Done returns a channel that's closed when the connection is
// permanently closed (via Close). It is NOT closed during transient
// disconnects that trigger reconnection.
func (c *Client) Done() <-chan struct{} {
	return c.closed
}

// readLoop reads messages from the WebSocket and dispatches them.
func (c *Client) readLoop() {
	defer func() {
		// Fail all pending commands.
		c.mu.Lock()
		for id, pc := range c.pending {
			pc.ch <- &CDPResponse{
				Error: &CDPError{Code: -1, Message: "connection closed"},
			}
			delete(c.pending, id)
		}
		c.mu.Unlock()

		// If user intentionally closed, mark as permanently closed.
		if c.userClosed.Load() {
			select {
			case <-c.closed:
			default:
				close(c.closed)
			}
			return
		}

		// Otherwise, trigger reconnection.
		go c.reconnectLoop()
	}()

	for {
		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()

		_, data, err := conn.ReadMessage()
		if err != nil {
			if c.userClosed.Load() {
				return
			}
			log.Printf("cdpclient: read error (will reconnect): %v", err)
			return
		}

		var resp CDPResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			log.Printf("cdpclient: unmarshal error: %v", err)
			continue
		}

		if resp.ID != 0 {
			// This is a response to a command.
			c.mu.Lock()
			pc, ok := c.pending[resp.ID]
			if ok {
				delete(c.pending, resp.ID)
			}
			c.mu.Unlock()
			if ok {
				pc.ch <- &resp
			}
		} else if resp.Method != "" {
			// This is an event.
			c.dispatchEvent(resp.Method, resp.Params, resp.SessionID)
		}
	}
}

// reconnectLoop attempts to re-establish the WebSocket connection with
// exponential backoff. It runs until the connection is restored or
// Close() is called.
func (c *Client) reconnectLoop() {
	// Mark connection as not ready.
	c.readyMu.Lock()
	c.ready = make(chan struct{})
	c.readyMu.Unlock()

	// Close the old connection.
	c.connMu.Lock()
	c.conn.Close()
	c.connMu.Unlock()

	backoff := 100 * time.Millisecond
	maxBackoff := 5 * time.Second

	for {
		if c.userClosed.Load() {
			select {
			case <-c.closed:
			default:
				close(c.closed)
			}
			return
		}

		log.Printf("cdpclient: reconnecting to %s (backoff %v)...", c.wsURL, backoff)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		dialer := websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		}
		conn, _, err := dialer.DialContext(ctx, c.wsURL, nil)
		cancel()

		if err != nil {
			log.Printf("cdpclient: reconnect failed: %v", err)
			select {
			case <-c.closed:
				return
			case <-time.After(backoff):
			}
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Replace the connection.
		c.connMu.Lock()
		c.conn = conn
		c.connMu.Unlock()

		log.Printf("cdpclient: reconnected to %s", c.wsURL)

		// Call the reconnect callback to re-establish sessions.
		if c.OnReconnect != nil {
			rctx, rcancel := context.WithTimeout(context.Background(), 30*time.Second)
			// Mark ready temporarily so the callback can send commands.
			c.readyMu.Lock()
			tmpReady := c.ready
			immediateReady := make(chan struct{})
			close(immediateReady)
			c.ready = immediateReady
			c.readyMu.Unlock()

			if err := c.OnReconnect(rctx, c); err != nil {
				log.Printf("cdpclient: reconnect callback error: %v", err)
			}
			rcancel()

			// Restore the real ready channel if callback failed,
			// or keep the immediate one if it succeeded.
			_ = tmpReady
		}

		// Mark as ready and start the read loop.
		c.readyMu.Lock()
		ready := c.ready
		// If ready isn't already closed (from callback path), close it.
		select {
		case <-ready:
			// already closed, we're good
		default:
			close(ready)
		}
		c.readyMu.Unlock()

		// Start new readLoop (which will trigger reconnectLoop again if it fails).
		c.readLoop()
		return
	}
}

// dispatchEvent calls registered event handlers.
func (c *Client) dispatchEvent(method string, params json.RawMessage, sessionID string) {
	c.eventMu.RLock()
	handlers := make([]EventHandler, 0, len(c.handlers[method])+len(c.wildcard))
	handlers = append(handlers, c.handlers[method]...)
	handlers = append(handlers, c.wildcard...)
	c.eventMu.RUnlock()

	for _, h := range handlers {
		h(method, params, sessionID)
	}
}
