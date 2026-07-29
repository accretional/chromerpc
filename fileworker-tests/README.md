# Fileworker browser test suite

This directory contains three complementary test layers. The top-level runner
executes them **sequentially** and gives every layer a hard wall-clock deadline.
Each browser-owning suite creates a fresh Chrome profile and local HTTP origin,
then tears them down before the next suite starts. It never uses query
parameters or demo-only application behavior.

```sh
make test-fileworker
```

The default run includes:

1. `fixed/`: deterministic drag/drop, cross-client refresh, More-button, and
   inspector regression tests in a real Chrome;
2. `validation/`: three-client, service-worker, layout, accessibility, console,
   overflow, and stale-peer checks;
3. `dynamic/` unit tests: deterministic validation of the Claude harness itself
   without requiring Claude credentials or a running proxy.

The live Claude-driven navigation is intentionally opt-in because it requires
an already-running chromerpc/chrome-proxy pair and incurs model cost:

```sh
fileworker-tests/run_all.sh --with-claude -- \
  --proxy http://127.0.0.1:8099 \
  --app http://127.0.0.1:8765/
```

See each subdirectory's README for coverage and prerequisites. Suite deadlines
can be adjusted using `FILEWORKER_FIXED_TIMEOUT`,
`FILEWORKER_VALIDATION_TIMEOUT`, `FILEWORKER_DYNAMIC_UNIT_TIMEOUT`, and
`FILEWORKER_DYNAMIC_TIMEOUT`; values are seconds. A timed-out process and its
entire process group are terminated, preventing an orphaned Chrome or agent
from contaminating later runs.

## Why `go test ./... -count=1` takes a long time

`internal/integration` contains roughly 170 serial browser integration tests.
Most call `setupTestEnv`, which starts and stabilizes a brand-new Chrome process
for that individual test. At roughly one to two seconds per launch, the package
normally needs several minutes; this is work in progress rather than a silent
hang. Running `./...` also discovers the fixed Fileworker browser package.

The repository `make test` target sets Go's package timeout explicitly (default
`10m`) so a genuinely stuck package produces goroutine stacks instead of
running under Go's much looser default. Override it when profiling:

```sh
make test GO_TEST_TIMEOUT=15m
```

For the Fileworker release gate, prefer `make test-fileworker`; it is narrowly
scoped, reports each layer separately, and kills complete process trees at its
documented deadlines.
