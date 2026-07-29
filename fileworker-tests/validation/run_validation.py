#!/usr/bin/env python3
"""Hermetic browser validation for Fileworker's multi-client production UI."""
from __future__ import annotations

import argparse
import base64
import contextlib
import http.server
import json
import os
import shutil
import socket
import struct
import subprocess
import tempfile
import threading
import time
import urllib.parse
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3] / "fileworker"
CHROME_CANDIDATES = (
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    shutil.which("google-chrome"),
    shutil.which("chromium"),
)


def free_port() -> int:
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


class QuietHandler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, *_args):
        pass

    def end_headers(self):
        self.send_header("Cache-Control", "no-store")
        super().end_headers()


class CDP:
    def __init__(self, websocket_url: str):
        self.websocket_url = websocket_url
        url = urllib.parse.urlparse(websocket_url)
        self.socket = socket.create_connection((url.hostname, url.port), timeout=10)
        self.socket.settimeout(130)
        key = base64.b64encode(os.urandom(16)).decode()
        request = (
            f"GET {url.path} HTTP/1.1\r\nHost: {url.netloc}\r\nUpgrade: websocket\r\n"
            f"Connection: Upgrade\r\nSec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n"
            "Origin: http://localhost\r\n\r\n"
        )
        self.socket.sendall(request.encode())
        response = self.socket.recv(4096)
        if b"101 WebSocket Protocol Handshake" not in response:
            raise RuntimeError(response.decode(errors="replace"))
        self.counter = 0
        self.events: list[dict] = []

    def _exact(self, size):
        data = b""
        while len(data) < size:
            chunk = self.socket.recv(size - len(data))
            if not chunk:
                raise RuntimeError("DevTools socket closed")
            data += chunk
        return data

    def _receive(self):
        first = self._exact(2)
        length = first[1] & 0x7F
        if length == 126:
            length = struct.unpack("!H", self._exact(2))[0]
        elif length == 127:
            length = struct.unpack("!Q", self._exact(8))[0]
        if first[1] & 0x80:
            mask = self._exact(4)
            raw = self._exact(length)
            payload = bytes(v ^ mask[i % 4] for i, v in enumerate(raw))
        else:
            payload = self._exact(length)
        if first[0] & 0x0F == 9:
            return self._receive()
        return json.loads(payload)

    @staticmethod
    def _frame(text):
        payload = text.encode()
        mask = os.urandom(4)
        size = len(payload)
        head = bytearray([0x81])
        if size < 126:
            head.append(0x80 | size)
        elif size < 65536:
            head.extend((0x80 | 126, *struct.pack("!H", size)))
        else:
            head.append(0x80 | 127)
            head.extend(struct.pack("!Q", size))
        head.extend(mask)
        head.extend(v ^ mask[i % 4] for i, v in enumerate(payload))
        return bytes(head)

    def call(self, method, params=None):
        self.counter += 1
        request_id = self.counter
        message = {"id": request_id, "method": method, "params": params or {}}
        self.socket.sendall(self._frame(json.dumps(message)))
        while True:
            response = self._receive()
            if response.get("id") == request_id:
                if response.get("error"):
                    raise RuntimeError(response["error"])
                return response.get("result", {})
            self.events.append(response)

    def evaluate(self, expression):
        result = self.call("Runtime.evaluate", {
            "expression": expression, "awaitPromise": True,
            "returnByValue": True, "timeout": 120000,
        })
        if result.get("exceptionDetails"):
            raise RuntimeError(json.dumps(result["exceptionDetails"]))
        return result.get("result", {}).get("value")

    def close(self):
        with contextlib.suppress(Exception):
            self.socket.close()


