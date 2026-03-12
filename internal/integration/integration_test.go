// Package integration contains end-to-end tests that launch headless Chrome
// and exercise the gRPC service implementations against real CDP.
//
// These tests require Chrome/Chromium to be installed. They are skipped
// automatically if Chrome is not found. Run with:
//
//	go test ./internal/integration/ -v -count=1 -timeout=60s
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/accretional/chromerpc/internal/cdpclient"
	accessibilityserver "github.com/accretional/chromerpc/internal/server/accessibility"
	browserserver "github.com/accretional/chromerpc/internal/server/browser"
	cachestorageserver "github.com/accretional/chromerpc/internal/server/cachestorage"
	consoleserver "github.com/accretional/chromerpc/internal/server/console"
	cssserver "github.com/accretional/chromerpc/internal/server/css"
	debuggerserver "github.com/accretional/chromerpc/internal/server/debugger"
	domserver "github.com/accretional/chromerpc/internal/server/dom"
	domstorageserver "github.com/accretional/chromerpc/internal/server/domstorage"
	emulationserver "github.com/accretional/chromerpc/internal/server/emulation"
	fetchserver "github.com/accretional/chromerpc/internal/server/fetch"
	heapprofilerserver "github.com/accretional/chromerpc/internal/server/heapprofiler"
	indexeddbserver "github.com/accretional/chromerpc/internal/server/indexeddb"
	inputserver "github.com/accretional/chromerpc/internal/server/input"
	ioserver "github.com/accretional/chromerpc/internal/server/io"
	logserver "github.com/accretional/chromerpc/internal/server/log"
	networkserver "github.com/accretional/chromerpc/internal/server/network"
	overlayserver "github.com/accretional/chromerpc/internal/server/overlay"
	pageserver "github.com/accretional/chromerpc/internal/server/page"
	performanceserver "github.com/accretional/chromerpc/internal/server/performance"
	profilerserver "github.com/accretional/chromerpc/internal/server/profiler"
	runtimeserver "github.com/accretional/chromerpc/internal/server/runtime"
	securityserver "github.com/accretional/chromerpc/internal/server/security"
	serviceworkerserver "github.com/accretional/chromerpc/internal/server/serviceworker"
	storageserver "github.com/accretional/chromerpc/internal/server/storage"
	targetserver "github.com/accretional/chromerpc/internal/server/target"
	accessibilitypb "github.com/accretional/chromerpc/proto/cdp/accessibility"
	browserpb "github.com/accretional/chromerpc/proto/cdp/browser"
	cachestoragepb "github.com/accretional/chromerpc/proto/cdp/cachestorage"
	consolepb "github.com/accretional/chromerpc/proto/cdp/console"
	csspb "github.com/accretional/chromerpc/proto/cdp/css"
	debuggerpb "github.com/accretional/chromerpc/proto/cdp/debugger"
	dompb "github.com/accretional/chromerpc/proto/cdp/dom"
	domstoragepb "github.com/accretional/chromerpc/proto/cdp/domstorage"
	emulationpb "github.com/accretional/chromerpc/proto/cdp/emulation"
	fetchpb "github.com/accretional/chromerpc/proto/cdp/fetch"
	heapprofilerpb "github.com/accretional/chromerpc/proto/cdp/heapprofiler"
	indexeddbpb "github.com/accretional/chromerpc/proto/cdp/indexeddb"
	inputpb "github.com/accretional/chromerpc/proto/cdp/input"
	iopb "github.com/accretional/chromerpc/proto/cdp/io"
	logpb "github.com/accretional/chromerpc/proto/cdp/log"
	networkpb "github.com/accretional/chromerpc/proto/cdp/network"
	overlaypb "github.com/accretional/chromerpc/proto/cdp/overlay"
	pagepb "github.com/accretional/chromerpc/proto/cdp/page"
	performancepb "github.com/accretional/chromerpc/proto/cdp/performance"
	profilerpb "github.com/accretional/chromerpc/proto/cdp/profiler"
	runtimepb "github.com/accretional/chromerpc/proto/cdp/runtime"
	securitypb "github.com/accretional/chromerpc/proto/cdp/security"
	serviceworkerpb "github.com/accretional/chromerpc/proto/cdp/serviceworker"
	storagepb "github.com/accretional/chromerpc/proto/cdp/storage"
	targetpb "github.com/accretional/chromerpc/proto/cdp/target"
)

