// Command isession is an interactive client for InteractiveSessionService.
//
// It opens a single bidi Session stream, reads one JSON SessionRequest per line
// from stdin, sends each, and prints every SessionResponse as it arrives. The
// stream (and the server-side Chrome) lives until stdin closes.
//
// Local:
//
//	go run ./cmd/isession -addr localhost:50051 <<'EOF'
//	{"id":1,"step":{"navigate":{"url":"https://example.com","wait_until":"load"}}}
//	{"id":2,"step":{"evaluate_script":{"expression":"document.title"}}}
//	EOF
//
// Cloud Run (TLS + identity token):
//
//	go run ./cmd/isession -addr host:443 -tls -audience https://host < script.jsonl
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/accretional/chromerpc/proto/cdp/headlessbrowser"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "server address")
	useTLS := flag.Bool("tls", false, "use TLS (required for Cloud Run)")
	audience := flag.String("audience", "", "identity-token audience (the service URL, for Cloud Run auth)")
	flag.Parse()

	ctx := context.Background()
	var opts []grpc.DialOption
	if *useTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
		if *audience != "" {
			ts, err := idtoken.NewTokenSource(ctx, *audience)
			if err != nil {
				log.Fatalf("idtoken source: %v", err)
			}
			opts = append(opts, grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: ts}))
		}
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(*addr, opts...)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	stream, err := pb.NewInteractiveSessionServiceClient(conn).Session(ctx)
	if err != nil {
		log.Fatalf("open session: %v", err)
	}

	marshal := protojson.MarshalOptions{EmitUnpopulated: false}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Printf("recv: %v", err)
				return
			}
			// Trim large screenshot blobs for readability.
			out := marshal.Format(resp)
			fmt.Println(out)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var req pb.SessionRequest
		if err := protojson.Unmarshal([]byte(line), &req); err != nil {
			log.Printf("bad request %q: %v", line, err)
			continue
		}
		if err := stream.Send(&req); err != nil {
			log.Printf("send: %v", err)
			break
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("stdin: %v", err)
	}
	stream.CloseSend()

	// Wait for the server to drain responses (and recycle Chrome) before exiting.
	select {
	case <-done:
	case <-time.After(60 * time.Second):
		log.Printf("timeout waiting for responses")
	}
}
