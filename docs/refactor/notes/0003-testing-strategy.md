# Testing strategy — single end-to-end harness + chrome-proxy meta-test

- Date: 2026-07-03
- Author: Claude (Opus 4.8) — initial session
- Status: active (design; some decision points pending user input)
- Related: [[0001-kickoff]], [[0002-environment-setup]], BUG-001, ../TODO.md

The Phase-1 deliverable: **one script** that gates every future refactor step. This
note is the design, grounded in exploration of the current repo. Decision points
flagged **[DECIDE]** need user input (see kickoff open questions).

## Requirements (restated)

1. Local CDP-level proto smoke — **hard gate**, fail the whole run if it fails.
2. Full chromium containerized build + deploy to Cloud Run — **hard gate**, fail if
   build/deploy fails. Build must reuse the components' **sub-build scripts**.
3. If gates pass, run the **full agentic + automation suite** and **collect ALL
   failures** (don't stop at first) so we know everything to fix before retrying.
   - Multi-agent workflow dynamic navigation via `chrome-proxy`.
   - Automation runs against the deployed proto service.
4. **Meta-test:** open the `chrome-proxy` UI from a **local client → remote
   instance**; validate its **visual + functional** aspects.

## Existing assets we reuse (don't reinvent)

From exploration (see 0001 inventory). Build/test units that already exist:

- **Local CDP tests:** `internal/cdpclient/cdpclient_test.go` (hermetic, no Chrome —
  9 tests: dial/send, events, errors, sessions, 50-way concurrency, reconnect).
  `internal/integration/*_test.go` (all ~55 domains against real headless Chrome,
  `data:`/`about:blank` URLs → **no network**; auto-`t.Skip` if Chrome absent).
  `internal/integration/smoke_test.go::TestSmokeRawCDP` (raw CDP, no gRPC).
- **Build sub-units:** `make proto` (protoc codegen, hand-listed 55 protos),
  `make build` (`go build ./cmd/chromerpc`), `make docker` / `cloudbuild.yaml`
  (container). Only `cmd/chromerpc` is containerized today.
- **Deploy sub-scripts:** `scripts/deploy-cloudrun.sh` (Cloud Build → version-tag →
  push → deploy main `chromerpc`, concurrency 8). `scripts/deploy-bidi.sh`
  (`chromerpc-bidi-interactive`, concurrency 1, `--interactive`),
  `scripts/deploy-pool.sh` (`chromerpc-bidi-pool`, `--interactive --pool-size=N`).
  **Ordering constraint:** bidi/pool reuse the image pushed by deploy-cloudrun; they
  do NOT build — deploy-cloudrun must run first.
- **Deployed-service tests:** `scripts/smoke-test.sh` (reflection + RunAutomation +
  **per-call isolation** no-leak assert), `scripts/recipe-run.sh` (recipe → grpcurl
  → save screenshot), `loadtest/run.sh` (pool under concurrency; `check-metrics.sh`
  is **advisory-only**, never gates).
- **Automation recipes** (`recipes/`): `screenshot_after_load` (cloudflare,
  networkidle — deterministic ✅), `dismiss_consent_and_screenshot` (cnn consent),
  `scroll_to_load_lazy` (full-page), `search_and_screenshot` (Amazon — **flaky/bot**
  ⚠️ per BUG-001, best-effort only).
- **chrome-proxy:** resident local HTTP tool (`go run ./chrome-proxy`), loopback
  `:8099`, holds one TLS gRPC bidi `Session` to a remote `--interactive` instance.
  UI at `/capture`; endpoints `/health`, `/steps`, `/shot.png`, `/clickxy`, `/key`,
  `/close`. **Not built/deployed** — it's the local client for the meta-test.

## Harness design — `scripts/refactor-e2e.sh` (proposed)

Three phases; A and B are hard gates, C collects all failures. A results table is
printed at the end; exit non-zero if anything failed.

```
PHASE A — LOCAL (hard gate, fail-fast)
  A1  go test ./internal/cdpclient/ -count=1          # hermetic backbone, no Chrome
  A2  guard: Chrome present? (else FAIL, not skip)    # treat skip-as-fail
  A3  go test ./internal/integration/ -count=1        # all ~55 CDP domains + raw-CDP smoke
  A4  local low-level proto smoke: launch ./bin/chromerpc --headless,
      screenshot a LOCAL fixture page via cmd/screenshot, assert PNG          [fixture avoids BUG-001]
  --> any failure: exit 1 immediately

PHASE B — BUILD + DEPLOY (hard gate, fail-fast)
  B1  make proto   (if protoc present; else verify generated code compiles: go build ./...)
  B2  make build   (server binary)
  B3  INVOKER_AUTH=require ./scripts/deploy-cloudrun.sh     # Cloud Build: build→tag→push→deploy main svc
  B4  ./scripts/deploy-bidi.sh    # interactive svc (for meta-test + agentic nav)
  B5  ./scripts/deploy-pool.sh    # pool svc (for loadtest + concurrency)
  --> any failure: exit 1
  [Cloud Build path builds the image in-cloud → no LOCAL docker needed.]

PHASE C — AGENTIC + AUTOMATION SUITE (collect ALL failures, never fail-fast)
  C1  scripts/smoke-test.sh                    # deployed reflection + RunAutomation + isolation
  C2  automation runs: recipe-run.sh over screenshot_after_load, dismiss_consent,
      scroll_to_load_lazy, + a local-fixture recipe; Amazon recipes best-effort
  C3  loadtest/run.sh                          # pool concurrency; check-metrics advisory
  C4  META-TEST: scripts/metatest-chrome-proxy.sh   # local chrome-proxy -> remote (see below)
  C5  MULTI-AGENT dynamic-navigation suite     # agents drive chrome-proxy HTTP API  [DECIDE #2]
  --> record pass/fail per item; print summary; exit nonzero if any failed
```

Flags: `--local-only` (Phase A only, fast iteration), `--skip-deploy` (reuse an
already-deployed HOST), `--keep` (don't tear down deployed services). Default =
full run.

## chrome-proxy meta-test — `scripts/metatest-chrome-proxy.sh` (new)

Local client → remote instance; validates visual + functional. Concrete, checkable
signals (from the chrome-proxy exploration):

**Setup:** `go run ./chrome-proxy -addr $BIDI_HOST:443 -tls -token $(id-token)
-listen 127.0.0.1:8099 &` → wait for stdout `session ready: <id>`.

**Functional asserts** (curl the loopback API):
- `GET /health` → body `ok`.
- `POST /steps {"steps":[{navigate example/fixture},{evaluate_script "document.title"}]}`
  → JSON `results[].success == true`, `script_result` == expected title.
- `GET /shot.png` → HTTP 200, body starts with PNG magic `\x89PNG`, headers
  `X-Css-Width`/`X-Css-Height` are integers > 0.
- `POST /clickxy {"x":..,"y":..}` → `{"ok":true}`.
- `POST /key {"key":"Enter"}` → `{"ok":true}`.
- `GET /capture` → HTML contains `<title>chrome-proxy capture</title>` and ids
  `#shot`, `#refresh`, `#enter`, `#auto`, `#status`.

**Visual assert (self-hosting):** use a LOCAL `chromerpc` to open
`http://127.0.0.1:8099/capture`, screenshot it, and confirm the live remote frame
actually renders inside `#shot` (evaluate `document.querySelector('#shot').naturalWidth
> 0` and/or assert the screenshot region is non-blank). This literally "opens the
chrome-proxy UI from a local client connected to a remote instance" and proves the
end-to-end visual path. **Teardown:** `POST /close`.

## Multi-agent dynamic-navigation suite [DECIDE #2]

"Agentic" dynamic navigation via chrome-proxy — candidate shapes (pick one):
1. **Headless `claude` agents** (the `claude` CLI IS installed here): the harness
   shells out to N `claude -p` agents, each given the chrome-proxy HTTP base URL and
   a goal ("navigate to X, find Y, screenshot"), each asserts its own outcome. Most
   faithful to "multi-agent workflow"; needs cost/time budget + determinism guard.
2. **Claude Code orchestration (Workflow tool):** this session (or a saved workflow)
   fans out agents that drive chrome-proxy; the shell harness owns only the
   deterministic legs and invokes the agentic layer as a step.
3. **Deterministic pseudo-agent (no LLM):** a Go/bash driver does fixed
   dynamic-navigation heuristics against chrome-proxy. Cheap, reproducible, not
   truly agentic.

Regardless of choice, the target is the **deployed bidi/pool** instance, driven
through **chrome-proxy** (per the redsocks/clown-nose precedent), asserting that a
long-lived session survives think-time gaps (keepalive) and that pool recycle always
returns a ready browser (the two defects those narrative logs documented — worth
encoding as explicit assertions).

## Environment prerequisites — CURRENT GAPS on this machine

Probed 2026-07-03. Missing tooling the harness needs:

| Tool | Needed for | Status | Plan |
|------|-----------|--------|------|
| Go 1.26 | everything | ✅ present | — |
| Chrome | local CDP tests | ✅ present (150) | — |
| `claude` CLI | agentic suite (option 1/2) | ✅ present | — |
| **gcloud SDK** | Phase B deploy, Phase C remote tests | ❌ missing | install; auth; set project **[DECIDE #1]** |
| GCP project/billing/auth | Cloud Run deploy | ❌ none configured | **[DECIDE #1]** |
| **grpcurl** | smoke-test.sh, recipe-run.sh | ❌ missing | `brew install grpcurl` (low-stakes, will just do) |
| docker | only deploy-local-image path | ❌ missing | **not needed** if we use Cloud Build path |
| protoc | `make proto` codegen | ❌ missing | optional; generated .pb.go already committed → `go build` suffices unless regenerating |
| buf | future proto tooling (Phase 3/4) | ❌ missing | later |

**Consequence:** Phase A (local) can run today. Phases B/C need gcloud + a GCP
project + grpcurl before the harness is "fully working." gcloud auth is interactive
(`gcloud auth login`) — the user runs it via the `!` prefix.

## Fixture strategy for BUG-001 (decided)

Automation/meta cases prefer **deterministic** targets, not bot-gated ones:
`example.com` and `cloudflare.com`(networkidle) as today, plus a **local HTML
fixture** served like `chrome-testing/snap.sh` does (python http.server) for fully
hermetic automation asserts. Amazon/Google search cases stay **best-effort**
(reported, non-gating) because headless trips CAPTCHA.

## Open decision points

- **[DECIDE #1]** GCP target (no gcloud/project on this machine). See TODO Phase-1.
- **[DECIDE #2]** Shape of the multi-agent dynamic-navigation suite (3 options above).
- Minor (decided by us unless told otherwise): use Cloud Build (skip local docker);
  install grpcurl; add `--local-only`/`--skip-deploy` flags; local fixture for
  hermetic automation.
