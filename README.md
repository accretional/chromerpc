# ChromeRPC

Writing gRPC adapters for https://chromedevtools.github.io/devtools-protocol/ (definition at eg https://source.chromium.org/chromium/chromium/src/+/main:third_party/blink/public/devtools_protocol/domains/Page.pdl) in a way that's compatible with the rest of our rpc tooling.

The dream:

```bash
./runrpc Stream.captureScreenshotRequest pages.binarypb | ./runrpc Page.captureScreenshot > screenshots.binarypb
```

## Starting Out

Milestone1: able to send a grpc to headless multiclient (https://developer.chrome.com/blog/new-in-devtools-63/#multi-client) chrome to:

* captureSnapshot (https://source.chromium.org/chromium/chromium/src/+/main:third_party/blink/public/devtools_protocol/domains/Page.pdl;l=632-642)

* captureScreenshot (https://source.chromium.org/chromium/chromium/src/+/main:third_party/blink/public/devtools_protocol/domains/Page.pdl;l=611-630)

* maybe printToPdf (https://source.chromium.org/chromium/chromium/src/+/main:third_party/blink/public/devtools_protocol/domains/Page.pdl;l=940-998)

* any other commands/infrastructure to get these working

We're writing our binaries in Go, and setting up a common linker in https://github.com/accretional/rpcfun - invest as little as possible in main.go, we want to basically just define and implement grpc services with one service per "domain" per directory, one .go implementation of that service per directory.

Might be worth using https://github.com/bitfield/script to chain commands/convert to http calls.

## HeadlessBrowser Automation

The `HeadlessBrowserService` is a high-level automation layer built on top of the CDP domain services. Instead of wiring together individual gRPC calls, you define automation as a **sequence of steps in a text proto file**, then execute the whole sequence with a single RPC.

### Quick Start

1. Start the server:

```bash
make run   # launches headless Chrome + gRPC on :50051
```

Or connect to an existing Chrome instance with remote debugging enabled:

```bash
# Get the WebSocket URL from a running Chrome
WS_URL=$(curl -s http://127.0.0.1:9222/json/version | python3 -c \
  "import sys,json; print(json.load(sys.stdin)['webSocketDebuggerUrl'])")

./bin/chromerpc --ws-url "$WS_URL" --port 50051
```

2. Write an automation file (`my_automation.textproto`):

```textproto
name: "screenshot_example"

steps: {
  label: "set_viewport"
  set_viewport: {
    width: 1280
    height: 800
    device_scale_factor: 2
  }
}

steps: {
  label: "navigate"
  navigate: {
    url: "https://example.com"
  }
}

steps: {
  label: "wait_for_render"
  wait: {
    milliseconds: 500
  }
}

steps: {
  label: "capture"
  screenshot: {
    output_path: "screenshot.png"
    format: "png"
  }
}
```

3. Run it:

```bash
go run ./cmd/automate -input my_automation.textproto
```

### Available Step Types

| Step | Description | Key Fields |
|------|-------------|------------|
| `set_viewport` | Set browser viewport size | `width`, `height`, `device_scale_factor`, `mobile` |
| `navigate` | Navigate to a URL | `url` |
| `wait` | Pause for a fixed duration | `milliseconds` |
| `screenshot` | Capture the visible page as an image | `output_path`, `format` (png/jpeg), `quality`, `full_page` |
| `full_page_screenshot` | Capture the entire scrollable page | `output_path`, `format`, `quality` |
| `evaluate_script` | Run JavaScript in the page | `expression` |
| `click` | Click at coordinates or a CSS selector | `x`, `y`, `selector` |
| `type_text` | Insert text into a focused element or selector | `text`, `selector` |
| `type_key_by_key` | Type text character-by-character with realistic delays | `text`, `delay_ms`, `selector` |
| `press_key` | Press a special key (Enter, Tab, Escape, arrows, etc.) | `key` |
| `wait_for_selector` | Wait until a CSS selector appears in the DOM | `selector`, `timeout_ms` |
| `reload` | Reload the current page | `ignore_cache` |
| `scroll_to` | Scroll to coordinates | `x`, `y` |
| `open_tab` | Open a URL in a new browser tab | `url` |
| `switch_tab` | Switch CDP session to a different tab | `target_id` |
| `close_tab` | Close a browser tab | `target_id` |
| `download_file` | Download a file via browser-native download | `url`, `output_path` |

### RPCs

The service exposes two RPCs:

```protobuf
service HeadlessBrowserService {
  // Run a full sequence of steps.
  rpc RunAutomation(AutomationSequence) returns (AutomationResult);
  // Run a single step (for orchestrators that need to branch on results).
  rpc ExecuteStep(AutomationStep) returns (StepResult);
}
```

`RunAutomation` executes a linear sequence and stops on first failure. `ExecuteStep` runs one step at a time, returning the result so the caller can make decisions (e.g., extract links from a page, then open each in a loop). This makes it possible to build complex orchestrators as standalone Go programs that call `ExecuteStep` in a loop.

### Multi-Tab Support

The `open_tab`, `switch_tab`, and `close_tab` steps enable multi-tab workflows. When you open a new tab, the returned `StepResult.script_result` contains the target ID. Pass this to `switch_tab` to route subsequent commands to that tab, and `close_tab` to clean up.

```
open_tab(url) → target_id
switch_tab(target_id) → session_id (commands now go to this tab)
... do work in the tab ...
close_tab(target_id) → tab destroyed
```

The server manages CDP sessions internally via `Target.attachToTarget` with `flatten=true`.

### File Downloads

The `download_file` step handles browser-native downloads. It opens the URL in a new tab, sets `Browser.setDownloadBehavior` to auto-save to the output directory, finds and clicks the download button (supporting pdf.js viewer's `#download` button, generic download buttons, and `<a download>` links), then waits for the file to appear on disk. This preserves the browser's cookies and session, avoiding issues with authenticated or CDN-protected resources.

### Modularity

Automations are plain text proto files (`AutomationSequence` messages). This means you can:

- **Reorder steps** by moving `steps: { ... }` blocks around.
- **Compose sequences** by concatenating multiple `.textproto` files or merging them with tooling.
- **Version control** your automations alongside code — they're human-readable diffs.
- **Extend** with new step types by adding a new action to the `AutomationStep` oneof in `proto/cdp/headlessbrowser/headlessbrowser.proto` and implementing the handler in `internal/server/headlessbrowser/headlessbrowser.go`.

### Example Automations

See the [`automations/`](automations/) directory for ready-to-use text proto files.

### Connecting to an Existing Chrome

For sites with bot detection, you can connect to a real (non-headless) Chrome instance:

```bash
# Launch Chrome with remote debugging
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --remote-debugging-port=9222 &

# Connect chromerpc to it
WS_URL=$(curl -s http://127.0.0.1:9222/json/version | python3 -c \
  "import sys,json; print(json.load(sys.stdin)['webSocketDebuggerUrl'])")
./bin/chromerpc --ws-url "$WS_URL"
```

The server includes `--disable-blink-features=AutomationControlled` by default and supports `--user-agent` overrides.

## Testing

Let's invest in some utility for testing that will make it easier to validate our implementations in a way that is uniform and agnostic/not tied to the implementations.

## Resources / Notes

Nodejs implementaiton of the chrome remote interface: https://github.com/cyrus-and/chrome-remote-interface

**VERY USEFUL**: entire browser_protocol.json for the chrome remote interface https://github.com/ChromeDevTools/devtools-protocol/blob/master/json/browser_protocol.json

https://buf.build/docs/reference/descriptors/#what-are-descriptors this could be useful for converting individual domains or commands into .protos programmatically via https://github.com/protocolbuffers/protobuf/blob/main/src/google/protobuf/descriptor.proto and https://pkg.go.dev/google.golang.org/protobuf/reflect/protoreflect and https://github.com/jhump/protoreflect/tree/main/protoprint
