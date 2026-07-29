package fixed

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/accretional/chromerpc/internal/cdpclient"
)

const defaultFileworkerDir = "/Users/mini2/repos/fileworker"

type browserHarness struct {
	t      *testing.T
	ctx    context.Context
	client *cdpclient.Client
	chrome *cdpclient.LaunchResult
	server *httptest.Server
	url    string
	outDir string
}

type page struct {
	h         *browserHarness
	sessionID string
}

func newHarness(t *testing.T) *browserHarness {
	t.Helper()
	root := os.Getenv("FILEWORKER_DIR")
	if root == "" {
		root = defaultFileworkerDir
	}
	if _, err := os.Stat(filepath.Join(root, "index.html")); err != nil {
		t.Fatalf("FILEWORKER_DIR is not a Fileworker checkout: %v", err)
	}

	server := httptest.NewServer(http.FileServer(http.Dir(root)))
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	t.Cleanup(cancel)
	chrome, err := cdpclient.Launch(ctx, cdpclient.LaunchConfig{
		Headless: true,
		ExtraArgs: []string{
			"--window-size=1440,900",
			"--force-device-scale-factor=1",
		},
	})
	if err != nil {
		server.Close()
		t.Fatalf("launch Chrome: %v", err)
	}
	client, err := cdpclient.Dial(ctx, chrome.WebSocketURL)
	if err != nil {
		chrome.Cleanup()
		server.Close()
		t.Fatalf("connect to Chrome: %v", err)
	}
	outDir := filepath.Join("artifacts")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	h := &browserHarness{t: t, ctx: ctx, client: client, chrome: chrome, server: server, url: server.URL + "/", outDir: outDir}
	t.Cleanup(func() {
		client.Close()
		chrome.Cleanup()
		server.Close()
	})
	return h
}

func (h *browserHarness) send(method string, params any) json.RawMessage {
	h.t.Helper()
	raw, err := h.client.Send(h.ctx, method, params)
	if err != nil {
		h.t.Fatalf("%s: %v", method, err)
	}
	return raw
}

func (h *browserHarness) newPage() *page {
	h.t.Helper()
	var created struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(h.send("Target.createTarget", map[string]any{"url": "about:blank"}), &created); err != nil {
		h.t.Fatal(err)
	}
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(h.send("Target.attachToTarget", map[string]any{"targetId": created.TargetID, "flatten": true}), &attached); err != nil {
		h.t.Fatal(err)
	}
	p := &page{h: h, sessionID: attached.SessionID}
	p.command("Page.enable", nil)
	p.command("Runtime.enable", nil)
	p.command("Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 1440, "height": 900, "deviceScaleFactor": 1, "mobile": false,
	})
	p.command("Page.navigate", map[string]any{"url": h.url})
	p.wait("document.readyState === 'complete'", 10*time.Second)
	p.wait("document.querySelector('#worker-status')?.textContent.includes('ONLINE') && document.querySelector('#storage-status')?.textContent.includes('READY')", 15*time.Second)
	return p
}

func (p *page) command(method string, params any) json.RawMessage {
	p.h.t.Helper()
	raw, err := p.h.client.SendWithSession(p.h.ctx, method, params, p.sessionID)
	if err != nil {
		p.h.t.Fatalf("%s: %v", method, err)
	}
	return raw
}

func (p *page) eval(expression string) any {
	p.h.t.Helper()
	var response struct {
		Result struct {
			Type        string          `json:"type"`
			Value       json.RawMessage `json:"value"`
			Description string          `json:"description"`
		} `json:"result"`
		ExceptionDetails json.RawMessage `json:"exceptionDetails"`
	}
	raw := p.command("Runtime.evaluate", map[string]any{
		"expression": expression, "awaitPromise": true, "returnByValue": true,
	})
	if err := json.Unmarshal(raw, &response); err != nil {
		p.h.t.Fatal(err)
	}
	if len(response.ExceptionDetails) > 0 && string(response.ExceptionDetails) != "null" {
		p.h.t.Fatalf("JavaScript exception evaluating %q: %s", expression, response.ExceptionDetails)
	}
	if response.Result.Type == "undefined" {
		return nil
	}
	var value any
	if len(response.Result.Value) > 0 {
		if err := json.Unmarshal(response.Result.Value, &value); err != nil {
			p.h.t.Fatal(err)
		}
	}
	return value
}

func (p *page) wait(predicate string, timeout time.Duration) {
	p.h.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if value, _ := p.eval("Boolean(" + predicate + ")").(bool); value {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	p.h.t.Fatalf("timed out waiting for: %s", predicate)
}

func (p *page) screenshot(name string) {
	p.h.t.Helper()
	var result struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(p.command("Page.captureScreenshot", map[string]any{"format": "png", "fromSurface": true}), &result); err != nil {
		p.h.t.Fatal(err)
	}
	data, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		p.h.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.h.outDir, name), data, 0644); err != nil {
		p.h.t.Fatal(err)
	}
}

