# Environment setup — getting chromerpc running & verified locally

- Date: 2026-07-03
- Author: Claude (Opus 4.8) — initial session
- Status: done
- Related: [[0001-kickoff]], BUG-001

What it took to get a working chromerpc on a clean macOS (Apple Silicon) machine,
and one important finding for automation testing.

## Steps that worked

1. **Repo** already cloned at `/Users/fred2/repos/chromerpc` (git repo, on `main`).
2. **Chrome was missing** — the machine had no Chrome/Chromium anywhere.
   chromerpc's `internal/cdpclient/launcher.go:272 findChrome()` probes
   `google-chrome`/`chromium` on PATH and the standard macOS app paths, then errors
   `chrome/chromium not found; set ChromePath or install Chrome`.
   Fix: `brew install --cask google-chrome` → Google Chrome 150 at
   `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome` (a path
   `findChrome()` checks).
3. **Build:** `make build` → `bin/chromerpc` (~27 MB, arm64). Pulls grpc/protobuf/
   websocket deps via Go modules. Go 1.26.4 present.
4. **Run (headless):** `./bin/chromerpc --headless --addr :50051` → logs
   `Chrome launched, WebSocket: ws://127.0.0.1:<port>/…` then
   `gRPC server listening on :50051`. Harmless noise: "Trying to load the allocator
   multiple times" and a GCM `DEPRECATED_ENDPOINT` registration error.
5. **Verify:** `go run ./cmd/screenshot -url <URL> -out out.png` navigates via the
   `PageService` gRPC (process-wide default session) and writes the PNG. `open out.png`.

## Toolchain state on this machine

- Go 1.26.4 (darwin/arm64) ✅
- Homebrew ✅, `jq` ✅, `python3` ✅
- `grpcurl` **not installed** — README examples using `make recipe`/`grpcurl`
  discovery won't run as-is until `brew install grpcurl`. `cmd/screenshot` and
  `cmd/automate` are pure-Go clients that don't need it.
- Chrome 150 installed this session.

## Finding → BUG-001: headless trips Google bot detection

Screenshotting `https://www.google.com/search?q=css` in **headless** mode returned
Google's reCAPTCHA "unusual traffic" wall (89 KB PNG of the interstitial), not
results. Relaunching a **non-headless** real Chrome (`--remote-debugging-port=9222`)
and connecting chromerpc with `--ws-url` returned the genuine results page (606 KB
PNG: search box, AI Overview, W3Schools/MDN results, CSS knowledge panel).

Implication for Phase 1: automation-suite cases can't rely on bot-gated sites in
headless mode. Prefer a controlled fixture (the repo already serves local HTML in
`chrome-testing/snap.sh`) or the connect-to-real-Chrome path for realism cases.
See BUG-001.

## Still-running processes from this session (clean up as needed)

- Non-headless Chrome with remote debugging on `:9222` (profile under scratchpad).
- A `chromerpc --ws-url … --addr :50051` server (background task).
