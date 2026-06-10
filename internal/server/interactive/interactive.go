// Package interactive implements InteractiveSessionService: a bidirectional
// streaming interface for stateful, interactive browser sessions.
//
// Each Session stream owns a dedicated Chrome whose state persists for the life
// of the stream. When the stream closes, the manager recycles (kills and
// relaunches) the Chrome child process so the next client gets a completely
// fresh browser — process-level isolation between tenants. Intended to run on a
// dedicated Cloud Run service with concurrency=1.
package interactive

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/accretional/chromerpc/internal/cdpclient"
	"github.com/accretional/chromerpc/internal/cdpsession"
	hbserver "github.com/accretional/chromerpc/internal/server/headlessbrowser"
	pb "github.com/accretional/chromerpc/proto/cdp/headlessbrowser"
)

// ChromeManager owns a dedicated Chrome process plus the CDP client and headless
// server bound to it, and can recycle Chrome to wipe all state at the process
// level.
type ChromeManager struct {
	cfg     cdpclient.LaunchConfig
	rootCtx context.Context // bounds the Chrome process lifetime (server lifetime)

	mu     sync.Mutex
	client *cdpclient.Client
	hb     *hbserver.Server
	launch *cdpclient.LaunchResult
}

func NewChromeManager(cfg cdpclient.LaunchConfig) *ChromeManager {
	return &ChromeManager{cfg: cfg}
}

// Start launches the initial Chrome and establishes the default session.
// rootCtx bounds the lifetime of every Chrome process the manager launches.
func (m *ChromeManager) Start(rootCtx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rootCtx = rootCtx
	return m.launchLocked(rootCtx)
}

// launchLocked launches Chrome (process tied to rootCtx), dials a client, and
// sets up the default page session. opCtx bounds the dial/setup operations only.
// Caller must hold m.mu.
func (m *ChromeManager) launchLocked(opCtx context.Context) error {
	res, err := cdpclient.Launch(m.rootCtx, m.cfg)
	if err != nil {
		return fmt.Errorf("launch chrome: %w", err)
	}

	client, err := cdpclient.Dial(opCtx, res.WebSocketURL)
	if err != nil {
		res.Cleanup()
		return fmt.Errorf("dial chrome: %w", err)
	}
	if err := cdpsession.SetupDefault(opCtx, client); err != nil {
		client.Close()
		res.Cleanup()
		return fmt.Errorf("setup session: %w", err)
	}

	m.client = client
	m.hb = hbserver.New(client)
	m.launch = res
	log.Printf("interactive: Chrome ready (ws=%s)", res.WebSocketURL)
	return nil
}

// teardownLocked closes the client and kills the current Chrome. Caller holds mu.
func (m *ChromeManager) teardownLocked() {
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}
	m.hb = nil
	if m.launch != nil {
		// Kills Chrome and removes its per-process temp tree (freeing tmpfs
		// memory on each recycle), unless --no-tmp-cleanup is set.
		m.launch.Cleanup()
		m.launch = nil
	}
}

// Recycle kills the current Chrome and launches a fresh one, wiping all browser
// state. opCtx bounds the dial/setup of the new Chrome.
func (m *ChromeManager) Recycle(opCtx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	log.Printf("interactive: recycling Chrome...")
	m.teardownLocked()
	return m.launchLocked(opCtx)
}

// HB returns the headless server bound to the current Chrome client.
func (m *ChromeManager) HB() *hbserver.Server {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hb
}

// Close tears down Chrome and the client (server shutdown).
func (m *ChromeManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.teardownLocked()
}

// recycleTimeout bounds the dial/setup work when relaunching Chrome.
const recycleTimeout = 30 * time.Second

// provider hands a ready Chrome to a session and takes it back when the session
// ends. Two implementations: a single Chrome (concurrency=1) and a pool of N.
type provider interface {
	Acquire(ctx context.Context) (*ChromeManager, error)
	Release(m *ChromeManager)
}

