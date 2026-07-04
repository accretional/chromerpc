# First Cloud Run deployment + live Phase-C validation

- Date: 2026-07-03
- Author: Claude (Opus 4.8) — initial session
- Status: active (results landing as tests run)
- Related: [[0003-testing-strategy]], [[0006-bidi-agent-lessons]], ../TODO.md §1.5b

First real deploy of the refactor harness's Phase B, to the **2-service model** the
user chose (dropped the redundant concurrency-1 bidi).

## GCP target
- Account `fred@accretional.com`, project **`scratchpad-427517`**, region `us-central1`.
- First-run auto-provisioned: enabled run/cloudbuild/artifactregistry APIs, created AR
  repo `chromerpc`, granted Cloud Build SAs run.admin/iam.serviceAccountUser/artifactregistry.writer.
- **min-instances 0 everywhere** (verified by harness pre-flight grep). Never >0.

## Services deployed (both IAM-gated, `--no-allow-unauthenticated`)
| Service | Host | Config |
|---------|------|--------|
| `chromerpc` (main: CDP domains + HeadlessBrowser) | `chromerpc-dru5ubu46q-uc.a.run.app` | concurrency 8, cpu4/4Gi, min0 |
| `chromerpc-interactive` (unified bidi+pool) | `chromerpc-interactive-dru5ubu46q-uc.a.run.app` | `--interactive --pool-size=2`, concurrency 2, cpu4/4Gi, min0 |

- Cloud Build: SUCCESS in ~3m24s; image tagged by Chrome version
  `…/chromerpc/chromerpc:150.0.7871.46` (+ `latest`). Container installs
  google-chrome-stable at build (floats with stable).
- "bidi" and "pool" are the **same binary** (`--pool-size` 1 vs N) → one interactive
  service is the unified target. deploy-bidi.sh is now redundant for our purposes.

## Auth (important — resolves the token question)
- Plain `gcloud auth print-identity-token` (user creds, no `--audiences`) **works**
  against the IAM-gated services — reflection `list` returned all CDP services, exit 0.
  (User creds can't set `--audiences`; Cloud Run accepts the user identity token for a
  principal with run.invoker. Owner has it.) So the meta-test/agentic token fallback
  (try `--audiences` → fall back to plain) lands on the plain token.

## Live Phase-C results
- **C1 smoke-test.sh** (main) — ✅ PASS: reflection + RunAutomation(navigate+evaluate)
  + per-call isolation (no cookie/localStorage leak between calls).
- **C4 chrome-proxy meta-test** (interactive) — ✅ PASS (all 8 checks): local proxy →
  remote bidi session; `/health`, `/steps` (navigate+evaluate → "Example Domain"),
  `/shot.png` (valid PNG, CSS 780x437), `/clickxy`, `/key`, `/capture` DOM contract,
  and the **visual capture of the live UI (82 KB PNG)**. Proves the unified
  interactive+pool service + chrome-proxy end to end. (One iteration: a whitespace-
  sensitive grep on the pretty-printed `"success": true` gave a false negative; fixed.)
- **C5 agentic suite** (interactive) — ✅ PASS: 2/2 headless `claude` (sonnet) agents,
  each holding its own chrome-proxy session on the pool, completed genuine dynamic
  navigation (example.com → read link href → navigate → confirm IANA).
- **C2 automation recipes** (main) — ✅ PASS: `screenshot_after_load` (cloudflare
  networkidle) success=true; `dismiss_consent_and_screenshot` (cnn) success=true
  ("no-consent-button" → screenshot). (Made `recipe-run.sh` CI-friendly: `NO_OPEN=1`
  to skip `open`, and exit nonzero on `success!=true`.)
- **C3 loadtest** (unified pool, 4×2) — ✅ PASS: pulse1 4/4 ok p95 6.2s, pulse2 4/4 ok
  p95 5.9s, TOTAL 8/8, 0 failures. Validates pool acquire/recycle under concurrency on
  the unified service (the notes/0006 pool-recycle behavior). `check-metrics` is
  advisory-only.

## Milestone: entire harness green end-to-end (2026-07-03)
Phase A (local, all 54 CDP domains + fixture) + Phase B (build+deploy 2 services) +
Phase C (C1 smoke, C2 recipes, C3 loadtest, C4 meta-test, C5 agentic) ALL pass.
Phase 1 (testing strategy) objective met. The harness is ready to gate the refactor.

## Portability gotcha (fixed)
- macOS has **no `timeout`/`gtimeout`** by default. `agentic-suite.sh` now uses a
  portable `run_timeout` (timeout→gtimeout→bash fallback). `metatest`/`smoke-test`
  rely on curl/grpcurl `-max-time` instead. (Do NOT wrap deployed-test scripts in
  `timeout` on macOS.)

## Teardown note
- Services scale to zero (min0) so idle cost ≈ 0. To fully remove:
  `gcloud run services delete chromerpc chromerpc-interactive --region us-central1`.