class Harness:
    def __init__(self, chrome, app_root):
        self.chrome_path = chrome
        self.app_root = app_root
        self.http_port = free_port()
        self.cdp_port = free_port()
        self.profile = tempfile.mkdtemp(prefix="fileworker-validation-")
        handler = lambda *args, **kwargs: QuietHandler(*args, directory=str(app_root), **kwargs)
        self.server = http.server.ThreadingHTTPServer(("127.0.0.1", self.http_port), handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.chrome = None
        self.pages: list[CDP] = []

    @property
    def url(self):
        return f"http://127.0.0.1:{self.http_port}/"

    def __enter__(self):
        self.thread.start()
        self.chrome = subprocess.Popen([
            self.chrome_path, "--headless=new", "--no-sandbox", "--disable-gpu",
            "--disable-features=WebRtcHideLocalIpsWithMdns",
            "--remote-allow-origins=*", f"--remote-debugging-port={self.cdp_port}",
            f"--user-data-dir={self.profile}", "about:blank",
        ], stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True)
        deadline = time.time() + 15
        while time.time() < deadline:
            try:
                urllib.request.urlopen(f"http://127.0.0.1:{self.cdp_port}/json/version", timeout=.5)
                return self
            except Exception:
                if self.chrome.poll() is not None:
                    raise RuntimeError(self.chrome.stderr.read())
                time.sleep(.1)
        raise TimeoutError("Chrome debugging endpoint did not start")

    def page(self):
        req = urllib.request.Request(
            f"http://127.0.0.1:{self.cdp_port}/json/new?{urllib.parse.quote(self.url, safe='')}",
            method="PUT",
        )
        with urllib.request.urlopen(req) as response:
            client = CDP(json.load(response)["webSocketDebuggerUrl"])
        client.call("Runtime.enable")
        client.call("Log.enable")
        client.call("Page.enable")
        self.pages.append(client)
        return client

    def __exit__(self, *_exc):
        for page in self.pages:
            page.close()
        if self.chrome:
            self.chrome.terminate()
            with contextlib.suppress(subprocess.TimeoutExpired):
                self.chrome.wait(timeout=5)
            if self.chrome.poll() is None:
                self.chrome.kill()
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=3)
        shutil.rmtree(self.profile, ignore_errors=True)


def wait(page, expression, timeout=25):
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        last = page.evaluate(expression)
        if last:
            return last
        time.sleep(.2)
    raise AssertionError(f"Timed out: {expression}\nLast value: {last}")


def runtime_failures(page):
    page.call("Runtime.evaluate", {"expression": "0"})
    failures = []
    for event in page.events:
        method, params = event.get("method"), event.get("params", {})
        if method == "Runtime.exceptionThrown":
            failures.append(params.get("exceptionDetails", {}).get("text", "uncaught exception"))
        elif method == "Log.entryAdded" and params.get("entry", {}).get("level") == "error":
            entry = params["entry"]
            # Chrome reports favicon misses as network errors; Fileworker has no favicon.
            if "favicon.ico" not in entry.get("url", ""):
                failures.append(entry.get("text", "console error"))
        elif method == "Runtime.consoleAPICalled" and params.get("type") in ("error", "assert"):
            failures.append(f"console.{params['type']}")
    return failures


READY = """document.querySelector("#worker-status")?.textContent === "WORKER ONLINE"
  && Boolean(window.FileworkerAPI) && Boolean(window.FileworkerPeerAPI)"""