// singleProvider serves one session at a time from a single Chrome and recycles
// it synchronously on release. The recycle runs while the streaming handler is
// still executing (CPU allocated on Cloud Run). For the concurrency=1 service.
type singleProvider struct {
	mgr *ChromeManager
	sem chan struct{}
}

func newSingleProvider(mgr *ChromeManager) *singleProvider {
	return &singleProvider{mgr: mgr, sem: make(chan struct{}, 1)}
}

func (s *singleProvider) Acquire(ctx context.Context) (*ChromeManager, error) {
	select {
	case s.sem <- struct{}{}:
		return s.mgr, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *singleProvider) Release(m *ChromeManager) {
	rctx, cancel := context.WithTimeout(context.Background(), recycleTimeout)
	defer cancel()
	if err := m.Recycle(rctx); err != nil {
		log.Printf("interactive: recycle error: %v", err)
	}
	<-s.sem
}

// ChromePool maintains a set of independent, pre-warmed Chrome processes. A
// session acquires a ready one and, on release, the Chrome is recycled
// asynchronously and returned to the pool — so a warm Chrome is (almost) always
// available with no cold-start or recycle latency on the request path, and
// sessions are isolated at the process level. For the concurrency=N service.
//
// Admission control: at most `max` clients are admitted at once (max = pool
// size). A client is admitted while admitted-count < max, then waits on the
// `ready` channel for a Chrome — so a client that arrives while a Chrome is
// mid-recycle stays PENDING and is handed that Chrome the instant its recycle
// completes (the channel's blocked receiver is the wait). Only when all `max`
// slots are already taken is a new client rejected with ResourceExhausted.
type ChromePool struct {
	max   int
	ready chan *ChromeManager

	mu      sync.Mutex
	clients int // admitted clients (pending + active); guarded by mu
}

func NewChromePool() *ChromePool { return &ChromePool{} }

// Start sizes the pool and warms `size` Chrome processes in the BACKGROUND, so
// the gRPC server can listen immediately (passing the Cloud Run startup probe)
// instead of blocking on — or dying from — slow Chrome warmup. Sessions that
// arrive before a Chrome is ready simply wait (pending). Launch failures are
// retried, never fatal, so a slow/contended launch can't take the instance down.
// rootCtx bounds the lifetime of every Chrome the pool launches.
func (p *ChromePool) Start(rootCtx context.Context, cfg cdpclient.LaunchConfig, size int) {
	p.max = size
	p.ready = make(chan *ChromeManager, size)
	for i := 0; i < size; i++ {
		go func(idx int) {
			m := NewChromeManager(cfg)
			for {
				if err := m.Start(rootCtx); err != nil {
					if rootCtx.Err() != nil {
						return
					}
					log.Printf("interactive pool: chrome %d warmup failed (%v); retrying", idx, err)
					select {
					case <-rootCtx.Done():
						return
					case <-time.After(2 * time.Second):
					}
					continue
				}
				break
			}
			p.ready <- m
			log.Printf("interactive pool: chrome %d ready", idx)
		}(i)
	}
}

func (p *ChromePool) Acquire(ctx context.Context) (*ChromeManager, error) {
	// Admit the client iff there is capacity (mutex-guarded — no data race).
	p.mu.Lock()
	if p.clients >= p.max {
		p.mu.Unlock()
		return nil, status.Errorf(codes.ResourceExhausted, "all %d browser sessions are in use; retry shortly", p.max)
	}
	p.clients++
	p.mu.Unlock()

	// Wait (pending) for a ready Chrome — one may currently be recycling.
	select {
	case m := <-p.ready:
		return m, nil
	case <-ctx.Done():
		p.mu.Lock()
		p.clients--
		p.mu.Unlock()
		return nil, ctx.Err()
	}
}

func (p *ChromePool) Release(m *ChromeManager) {
	// The client is done -> free its admission slot AND return the handler
	// immediately (don't hold the Cloud Run concurrency slot open).
	p.mu.Lock()
	p.clients--
	p.mu.Unlock()

	// Start recycling in the BACKGROUND the instant the client disconnects — the
	// earliest opportunity, and non-blocking. The relaunch opportunistically uses
	// whatever CPU the instance has provisioned for *any* active request: while
	// peers are still serving, or — for an otherwise-idle instance — the next
	// client that lands here keeps CPU on and lets the recycle finish. That is
	// strictly the more optimal time/place than tying it to this one dead
	// request's window. Retries so a throttled/transient failure can't
	// permanently shrink the pool.
	go func() {
		for {
			rctx, cancel := context.WithTimeout(context.Background(), recycleTimeout)
			err := m.Recycle(rctx)
			cancel()
			if err == nil {
				break
			}
			if m.rootCtx.Err() != nil {
				return // server shutting down
			}
			log.Printf("interactive pool: recycle failed (%v); retrying", err)
			time.Sleep(2 * time.Second)
		}
		p.ready <- m
	}()
}

// Server implements InteractiveSessionService, backed by a provider (a single
// Chrome or a pool).
type Server struct {
	pb.UnimplementedInteractiveSessionServiceServer
	prov    provider
	counter atomic.Int64
}

// NewSingle backs the service with one Chrome, one session at a time (deploy
// with concurrency=1).
func NewSingle(mgr *ChromeManager) *Server { return &Server{prov: newSingleProvider(mgr)} }

// NewPool backs the service with a pool of Chromes (deploy with concurrency
// matching, or below, the pool size).
func NewPool(pool *ChromePool) *Server { return &Server{prov: pool} }

// Session handles one interactive bidi stream. It acquires a dedicated Chrome,
// runs steps against it (state persists for the stream), and releases it (which
// recycles the Chrome) when the stream closes.
func (s *Server) Session(stream pb.InteractiveSessionService_SessionServer) error {
	ctx := stream.Context()

	mgr, err := s.prov.Acquire(ctx)
	if err != nil {
		return err
	}
	sid := fmt.Sprintf("isess-%d", s.counter.Add(1))
	log.Printf("interactive: session %s open", sid)
	defer func() {
		s.prov.Release(mgr)
		log.Printf("interactive: session %s closed", sid)
	}()

	// Announce readiness.
	if err := stream.Send(&pb.SessionResponse{
		Event: &pb.SessionResponse_Ready{Ready: &pb.SessionReady{SessionId: sid}},
	}); err != nil {
		return err
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil // client closed the send side cleanly
		}
		if err != nil {
			return err // disconnect / cancellation
		}
		if resp := s.handle(ctx, mgr, req); resp != nil {
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}

func (s *Server) handle(ctx context.Context, mgr *ChromeManager, req *pb.SessionRequest) *pb.SessionResponse {
	sessErr := func(msg string) *pb.SessionResponse {
		return &pb.SessionResponse{Id: req.Id, Event: &pb.SessionResponse_Error{Error: &pb.SessionError{Message: msg}}}
	}
	switch c := req.Command.(type) {
	case *pb.SessionRequest_Step:
		hb := mgr.HB()
		if hb == nil {
			return sessErr("browser not ready")
		}
		res := hb.ExecuteStepShared(ctx, c.Step)
		return &pb.SessionResponse{Id: req.Id, Event: &pb.SessionResponse_Result{Result: res}}
	case *pb.SessionRequest_Ping:
		return &pb.SessionResponse{Id: req.Id, Event: &pb.SessionResponse_Pong{Pong: &pb.Pong{}}}
	case *pb.SessionRequest_Reset_:
		// Wipe this session's Chrome mid-stream (synchronous).
		rctx, cancel := context.WithTimeout(context.Background(), recycleTimeout)
		defer cancel()
		if err := mgr.Recycle(rctx); err != nil {
			return sessErr("reset failed: " + err.Error())
		}
		return &pb.SessionResponse{Id: req.Id, Event: &pb.SessionResponse_Ready{Ready: &pb.SessionReady{SessionId: "reset"}}}
	default:
		return sessErr("unknown or empty command")
	}
}
