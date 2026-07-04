# Refactor kickoff — scope, phases, decisions, open questions

- Date: 2026-07-03
- Author: Claude (Opus 4.8) — initial session
- Status: active
- Related: [[0002-environment-setup]], [[0003-testing-strategy]], ../TODO.md, ../BUGS.md

## Goal (as stated by the user)

Undertake a major refactor of chromerpc. Before touching product code, stand up
coordination/tracking (this `docs/refactor/` tree) and a **testing strategy** that
can prove the whole system works as we iterate. Only once the testing harness is
planned and fully working do we start the actual refactor.

## The testing strategy the user asked for (Phase 1 — build this first)

A **single script** that runs, in order, failing fast at the gates:

1. **Local CDP-level proto smoke test** — exercise the low-level CDP gRPC APIs +
   basic smoke. **If this fails, the whole script fails immediately.**
2. **Full chromium containerized build + deploy to Google Cloud Run.** Build must
   reuse the **sub-build scripts of the various components** (not a single
   monolithic step). **If build or deploy fails, the whole script fails.**
3. If (1) and (2) pass, run the **full agentic + automation suite** and collect
   **all** failures (do NOT stop at first failure here) so we know everything to
   fix before the next attempt:
   - **Multi-agent workflow testing for dynamic navigation** using `chrome-proxy`.
   - **Automation runs** against the deployed proto service.
4. A **meta-test**: open the `chrome-proxy` UI from a **local client connected to a
   remote instance** and validate `chrome-proxy`'s **visual and functional**
   aspects.

## Component inventory (current state, from exploration)

- **Module** `github.com/accretional/chromerpc`, Go 1.25.8. Deps: grpc v1.81.1,
  protobuf v1.36.11, gorilla/websocket v1.5.3, google.golang.org/api (idtoken).
- **proto/** has only `cdp/`. `proto/cdp/` = **55 dirs**: 54 real CDP domains +
  `headlessbrowser` (the custom automation API, NOT a CDP domain). No
  `chromescript/`, no `chromerun/`, no buf, no PDL files, no PDL→proto generator.
- **cmd/** (5): `chromerpc` (the server), `screenshot` (low-level CDP client, the
  model for post-refactor usage), `automate` + `recipe2json` (clients of the
  headlessbrowser API — die with it), `isession` (bidi `InteractiveSessionService`
  client — moves to chromerun).
- **internal/server/** = 56 packages: one per CDP domain + `headlessbrowser` +
  `interactive`. Main server (`runServer`) registers 55 services on one
  `*cdpclient.Client` sharing a **process-wide default session**
  (`internal/cdpsession`). Interactive mode (`runInteractive`) registers only
  `InteractiveSessionService`, backed by a `ChromeManager`/`ChromePool` (dedicated
  Chrome per bidi stream, killed+relaunched on close for tenant isolation).
- **Custom automation API to remove:** `proto/cdp/headlessbrowser/headlessbrowser.proto`
  defines `HeadlessBrowserService` (`RunAutomation`, `ExecuteStep`) + a
  22-step-type `AutomationStep` oneof, implemented in
  `internal/server/headlessbrowser/headlessbrowser.go` (per-call incognito
  `runContext` isolation). The bidi `InteractiveSessionService` is also defined in
  this same proto and is coupled to the step implementations via `ExecuteStepShared`.
- **Codegen** is a single hand-maintained `protoc` invocation in the Makefile that
  lists all 55 `.proto` files explicitly; new domains added by hand. The main
  server hand-wires 55 `Register…ServiceServer` calls.
- **chrome-proxy/** = one `main.go` + README. A resident **local** HTTP tool
  (loopback `:8099`) that holds one authenticated TLS gRPC bidi `Session` stream to
  a **remote** `--interactive` Cloud Run instance, and serves an HTML operator
  console at `/capture` (live still `/shot.png`, clickable image → `/clickxy`,
  `/key`, `/steps`, `/health`, `/close`). **Never built or deployed** — run via
  `go run ./chrome-proxy -addr HOST:443 -tls -token …`. Full detail in the
  testing-strategy note.

## Full roadmap (what's coming — for anticipation/tracking, not now)

See ../README.md for the phase table. In brief, after Phase 1 (testing):
- **Docs sweep** — preserve useful resources/techniques, remove most.
- **CDP proto sweep + generation** — audit domains; set up tracking + auto-pull +
  auto-generation from CDP **PDL** changes (nothing exists today).
- **Single CDP server via proto-linker** — remove `HeadlessBrowserService`; use the
  user's external "proto linker" build project to combine all CDP gRPC services
  into one low-level CDP server (replaces hand-listed protoc + hand-wired registers).
- **chromescript** — new automation impl under `proto/chromescript/` (re-homes the
  automation semantics currently in `headlessbrowser.go`).
- **chromerun** — move bidi navigation (`InteractiveSessionService` + `isession`)
  to `proto/chromerun/`; break the `hbserver` coupling.
- **Deps + tooling cleanup**, then **make it all work**, then **build/test/docs v2
  incl. fuzzing** + whole-repo cleanup.
- **Agents-in-Chrome** — a later set of integration projects.

## Decisions so far

- Testing-first: no product refactoring until the Phase-1 harness is green.
- Coordination lives in `docs/refactor/` (committed in the chromerpc repo); notes
  convention in `notes/README.md`; all agent instances write notes.

## Open questions — RESOLVED

1. **GCP deploy target** → ✅ `gcloud` authed as `fred@accretional.com`, project
   **`scratchpad-427517`** (region unset → scripts default `us-central1`). Deploy
   scripts self-provision APIs + Artifact Registry repo + Cloud Build IAM on first run.
2. **Agentic suite shape** → ✅ headless `claude` agents shelling out from the harness
   (DECIDE #2).
3. **proto-linker identity** → ✅ **`accretional/proto-go`** (`src/linker`); gluon is
   the astkit codegen substrate, not the linker. See [[0004-accretional-repos]].
4. **Bot-gated fixtures (BUG-001)** → ✅ decided: local fixture (`scripts/fixtures/
   smoke.html`) for hermetic cases; deterministic sites (example/cloudflare) for
   remote; Amazon/Google best-effort, non-gating.
5. **Per-run cost/time** → ✅ full build+deploy is the default; `--local-only` and
   `--skip-deploy --host` provide fast iteration paths.

Remaining nod needed: confirm Phase-4 entrypoint is proto-go's `src/linker` directly.