// testEnv holds a running Chrome + gRPC server for tests.
type testEnv struct {
	grpcAddr            string
	grpcServer          *grpc.Server
	client              *cdpclient.Client
	launchResult        *cdpclient.LaunchResult
	pageClient          pagepb.PageServiceClient
	targetClient        targetpb.TargetServiceClient
	runtimeClient       runtimepb.RuntimeServiceClient
	networkClient       networkpb.NetworkServiceClient
	domClient           dompb.DOMServiceClient
	emulationClient     emulationpb.EmulationServiceClient
	inputClient         inputpb.InputServiceClient
	browserClient       browserpb.BrowserServiceClient
	fetchClient         fetchpb.FetchServiceClient
	cssClient           csspb.CSSServiceClient
	logClient           logpb.LogServiceClient
	performanceClient   performancepb.PerformanceServiceClient
	accessibilityClient accessibilitypb.AccessibilityServiceClient
	ioClient            iopb.IOServiceClient
	securityClient      securitypb.SecurityServiceClient
	storageClient       storagepb.StorageServiceClient
	overlayClient       overlaypb.OverlayServiceClient
	domstorageClient    domstoragepb.DOMStorageServiceClient
	debuggerClient      debuggerpb.DebuggerServiceClient
	profilerClient      profilerpb.ProfilerServiceClient
	consoleClient       consolepb.ConsoleServiceClient
	heapProfilerClient  heapprofilerpb.HeapProfilerServiceClient
	serviceWorkerClient serviceworkerpb.ServiceWorkerServiceClient
	indexedDBClient     indexeddbpb.IndexedDBServiceClient
	cacheStorageClient  cachestoragepb.CacheStorageServiceClient
	conn                *grpc.ClientConn
}

func (e *testEnv) cleanup() {
	if e.conn != nil {
		e.conn.Close()
	}
	if e.grpcServer != nil {
		e.grpcServer.Stop()
	}
	if e.client != nil {
		e.client.Close()
	}
	if e.launchResult != nil {
		e.launchResult.Process.Kill()
		e.launchResult.Cmd.Wait()
		if e.launchResult.TempDir != "" {
			os.RemoveAll(e.launchResult.TempDir)
		}
	}
}

