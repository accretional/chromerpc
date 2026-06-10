// Command chrome-proxy holds ONE interactive bidi Session open to a remote
// chromerpc service and fronts it with a tiny local HTTP API, so an operator (a
// human or an agent making one shell call at a time) can steer a single live
// browser session iteratively — the session, and its server-side Chrome, stay
// alive across many separate HTTP calls.
//
// Why a proxy: a gRPC bidi stream is one long-lived connection; a caller that
// can only make independent one-shot requests (like an agent issuing separate
// tool calls) can't hold that stream. The proxy holds it and exposes:
//
//	POST /steps   body: {"steps":[<AutomationStep protojson>, ...]}
//	              runs them in order on the open session, returns each result,
//	              and writes any screenshot bytes to <shots>/NNNN-<label>.png
//	              (the response carries the file path + size, not the blob).
//	GET  /health  liveness
//	POST /close   close the session and exit
//
// For human "click-through" (e.g. solving a CAPTCHA in the live session):
//
//	GET  /capture  a self-contained HTML console: live screenshot you click on
//	GET  /shot.png a fresh PNG of the session; X-Css-Width/X-Css-Height headers
//	               give the CSS viewport size the PNG should be displayed at
//	POST /clickxy  body {"x":<css>,"y":<css>} replays one Click at CSS coords
//	POST /key      body {"key":"Enter"} replays one key press
//
// Each step is a single cdp.headlessbrowser.AutomationStep in protojson, e.g.
//	{"navigate":{"url":"https://amazon.com","wait_until":"networkidle"}}
//	{"screenshot":{"format":"png"}}
//	{"evaluate_script":{"expression":"document.title"}}
//
// This is intentionally a focused, hand-written proxy over the reflected gRPC
// surface. A fully generated reflection->HTTP gateway (e.g. via accretional's
// openapi2proto / gluon / astkit tooling) would generalize this to every
// service/method; see chrome-proxy/README.md.
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/accretional/chromerpc/proto/cdp/headlessbrowser"
)

type proxy struct {
	mu       sync.Mutex // serializes access to the single stream
	stream   pb.InteractiveSessionService_SessionClient
	shotsDir string
	stepN    int
	shotN    int
}

type stepResultJSON struct {
	Index        int    `json:"index"`
	Label        string `json:"label"`
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
	ScriptResult string `json:"script_result,omitempty"`
	Screenshot   string `json:"screenshot,omitempty"` // saved file path
	ScreenshotKB int    `json:"screenshot_kb,omitempty"`
}

func (p *proxy) runSteps(steps []*pb.AutomationStep) ([]stepResultJSON, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]stepResultJSON, 0, len(steps))
	for _, st := range steps {
		p.stepN++
		if err := p.stream.Send(&pb.SessionRequest{Id: uint64(p.stepN), Command: &pb.SessionRequest_Step{Step: st}}); err != nil {
			return out, fmt.Errorf("send: %w", err)
		}
		resp, err := p.stream.Recv()
		if err != nil {
			return out, fmt.Errorf("recv: %w", err)
		}
		r := stepResultJSON{Index: p.stepN}
		if e := resp.GetError(); e != nil {
			r.Error = "session: " + e.Message
			out = append(out, r)
			continue
		}
		res := resp.GetResult()
		if res == nil {
			r.Error = "no result"
			out = append(out, r)
			continue
		}
		r.Label = res.Label
		r.Success = res.Success
		r.Error = res.Error
		r.ScriptResult = res.ScriptResult
		if len(res.ScreenshotData) > 0 {
			p.shotN++
			name := fmt.Sprintf("%04d-%s.png", p.shotN, sanitize(res.Label))
			path := filepath.Join(p.shotsDir, name)
			if err := os.WriteFile(path, res.ScreenshotData, 0644); err != nil {
				r.Error = "save screenshot: " + err.Error()
			} else {
				r.Screenshot = path
				r.ScreenshotKB = len(res.ScreenshotData) / 1024
			}
		}
		out = append(out, r)
	}
	return out, nil
}

// sendStep serializes a single step onto the one live stream, the same way
// runSteps does, and returns the raw StepResult. All stream access — here and in
// runSteps — must hold p.mu; the bidi stream cannot be used concurrently.
func (p *proxy) sendStep(st *pb.AutomationStep) (*pb.StepResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stepN++
	if err := p.stream.Send(&pb.SessionRequest{Id: uint64(p.stepN), Command: &pb.SessionRequest_Step{Step: st}}); err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}
	resp, err := p.stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("recv: %w", err)
	}
	if e := resp.GetError(); e != nil {
		return nil, fmt.Errorf("session: %s", e.Message)
	}
	res := resp.GetResult()
	if res == nil {
		return nil, fmt.Errorf("no result")
	}
	return res, nil
}

