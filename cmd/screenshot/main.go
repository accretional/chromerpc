// Command screenshot takes a screenshot of a URL using the chromerpc gRPC server.
// Usage: screenshot -addr localhost:50051 -url https://example.com -out screenshot.png
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pagepb "github.com/accretional/chromerpc/proto/cdp/page"
	emulationpb "github.com/accretional/chromerpc/proto/cdp/emulation"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "gRPC server address")
	url := flag.String("url", "", "URL to screenshot")
	out := flag.String("out", "screenshot.png", "output file path")
	width := flag.Int("width", 1280, "viewport width")
	height := flag.Int("height", 800, "viewport height")
	wait := flag.Duration("wait", 3*time.Second, "time to wait after navigation")
	flag.Parse()

	if *url == "" {
		fmt.Fprintln(os.Stderr, "usage: screenshot -url <URL> [-out file.png] [-width 1280] [-height 800]")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	pageClient := pagepb.NewPageServiceClient(conn)
	emulationClient := emulationpb.NewEmulationServiceClient(conn)

	// Set viewport size.
	_, err = emulationClient.SetDeviceMetricsOverride(ctx, &emulationpb.SetDeviceMetricsOverrideRequest{
		Width:             int32(*width),
		Height:            int32(*height),
		DeviceScaleFactor: 2,
		Mobile:            false,
	})
	if err != nil {
		log.Fatalf("SetDeviceMetricsOverride: %v", err)
	}

	// Enable page domain.
	pageClient.Enable(ctx, &pagepb.EnableRequest{})

	// Navigate.
	log.Printf("Navigating to %s ...", *url)
	navResp, err := pageClient.Navigate(ctx, &pagepb.NavigateRequest{Url: *url})
	if err != nil {
		log.Fatalf("Navigate: %v", err)
	}
	if navResp.ErrorText != "" {
		log.Printf("Navigation error: %s", navResp.ErrorText)
	}

	// Wait for page to load.
	time.Sleep(*wait)

	// Capture screenshot.
	log.Printf("Capturing screenshot...")
	ssResp, err := pageClient.CaptureScreenshot(ctx, &pagepb.CaptureScreenshotRequest{
		Format: pagepb.ScreenshotFormat_SCREENSHOT_FORMAT_PNG,
	})
	if err != nil {
		log.Fatalf("CaptureScreenshot: %v", err)
	}

	if err := os.WriteFile(*out, ssResp.Data, 0644); err != nil {
		log.Fatalf("write %s: %v", *out, err)
	}
	log.Printf("Saved %s (%d bytes)", *out, len(ssResp.Data))
}