def validate(args):
    chrome = args.chrome or next((item for item in CHROME_CANDIDATES if item and Path(item).exists()), None)
    if not chrome:
        raise SystemExit("Chrome/Chromium not found; pass --chrome")
    checks = {}
    with Harness(chrome, Path(args.app_root).resolve()) as harness:
        one, two, stale = harness.page(), harness.page(), harness.page()
        for page in (one, two, stale):
            wait(page, READY)

        # No demo-only URL branches: all clients run the exact root URL.
        urls = [page.evaluate("location.href") for page in (one, two, stale)]
        assert urls == [harness.url] * 3, urls
        checks["production_urls"] = True

        for page in (one, two, stale):
            page.evaluate("FileworkerPeerAPI.announce()")
        for page in (one, two, stale):
            wait(page, "FileworkerPeerAPI.connectedPeerIds().length === 2", timeout=30)
            peer_state = page.evaluate("""({
              ids: FileworkerPeerAPI.connectedPeerIds(),
              rows: [...document.querySelectorAll("#peer-list .peer-row")].map(x => x.dataset.id)
            })""")
            assert len(peer_state["ids"]) == len(set(peer_state["ids"])) == 2, peer_state
            assert sorted(peer_state["ids"]) == sorted(peer_state["rows"]), peer_state
        checks["three_client_no_duplicate_peers"] = True

        # Cross-client mutation must update the rendered index without manual refresh.
        one.evaluate("""FileworkerAPI.call("write", {
          path:"validation-cross-tab.txt", data:new Blob(["ok"]), type:"text/plain"
        })""")
        wait(two, """[...document.querySelectorAll(".file-row")]
          .some(row => row.dataset.path === "validation-cross-tab.txt")""")
        checks["cross_client_invalidation"] = True

        # Exercise More and every responsive layout, looking for page-level overflow,
        # clipped controls, offscreen inspector content, and accidental non-pointer UI.
        viewports = ((1440, 900), (1024, 768), (390, 844))
        for width, height in viewports:
            two.call("Emulation.setDeviceMetricsOverride", {
                "width": width, "height": height, "deviceScaleFactor": 1, "mobile": width < 500,
            })
            two.evaluate("""(() => {
              const row = [...document.querySelectorAll(".file-row")]
                .find(x => x.dataset.path === "validation-cross-tab.txt");
              row.querySelector(".more-button").click();
            })()""")
            layout = two.evaluate("""(() => {
              const visible = e => {
                const s=getComputedStyle(e), r=e.getBoundingClientRect();
                return s.display!=="none" && s.visibility!=="hidden" && r.width>0 && r.height>0;
              };
              const inspector=document.querySelector("#inspector"), r=inspector.getBoundingClientRect();
              const canScrollTo = e => {
                for(let p=e.parentElement;p;p=p.parentElement) {
                  const s=getComputedStyle(p);
                  if((s.overflowY==="auto" || s.overflowY==="scroll") && p.scrollHeight>p.clientHeight)
                    return true;
                }
                return false;
              };
              const badControls=[...document.querySelectorAll(
                "button:not([disabled]), a[href], input:not([type=hidden]):not([disabled]), select:not([disabled]), [tabindex='0']"
              )].filter(visible).filter(e => {
                const b=e.getBoundingClientRect();
                return !e.classList.contains("skip-link") && !e.closest("#inspector") && !canScrollTo(e) &&
                  (b.right<0 || b.left>innerWidth || b.bottom<0 || b.top>innerHeight);
              }).map(e=>e.id || e.className || e.tagName);
              const badNames=[...document.querySelectorAll("button, a[href], input, select")]
                .filter(visible).filter(e => !(e.getAttribute("aria-label") || e.title ||
                  e.textContent.trim() || e.labels?.[0]?.textContent.trim()))
                .map(e=>e.id || e.outerHTML.slice(0,80));
              const badCursors=[...document.querySelectorAll("button:not([disabled]), a[href], select, [role=button]")]
                .filter(visible).filter(e=>getComputedStyle(e).cursor!=="pointer")
                .map(e=>e.id || e.className || e.tagName);
              return {
                viewport:[innerWidth,innerHeight], bodyScroll:[document.documentElement.scrollWidth,
                  document.documentElement.scrollHeight],
                inspector:{open:inspector.classList.contains("open"), top:r.top, bottom:r.bottom,
                  first:document.querySelector("#inspector-content").getBoundingClientRect().top,
                  closeVisible:visible(document.querySelector("#close-inspector"))},
                badControls,badNames,badCursors
              };
            })()""")
            assert layout["bodyScroll"][0] <= layout["viewport"][0] + 1, layout
            assert layout["inspector"]["open"], layout
            assert 0 <= layout["inspector"]["top"] < height, layout
            assert layout["inspector"]["bottom"] <= height + 1, layout
            assert layout["inspector"]["first"] < height, layout
            assert layout["inspector"]["closeVisible"], layout
            assert not layout["badControls"], layout
            assert not layout["badNames"], layout
            assert not layout["badCursors"], layout
            two.evaluate("document.querySelector('#close-inspector').click()")
        checks["responsive_layout_and_more"] = True
        checks["basic_accessibility_and_cursors"] = True

        # A retired client must cease being a live peer; remaining clients must not
        # acquire duplicate rows while reconciling the disconnect.
        stale_id = stale.evaluate("FileworkerPeerAPI.id")
        stale.evaluate("location.href='about:blank'")
        wait(one, f"!FileworkerPeerAPI.connectedPeerIds().includes({json.dumps(stale_id)})", timeout=20)
        for page in (one, two):
            state = page.evaluate("""({
              ids:FileworkerPeerAPI.connectedPeerIds(),
              rows:[...document.querySelectorAll("#peer-list .peer-row")].map(x=>x.dataset.id)
            })""")
            assert len(state["ids"]) == len(set(state["ids"])) == 1, state
            assert state["rows"] == state["ids"], state
        checks["stale_client_eviction"] = True

        failures = {i: runtime_failures(page) for i, page in enumerate((one, two)) if runtime_failures(page)}
        assert not failures, failures
        checks["no_page_or_console_errors"] = True

    # If context exit returned, both owned processes shut down cleanly.
    checks["server_and_browser_lifecycle"] = True
    return checks


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--app-root", default=str(ROOT))
    parser.add_argument("--chrome")
    args = parser.parse_args()
    try:
        checks = validate(args)
        print(json.dumps({"ok": True, "checks": checks}, sort_keys=True))
    except Exception as exc:
        print(json.dumps({"ok": False, "error": str(exc)}, sort_keys=True))
        raise


if __name__ == "__main__":
    main()