// shot returns a fresh PNG screenshot of the live session along with the CSS
// viewport size (window.innerWidth/innerHeight). The PNG is rendered at
// devicePixelRatio, so its pixel size is (cssW*DPR) x (cssH*DPR); a human click
// must be scaled back to CSS px before being replayed as a remote Click (CDP
// Input.dispatchMouseEvent uses CSS px). Callers display the PNG at cssW px wide.
func (p *proxy) shot() (png []byte, cssW int, cssH int, err error) {
	vp, err := p.sendStep(&pb.AutomationStep{Action: &pb.AutomationStep_EvaluateScript{
		EvaluateScript: &pb.EvaluateScript{Expression: "JSON.stringify({w:window.innerWidth,h:window.innerHeight})"},
	}})
	if err != nil {
		return nil, 0, 0, fmt.Errorf("viewport: %w", err)
	}
	// ScriptResult is double-encoded: the expression already returns a JSON
	// string, and the result is itself delivered JSON-stringified (so quoted).
	// Peel one layer of string quoting if present, then parse the dims.
	raw := vp.ScriptResult
	var inner string
	if json.Unmarshal([]byte(raw), &inner) == nil {
		raw = inner
	}
	var dims struct {
		W int `json:"w"`
		H int `json:"h"`
	}
	if err := json.Unmarshal([]byte(raw), &dims); err != nil {
		return nil, 0, 0, fmt.Errorf("parse viewport %q: %w", vp.ScriptResult, err)
	}
	sh, err := p.sendStep(&pb.AutomationStep{Action: &pb.AutomationStep_Screenshot{
		Screenshot: &pb.Screenshot{Format: "png"},
	}})
	if err != nil {
		return nil, 0, 0, fmt.Errorf("screenshot: %w", err)
	}
	return sh.ScreenshotData, dims.W, dims.H, nil
}

func sanitize(s string) string {
	if s == "" {
		return "step"
	}
	b := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			b = append(b, c)
		} else {
			b = append(b, '_')
		}
	}
	return string(b)
}

func main() {
	addr := flag.String("addr", "", "remote service host:port")
	useTLS := flag.Bool("tls", true, "use TLS (Cloud Run)")
	token := flag.String("token", "", "bearer identity token")
	listen := flag.String("listen", "127.0.0.1:8099", "local HTTP control address")
	shots := flag.String("shots", "chrome-proxy/shots", "directory to write screenshots")
	flag.Parse()
	if *addr == "" {
		log.Fatal("-addr required")
	}
	if err := os.MkdirAll(*shots, 0755); err != nil {
		log.Fatalf("shots dir: %v", err)
	}

	var tc grpc.DialOption
	if *useTLS {
		tc = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	} else {
		tc = grpc.WithTransportCredentials(insecure.NewCredentials())
	}
	// Keepalive pings keep the long-lived bidi stream alive across idle gaps
	// between steps (an operator thinking, or an agent between tool calls) — the
	// HTTP/2 connection would otherwise be idle-closed, ending the session.
	ka := grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                30 * time.Second,
		Timeout:             10 * time.Second,
		PermitWithoutStream: true,
	})
	conn, err := grpc.NewClient(*addr, tc, ka)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	if *token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+*token)
	}
	stream, err := pb.NewInteractiveSessionServiceClient(conn).Session(ctx)
	if err != nil {
		log.Fatalf("open session: %v", err)
	}
	ready, err := stream.Recv()
	if err != nil {
		log.Fatalf("await ready: %v", err)
	}
	log.Printf("session ready: %v", ready.GetReady().GetSessionId())

	p := &proxy{stream: stream, shotsDir: *shots}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "ok") })
	mux.HandleFunc("/steps", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// Accept either {"steps":[...]} or a bare [...] array.
		var wrap struct {
			Steps []json.RawMessage `json:"steps"`
		}
		var raws []json.RawMessage
		if json.Unmarshal(body, &wrap) == nil && len(wrap.Steps) > 0 {
			raws = wrap.Steps
		} else if json.Unmarshal(body, &raws) != nil {
			http.Error(w, "body must be {\"steps\":[...]} or [...]", 400)
			return
		}
		steps := make([]*pb.AutomationStep, 0, len(raws))
		for _, rm := range raws {
			st := &pb.AutomationStep{}
			if err := protojson.Unmarshal(rm, st); err != nil {
				http.Error(w, "bad step "+string(rm)+": "+err.Error(), 400)
				return
			}
			steps = append(steps, st)
		}
		results, err := p.runSteps(steps)
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		resp := map[string]any{"results": results}
		if err != nil {
			resp["error"] = err.Error()
		}
		enc.Encode(resp)
	})
	// Human "click-through" capture: open /capture in a browser to see a live
	// screenshot of the remote session and click it (e.g. to solve a CAPTCHA);
	// each click is replayed as a real Click at the right CSS coordinates.
	mux.HandleFunc("/shot.png", func(w http.ResponseWriter, r *http.Request) {
		png, cssW, cssH, err := p.shot()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Css-Width", fmt.Sprintf("%d", cssW))
		w.Header().Set("X-Css-Height", fmt.Sprintf("%d", cssH))
		w.Write(png)
	})
	mux.HandleFunc("/clickxy", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "body must be {\"x\":<num>,\"y\":<num>}", 400)
			return
		}
		if _, err := p.sendStep(&pb.AutomationStep{Action: &pb.AutomationStep_Click{
			Click: &pb.Click{X: body.X, Y: body.Y},
		}}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"ok":true}`)
	})
	mux.HandleFunc("/key", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Key string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
			http.Error(w, "body must be {\"key\":\"...\"}", 400)
			return
		}
		if _, err := p.sendStep(&pb.AutomationStep{Action: &pb.AutomationStep_PressKey{
			PressKey: &pb.PressKey{Key: body.Key},
		}}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"ok":true}`)
	})
	mux.HandleFunc("/capture", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, capturePage)
	})

	srv := &http.Server{Addr: *listen, Handler: mux}
	mux.HandleFunc("/close", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		p.stream.CloseSend()
		p.mu.Unlock()
		fmt.Fprintln(w, "closing")
		go func() { time.Sleep(300 * time.Millisecond); srv.Close() }()
	})

	log.Printf("chrome-proxy: steering %s  ->  http://%s  (shots in %s)", *addr, *listen, *shots)
	log.Printf("human click-through capture: http://%s/capture", *listen)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http: %v", err)
	}
}

