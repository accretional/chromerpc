# Refactor TODO

Task-level tracking. Checkboxes = concrete work items. Keep the phase headers in
sync with [README.md](README.md) roadmap. Detail/rationale lives in `notes/`.

Legend: `[ ]` todo · `[~]` in progress · `[x]` done · `[!]` blocked (needs input)

---

## Phase 0 — Coordination & tracking ✅

- [x] Create `docs/refactor/` structure (README, TODO, BUGS, notes/).
- [x] Establish note conventions (`notes/README.md`).
- [x] Record kickoff scope, component inventory, roadmap (`notes/0001`).
- [x] Record environment setup + BUG-001 (`notes/0002`).
- [x] Draft testing-strategy design (`notes/0003`).

## Phase 1 — Testing strategy (CURRENT)

Design in [notes/0003-testing-strategy.md](notes/0003-testing-strategy.md).

### 1.0 Decisions — RESOLVED (2026-07-03)
- [x] **DECIDE #1 → Use an EXISTING GCP project.** Install gcloud; user authenticates
  interactively (`gcloud auth login`) and provides the project ID (billing + Cloud
  Run/Build/Artifact Registry). Deploy leg wires to it. _Project ID still needed._
- [x] **DECIDE #2 → Headless `claude` agents.** The harness shells out to `claude -p`
  agents that drive chrome-proxy's HTTP API for dynamic navigation and assert
  outcomes (with cost/time + determinism guards).
- [x] **DECIDE #3 → proto-linker = `accretional/proto-go`** (`src/linker`: `linker.proto`,
  `Link`/`Build`). gluon = astkit codegen substrate (not the linker); go2proto +
  proto-type are substrate; katarche = older ancestor; "protosh" = proto-expr's
  scripting service (unrelated). Identified via org code search — see
  [notes/0004](notes/0004-accretional-repos.md). User to give final nod on entrypoint.

### 1.1 Environment prerequisites — captured in `scripts/refactor-setup.sh`
- [x] Author idempotent setup script (installs grpcurl, gcloud; verifies Go/Chrome/
      claude; guides auth+project). `scripts/refactor-setup.sh`.
- [x] Run setup script — grpcurl 1.9.3 + gcloud 575.0.0 installed & on PATH.
- [!] `gcloud auth login` + set project — **needs user** (interactive; project ID).
- [ ] Confirm Cloud Build path works (avoids needing local docker).
- [ ] (optional) `--with-protoc`/`--with-buf` only if regenerating protos this phase.

### 1.2 Harness — `scripts/refactor-e2e.sh` ✅ authored
- [x] Phase A (local, hard gate): cdpclient unit tests → Chrome-present guard →
      integration suite (all domains + raw-CDP smoke) → local low-level proto
      screenshot against a **local fixture** (`scripts/fixtures/smoke.html`).
- [x] Make integration "skip on no-Chrome" a hard FAIL in harness context (A2 guard).
- [x] Phase B (build+deploy, hard gate): `go build ./...` → `deploy-cloudrun.sh` →
      `deploy-bidi.sh` → `deploy-pool.sh` (respects ordering). _Wired; needs GCP._
- [x] Phase C (collect-all): smoke-test.sh, recipe automation runs, loadtest,
      meta-test(optional), multi-agent nav suite(optional) → results table →
      nonzero exit on any fail. _Wired; C4/C5 scripts pending._
- [x] Flags: `--local-only`, `--skip-deploy`, `--host`, `--keep`.
- [x] Local HTML fixture (`scripts/fixtures/smoke.html`) for hermetic A4.
- [x] **Phase A GREEN end-to-end** (A1 unit → A2 Chrome guard → A3 full 54-domain
      integration suite + raw-CDP smoke → A4 fixture screenshot 86 KB PNG). exit 0.

### 1.3 chrome-proxy meta-test — `scripts/metatest-chrome-proxy.sh` ✅ authored
- [x] Launch local chrome-proxy → remote interactive instance; wait `session ready`.
- [x] Functional asserts: `/health`, `/steps` (navigate+evaluate title), `/shot.png`
      (PNG magic + `X-Css-Width/Height`), `/clickxy`, `/key`, `/capture` DOM ids.
- [x] Visual assert: local headless chromerpc opens `/capture`, screenshots it,
      asserts valid PNG (remote frame rendered in the console).
- [x] Teardown via `/close`.
- [ ] Validate against the live deployed service (once B lands) + tune auth token.

### 1.4 Multi-agent dynamic-navigation suite — `scripts/agentic-suite.sh` ✅ authored
- [x] Headless `claude -p` agents (model sonnet, time-boxed), one chrome-proxy
      session each → concurrent load on the unified pool.
- [x] Genuine dynamic nav task (example.com → read link href → navigate → confirm
      IANA); deterministic, non-bot-gated (BUG-001); machine-parseable verdicts.
- [ ] Validate against live service; add explicit keepalive (>30s idle) + pool-recycle
      regression assertions (lessons from notes/0006).

### 1.5b Service model change (per user 2026-07-03)
- [x] Collapse to **2 deployed services**: `chromerpc` (CDP domains + HeadlessBrowser)
      + `chromerpc-interactive` (unified bidi+pool; `--interactive --pool-size=N`).
      Dropped the redundant concurrency-1 bidi. Harness Phase B updated.
- [x] **Rule enforced:** never `--min-instances > 0` (scale-to-zero); harness B has a
      pre-flight grep guard. Saved to cross-session memory.
- [ ] Longer-term (Phase 4+): converge toward a single CDP server.

