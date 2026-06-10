// Command loadtest drives the interactive pool service under concurrent load.
//
// It runs N concurrent bidi sessions, each executing the SAME multi-step
// workflow (visit accretional.com + screenshot, visit statue.dev + screenshot,
// click, read), sending each step and waiting for its response before sending
// the next — i.e. a realistic interactive client, not a fire-and-forget batch.
//
// Sessions run in "pulses": all N launched at once, then a gap measured from the
// last finisher before the next pulse. Reports per-pulse success/failure and
// session latency percentiles so you can spot regressions across pulses.
//
// Usage:
//
//	go run ./loadtest -addr HOST:443 -tls -token "$(gcloud auth print-identity-token)" \
//	    -concurrency 20 -pulses 3 -gap 30s
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/accretional/chromerpc/proto/cdp/headlessbrowser"
)

// workflow is the fixed multi-step sequence every session runs.
func workflow() []*pb.AutomationStep {
	return []*pb.AutomationStep{
		{Label: "viewport", Action: &pb.AutomationStep_SetViewport{SetViewport: &pb.SetViewport{Width: 1280, Height: 800, DeviceScaleFactor: 1}}},
		{Label: "nav_accretional", Action: &pb.AutomationStep_Navigate{Navigate: &pb.Navigate{Url: "https://accretional.com", WaitUntil: "networkidle", TimeoutMs: 30000}}},
		{Label: "screenshot_1", Action: &pb.AutomationStep_Screenshot{Screenshot: &pb.Screenshot{Format: "png"}}},
		{Label: "nav_statue", Action: &pb.AutomationStep_Navigate{Navigate: &pb.Navigate{Url: "https://statue.dev", WaitUntil: "networkidle", TimeoutMs: 30000}}},
		{Label: "screenshot_2", Action: &pb.AutomationStep_Screenshot{Screenshot: &pb.Screenshot{Format: "png"}}},
		{Label: "click", Action: &pb.AutomationStep_Click{Click: &pb.Click{X: 100, Y: 100}}},
		{Label: "read", Action: &pb.AutomationStep_EvaluateScript{EvaluateScript: &pb.EvaluateScript{Expression: "JSON.stringify({title: document.title, url: location.href})"}}},
	}
}

type result struct {
	ok  bool
	dur time.Duration
	err string
}

func runSession(ctx context.Context, client pb.InteractiveSessionServiceClient) result {
	start := time.Now()
	stream, err := client.Session(ctx)
	if err != nil {
		return result{err: "open: " + err.Error(), dur: time.Since(start)}
	}
	// The server sends a SessionReady first.
	if _, err := stream.Recv(); err != nil {
		return result{err: "ready: " + err.Error(), dur: time.Since(start)}
	}
	for i, st := range workflow() {
		if err := stream.Send(&pb.SessionRequest{Id: uint64(i + 1), Command: &pb.SessionRequest_Step{Step: st}}); err != nil {
			return result{err: fmt.Sprintf("send %s: %v", st.Label, err), dur: time.Since(start)}
		}
		resp, err := stream.Recv()
		if err != nil {
			return result{err: fmt.Sprintf("recv %s: %v", st.Label, err), dur: time.Since(start)}
		}
		if e := resp.GetError(); e != nil {
			return result{err: fmt.Sprintf("session error at %s: %s", st.Label, e.Message), dur: time.Since(start)}
		}
		if r := resp.GetResult(); r != nil && !r.Success {
			return result{err: fmt.Sprintf("step %s failed: %s", st.Label, r.Error), dur: time.Since(start)}
		}
	}
	stream.CloseSend()
	for { // drain to EOF so the server-side session closes cleanly
		if _, err := stream.Recv(); err != nil {
			break
		}
	}
	return result{ok: true, dur: time.Since(start)}
}

func pct(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	i := int(p / 100 * float64(len(sorted)))
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i]
}

func main() {
	addr := flag.String("addr", "", "service host:port")
	useTLS := flag.Bool("tls", true, "use TLS (Cloud Run)")
	token := flag.String("token", "", "bearer identity token (gcloud auth print-identity-token)")
	conc := flag.Int("concurrency", 20, "concurrent sessions per pulse")
	pulses := flag.Int("pulses", 3, "number of pulses")
	gap := flag.Duration("gap", 30*time.Second, "wait after the last finisher before the next pulse")
	flag.Parse()
	if *addr == "" {
		log.Fatal("-addr is required")
	}

	var tc grpc.DialOption
	if *useTLS {
		tc = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	} else {
		tc = grpc.WithTransportCredentials(insecure.NewCredentials())
	}
	conn, err := grpc.NewClient(*addr, tc)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewInteractiveSessionServiceClient(conn)

	baseCtx := context.Background()
	if *token != "" {
		baseCtx = metadata.AppendToOutgoingContext(baseCtx, "authorization", "Bearer "+*token)
	}

	fmt.Printf("loadtest -> %s | %d concurrent x %d pulses, %s gap\n", *addr, *conc, *pulses, *gap)
	fmt.Printf("test window start (UTC): %s\n", time.Now().UTC().Format(time.RFC3339))

	var totalOK, totalFail int
	for p := 1; p <= *pulses; p++ {
		fmt.Printf("\n=== PULSE %d/%d — launching %d sessions simultaneously ===\n", p, *pulses, *conc)
		results := make([]result, *conc)
		var wg sync.WaitGroup
		pulseStart := time.Now()
		for i := 0; i < *conc; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(baseCtx, 5*time.Minute)
				defer cancel()
				results[idx] = runSession(ctx, client)
			}(i)
		}
		wg.Wait()
		pulseWall := time.Since(pulseStart)

		ok, fail := 0, 0
		var durs []time.Duration
		failMsgs := map[string]int{}
		for _, r := range results {
			if r.ok {
				ok++
				durs = append(durs, r.dur)
			} else {
				fail++
				failMsgs[r.err]++
			}
		}
		totalOK += ok
		totalFail += fail
		sort.Slice(durs, func(a, b int) bool { return durs[a] < durs[b] })
		maxd := time.Duration(0)
		if len(durs) > 0 {
			maxd = durs[len(durs)-1]
		}
		fmt.Printf("pulse %d: ok=%d fail=%d | wall=%s | session p50=%s p95=%s max=%s\n",
			p, ok, fail, pulseWall.Round(time.Millisecond),
			pct(durs, 50).Round(time.Millisecond), pct(durs, 95).Round(time.Millisecond), maxd.Round(time.Millisecond))
		for msg, n := range failMsgs {
			fmt.Printf("   FAIL x%d: %s\n", n, msg)
		}
		if p < *pulses {
			fmt.Printf("... cooling down %s before next pulse\n", *gap)
			time.Sleep(*gap)
		}
	}

	fmt.Printf("\ntest window end (UTC): %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Printf("TOTAL: ok=%d fail=%d\n", totalOK, totalFail)
	if totalFail > 0 {
		log.Fatalf("FAILURES: %d sessions failed", totalFail)
	}
	fmt.Println("ALL SESSIONS OK")
}