// capturePage is a self-contained (no external deps) operator console: it shows
// a live screenshot of the remote session at CSS scale and replays each click on
// it as a real remote Click at the matching CSS coordinates.
const capturePage = `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>chrome-proxy capture</title>
<style>
  body { font-family: system-ui, sans-serif; margin: 12px; background: #1e1e1e; color: #ddd; }
  #shot { display: block; border: 1px solid #555; cursor: crosshair; image-rendering: auto; }
  button { font-size: 14px; padding: 6px 12px; margin-right: 6px; }
  #status { margin: 8px 0; font-family: monospace; white-space: pre-wrap; }
  .bar { margin-bottom: 8px; }
</style>
</head>
<body>
<div class="bar">
  <button id="refresh">Refresh</button>
  <button id="enter">Press Enter</button>
  <label><input type="checkbox" id="auto" checked> auto-refresh (2.5s)</label>
</div>
<div id="status">loading...</div>
<img id="shot" alt="remote screenshot">
<script>
(function () {
  var img = document.getElementById('shot');
  var statusEl = document.getElementById('status');
  var autoEl = document.getElementById('auto');
  var cssW = 0, cssH = 0;
  var pauseUntil = 0; // ms epoch; suppress auto-refresh briefly after a click
  var lastBlob = null;

  function setStatus(s) { statusEl.textContent = s; }

  function loadShot() {
    return fetch('/shot.png', { cache: 'no-store' })
      .then(function (resp) {
        if (!resp.ok) { throw new Error('shot ' + resp.status); }
        cssW = parseInt(resp.headers.get('X-Css-Width'), 10) || cssW;
        cssH = parseInt(resp.headers.get('X-Css-Height'), 10) || cssH;
        return resp.blob();
      })
      .then(function (blob) {
        if (lastBlob) { URL.revokeObjectURL(lastBlob); }
        lastBlob = URL.createObjectURL(blob);
        img.src = lastBlob;
        img.style.width = cssW + 'px';
        setStatus('css viewport ' + cssW + 'x' + cssH + ' — click the image to act');
      })
      .catch(function (e) { setStatus('error: ' + e.message); });
  }

  img.addEventListener('click', function (e) {
    if (!cssW || !cssH) { return; }
    var rect = img.getBoundingClientRect();
    var cssX = Math.round((e.clientX - rect.left) / rect.width * cssW);
    var cssY = Math.round((e.clientY - rect.top) / rect.height * cssH);
    pauseUntil = Date.now() + 4000;
    setStatus('clicking css (' + cssX + ', ' + cssY + ')...');
    fetch('/clickxy', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ x: cssX, y: cssY })
    })
      .then(function (resp) {
        if (!resp.ok) { throw new Error('click ' + resp.status); }
        setStatus('clicked css (' + cssX + ', ' + cssY + ') — refreshing');
        return loadShot();
      })
      .catch(function (err) { setStatus('error: ' + err.message); });
  });

  document.getElementById('refresh').addEventListener('click', loadShot);

  document.getElementById('enter').addEventListener('click', function () {
    pauseUntil = Date.now() + 4000;
    fetch('/key', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ key: 'Enter' })
    })
      .then(function () { return loadShot(); })
      .catch(function (err) { setStatus('error: ' + err.message); });
  });

  setInterval(function () {
    if (autoEl.checked && Date.now() >= pauseUntil) { loadShot(); }
  }, 2500);

  loadShot();
})();
</script>
</body>
</html>
`
