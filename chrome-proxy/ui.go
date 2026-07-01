package main

// ui.go serves the read-only monitoring dashboard: a static, CSS-driven page
// (monitor.html, embedded) plus a websocket that bridges ChromeMan.History into
// the browser. On connect it opens a History stream and forwards each
// ConnectionHistory to the socket as a protojson text frame (bytes fields become
// base64 automatically, which the page turns into data: URLs). All navigation,
// window sizing, modal expansion and history expansion are pure CSS in the page.

import (
	"context"
	_ "embed"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"

	cmpb "github.com/accretional/chromerpc/proto/chromeman"
)

//go:embed monitor.html
var monitorHTML []byte

var upgrader = websocket.Upgrader{
	// Same-origin dashboard; allow any origin for local/dev convenience.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// registerUI mounts the dashboard ("/") and its data socket ("/ws"). It leaves
// the proxy's existing routes (/steps, /shot.png, /capture, ...) untouched.
func registerUI(mux *http.ServeMux, cm cmpb.ChromeManClient) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(monitorHTML)
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return // Upgrade already wrote an error response
		}
		defer conn.Close()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		stream, err := cm.History(ctx, &emptypb.Empty{})
		if err != nil {
			log.Printf("ui: ChromeMan.History: %v", err)
			return
		}

		// Reading is required to observe client close / pings; a read error means
		// the browser went away, so cancel the History stream.
		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					cancel()
					return
				}
			}
		}()

		marshal := protojson.MarshalOptions{EmitUnpopulated: false}
		for {
			msg, err := stream.Recv()
			if err != nil {
				return // History ended (ctx cancelled, slow-subscriber drop, or shutdown)
			}
			b, err := marshal.Marshal(msg)
			if err != nil {
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
				return
			}
		}
	})
}
