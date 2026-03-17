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
| `screenshot` | Capture the page as an image | `output_path`, `format` (png/jpeg), `quality`, `full_page` |
| `evaluate_script` | Run JavaScript in the page | `expression` |
| `click` | Click at coordinates or a CSS selector | `x`, `y`, `selector` |
| `type_text` | Type text into a focused element or selector | `text`, `selector` |
| `wait_for_selector` | Wait until a CSS selector appears in the DOM | `selector`, `timeout_ms` |
| `reload` | Reload the current page | `ignore_cache` |
| `scroll_to` | Scroll to coordinates | `x`, `y` |

### Modularity

Automations are plain text proto files (`AutomationSequence` messages). This means you can:

- **Reorder steps** by moving `steps: { ... }` blocks around.
- **Compose sequences** by concatenating multiple `.textproto` files or merging them with tooling.
- **Version control** your automations alongside code — they're human-readable diffs.
- **Extend** with new step types by adding a new action to the `AutomationStep` oneof in `proto/cdp/headlessbrowser/headlessbrowser.proto` and implementing the handler in `internal/server/headlessbrowser/headlessbrowser.go`.

### Proto Definition

The full service definition is at [`proto/cdp/headlessbrowser/headlessbrowser.proto`](proto/cdp/headlessbrowser/headlessbrowser.proto). The service exposes a single RPC:

```protobuf
service HeadlessBrowserService {
  rpc RunAutomation(AutomationSequence) returns (AutomationResult);
}
```

### Example Automations

See the [`automations/`](automations/) directory for ready-to-use text proto files.

## Testing

Let's invest in some utility for testing that will make it easier to validate our implementations in a way that is uniform and agnostic/not tied to the implementations.

## Resources / Notes

Nodejs implementaiton of the chrome remote interface: https://github.com/cyrus-and/chrome-remote-interface

**VERY USEFUL**: entire browser_protocol.json for the chrome remote interface https://github.com/ChromeDevTools/devtools-protocol/blob/master/json/browser_protocol.json

https://buf.build/docs/reference/descriptors/#what-are-descriptors this could be useful for converting individual domains or commands into .protos programmatically via https://github.com/protocolbuffers/protobuf/blob/main/src/google/protobuf/descriptor.proto and https://pkg.go.dev/google.golang.org/protobuf/reflect/protoreflect and https://github.com/jhump/protoreflect/tree/main/protoprint
