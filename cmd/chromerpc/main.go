// Command chromerpc starts the gRPC server that bridges Chrome DevTools Protocol.
//
// Usage:
//
//	chromerpc [flags]
//	  -addr       gRPC listen address (default ":50051")
//	  -ws-url     Connect to existing CDP WebSocket URL (skip Chrome launch)
//	  -chrome     Path to Chrome/Chromium binary
//	  -headless   Run Chrome in headless mode (default true)
//	  -port       Chrome remote debugging port (0=auto, default 0)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/anthropics/chromerpc/internal/cdpclient"
	pageserver "github.com/anthropics/chromerpc/internal/server/page"
	targetserver "github.com/anthropics/chromerpc/internal/server/target"
	pagepb "github.com/anthropics/chromerpc/proto/cdp/page"
	targetpb "github.com/anthropics/chromerpc/proto/cdp/target"
)

func main() {
	addr := flag.String("addr", ":50051", "gRPC listen address")
	wsURL := flag.String("ws-url", "", "CDP WebSocket URL (skip Chrome launch)")
	chromePath := flag.String("chrome", "", "Path to Chrome/Chromium binary")
	headless := flag.Bool("headless", true, "Run Chrome in headless mode")
	port := flag.Int("port", 0, "Chrome remote debugging port (0=auto)")
	flag.Parse()

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
	client, launchResult, err := cdpclient.ConnectOrLaunch(ctx, *wsURL, cdpclient.LaunchConfig{
		ChromePath: *chromePath,
		Port:       *port,
		Headless:   *headless,
		Stderr:     os.Stderr,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Chrome: %v", err)
	}
	defer client.Close()

	if launchResult != nil {
		log.Printf("Chrome launched, WebSocket: %s", launchResult.WebSocketURL)
		defer func() {
			launchResult.Process.Kill()
			launchResult.Cmd.Wait()
			if launchResult.TempDir != "" {
				os.RemoveAll(launchResult.TempDir)
			}
		}()
	}

	// For Page domain commands, we need to attach to a page target.
	// Discover targets and attach to the first page.
	if err := setupDefaultSession(ctx, client); err != nil {
		log.Printf("Warning: could not set up default session: %v", err)
		log.Printf("Page commands may fail without a session. Use Target.AttachToTarget first.")
	}

	// Start gRPC server.
	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", *addr, err)
	}

	grpcServer := grpc.NewServer()
	targetpb.RegisterTargetServiceServer(grpcServer, targetserver.New(client))
	pagepb.RegisterPageServiceServer(grpcServer, pageserver.New(client))

	// Enable gRPC reflection for tools like grpcurl.
	reflection.Register(grpcServer)

	log.Printf("gRPC server listening on %s", *addr)

	// Shutdown on context cancellation.
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}
}

// setupDefaultSession discovers page targets and attaches to the first one
// with flatten=true, setting the session ID on the client so that Page
// domain commands are routed to the correct target.
func setupDefaultSession(ctx context.Context, client *cdpclient.Client) error {
	// Get all targets.
	result, err := client.Send(ctx, "Target.getTargets", nil)
	if err != nil {
		return fmt.Errorf("getTargets: %w", err)
	}

	type targetInfo struct {
		TargetID string `json:"targetId"`
		Type     string `json:"type"`
		URL      string `json:"url"`
	}
	var resp struct {
		TargetInfos []targetInfo `json:"targetInfos"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return fmt.Errorf("unmarshal targets: %w", err)
	}

	// Find first page target.
	var pageTarget *targetInfo
	for _, t := range resp.TargetInfos {
		if t.Type == "page" {
			t := t
			pageTarget = &t
			break
		}
	}

	if pageTarget == nil {
		return fmt.Errorf("no page target found")
	}

	log.Printf("Attaching to page target %s (%s)", pageTarget.TargetID, pageTarget.URL)

	// Attach with flatten=true.
	attachResult, err := client.Send(ctx, "Target.attachToTarget", map[string]interface{}{
		"targetId": pageTarget.TargetID,
		"flatten":  true,
	})
	if err != nil {
		return fmt.Errorf("attachToTarget: %w", err)
	}

	var attachResp struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(attachResult, &attachResp); err != nil {
		return fmt.Errorf("unmarshal attach: %w", err)
	}

	client.SetSessionID(attachResp.SessionID)
	log.Printf("Session established: %s", attachResp.SessionID)
	return nil
}
