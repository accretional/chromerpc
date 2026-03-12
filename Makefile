.PHONY: build test proto clean docker run

# Build the chromerpc binary
build:
	go build -o bin/chromerpc ./cmd/chromerpc

# Run all tests
test:
	go test ./... -v -count=1

# Regenerate protobuf Go code
proto:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/cdp/target/target.proto \
		proto/cdp/page/page.proto \
		proto/cdp/runtime/runtime.proto \
		proto/cdp/network/network.proto \
		proto/cdp/dom/dom.proto \
		proto/cdp/emulation/emulation.proto \
		proto/cdp/input/input.proto \
		proto/cdp/browser/browser.proto \
		proto/cdp/fetch/fetch.proto \
		proto/cdp/css/css.proto \
		proto/cdp/log/log.proto \
		proto/cdp/performance/performance.proto \
		proto/cdp/accessibility/accessibility.proto \
		proto/cdp/io/io.proto \
		proto/cdp/security/security.proto \
		proto/cdp/storage/storage.proto \
		proto/cdp/overlay/overlay.proto \
		proto/cdp/domstorage/domstorage.proto \
		proto/cdp/debugger/debugger.proto \
		proto/cdp/profiler/profiler.proto \
		proto/cdp/console/console.proto \
		proto/cdp/heapprofiler/heapprofiler.proto \
		proto/cdp/serviceworker/serviceworker.proto \
		proto/cdp/indexeddb/indexeddb.proto \
		proto/cdp/cachestorage/cachestorage.proto \
		proto/cdp/tracing/tracing.proto \
		proto/cdp/systeminfo/systeminfo.proto \
		proto/cdp/audits/audits.proto \
		proto/cdp/layertree/layertree.proto \
		proto/cdp/animation/animation.proto \
		proto/cdp/media/media.proto

# Build Docker image
docker:
	docker build -t chromerpc .

# Run with Docker Compose
docker-run:
	docker compose up --build

# Run locally (requires Chrome installed)
run: build
	./bin/chromerpc --headless --addr :50051

# Run locally connecting to existing Chrome instance
run-connect: build
	@echo "Start Chrome with: google-chrome --remote-debugging-port=9222 --headless=new"
	./bin/chromerpc --ws-url ws://127.0.0.1:9222/json/version --addr :50051

# Clean build artifacts
clean:
	rm -rf bin/