// setupTestEnv launches Chrome, connects, starts the gRPC server, and returns
// gRPC clients ready for testing.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Use context.Background() for Chrome's process lifetime — Chrome must
	// stay alive for the entire test duration. Cleanup happens via t.Cleanup.
	chromeCtx := context.Background()

	// Launch headless Chrome.
	client, launchResult, err := cdpclient.ConnectOrLaunch(chromeCtx, "", cdpclient.LaunchConfig{
		Headless: true,
		ExtraArgs: []string{
			"--no-sandbox",
			"--disable-gpu",
			"--disable-dev-shm-usage",
		},
	})
	if err != nil {
		t.Skipf("Chrome not available: %v", err)
	}

	t.Logf("Chrome launched at %s", launchResult.WebSocketURL)

	// Give Chrome a moment to stabilize.
	time.Sleep(200 * time.Millisecond)

	// Attach to a page target for Page domain commands.
	ctx := context.Background()
	if err := attachToFirstPage(ctx, client); err != nil {
		client.Close()
		launchResult.Process.Kill()
		launchResult.Cmd.Wait()
		if launchResult.TempDir != "" {
			os.RemoveAll(launchResult.TempDir)
		}
		t.Fatalf("Failed to attach to page: %v", err)
	}

	// Start gRPC server on a random port.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	targetpb.RegisterTargetServiceServer(grpcServer, targetserver.New(client))
	pagepb.RegisterPageServiceServer(grpcServer, pageserver.New(client))
	runtimepb.RegisterRuntimeServiceServer(grpcServer, runtimeserver.New(client))
	networkpb.RegisterNetworkServiceServer(grpcServer, networkserver.New(client))
	dompb.RegisterDOMServiceServer(grpcServer, domserver.New(client))
	emulationpb.RegisterEmulationServiceServer(grpcServer, emulationserver.New(client))
	inputpb.RegisterInputServiceServer(grpcServer, inputserver.New(client))
	browserpb.RegisterBrowserServiceServer(grpcServer, browserserver.New(client))
	fetchpb.RegisterFetchServiceServer(grpcServer, fetchserver.New(client))
	csspb.RegisterCSSServiceServer(grpcServer, cssserver.New(client))
	logpb.RegisterLogServiceServer(grpcServer, logserver.New(client))
	performancepb.RegisterPerformanceServiceServer(grpcServer, performanceserver.New(client))
	accessibilitypb.RegisterAccessibilityServiceServer(grpcServer, accessibilityserver.New(client))
	iopb.RegisterIOServiceServer(grpcServer, ioserver.New(client))
	securitypb.RegisterSecurityServiceServer(grpcServer, securityserver.New(client))
	storagepb.RegisterStorageServiceServer(grpcServer, storageserver.New(client))
	overlaypb.RegisterOverlayServiceServer(grpcServer, overlayserver.New(client))
	domstoragepb.RegisterDOMStorageServiceServer(grpcServer, domstorageserver.New(client))
	debuggerpb.RegisterDebuggerServiceServer(grpcServer, debuggerserver.New(client))
	profilerpb.RegisterProfilerServiceServer(grpcServer, profilerserver.New(client))
	consolepb.RegisterConsoleServiceServer(grpcServer, consoleserver.New(client))
	heapprofilerpb.RegisterHeapProfilerServiceServer(grpcServer, heapprofilerserver.New(client))
	serviceworkerpb.RegisterServiceWorkerServiceServer(grpcServer, serviceworkerserver.New(client))
	indexeddbpb.RegisterIndexedDBServiceServer(grpcServer, indexeddbserver.New(client))
	cachestoragepb.RegisterCacheStorageServiceServer(grpcServer, cachestorageserver.New(client))

	go grpcServer.Serve(lis)

	// Connect gRPC client.
	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		grpcServer.Stop()
		t.Fatalf("grpc dial: %v", err)
	}

	env := &testEnv{
		grpcAddr:            lis.Addr().String(),
		grpcServer:          grpcServer,
		client:              client,
		launchResult:        launchResult,
		pageClient:          pagepb.NewPageServiceClient(conn),
		targetClient:        targetpb.NewTargetServiceClient(conn),
		runtimeClient:       runtimepb.NewRuntimeServiceClient(conn),
		networkClient:       networkpb.NewNetworkServiceClient(conn),
		domClient:           dompb.NewDOMServiceClient(conn),
		emulationClient:     emulationpb.NewEmulationServiceClient(conn),
		inputClient:         inputpb.NewInputServiceClient(conn),
		browserClient:       browserpb.NewBrowserServiceClient(conn),
		fetchClient:         fetchpb.NewFetchServiceClient(conn),
		cssClient:           csspb.NewCSSServiceClient(conn),
		logClient:           logpb.NewLogServiceClient(conn),
		performanceClient:   performancepb.NewPerformanceServiceClient(conn),
		accessibilityClient: accessibilitypb.NewAccessibilityServiceClient(conn),
		ioClient:            iopb.NewIOServiceClient(conn),
		securityClient:      securitypb.NewSecurityServiceClient(conn),
		storageClient:       storagepb.NewStorageServiceClient(conn),
		overlayClient:       overlaypb.NewOverlayServiceClient(conn),
		domstorageClient:    domstoragepb.NewDOMStorageServiceClient(conn),
		debuggerClient:      debuggerpb.NewDebuggerServiceClient(conn),
		profilerClient:      profilerpb.NewProfilerServiceClient(conn),
		consoleClient:       consolepb.NewConsoleServiceClient(conn),
		heapProfilerClient:  heapprofilerpb.NewHeapProfilerServiceClient(conn),
		serviceWorkerClient: serviceworkerpb.NewServiceWorkerServiceClient(conn),
		indexedDBClient:     indexeddbpb.NewIndexedDBServiceClient(conn),
		cacheStorageClient:  cachestoragepb.NewCacheStorageServiceClient(conn),
		conn:                conn,
	}

	t.Cleanup(env.cleanup)
	return env
}