### 1.5 Wire-up & docs
- [ ] `make e2e` target (or documented entrypoint) for the harness.
- [ ] README/DEPLOY note pointing at the harness as the pre-refactor gate.
- [x] **Whole harness GREEN once, end to end** (2026-07-03): Phase A local (54 domains
      + fixture) ✅ · Phase B deploy 2 services ✅ · Phase C — C1 smoke ✅, C2 recipes
      2/2 ✅, C3 loadtest 8/8 ✅, C4 meta-test ✅, C5 agentic 2/2 ✅. See notes/0007.
- [ ] Capstone: single `refactor-e2e.sh` orchestrated run (add `--interactive-host`
      done) — optional; each leg validated individually.

---

## Cleanup candidates (junk sweep — see notes/0005) — DONE 2026-07-03
Executed after user greenlight. Lessons preserved in notes/0006 first.
- [x] `clown-nose-test/` + `redsocks-test/` deleted — keepalive + pool-recycle +
      CAPTCHA-auto-resume lessons preserved in [notes/0006](notes/0006-bidi-agent-lessons.md).
- [x] `automations/screenshot_site.textproto` deleted (hardcoded `/Users/fred…`);
      empty `automations/` dir removed.
- [x] `Database` CDP domain removed (dead upstream, BUG-002): `proto/cdp/database`,
      `internal/server/database`, main.go registration (2 imports + register),
      integration_test.go wiring (5 sites), `TestDatabaseEnableDisable`, Makefile
      proto entry. `go build ./...` + `go vet` clean. **54 CDP domains now.**
- [x] Fixed broken refs I caused: `.dockerignore` (`automations/*/output/`),
      `chrome-proxy/README.md` (redsocks-test links → notes/0006).
- [ ] README.md still has a dangling `automations/` link + "the dream" content →
      Phase 2 docs sweep (left intentionally; part of the larger README rework).

## Phase 2 — Docs sweep (DONE)
Executed from a full doc audit:
- [x] Slimmed root README 354→~180 lines: cut the origin story ("dream/Starting
      Out/Milestone1"); fixed 3 broken links (automations/, testing/ paths, demo
      image); 55→54 counts; reconciled the default-auth story; added refactor pointer.
      Preserved the CDP reflection-discovery, Cloud Run calling, and Go-client techniques.
- [x] Moved `DEPLOY.md` → `docs/deploy.md` (best-written doc; kept knobs table,
      env tunables, image-tag scheme) + noted `make deploy` uses `require`.
- [x] `loadtest/README.md`: `chromerpc-bidi-pool` → unified `chromerpc-interactive`.
- [x] `chrome-proxy/README.md`: 55→54, trimmed speculative gateway para.
- [x] Left `recipes/README.md` + `chrome-testing/USAGE_INSTRUCTIONS.md` in place
      (accurate; recipes removal-slated for Phase 4/5).

## Phase 3 — CDP proto sweep + generation (CORE MECHANISMS WORKING)
Full analysis in [notes/0008](notes/0008-phase3-cdp-diff.md). No CDP→proto generator
existed → built net-new here.
- [x] **Diff vs upstream** (agent) + **vs the running Chrome 150** (`cdp-pull.sh`):
      57 domains in 150; 4 missing (all experimental); 0 dead domains; 4 stale calls
      confirmed absent from 150.
- [x] **3a Tracking/pull:** `scripts/cdp-pull.sh` dumps the running Chrome's exact
      `/json/protocol` + master, vendors under `proto/cdp/_upstream/` (chrome-protocol
      pinned/committed; master gitignored), prints domain/command diff. `make cdp-pull`.
- [x] **3b Decisions:** skip the 5/4 experimental domains; fix the stale calls.
      Applied `Media.playersCreated`→`playerCreated`. Deferred 3 dead-method removals
      to the cdpgen migration (need regen, tracked).
- [x] **3c Generator:** `tools/cdpgen` reads CDP JSON → `.proto` per domain. **All 57
      domains generate protoc-valid** (`make cdp-gen DOMAIN=all`). RPC surface matches
      hand-written. Fixed sessionId + `<Domain>Event` collisions.
- [ ] **Next (3→4):** full cross-domain `$ref` imports + byte-fidelity to hand-written
      protos; wire regenerated `.pb.go` into the build; auto-emit the Makefile `proto:`
      list (or switch to buf); then use regen to drop the 3 dead methods cleanly.

## Phase 4 — Single CDP server via proto-linker (not started)
- [ ] Integrate the external proto-linker/build project (repo TBD — DECIDE #3).
- [ ] Combine all CDP gRPC services into one low-level CDP server.
- [ ] Remove `HeadlessBrowserService` (`proto/cdp/headlessbrowser`,
      `internal/server/headlessbrowser`, `cmd/automate`, `cmd/recipe2json`, the
      `main.go` registration, the `recipe` Makefile target, `recipes/`,
      `automations/`).

## Phase 5 — chromescript automation (not started)
- [ ] New automation impl under `proto/chromescript/`; re-home the 22 step-type
      semantics currently in `headlessbrowser.go`.

## Phase 6 — chromerun bidi (not started)
- [ ] Move `InteractiveSessionService` + `cmd/isession` to `proto/chromerun/`;
      break the `hbserver` step-execution coupling in `internal/server/interactive`.

## Phase 7 — Deps + tooling cleanup (not started)
- [ ] Update all deps; simplify/remove unneeded tools.

## Phase 8 — Make it all work (not started)
- [ ] End-to-end green on the harness after refactor.

## Phase 9 — Build/test/docs strategy v2 (not started)
- [ ] Add fuzzing; execute; clean up the whole repo.

## Phase 10 — Agents-in-Chrome (future)
- [ ] Scope the follow-on integration projects.
