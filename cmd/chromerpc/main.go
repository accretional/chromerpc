// Command chromerpc starts the gRPC server that bridges Chrome DevTools Protocol.
//
// Usage:
//
//	chromerpc [flags]
//	  -addr                   gRPC listen address (default: :$PORT, else :50051)
//	  -ws-url                 Connect to existing CDP WebSocket URL (skip Chrome launch)
//	  -chrome                 Path to Chrome/Chromium binary
//	  -headless               Run Chrome in headless mode (default true)
//	  -port                   Chrome remote debugging port (0=auto, default 0)
//	  -no-sandbox             Pass --no-sandbox to Chrome (required on Cloud Run)
//	  -disable-dev-shm-usage  Pass --disable-dev-shm-usage to Chrome
//	  -autoplay               Allow media autoplay without a user gesture (for A/V recording)
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/accretional/chromerpc/internal/cdpclient"
	"github.com/accretional/chromerpc/internal/cdpsession"
	interactive "github.com/accretional/chromerpc/internal/server/interactive"
	accessibilityserver "github.com/accretional/chromerpc/internal/server/accessibility"
	autofillserver "github.com/accretional/chromerpc/internal/server/autofill"
	bluetoothemulationserver "github.com/accretional/chromerpc/internal/server/bluetoothemulation"
	headlessbrowserserver "github.com/accretional/chromerpc/internal/server/headlessbrowser"
	animationserver "github.com/accretional/chromerpc/internal/server/animation"
	auditsserver "github.com/accretional/chromerpc/internal/server/audits"
	backgroundserviceserver "github.com/accretional/chromerpc/internal/server/backgroundservice"
	browserserver "github.com/accretional/chromerpc/internal/server/browser"
	cachestorageserver "github.com/accretional/chromerpc/internal/server/cachestorage"
	castserver "github.com/accretional/chromerpc/internal/server/cast"
	consoleserver "github.com/accretional/chromerpc/internal/server/console"
	cssserver "github.com/accretional/chromerpc/internal/server/css"
	databaseserver "github.com/accretional/chromerpc/internal/server/database"
	deviceaccessserver "github.com/accretional/chromerpc/internal/server/deviceaccess"
	debuggerserver "github.com/accretional/chromerpc/internal/server/debugger"
	deviceorientationserver "github.com/accretional/chromerpc/internal/server/deviceorientation"
	domserver "github.com/accretional/chromerpc/internal/server/dom"
	domdebuggerserver "github.com/accretional/chromerpc/internal/server/domdebugger"
	domsnapshotserver "github.com/accretional/chromerpc/internal/server/domsnapshot"
	domstorageserver "github.com/accretional/chromerpc/internal/server/domstorage"
	emulationserver "github.com/accretional/chromerpc/internal/server/emulation"
	fetchserver "github.com/accretional/chromerpc/internal/server/fetch"
	heapprofilerserver "github.com/accretional/chromerpc/internal/server/heapprofiler"
	indexeddbserver "github.com/accretional/chromerpc/internal/server/indexeddb"
	inputserver "github.com/accretional/chromerpc/internal/server/input"
	inspectorserver "github.com/accretional/chromerpc/internal/server/inspector"
	ioserver "github.com/accretional/chromerpc/internal/server/io"
	layertreeserver "github.com/accretional/chromerpc/internal/server/layertree"
	logserver "github.com/accretional/chromerpc/internal/server/log"
	mediaserver "github.com/accretional/chromerpc/internal/server/media"
	memoryserver "github.com/accretional/chromerpc/internal/server/memory"
	networkserver "github.com/accretional/chromerpc/internal/server/network"
	overlayserver "github.com/accretional/chromerpc/internal/server/overlay"
	pageserver "github.com/accretional/chromerpc/internal/server/page"
	performanceserver "github.com/accretional/chromerpc/internal/server/performance"
	profilerserver "github.com/accretional/chromerpc/internal/server/profiler"
	runtimeserver "github.com/accretional/chromerpc/internal/server/runtime"
	securityserver "github.com/accretional/chromerpc/internal/server/security"
	serviceworkerserver "github.com/accretional/chromerpc/internal/server/serviceworker"
	storageserver "github.com/accretional/chromerpc/internal/server/storage"
	systeminfoserver "github.com/accretional/chromerpc/internal/server/systeminfo"
	targetserver "github.com/accretional/chromerpc/internal/server/target"
	tracingserver "github.com/accretional/chromerpc/internal/server/tracing"
	eventbreakpointsserver "github.com/accretional/chromerpc/internal/server/eventbreakpoints"
	extensionsserver "github.com/accretional/chromerpc/internal/server/extensions"
	filesystemserver "github.com/accretional/chromerpc/internal/server/filesystem"
	fedcmserver "github.com/accretional/chromerpc/internal/server/fedcm"
	headlessexperimentalserver "github.com/accretional/chromerpc/internal/server/headlessexperimental"
	performancetimelineserver "github.com/accretional/chromerpc/internal/server/performancetimeline"
	preloadserver "github.com/accretional/chromerpc/internal/server/preload"
	pwaserver "github.com/accretional/chromerpc/internal/server/pwa"
	schemaserver "github.com/accretional/chromerpc/internal/server/schema"
	tetheringserver "github.com/accretional/chromerpc/internal/server/tethering"
	webaudioserver "github.com/accretional/chromerpc/internal/server/webaudio"
	webauthnserver "github.com/accretional/chromerpc/internal/server/webauthn"
	accessibilitypb "github.com/accretional/chromerpc/proto/cdp/accessibility"
	autofillpb "github.com/accretional/chromerpc/proto/cdp/autofill"
	bluetoothemulationpb "github.com/accretional/chromerpc/proto/cdp/bluetoothemulation"
	headlessbrowserpb "github.com/accretional/chromerpc/proto/cdp/headlessbrowser"
	animationpb "github.com/accretional/chromerpc/proto/cdp/animation"
	auditspb "github.com/accretional/chromerpc/proto/cdp/audits"
	backgroundservicepb "github.com/accretional/chromerpc/proto/cdp/backgroundservice"
	browserpb "github.com/accretional/chromerpc/proto/cdp/browser"
	cachestoragepb "github.com/accretional/chromerpc/proto/cdp/cachestorage"
	castpb "github.com/accretional/chromerpc/proto/cdp/cast"
	consolepb "github.com/accretional/chromerpc/proto/cdp/console"
	csspb "github.com/accretional/chromerpc/proto/cdp/css"
	databasepb "github.com/accretional/chromerpc/proto/cdp/database"
	deviceaccesspb "github.com/accretional/chromerpc/proto/cdp/deviceaccess"
	debuggerpb "github.com/accretional/chromerpc/proto/cdp/debugger"
	deviceorientationpb "github.com/accretional/chromerpc/proto/cdp/deviceorientation"
	dompb "github.com/accretional/chromerpc/proto/cdp/dom"
	domdebuggerpb "github.com/accretional/chromerpc/proto/cdp/domdebugger"
	domsnapshotpb "github.com/accretional/chromerpc/proto/cdp/domsnapshot"
	domstoragepb "github.com/accretional/chromerpc/proto/cdp/domstorage"
	emulationpb "github.com/accretional/chromerpc/proto/cdp/emulation"
	fetchpb "github.com/accretional/chromerpc/proto/cdp/fetch"
	heapprofilerpb "github.com/accretional/chromerpc/proto/cdp/heapprofiler"
	indexeddbpb "github.com/accretional/chromerpc/proto/cdp/indexeddb"
	inputpb "github.com/accretional/chromerpc/proto/cdp/input"
	inspectorpb "github.com/accretional/chromerpc/proto/cdp/inspector"
	iopb "github.com/accretional/chromerpc/proto/cdp/io"
	layertreepb "github.com/accretional/chromerpc/proto/cdp/layertree"
	logpb "github.com/accretional/chromerpc/proto/cdp/log"
	mediapb "github.com/accretional/chromerpc/proto/cdp/media"
	memorypb "github.com/accretional/chromerpc/proto/cdp/memory"
	networkpb "github.com/accretional/chromerpc/proto/cdp/network"
	overlaypb "github.com/accretional/chromerpc/proto/cdp/overlay"
	pagepb "github.com/accretional/chromerpc/proto/cdp/page"
	performancepb "github.com/accretional/chromerpc/proto/cdp/performance"
	profilerpb "github.com/accretional/chromerpc/proto/cdp/profiler"
	runtimepb "github.com/accretional/chromerpc/proto/cdp/runtime"
	securitypb "github.com/accretional/chromerpc/proto/cdp/security"
	serviceworkerpb "github.com/accretional/chromerpc/proto/cdp/serviceworker"
	storagepb "github.com/accretional/chromerpc/proto/cdp/storage"
	systeminfopb "github.com/accretional/chromerpc/proto/cdp/systeminfo"
	targetpb "github.com/accretional/chromerpc/proto/cdp/target"
	tracingpb "github.com/accretional/chromerpc/proto/cdp/tracing"
	eventbreakpointspb "github.com/accretional/chromerpc/proto/cdp/eventbreakpoints"
	extensionspb "github.com/accretional/chromerpc/proto/cdp/extensions"
	filesystempb "github.com/accretional/chromerpc/proto/cdp/filesystem"
	fedcmpb "github.com/accretional/chromerpc/proto/cdp/fedcm"
	headlessexperimentalpb "github.com/accretional/chromerpc/proto/cdp/headlessexperimental"
	performancetimelinepb "github.com/accretional/chromerpc/proto/cdp/performancetimeline"
	preloadpb "github.com/accretional/chromerpc/proto/cdp/preload"
	pwapb "github.com/accretional/chromerpc/proto/cdp/pwa"
	schemapb "github.com/accretional/chromerpc/proto/cdp/schema"
	tetheringpb "github.com/accretional/chromerpc/proto/cdp/tethering"
	webaudiopb "github.com/accretional/chromerpc/proto/cdp/webaudio"
	webauthnpb "github.com/accretional/chromerpc/proto/cdp/webauthn"
)