func attachToFirstPage(ctx context.Context, client *cdpclient.Client) error {
	result, err := client.Send(ctx, "Target.getTargets", nil)
	if err != nil {
		return fmt.Errorf("getTargets: %w", err)
	}
	type targetInfo struct {
		TargetID string `json:"targetId"`
		Type     string `json:"type"`
	}
	var resp struct {
		TargetInfos []targetInfo `json:"targetInfos"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	for _, t := range resp.TargetInfos {
		if t.Type == "page" {
			attachResult, err := client.Send(ctx, "Target.attachToTarget", map[string]interface{}{
				"targetId": t.TargetID,
				"flatten":  true,
			})
			if err != nil {
				return fmt.Errorf("attach: %w", err)
			}
			var ar struct {
				SessionID string `json:"sessionId"`
			}
			json.Unmarshal(attachResult, &ar)
			client.SetSessionID(ar.SessionID)
			return nil
		}
	}
	return fmt.Errorf("no page target found")
}

// =============================================================
// Target Domain Tests
// =============================================================

func TestTargetGetTargets(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	resp, err := env.targetClient.GetTargets(ctx, &targetpb.GetTargetsRequest{})
	if err != nil {
		t.Fatalf("GetTargets: %v", err)
	}
	if len(resp.TargetInfos) == 0 {
		t.Fatal("expected at least one target")
	}

	// Should have at least one page target (the about:blank we opened).
	var foundPage bool
	for _, ti := range resp.TargetInfos {
		t.Logf("Target: id=%s type=%s url=%s", ti.TargetId, ti.Type, ti.Url)
		if ti.Type == "page" {
			foundPage = true
		}
	}
	if !foundPage {
		t.Error("expected at least one page target")
	}
}

func TestTargetCreateAndCloseTarget(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create a new page target.
	createResp, err := env.targetClient.CreateTarget(ctx, &targetpb.CreateTargetRequest{
		Url: "about:blank",
	})
	if err != nil {
		t.Fatalf("CreateTarget: %v", err)
	}
	if createResp.TargetId == "" {
		t.Fatal("expected non-empty target ID")
	}
	t.Logf("Created target: %s", createResp.TargetId)

	// Verify it exists.
	targets, err := env.targetClient.GetTargets(ctx, &targetpb.GetTargetsRequest{})
	if err != nil {
		t.Fatalf("GetTargets: %v", err)
	}
	var found bool
	for _, ti := range targets.TargetInfos {
		if ti.TargetId == createResp.TargetId {
			found = true
			break
		}
	}
	if !found {
		t.Error("newly created target not found in target list")
	}

	// Close it.
	closeResp, err := env.targetClient.CloseTarget(ctx, &targetpb.CloseTargetRequest{
		TargetId: createResp.TargetId,
	})
	if err != nil {
		t.Fatalf("CloseTarget: %v", err)
	}
	if !closeResp.Success {
		t.Error("CloseTarget returned success=false")
	}
}

func TestTargetGetBrowserContexts(t *testing.T) {
	// Browser context operations require the browser-level session (no sessionId).
	// setupTestEnv attaches to a page target. We need a separate env without
	// the page-level session to test browser context commands.
	t.Helper()
	ctx := context.Background()

	client, launchResult, err := cdpclient.ConnectOrLaunch(ctx, "", cdpclient.LaunchConfig{
		Headless: true,
		ExtraArgs: []string{
			"--no-sandbox",
			"--disable-gpu",
			"--disable-dev-shm-usage",
		},
	})
	if err != nil {
		t.Skipf("Chrome not available: %v", err)
	}
	defer func() {
		client.Close()
		launchResult.Process.Kill()
		launchResult.Cmd.Wait()
		if launchResult.TempDir != "" {
			os.RemoveAll(launchResult.TempDir)
		}
	}()
	time.Sleep(200 * time.Millisecond)

	// Use the Target server without a page-level session — browser-level.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	targetpb.RegisterTargetServiceServer(grpcServer, targetserver.New(client))
	go grpcServer.Serve(lis)
	defer grpcServer.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	tc := targetpb.NewTargetServiceClient(conn)

	resp, err := tc.GetBrowserContexts(ctx, &targetpb.GetBrowserContextsRequest{})
	if err != nil {
		t.Fatalf("GetBrowserContexts: %v", err)
	}
	t.Logf("Browser contexts: %v", resp.BrowserContextIds)
}

func TestTargetCreateBrowserContext(t *testing.T) {
	// Browser context operations require browser-level connection (no sessionId).
	ctx := context.Background()

	client, launchResult, err := cdpclient.ConnectOrLaunch(ctx, "", cdpclient.LaunchConfig{
		Headless: true,
		ExtraArgs: []string{
			"--no-sandbox",
			"--disable-gpu",
			"--disable-dev-shm-usage",
		},
	})
	if err != nil {
		t.Skipf("Chrome not available: %v", err)
	}
	defer func() {
		client.Close()
		launchResult.Process.Kill()
		launchResult.Cmd.Wait()
		if launchResult.TempDir != "" {
			os.RemoveAll(launchResult.TempDir)
		}
	}()
	time.Sleep(200 * time.Millisecond)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	targetpb.RegisterTargetServiceServer(grpcServer, targetserver.New(client))
	go grpcServer.Serve(lis)
	defer grpcServer.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	tc := targetpb.NewTargetServiceClient(conn)

	// Create a new browser context (incognito).
	createResp, err := tc.CreateBrowserContext(ctx, &targetpb.CreateBrowserContextRequest{
		DisposeOnDetach: true,
	})
	if err != nil {
		t.Fatalf("CreateBrowserContext: %v", err)
	}
	if createResp.BrowserContextId == "" {
		t.Fatal("expected non-empty browser context ID")
	}
	t.Logf("Created browser context: %s", createResp.BrowserContextId)

	// Dispose it.
	_, err = tc.DisposeBrowserContext(ctx, &targetpb.DisposeBrowserContextRequest{
		BrowserContextId: createResp.BrowserContextId,
	})
	if err != nil {
		t.Fatalf("DisposeBrowserContext: %v", err)
	}
}

func TestTargetAttachDetach(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create a target to attach to.
	createResp, err := env.targetClient.CreateTarget(ctx, &targetpb.CreateTargetRequest{
		Url: "about:blank",
	})
	if err != nil {
		t.Fatalf("CreateTarget: %v", err)
	}
	defer env.targetClient.CloseTarget(ctx, &targetpb.CloseTargetRequest{TargetId: createResp.TargetId})

	// Attach.
	attachResp, err := env.targetClient.AttachToTarget(ctx, &targetpb.AttachToTargetRequest{
		TargetId: createResp.TargetId,
		Flatten:  true,
	})
	if err != nil {
		t.Fatalf("AttachToTarget: %v", err)
	}
	if attachResp.SessionId == "" {
		t.Fatal("expected non-empty session ID")
	}
	t.Logf("Attached with session: %s", attachResp.SessionId)

	// Detach.
	_, err = env.targetClient.DetachFromTarget(ctx, &targetpb.DetachFromTargetRequest{
		SessionId: attachResp.SessionId,
	})
	if err != nil {
		t.Fatalf("DetachFromTarget: %v", err)
	}
}

// =============================================================
// Page Domain Tests
// =============================================================

func TestPageEnable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	if err != nil {
		t.Fatalf("Page.Enable: %v", err)
	}
}

func TestPageNavigate(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Enable page events first.
	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})

	resp, err := env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Hello ChromeRPC</h1>",
	})
	if err != nil {
		t.Fatalf("Page.Navigate: %v", err)
	}
	if resp.FrameId == "" {
		t.Error("expected non-empty frame ID")
	}
	t.Logf("Navigate: frameId=%s loaderId=%s", resp.FrameId, resp.LoaderId)

	// Give it a moment to load.
	time.Sleep(500 * time.Millisecond)
}

func TestPageCaptureScreenshot(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Navigate to a page with content.
	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1 style='font-size:72px;color:red'>Screenshot Test</h1>",
	})
	time.Sleep(500 * time.Millisecond)

	// Capture as PNG (default).
	resp, err := env.pageClient.CaptureScreenshot(ctx, &pagepb.CaptureScreenshotRequest{})
	if err != nil {
		t.Fatalf("CaptureScreenshot (default): %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("expected non-empty screenshot data")
	}
	// PNG starts with \x89PNG
	if len(resp.Data) < 4 || resp.Data[0] != 0x89 || resp.Data[1] != 'P' || resp.Data[2] != 'N' || resp.Data[3] != 'G' {
		t.Errorf("expected PNG data, got first 4 bytes: %x", resp.Data[:min(4, len(resp.Data))])
	}
	t.Logf("PNG screenshot: %d bytes", len(resp.Data))

	// Capture as JPEG.
	jpegResp, err := env.pageClient.CaptureScreenshot(ctx, &pagepb.CaptureScreenshotRequest{
		Format:  pagepb.ScreenshotFormat_SCREENSHOT_FORMAT_JPEG,
		Quality: 80,
	})
	if err != nil {
		t.Fatalf("CaptureScreenshot (jpeg): %v", err)
	}
	if len(jpegResp.Data) == 0 {
		t.Fatal("expected non-empty JPEG data")
	}
	// JPEG starts with \xff\xd8
	if len(jpegResp.Data) < 2 || jpegResp.Data[0] != 0xFF || jpegResp.Data[1] != 0xD8 {
		t.Errorf("expected JPEG data, got first 2 bytes: %x", jpegResp.Data[:min(2, len(jpegResp.Data))])
	}
	t.Logf("JPEG screenshot: %d bytes", len(jpegResp.Data))

	// Capture with clip region.
	clipResp, err := env.pageClient.CaptureScreenshot(ctx, &pagepb.CaptureScreenshotRequest{
		Format: pagepb.ScreenshotFormat_SCREENSHOT_FORMAT_PNG,
		Clip: &pagepb.Viewport{
			X:      0,
			Y:      0,
			Width:  200,
			Height: 200,
			Scale:  1,
		},
	})
	if err != nil {
		t.Fatalf("CaptureScreenshot (clip): %v", err)
	}
	if len(clipResp.Data) == 0 {
		t.Fatal("expected non-empty clipped screenshot")
	}
	t.Logf("Clipped screenshot: %d bytes", len(clipResp.Data))
}

func TestPageCaptureSnapshot(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Snapshot Test</h1><p>Some content</p>",
	})
	time.Sleep(500 * time.Millisecond)

	resp, err := env.pageClient.CaptureSnapshot(ctx, &pagepb.CaptureSnapshotRequest{
		Format: pagepb.CaptureSnapshotFormat_CAPTURE_SNAPSHOT_FORMAT_MHTML,
	})
	if err != nil {
		t.Fatalf("CaptureSnapshot: %v", err)
	}
	if resp.Data == "" {
		t.Fatal("expected non-empty snapshot data")
	}
	// MHTML files start with specific headers.
	if len(resp.Data) < 20 {
		t.Errorf("snapshot data too short: %d bytes", len(resp.Data))
	}
	t.Logf("MHTML snapshot: %d bytes", len(resp.Data))
}

func TestPagePrintToPDF(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>PDF Test</h1><p>Page content for PDF generation</p>",
	})
	time.Sleep(500 * time.Millisecond)

	// Default PDF.
	resp, err := env.pageClient.PrintToPDF(ctx, &pagepb.PrintToPDFRequest{})
	if err != nil {
		t.Fatalf("PrintToPDF (default): %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("expected non-empty PDF data")
	}
	// PDF starts with %PDF
	if len(resp.Data) < 4 || string(resp.Data[:4]) != "%PDF" {
		t.Errorf("expected PDF data, got first 4 bytes: %q", string(resp.Data[:min(4, len(resp.Data))]))
	}
	t.Logf("PDF: %d bytes", len(resp.Data))

	// PDF with options.
	optResp, err := env.pageClient.PrintToPDF(ctx, &pagepb.PrintToPDFRequest{
		Landscape:      true,
		PrintBackground: true,
		Scale:          0.5,
		PaperWidth:     11,
		PaperHeight:    17,
	})
	if err != nil {
		t.Fatalf("PrintToPDF (options): %v", err)
	}
	if len(optResp.Data) == 0 {
		t.Fatal("expected non-empty PDF data with options")
	}
	t.Logf("PDF with options: %d bytes", len(optResp.Data))
}

func TestPageGetFrameTree(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Frame Tree Test</h1>",
	})
	time.Sleep(500 * time.Millisecond)

	resp, err := env.pageClient.GetFrameTree(ctx, &pagepb.GetFrameTreeRequest{})
	if err != nil {
		t.Fatalf("GetFrameTree: %v", err)
	}
	if resp.FrameTree == nil {
		t.Fatal("expected non-nil frame tree")
	}
	if resp.FrameTree.Frame == nil {
		t.Fatal("expected non-nil root frame")
	}
	t.Logf("Root frame: id=%s url=%s mimeType=%s",
		resp.FrameTree.Frame.Id,
		resp.FrameTree.Frame.Url,
		resp.FrameTree.Frame.MimeType,
	)
}

func TestPageGetLayoutMetrics(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<div style='width:2000px;height:3000px'>Big content</div>",
	})
	time.Sleep(500 * time.Millisecond)

	resp, err := env.pageClient.GetLayoutMetrics(ctx, &pagepb.GetLayoutMetricsRequest{})
	if err != nil {
		t.Fatalf("GetLayoutMetrics: %v", err)
	}
	if resp.CssLayoutViewport == nil {
		t.Fatal("expected non-nil CSS layout viewport")
	}
	if resp.CssVisualViewport == nil {
		t.Fatal("expected non-nil CSS visual viewport")
	}
	if resp.CssContentSize == nil {
		t.Fatal("expected non-nil CSS content size")
	}
	t.Logf("Layout: viewport=%dx%d content=%.0fx%.0f",
		resp.CssLayoutViewport.ClientWidth,
		resp.CssLayoutViewport.ClientHeight,
		resp.CssContentSize.Width,
		resp.CssContentSize.Height,
	)
}

func TestPageGetNavigationHistory(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})

	// Navigate to create history.
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Page 1</h1>",
	})
	time.Sleep(300 * time.Millisecond)

	resp, err := env.pageClient.GetNavigationHistory(ctx, &pagepb.GetNavigationHistoryRequest{})
	if err != nil {
		t.Fatalf("GetNavigationHistory: %v", err)
	}
	if len(resp.Entries) == 0 {
		t.Fatal("expected at least one navigation entry")
	}
	t.Logf("Navigation history: %d entries, current=%d", len(resp.Entries), resp.CurrentIndex)
	for _, e := range resp.Entries {
		t.Logf("  entry: id=%d url=%s title=%s", e.Id, e.Url, e.Title)
	}
}

func TestPageSetDocumentContent(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	navResp, err := env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "about:blank",
	})
	if err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	// Set document content.
	_, err = env.pageClient.SetDocumentContent(ctx, &pagepb.SetDocumentContentRequest{
		FrameId: navResp.FrameId,
		Html:    "<html><body><h1>Injected Content</h1></body></html>",
	})
	if err != nil {
		t.Fatalf("SetDocumentContent: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// Verify by taking a screenshot (should show the injected content).
	ssResp, err := env.pageClient.CaptureScreenshot(ctx, &pagepb.CaptureScreenshotRequest{})
	if err != nil {
		t.Fatalf("CaptureScreenshot after SetDocumentContent: %v", err)
	}
	if len(ssResp.Data) == 0 {
		t.Error("expected non-empty screenshot")
	}
	t.Logf("Screenshot after content injection: %d bytes", len(ssResp.Data))
}

func TestPageReload(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Reload Test</h1>",
	})
	time.Sleep(300 * time.Millisecond)

	_, err := env.pageClient.Reload(ctx, &pagepb.ReloadRequest{
		IgnoreCache: true,
	})
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
}

func TestPageSetBypassCSP(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.pageClient.SetBypassCSP(ctx, &pagepb.SetBypassCSPRequest{
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("SetBypassCSP: %v", err)
	}

	// Disable it again.
	_, err = env.pageClient.SetBypassCSP(ctx, &pagepb.SetBypassCSPRequest{
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("SetBypassCSP (disable): %v", err)
	}
}

func TestPageAddRemoveScript(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})

	// Add a script.
	addResp, err := env.pageClient.AddScriptToEvaluateOnNewDocument(ctx, &pagepb.AddScriptToEvaluateOnNewDocumentRequest{
		Source: "window.__chromerpc_test = true;",
	})
	if err != nil {
		t.Fatalf("AddScriptToEvaluateOnNewDocument: %v", err)
	}
	if addResp.Identifier == "" {
		t.Fatal("expected non-empty script identifier")
	}
	t.Logf("Script identifier: %s", addResp.Identifier)

	// Remove it.
	_, err = env.pageClient.RemoveScriptToEvaluateOnNewDocument(ctx, &pagepb.RemoveScriptToEvaluateOnNewDocumentRequest{
		Identifier: addResp.Identifier,
	})
	if err != nil {
		t.Fatalf("RemoveScriptToEvaluateOnNewDocument: %v", err)
	}
}

func TestPageSetLifecycleEventsEnabled(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})

	_, err := env.pageClient.SetLifecycleEventsEnabled(ctx, &pagepb.SetLifecycleEventsEnabledRequest{
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("SetLifecycleEventsEnabled: %v", err)
	}

	// Navigate and let lifecycle events fire.
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Lifecycle Test</h1>",
	})
	time.Sleep(500 * time.Millisecond)

	_, err = env.pageClient.SetLifecycleEventsEnabled(ctx, &pagepb.SetLifecycleEventsEnabledRequest{
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("SetLifecycleEventsEnabled (disable): %v", err)
	}
}

func TestPageCreateIsolatedWorld(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	navResp, err := env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Isolated World Test</h1>",
	})
	if err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	resp, err := env.pageClient.CreateIsolatedWorld(ctx, &pagepb.CreateIsolatedWorldRequest{
		FrameId:   navResp.FrameId,
		WorldName: "chromerpc-test",
	})
	if err != nil {
		t.Fatalf("CreateIsolatedWorld: %v", err)
	}
	if resp.ExecutionContextId == 0 {
		t.Error("expected non-zero execution context ID")
	}
	t.Logf("Isolated world context: %d", resp.ExecutionContextId)
}

func TestPageGetAppManifest(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Manifest Test</h1>",
	})
	time.Sleep(300 * time.Millisecond)

	// Should return empty manifest for a data: URL (no errors expected).
	resp, err := env.pageClient.GetAppManifest(ctx, &pagepb.GetAppManifestRequest{})
	if err != nil {
		t.Fatalf("GetAppManifest: %v", err)
	}
	t.Logf("Manifest URL: %q, data length: %d, errors: %d", resp.Url, len(resp.Data), len(resp.Errors))
}

func TestPageStopLoading(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.pageClient.StopLoading(ctx, &pagepb.StopLoadingRequest{})
	if err != nil {
		t.Fatalf("StopLoading: %v", err)
	}
}

func TestPageBringToFront(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.pageClient.BringToFront(ctx, &pagepb.BringToFrontRequest{})
	if err != nil {
		t.Fatalf("BringToFront: %v", err)
	}
}

// TestFullWorkflow tests a realistic workflow: navigate → screenshot → PDF.
func TestFullWorkflow(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// 1. Enable page domain.
	_, err := env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}

	// 2. Enable lifecycle events.
	_, err = env.pageClient.SetLifecycleEventsEnabled(ctx, &pagepb.SetLifecycleEventsEnabledRequest{
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("SetLifecycleEventsEnabled: %v", err)
	}

	// 3. Navigate to a real-looking page.
	navResp, err := env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: `data:text/html,<!DOCTYPE html>
<html>
<head><title>ChromeRPC Test</title></head>
<body style="font-family:sans-serif;padding:40px">
  <h1>ChromeRPC Integration Test</h1>
  <p>This page was rendered by headless Chrome and captured via gRPC.</p>
  <ul>
    <li>Target management: working</li>
    <li>Page navigation: working</li>
    <li>Screenshot capture: working</li>
    <li>PDF generation: working</li>
  </ul>
</body>
</html>`,
	})
	if err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	t.Logf("Navigated: frameId=%s", navResp.FrameId)
	time.Sleep(500 * time.Millisecond)

	// 4. Get frame tree.
	ftResp, err := env.pageClient.GetFrameTree(ctx, &pagepb.GetFrameTreeRequest{})
	if err != nil {
		t.Fatalf("GetFrameTree: %v", err)
	}
	t.Logf("Frame: url=%s mime=%s", ftResp.FrameTree.Frame.Url, ftResp.FrameTree.Frame.MimeType)

	// 5. Capture screenshot.
	ssResp, err := env.pageClient.CaptureScreenshot(ctx, &pagepb.CaptureScreenshotRequest{
		Format: pagepb.ScreenshotFormat_SCREENSHOT_FORMAT_PNG,
	})
	if err != nil {
		t.Fatalf("CaptureScreenshot: %v", err)
	}
	if len(ssResp.Data) < 1000 {
		t.Errorf("screenshot seems too small: %d bytes", len(ssResp.Data))
	}
	t.Logf("Screenshot: %d bytes", len(ssResp.Data))

	// 6. Print to PDF.
	pdfResp, err := env.pageClient.PrintToPDF(ctx, &pagepb.PrintToPDFRequest{
		PrintBackground: true,
	})
	if err != nil {
		t.Fatalf("PrintToPDF: %v", err)
	}
	if len(pdfResp.Data) < 1000 {
		t.Errorf("PDF seems too small: %d bytes", len(pdfResp.Data))
	}
	t.Logf("PDF: %d bytes", len(pdfResp.Data))

	// 7. Capture MHTML snapshot.
	snapResp, err := env.pageClient.CaptureSnapshot(ctx, &pagepb.CaptureSnapshotRequest{
		Format: pagepb.CaptureSnapshotFormat_CAPTURE_SNAPSHOT_FORMAT_MHTML,
	})
	if err != nil {
		t.Fatalf("CaptureSnapshot: %v", err)
	}
	if len(snapResp.Data) < 100 {
		t.Errorf("snapshot seems too small: %d bytes", len(snapResp.Data))
	}
	t.Logf("MHTML snapshot: %d bytes", len(snapResp.Data))

	// 8. Get layout metrics.
	lmResp, err := env.pageClient.GetLayoutMetrics(ctx, &pagepb.GetLayoutMetricsRequest{})
	if err != nil {
		t.Fatalf("GetLayoutMetrics: %v", err)
	}
	t.Logf("Content size: %.0fx%.0f", lmResp.CssContentSize.Width, lmResp.CssContentSize.Height)
}