func (p *page) click(selector string) {
	p.h.t.Helper()
	value := p.eval(fmt.Sprintf(`(() => {
		const element = document.querySelector(%s);
		if (!element) throw new Error("click target not found");
		const rect = element.getBoundingClientRect();
		return {x: rect.left + rect.width / 2, y: rect.top + rect.height / 2};
	})()`, jsString(selector)))
	raw, _ := json.Marshal(value)
	var point struct{ X, Y float64 }
	if err := json.Unmarshal(raw, &point); err != nil {
		p.h.t.Fatal(err)
	}
	for _, eventType := range []string{"mouseMoved", "mousePressed", "mouseReleased"} {
		params := map[string]any{"type": eventType, "x": point.X, "y": point.Y}
		if eventType != "mouseMoved" {
			params["button"] = "left"
			params["clickCount"] = 1
		}
		p.command("Input.dispatchMouseEvent", params)
	}
}

func dropFile(p *page, name, contents string) {
	p.h.t.Helper()
	script := fmt.Sprintf(`(() => {
		const data = new DataTransfer();
		data.items.add(new File([%s], %s, {type: "text/plain", lastModified: 1700000000000}));
		const target = document.querySelector("#drop-zone");
		target.dispatchEvent(new DragEvent("dragenter", {bubbles:true, cancelable:true, dataTransfer:data}));
		target.dispatchEvent(new DragEvent("dragover", {bubbles:true, cancelable:true, dataTransfer:data}));
		target.dispatchEvent(new DragEvent("drop", {bubbles:true, cancelable:true, dataTransfer:data}));
		return true;
	})()`, jsString(contents), jsString(name))
	p.eval(script)
}

func jsString(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func TestInspectorIsVisibleAndMoreButtonIsInteractive(t *testing.T) {
	h := newHarness(t)
	p := h.newPage()
	name := fmt.Sprintf("inspector-%d.txt", time.Now().UnixNano())
	dropFile(p, name, "fixed inspector geometry")
	p.wait(fmt.Sprintf(`document.querySelector('[data-path="%s"]')`, name), 10*time.Second)
	p.click(fmt.Sprintf(`[data-path="%s"] .more-button`, name))
	p.wait(`document.querySelector("#inspector").classList.contains("open")`, 3*time.Second)

	value := p.eval(`(() => {
		const drawer = document.querySelector("#inspector");
		const more = document.querySelector(".more-button");
		const rect = drawer.getBoundingClientRect();
		const content = document.querySelector("#inspector-content").getBoundingClientRect();
		return {
			viewport: {width: innerWidth, height: innerHeight},
			drawer: {top: rect.top, left: rect.left, right: rect.right, bottom: rect.bottom, width: rect.width, height: rect.height},
			content: {top: content.top, bottom: content.bottom},
			scrollTop: drawer.scrollTop,
			position: getComputedStyle(drawer).position,
			moreCursor: getComputedStyle(more).cursor,
			moreTitle: more.title
		};
	})()`)
	data, _ := json.Marshal(value)
	var geometry struct {
		Viewport                        struct{ Width, Height float64 }
		Drawer                          struct{ Top, Left, Right, Bottom, Width, Height float64 }
		Content                         struct{ Top, Bottom float64 }
		ScrollTop                       float64
		Position, MoreCursor, MoreTitle string
	}
	if err := json.Unmarshal(data, &geometry); err != nil {
		t.Fatal(err)
	}
	if geometry.Drawer.Top < 0 || geometry.Drawer.Top >= geometry.Viewport.Height ||
		geometry.Drawer.Bottom > geometry.Viewport.Height+1 || geometry.Drawer.Height < 300 {
		t.Errorf("inspector is not contained in viewport: %+v", geometry)
	}
	if geometry.Content.Top < geometry.Drawer.Top || geometry.Content.Top >= geometry.Viewport.Height {
		t.Errorf("inspector content starts offscreen: %+v", geometry)
	}
	if geometry.ScrollTop != 0 {
		t.Errorf("inspector opened scrolled to %.0f, want 0", geometry.ScrollTop)
	}
	if geometry.Position != "fixed" {
		t.Errorf("inspector position = %q, want fixed", geometry.Position)
	}
	if geometry.MoreCursor != "pointer" {
		t.Errorf("more button cursor = %q, want pointer", geometry.MoreCursor)
	}
	if !strings.Contains(strings.ToLower(geometry.MoreTitle), "inspector") {
		t.Errorf("more button needs a descriptive title, got %q", geometry.MoreTitle)
	}
	p.screenshot("inspector-open.png")
}

func TestDropUploadRefreshesOtherClient(t *testing.T) {
	h := newHarness(t)
	receiver := h.newPage()
	sender := h.newPage()
	name := fmt.Sprintf("cross-client-%d.txt", time.Now().UnixNano())
	dropFile(sender, name, "cross-client invalidation")

	sender.wait(fmt.Sprintf(`document.querySelector('[data-path="%s"]')`, name), 10*time.Second)
	receiver.wait(fmt.Sprintf(`document.querySelector('[data-path="%s"]')`, name), 10*time.Second)

	if location, _ := receiver.eval("location.href").(string); location != h.url {
		t.Errorf("receiver URL changed or uses demo parameters: got %q, want %q", location, h.url)
	}
	receiver.screenshot("cross-client-refresh.png")
}
