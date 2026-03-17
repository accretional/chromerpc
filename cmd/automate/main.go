// Command automate reads a text proto file containing an AutomationSequence
// and sends it to the HeadlessBrowser gRPC service for execution.
//
// Usage:
//
//	automate -addr localhost:50051 -input automation.textproto
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
	"google.golang.org/protobuf/encoding/prototext"

	pb "github.com/accretional/chromerpc/proto/cdp/headlessbrowser"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "gRPC server address")
	input := flag.String("input", "", "path to text proto file (AutomationSequence)")
	timeout := flag.Duration("timeout", 60*time.Second, "overall timeout")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(os.Stderr, "usage: automate -input <file.textproto> [-addr host:port]")
		os.Exit(1)
	}

	// Read and parse the text proto file.
	data, err := os.ReadFile(*input)
	if err != nil {
		log.Fatalf("read %s: %v", *input, err)
	}

	var seq pb.AutomationSequence
	if err := prototext.Unmarshal(data, &seq); err != nil {
		log.Fatalf("parse text proto: %v", err)
	}

	log.Printf("Loaded automation %q with %d steps from %s", seq.Name, len(seq.Steps), *input)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewHeadlessBrowserServiceClient(conn)

	result, err := client.RunAutomation(ctx, &seq)
	if err != nil {
		log.Fatalf("RunAutomation RPC error: %v", err)
	}

	for _, sr := range result.StepResults {
		status := "OK"
		if !sr.Success {
			status = "FAIL: " + sr.Error
		}
		extra := ""
		if len(sr.ScreenshotData) > 0 {
			extra = fmt.Sprintf(" (%d bytes)", len(sr.ScreenshotData))
		}
		if sr.ScriptResult != "" {
			extra = fmt.Sprintf(" => %s", sr.ScriptResult)
		}
		log.Printf("  [%s] %s%s", sr.Label, status, extra)
	}

	if !result.Success {
		log.Fatalf("Automation failed: %s", result.Error)
	}
	log.Printf("Automation %q completed successfully", seq.Name)
}