func main() {
	addr := flag.String("addr", "", "gRPC listen address (default: :$PORT, else :50051)")
	wsURL := flag.String("ws-url", "", "CDP WebSocket URL (skip Chrome launch)")
	chromePath := flag.String("chrome", "", "Path to Chrome/Chromium binary")
	headless := flag.Bool("headless", true, "Run Chrome in headless mode")
	port := flag.Int("port", 0, "Chrome remote debugging port (0=auto)")
	userAgent := flag.String("user-agent", "", "Override Chrome user agent string")
	noSandbox := flag.Bool("no-sandbox", false, "Pass --no-sandbox to Chrome (required in restricted sandboxes like Cloud Run)")
	disableDevShmUsage := flag.Bool("disable-dev-shm-usage", false, "Pass --disable-dev-shm-usage to Chrome (avoid small /dev/shm in containers)")
	autoplay := flag.Bool("autoplay", false, "Allow media to autoplay without a user gesture (--autoplay-policy=no-user-gesture-required); needed to script <audio>/<video> playback for A/V recording.")
	interactive := flag.Bool("interactive", false, "Run only InteractiveSessionService (bidi streaming; recycles Chrome between sessions).")
	poolSize := flag.Int("pool-size", 1, "Interactive mode: number of pre-warmed Chrome processes. 1 = single Chrome (deploy concurrency=1); N>1 = pool (deploy concurrency<=N).")
	noTmpCleanup := flag.Bool("no-tmp-cleanup", false, "Leave per-process Chrome temp dirs on disk (local debugging only; on servers they live in tmpfs and would grow instance memory).")
	flag.Parse()

	// Resolve the listen address. Cloud Run (and similar platforms) inject the
	// port to listen on via the PORT env var; honor it when -addr is not set.
	listenAddr := *addr
	if listenAddr == "" {
		if p := os.Getenv("PORT"); p != "" {
			listenAddr = ":" + p
		} else {
			listenAddr = ":50051"
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	// Connect to (or launch) Chrome.
	var extraArgs []string
	if *userAgent != "" {
		extraArgs = append(extraArgs, "--user-agent="+*userAgent)
	}
	// Always add anti-detection flags.
	extraArgs = append(extraArgs, "--disable-blink-features=AutomationControlled")
	// Keep timers/rAF running at full rate even when the (offscreen, headless)
	// page is treated as backgrounded/occluded — otherwise requestAnimationFrame
	// throttles to a few Hz, which makes screencast A/V recordings choppy.
	extraArgs = append(extraArgs,
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-renderer-backgrounding")
	// Container/Cloud Run hardening flags.
	if *noSandbox {
		extraArgs = append(extraArgs, "--no-sandbox")
	}
	if *disableDevShmUsage {
		extraArgs = append(extraArgs, "--disable-dev-shm-usage")
	}
	if *autoplay {
		extraArgs = append(extraArgs, "--autoplay-policy=no-user-gesture-required")
	}

	// Interactive (bidi streaming) mode runs its own Chrome lifecycle so it can
	// recycle the browser between sessions. It serves only InteractiveSessionService.
	if *interactive {
		runInteractive(ctx, listenAddr, cdpclient.LaunchConfig{
			ChromePath: *chromePath,
			Port:       *port,
			Headless:   *headless,
			ExtraArgs:    extraArgs,
			Stderr:       os.Stderr,
			NoTmpCleanup: *noTmpCleanup,
		}, *poolSize)
		return
	}

	client, launchResult, err := cdpclient.ConnectOrLaunch(ctx, *wsURL, cdpclient.LaunchConfig{
		ChromePath: *chromePath,
		Port:       *port,
		Headless:   *headless,
		ExtraArgs:    extraArgs,
		Stderr:       os.Stderr,
		NoTmpCleanup: *noTmpCleanup,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Chrome: %v", err)
	}
	defer client.Close()

	if launchResult != nil {
		log.Printf("Chrome launched, WebSocket: %s", launchResult.WebSocketURL)
		defer launchResult.Cleanup()
	}

	// For Page domain commands, we need to attach to a page target.
	// Discover targets and attach to the first page.
	if err := cdpsession.SetupDefault(ctx, client); err != nil {
		log.Printf("Warning: could not set up default session: %v", err)
		log.Printf("Page commands may fail without a session. Use Target.AttachToTarget first.")
	}

	// Re-establish the default session after reconnection.
	client.OnReconnect = func(rctx context.Context, c *cdpclient.Client) error {
		log.Println("Re-establishing default session after reconnect...")
		if err := cdpsession.SetupDefault(rctx, c); err != nil {
			log.Printf("Warning: could not re-establish session: %v", err)
			return err
		}
		log.Println("Default session re-established.")
		return nil
	}

	// Start gRPC server.
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", listenAddr, err)
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
	auditspb.RegisterAuditsServiceServer(grpcServer, auditsserver.New(client))
	layertreepb.RegisterLayerTreeServiceServer(grpcServer, layertreeserver.New(client))
	animationpb.RegisterAnimationServiceServer(grpcServer, animationserver.New(client))
	mediapb.RegisterMediaServiceServer(grpcServer, mediaserver.New(client))
	tracingpb.RegisterTracingServiceServer(grpcServer, tracingserver.New(client))
	systeminfopb.RegisterSystemInfoServiceServer(grpcServer, systeminfoserver.New(client))
	memorypb.RegisterMemoryServiceServer(grpcServer, memoryserver.New(client))
	domdebuggerpb.RegisterDOMDebuggerServiceServer(grpcServer, domdebuggerserver.New(client))
	webaudiopb.RegisterWebAudioServiceServer(grpcServer, webaudioserver.New(client))
	inspectorpb.RegisterInspectorServiceServer(grpcServer, inspectorserver.New(client))
	databasepb.RegisterDatabaseServiceServer(grpcServer, databaseserver.New(client))
	backgroundservicepb.RegisterBackgroundServiceServiceServer(grpcServer, backgroundserviceserver.New(client))
	deviceorientationpb.RegisterDeviceOrientationServiceServer(grpcServer, deviceorientationserver.New(client))
	webauthnpb.RegisterWebAuthnServiceServer(grpcServer, webauthnserver.New(client))
	performancetimelinepb.RegisterPerformanceTimelineServiceServer(grpcServer, performancetimelineserver.New(client))
	preloadpb.RegisterPreloadServiceServer(grpcServer, preloadserver.New(client))
	eventbreakpointspb.RegisterEventBreakpointsServiceServer(grpcServer, eventbreakpointsserver.New(client))
	headlessexperimentalpb.RegisterHeadlessExperimentalServiceServer(grpcServer, headlessexperimentalserver.New(client))
	pwapb.RegisterPWAServiceServer(grpcServer, pwaserver.New(client))
	schemapb.RegisterSchemaServiceServer(grpcServer, schemaserver.New(client))
	tetheringpb.RegisterTetheringServiceServer(grpcServer, tetheringserver.New(client))
	castpb.RegisterCastServiceServer(grpcServer, castserver.New(client))
	domsnapshotpb.RegisterDOMSnapshotServiceServer(grpcServer, domsnapshotserver.New(client))
	fedcmpb.RegisterFedCmServiceServer(grpcServer, fedcmserver.New(client))
	autofillpb.RegisterAutofillServiceServer(grpcServer, autofillserver.New(client))
	extensionspb.RegisterExtensionsServiceServer(grpcServer, extensionsserver.New(client))
	deviceaccesspb.RegisterDeviceAccessServiceServer(grpcServer, deviceaccessserver.New(client))
	filesystempb.RegisterFileSystemServiceServer(grpcServer, filesystemserver.New(client))
	bluetoothemulationpb.RegisterBluetoothEmulationServiceServer(grpcServer, bluetoothemulationserver.New(client))
	headlessbrowserpb.RegisterHeadlessBrowserServiceServer(grpcServer, headlessbrowserserver.New(client))

	// Enable gRPC reflection for tools like grpcurl.
	reflection.Register(grpcServer)

	log.Printf("gRPC server listening on %s", listenAddr)

	// Shutdown on context cancellation.
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}
}

// runInteractive serves only InteractiveSessionService, backed by a ChromeManager
// that owns the Chrome lifecycle and recycles the browser between sessions.
func runInteractive(ctx context.Context, listenAddr string, cfg cdpclient.LaunchConfig, poolSize int) {
	var srv *interactive.Server
	if poolSize > 1 {
		log.Printf("interactive: warming a pool of %d Chrome processes in the background...", poolSize)
		pool := interactive.NewChromePool()
		pool.Start(ctx, cfg, poolSize) // non-blocking; server listens immediately
		srv = interactive.NewPool(pool)
	} else {
		mgr := interactive.NewChromeManager(cfg)
		if err := mgr.Start(ctx); err != nil {
			log.Fatalf("interactive: failed to start Chrome: %v", err)
		}
		defer mgr.Close()
		srv = interactive.NewSingle(mgr)
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", listenAddr, err)
	}

	// The interactive service is built for long-lived, often-idle bidi streams
	// (e.g. chrome-proxy holds one open for the life of a session, and an A/V
	// recording keeps a single step running for many seconds). Its client keeps
	// the HTTP/2 connection alive with keepalive pings — including between RPCs.
	// Permit those pings; the server default (min 5m, no pings without an active
	// stream) would otherwise GOAWAY the connection with "too_many_pings",
	// dropping the session.
	grpcServer := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    60 * time.Second,
			Timeout: 20 * time.Second,
		}),
	)
	headlessbrowserpb.RegisterInteractiveSessionServiceServer(grpcServer, srv)
	reflection.Register(grpcServer)

	log.Printf("interactive gRPC server listening on %s", listenAddr)

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}
}
