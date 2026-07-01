package main

// history.go records the proxy's connection/call history and serves it over the
// ChromeMan gRPC service. The service is implemented in-process and dialed via
// an in-memory bufconn listener (no external port); the HTTP/websocket layer in
// ui.go is its only client — it calls History and forwards each ConnectionHistory
// message into the operator's websocket.
//
// Streaming is APPEND-ONLY (see chromeman.proto): each ConnectionHistory carries
// the latest connection-level fields plus zero or more NEW calls to append, so a
// screenshot's bytes travel exactly once. Clients key by connection_id.

import (
	"context"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	cmpb "github.com/accretional/chromerpc/proto/chromeman"
)

// maxCalls bounds the per-connection call ring (and thus retained media blobs).
// New subscribers see at most this many past calls; the live stream is unbounded.
const maxCalls = 1000

func nowNs() uint64 { return uint64(time.Now().UnixNano()) }

// connState is one bidi session's live state plus a bounded ring of its calls.
type connState struct {
	id        uint32
	target    string
	sessionID string
	state     cmpb.ConnectionState
	createdNs uint64
	updatedNs uint64
	errMsg    string
	calls     []*cmpb.CallRecord
}

// meta returns a ConnectionHistory with the connection-level fields set and no
// calls (the caller fills calls for append messages).
func (c *connState) meta() *cmpb.ConnectionHistory {
	return &cmpb.ConnectionHistory{
		ConnectionId: c.id,
		Target:       c.target,
		SessionId:    c.sessionID,
		State:        c.state,
		CreatedAtNs:  c.createdNs,
		UpdatedAtNs:  c.updatedNs,
		Error:        c.errMsg,
	}
}

// snapshot returns meta plus a copy of the current call ring — used to seed a
// newly-subscribed client.
func (c *connState) snapshot() *cmpb.ConnectionHistory {
	m := c.meta()
	if len(c.calls) > 0 {
		m.Calls = make([]*cmpb.CallRecord, len(c.calls))
		copy(m.Calls, c.calls)
	}
	return m
}

// histStore records connections/calls and fans updates out to History subscribers.
type histStore struct {
	mu      sync.Mutex
	connSeq uint32
	conns   map[uint32]*connState
	order   []uint32
	subSeq  int
	subs    map[int]chan *cmpb.ConnectionHistory
}

func newHistStore() *histStore {
	return &histStore{
		conns: make(map[uint32]*connState),
		subs:  make(map[int]chan *cmpb.ConnectionHistory),
	}
}

// broadcast delivers msg to every subscriber. A subscriber whose buffer is full
// (a stalled browser) is dropped and closed — its History RPC ends and the client
// reconnects to a fresh snapshot. Caller must hold h.mu.
func (h *histStore) broadcast(msg *cmpb.ConnectionHistory) {
	for id, ch := range h.subs {
		select {
		case ch <- msg:
		default:
			delete(h.subs, id)
			close(ch)
		}
	}
}

// openConn registers a new connection in CONNECTING state and returns its id.
func (h *histStore) openConn(target string) uint32 {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connSeq++
	id := h.connSeq
	now := nowNs()
	c := &connState{id: id, target: target, state: cmpb.ConnectionState_CONNECTING, createdNs: now, updatedNs: now}
	h.conns[id] = c
	h.order = append(h.order, id)
	h.broadcast(c.meta())
	return id
}

// setState updates a connection's lifecycle state (and optional error) and fans
// a state-only update (empty calls).
func (h *histStore) setState(id uint32, state cmpb.ConnectionState, sessionID, errMsg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c := h.conns[id]
	if c == nil {
		return
	}
	c.state = state
	if sessionID != "" {
		c.sessionID = sessionID
	}
	if errMsg != "" {
		c.errMsg = errMsg
	}
	c.updatedNs = nowNs()
	h.broadcast(c.meta())
}

func (h *histStore) setReady(id uint32, sessionID string) {
	h.setState(id, cmpb.ConnectionState_READY, sessionID, "")
}
func (h *histStore) setClosed(id uint32)              { h.setState(id, cmpb.ConnectionState_CLOSED, "", "") }
func (h *histStore) setErrored(id uint32, msg string) { h.setState(id, cmpb.ConnectionState_ERRORED, "", msg) }

// appendCall stores rec and fans an append update carrying just that call.
func (h *histStore) appendCall(id uint32, rec *cmpb.CallRecord) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c := h.conns[id]
	if c == nil {
		return
	}
	c.calls = append(c.calls, rec)
	if len(c.calls) > maxCalls {
		c.calls = c.calls[len(c.calls)-maxCalls:]
	}
	c.updatedNs = nowNs()
	msg := c.meta()
	msg.Calls = []*cmpb.CallRecord{rec}
	h.broadcast(msg)
}

// subscribe atomically snapshots all connections and registers a fresh update
// channel, so no update is lost between the snapshot and the first receive.
func (h *histStore) subscribe() (initial []*cmpb.ConnectionHistory, ch chan *cmpb.ConnectionHistory, cancel func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, id := range h.order {
		initial = append(initial, h.conns[id].snapshot())
	}
	subID := h.subSeq
	h.subSeq++
	ch = make(chan *cmpb.ConnectionHistory, 64)
	h.subs[subID] = ch
	cancel = func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if existing, ok := h.subs[subID]; ok {
			delete(h.subs, subID)
			close(existing)
		}
	}
	return initial, ch, cancel
}

// chromeManServer serves ChromeMan.History from a histStore.
type chromeManServer struct {
	cmpb.UnimplementedChromeManServer
	store *histStore
}

func (s *chromeManServer) History(_ *emptypb.Empty, stream cmpb.ChromeMan_HistoryServer) error {
	initial, ch, cancel := s.store.subscribe()
	defer cancel()
	for _, m := range initial {
		if err := stream.Send(m); err != nil {
			return err
		}
	}
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case m, ok := <-ch:
			if !ok {
				return nil // dropped as a slow subscriber; client will reconnect
			}
			if err := stream.Send(m); err != nil {
				return err
			}
		}
	}
}

// startChromeMan serves ChromeMan over an in-memory bufconn and returns an
// in-process client plus a stop func. No TCP port is opened.
func startChromeMan(store *histStore) (cmpb.ChromeManClient, func(), error) {
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	cmpb.RegisterChromeManServer(srv, &chromeManServer{store: store})
	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		srv.Stop()
		_ = lis.Close()
		return nil, nil, err
	}
	stop := func() {
		_ = conn.Close()
		srv.Stop()
		_ = lis.Close()
	}
	return cmpb.NewChromeManClient(conn), stop, nil
}
