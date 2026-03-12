package cdpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// mockCDP starts a mock CDP WebSocket server for testing.
func mockCDP(t *testing.T, handler func(conn *websocket.Conn)) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		defer conn.Close()
		handler(conn)
	}))
	return srv
}

func TestDialAndSend(t *testing.T) {
	srv := mockCDP(t, func(conn *websocket.Conn) {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req CDPRequest
			if err := json.Unmarshal(msg, &req); err != nil {
				t.Errorf("unmarshal request: %v", err)
				return
			}

			// Echo back a response with the same ID.
			resp := map[string]interface{}{
				"id":     req.ID,
				"result": map[string]interface{}{"success": true},
			}
			data, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, data)
		}
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx := context.Background()

	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	result, err := client.Send(ctx, "Test.method", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	var resp struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestEvents(t *testing.T) {
	srv := mockCDP(t, func(conn *websocket.Conn) {
		// Send an event after a brief pause.
		time.Sleep(50 * time.Millisecond)
		evt := map[string]interface{}{
			"method": "Page.loadEventFired",
			"params": map[string]interface{}{"timestamp": 1234.5},
		}
		data, _ := json.Marshal(evt)
		conn.WriteMessage(websocket.TextMessage, data)

		// Keep connection alive.
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx := context.Background()

	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	eventCh := make(chan string, 1)
	client.On("Page.loadEventFired", func(method string, params json.RawMessage, sessionID string) {
		eventCh <- method
	})

	select {
	case method := <-eventCh:
		if method != "Page.loadEventFired" {
			t.Errorf("expected Page.loadEventFired, got %s", method)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestCDPError(t *testing.T) {
	srv := mockCDP(t, func(conn *websocket.Conn) {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req CDPRequest
			json.Unmarshal(msg, &req)

			resp := map[string]interface{}{
				"id": req.ID,
				"error": map[string]interface{}{
					"code":    -32000,
					"message": "Something went wrong",
				},
			}
			data, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, data)
		}
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx := context.Background()

	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	_, err = client.Send(ctx, "Test.fail", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	cdpErr, ok := err.(*CDPError)
	if !ok {
		t.Fatalf("expected CDPError, got %T: %v", err, err)
	}
	if cdpErr.Code != -32000 {
		t.Errorf("expected code -32000, got %d", cdpErr.Code)
	}
}

func TestSendWithSession(t *testing.T) {
	srv := mockCDP(t, func(conn *websocket.Conn) {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req CDPRequest
			json.Unmarshal(msg, &req)

			resp := map[string]interface{}{
				"id":     req.ID,
				"result": map[string]interface{}{"sessionId": req.SessionID},
			}
			data, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, data)
		}
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx := context.Background()

	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	result, err := client.SendWithSession(ctx, "Test.method", nil, "test-session-123")
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	var resp struct {
		SessionID string `json:"sessionId"`
	}
	json.Unmarshal(result, &resp)
	if resp.SessionID != "test-session-123" {
		t.Errorf("expected session test-session-123, got %s", resp.SessionID)
	}
}

func TestConcurrentSends(t *testing.T) {
	srv := mockCDP(t, func(conn *websocket.Conn) {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req CDPRequest
			json.Unmarshal(msg, &req)

			resp := map[string]interface{}{
				"id":     req.ID,
				"result": map[string]interface{}{"id": req.ID},
			}
			data, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, data)
		}
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx := context.Background()

	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	// Fire off 50 concurrent requests and verify each gets its own response.
	const n = 50
	type result struct {
		id  int64
		err error
	}
	results := make(chan result, n)

	for i := 0; i < n; i++ {
		go func() {
			res, err := client.Send(ctx, "Test.concurrent", nil)
			if err != nil {
				results <- result{err: err}
				return
			}
			var r struct {
				ID int64 `json:"id"`
			}
			json.Unmarshal(res, &r)
			results <- result{id: r.ID}
		}()
	}

	seen := make(map[int64]bool)
	for i := 0; i < n; i++ {
		r := <-results
		if r.err != nil {
			t.Errorf("concurrent send error: %v", r.err)
			continue
		}
		if seen[r.id] {
			t.Errorf("duplicate response id %d", r.id)
		}
		seen[r.id] = true
	}
}

func TestParseWSURLFromString(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{
			"DevTools listening on ws://127.0.0.1:9222/devtools/browser/abc-123",
			"ws://127.0.0.1:9222/devtools/browser/abc-123",
			true,
		},
		{
			"Some other output\nDevTools listening on ws://localhost:36775/devtools/browser/xyz\nmore output",
			"ws://localhost:36775/devtools/browser/xyz",
			true,
		},
		{
			"no websocket url here",
			"",
			false,
		},
	}
	for _, tt := range tests {
		got, ok := ParseWSURLFromString(tt.input)
		if ok != tt.ok || got != tt.want {
			t.Errorf("ParseWSURLFromString(%q) = (%q, %v), want (%q, %v)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestReconnectAfterDisconnect(t *testing.T) {
	var connCount atomic.Int32

	// Server that handles connections, tracks how many.
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connCount.Add(1)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req CDPRequest
			json.Unmarshal(msg, &req)
			resp := map[string]interface{}{
				"id":     req.ID,
				"result": map[string]interface{}{"conn": connCount.Load()},
			}
			data, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, data)
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx := context.Background()

	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	// Track reconnect callbacks.
	var reconnected atomic.Bool
	client.OnReconnect = func(ctx context.Context, c *Client) error {
		reconnected.Store(true)
		return nil
	}

	// First send should work.
	_, err = client.Send(ctx, "Test.method", nil)
	if err != nil {
		t.Fatalf("first send: %v", err)
	}
	if connCount.Load() != 1 {
		t.Fatalf("expected 1 connection, got %d", connCount.Load())
	}

	// Force-close the underlying connection to simulate a disconnect.
	client.connMu.RLock()
	client.conn.Close()
	client.connMu.RUnlock()

	// Wait for reconnection.
	time.Sleep(500 * time.Millisecond)

	// Send should work again after reconnection.
	sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = client.Send(sendCtx, "Test.method", nil)
	if err != nil {
		t.Fatalf("send after reconnect: %v", err)
	}

	if connCount.Load() < 2 {
		t.Fatalf("expected at least 2 connections, got %d", connCount.Load())
	}
	if !reconnected.Load() {
		t.Error("OnReconnect callback was not called")
	}
}

func TestNoReconnectAfterClose(t *testing.T) {
	var connCount atomic.Int32

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connCount.Add(1)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx := context.Background()

	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	// Close intentionally — should NOT reconnect.
	client.Close()

	time.Sleep(300 * time.Millisecond)

	if connCount.Load() != 1 {
		t.Fatalf("expected exactly 1 connection (no reconnect), got %d", connCount.Load())
	}

	// Done channel should be closed.
	select {
	case <-client.Done():
		// good
	default:
		t.Error("Done() channel should be closed after Close()")
	}
}

func TestContextCancellation(t *testing.T) {
	srv := mockCDP(t, func(conn *websocket.Conn) {
		// Never respond — just keep connection open.
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ctx := context.Background()

	client, err := Dial(ctx, wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	cancelCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err = client.Send(cancelCtx, "Test.hang", nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
