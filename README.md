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

## Testing

Let's invest in some utility for testing that will make it easier to validate our implementations in a way that is uniform and agnostic/not tied to the implementations.

## Resources / Notes

Nodejs implementaiton of the chrome remote interface: https://github.com/cyrus-and/chrome-remote-interface

**VERY USEFUL**: entire browser_protocol.json for the chrome remote interface https://github.com/ChromeDevTools/devtools-protocol/blob/master/json/browser_protocol.json

https://buf.build/docs/reference/descriptors/#what-are-descriptors this could be useful for converting individual domains or commands into .protos programmatically via https://github.com/protocolbuffers/protobuf/blob/main/src/google/protobuf/descriptor.proto and https://pkg.go.dev/google.golang.org/protobuf/reflect/protoreflect and https://github.com/jhump/protoreflect/tree/main/protoprint
