#!/usr/bin/env bash
#
# refactor-e2e.sh — the single end-to-end gate for the chromerpc refactor.
# Design: docs/refactor/notes/0003-testing-strategy.md
#
#   PHASE A  LOCAL CDP-level proto smoke      (HARD GATE — fail fast)
#   PHASE B  containerized build + Cloud Run  (HARD GATE — fail fast)
#   PHASE C  agentic + automation suite       (COLLECT ALL failures, then report)
#
# A or B failing aborts immediately. If both pass, every Phase-C item runs and the
# script reports the full list of failures at the end (nonzero exit if any failed),
# so we know everything to fix before the next attempt.
#
# Usage:
#   scripts/refactor-e2e.sh [--local-only] [--skip-deploy] [--host HOST] [--keep]
#
#   --local-only   Run PHASE A only (fast inner loop; no GCP needed).
#   --skip-deploy  Skip PHASE B; run A then C against an existing deployment.
#                  Requires --host (or HOST env) pointing at the main service.
#   --host HOST    Deployed main-service host (e.g. chromerpc-xxxx-uc.a.run.app)
#                  for PHASE C when --skip-deploy is used.
#   --keep         Don't tear down deployed services after the run.
#
# Env: PROJECT, REGION (default us-central1) forwarded to the deploy scripts.
#
# Prereqs: run scripts/refactor-setup.sh first (Go, Chrome, grpcurl, gcloud, claude).

set -uo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# ---- args ---------------------------------------------------------------------
LOCAL_ONLY=0; SKIP_DEPLOY=0; KEEP=0; HOST="${HOST:-}"
INTERACTIVE_HOST="${INTERACTIVE_HOST:-}"
REGION="${REGION:-us-central1}"
while [ $# -gt 0 ]; do
  case "$1" in
    --local-only) LOCAL_ONLY=1; shift ;;
    --skip-deploy) SKIP_DEPLOY=1; shift ;;
    --host) HOST="${2:-}"; shift 2 ;;
    --interactive-host) INTERACTIVE_HOST="${2:-}"; shift 2 ;;
    --keep) KEEP=1; shift ;;
    -h|--help) sed -n '2,30p' "$0"; exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

# ---- output helpers -----------------------------------------------------------
GREEN=$'\e[32m'; YELLOW=$'\e[33m'; RED=$'\e[31m'; BOLD=$'\e[1m'; RST=$'\e[0m'
hdr()  { printf '\n%s========== %s ==========%s\n' "$BOLD" "$*" "$RST"; }
ok()   { printf '  %s✔%s %s\n' "$GREEN" "$RST" "$*"; }
warn() { printf '  %s!%s %s\n' "$YELLOW" "$RST" "$*"; }
err()  { printf '  %s✗%s %s\n' "$RED" "$RST" "$*"; }
have() { command -v "$1" >/dev/null 2>&1; }

TMP="$(mktemp -d "${TMPDIR:-/tmp}/chromerpc-e2e.XXXXXX")"
declare -a BG_PIDS=()
declare -a RESULTS=()   # "PASS|NAME" / "FAIL|NAME" / "SKIP|NAME"

cleanup() {
  local pid
  for pid in "${BG_PIDS[@]:-}"; do [ -n "$pid" ] && kill "$pid" 2>/dev/null; done
  rm -rf "$TMP" 2>/dev/null
}
trap cleanup EXIT

fail_hard() { err "$*"; echo; echo "${RED}${BOLD}ABORTED at a hard gate.${RST}"; exit 1; }

# ---- low-level utilities ------------------------------------------------------
find_chrome() {
  local c
  for c in google-chrome google-chrome-stable chromium chromium-browser; do
    command -v "$c" >/dev/null 2>&1 && { command -v "$c"; return 0; }
  done
  for c in "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
           "/Applications/Chromium.app/Contents/MacOS/Chromium"; do
    [ -x "$c" ] && { echo "$c"; return 0; }
  done
  return 1
}
free_port() { python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()'; }
wait_tcp() { # host port [timeout_s]
  local h="$1" p="$2" t="${3:-20}" i
  for ((i=0; i<t*2; i++)); do
    (exec 3<>"/dev/tcp/$h/$p") 2>/dev/null && { exec 3>&- 3<&- 2>/dev/null; return 0; }
    sleep 0.5
  done
  return 1
}
is_png() { [ -f "$1" ] || return 1; [ "$(od -An -tx1 -N4 "$1" 2>/dev/null | tr -d ' \n')" = "89504e47" ]; }

# Run a hard-gate step: on failure, abort the whole run.
gate() { # "name" cmd...
  local name="$1"; shift
  printf '  … %s\n' "$name"
  if "$@" >"$TMP/last.log" 2>&1; then ok "$name"; else
    err "$name — FAILED"; echo "  --- last 30 log lines ---"; tail -30 "$TMP/last.log" | sed 's/^/    /'
    fail_hard "$name"
  fi
}
# Run a Phase-C step: record pass/fail, never abort.
collect() { # "name" cmd...
  local name="$1"; shift
  printf '  … %s\n' "$name"
  if "$@" >"$TMP/c.log" 2>&1; then ok "$name"; RESULTS+=("PASS|$name"); else
    warn "$name — failed (collected)"; tail -8 "$TMP/c.log" | sed 's/^/      /'; RESULTS+=("FAIL|$name")
  fi
}
collect_optional() { # "name" script cmd...  — SKIP (not fail) if the script file is missing
  local name="$1" script="$2"; shift 2
  if [ ! -f "$script" ]; then warn "$name — SKIP (not yet implemented: $script)"; RESULTS+=("SKIP|$name"); return; fi
  collect "$name" "$script" "$@"
}

# ============================== PHASE A ========================================
phase_a() {
  hdr "PHASE A — LOCAL CDP-level proto smoke (hard gate)"

  gate "A1 cdpclient unit tests (hermetic, no Chrome)" \
       go test ./internal/cdpclient/ -count=1

  local chrome
  if chrome="$(find_chrome)"; then ok "A2 Chrome present: $chrome"; else
    fail_hard "A2 Chrome NOT found — Phase A requires a local Chrome/Chromium (run scripts/refactor-setup.sh)"
  fi

  # The integration suite self-SKIPs without Chrome; A2 guarantees Chrome, so a
  # skip here would be anomalous. Treat a non-zero exit as failure.
  gate "A3 integration suite (all CDP domains + raw-CDP smoke)" \
       go test ./internal/integration/ -count=1 -timeout 240s

  a4_local_proto_smoke
}

# A4: prove the low-level Page/Emulation gRPC path end to end against a hermetic
# local fixture (no network, no bot detection — BUG-001).
a4_local_proto_smoke() {
  printf '  … %s\n' "A4 low-level proto screenshot vs local fixture"
  gate "A4a build server binary" make build
  local sport fport shot fixdir="$ROOT/scripts/fixtures"
  sport="$(free_port)"; fport="$(free_port)"
  ( cd "$fixdir" && exec python3 -m http.server "$fport" --bind 127.0.0.1 ) >/dev/null 2>&1 &
  BG_PIDS+=("$!")
  "$ROOT/bin/chromerpc" --headless --addr ":$sport" >"$TMP/a4-server.log" 2>&1 &
  BG_PIDS+=("$!")
  wait_tcp 127.0.0.1 "$fport" 10 || fail_hard "A4 fixture http server never came up on :$fport"
  wait_tcp 127.0.0.1 "$sport" 25 || { tail -20 "$TMP/a4-server.log" | sed 's/^/    /'; fail_hard "A4 chromerpc never listened on :$sport"; }
  shot="$TMP/a4.png"
  if go run ./cmd/screenshot -addr "127.0.0.1:$sport" -url "http://127.0.0.1:$fport/smoke.html" -out "$shot" -wait 2s >"$TMP/a4-shot.log" 2>&1; then :; else
    tail -20 "$TMP/a4-shot.log" | sed 's/^/    /'; fail_hard "A4 screenshot RPC failed"
  fi
  is_png "$shot" || fail_hard "A4 output is not a valid PNG"
  local sz; sz="$(wc -c <"$shot" | tr -d ' ')"
  [ "$sz" -gt 3000 ] || fail_hard "A4 screenshot suspiciously small ($sz bytes) — render may have failed"
  ok "A4 captured fixture screenshot ($sz bytes, valid PNG) via low-level Page+Emulation gRPC"
}

# ============================== PHASE B ========================================
require_gcloud_project() {
  have gcloud || fail_hard "gcloud not installed — run scripts/refactor-setup.sh"
  local acct proj
  acct="$(gcloud auth list --filter=status:ACTIVE --format='value(account)' 2>/dev/null | head -1)"
  [ -n "$acct" ] || fail_hard "no active gcloud account — run: gcloud auth login (in-session: ! gcloud auth login)"
  proj="${PROJECT:-$(gcloud config get-value project 2>/dev/null)}"
  { [ -n "$proj" ] && [ "$proj" != "(unset)" ]; } || fail_hard "no GCP project set — run: gcloud config set project <ID> (needs billing + Run/Build/Artifact Registry)"
  ok "gcloud account: $acct   project: $proj   region: $REGION"
}

# Service model: TWO deployed services (the old 3-way main/bidi/pool split was only
# for dev/test). "bidi" and "pool" are the same binary — --interactive --pool-size=N
# (N=1 single, N>1 pool) — so ONE interactive(pool) service is the unified bidi+pool.
#   - chromerpc              : low-level CDP domains + HeadlessBrowserService (C1/C2)
#   - chromerpc-interactive  : unified InteractiveSessionService pool (C3/C4/C5)
# HARD RULE: never set --min-instances > 0 (scale-to-zero always). Deploy scripts are
# already at 0; do not change them.
INTERACTIVE_SERVICE="${INTERACTIVE_SERVICE:-chromerpc-interactive}"
POOL_SIZE="${POOL_SIZE:-2}"
INTERACTIVE_HOST=""

phase_b() {
  hdr "PHASE B — containerized build + Cloud Run deploy (hard gate)"
  require_gcloud_project
  # Safety net for the min-instances rule: refuse to proceed if any deploy path set it >0.
  if grep -rnE "min-instances[ =]+[1-9]" "$ROOT/scripts" "$ROOT/cloudbuild.yaml" 2>/dev/null; then
    fail_hard "a deploy path sets --min-instances > 0 — forbidden (scale-to-zero only)"
  fi
  gate "B1 go build ./... (compile check)" go build ./...
  gate "B2 deploy main service (Cloud Build → tag → push → deploy)" \
       env INVOKER_AUTH=require REGION="$REGION" "$ROOT/scripts/deploy-cloudrun.sh"
  gate "B3 deploy unified interactive+pool service (bidi+pool in one)" \
       env INVOKER_AUTH=require REGION="$REGION" SERVICE="$INTERACTIVE_SERVICE" POOL_SIZE="$POOL_SIZE" \
       "$ROOT/scripts/deploy-pool.sh"
  # Resolve hosts for Phase C.
  HOST="$(gcloud run services describe chromerpc --region "$REGION" --format='value(status.url)' 2>/dev/null)"; HOST="${HOST#https://}"
  INTERACTIVE_HOST="$(gcloud run services describe "$INTERACTIVE_SERVICE" --region "$REGION" --format='value(status.url)' 2>/dev/null)"; INTERACTIVE_HOST="${INTERACTIVE_HOST#https://}"
  [ -n "$HOST" ] && ok "main service:        $HOST"
  [ -n "$INTERACTIVE_HOST" ] && ok "interactive service: $INTERACTIVE_HOST (pool-size $POOL_SIZE)"
}

# ============================== PHASE C ========================================
phase_c() {
  hdr "PHASE C — agentic + automation suite (collect all failures)"
  if [ -z "$HOST" ]; then warn "no HOST for Phase C — pass --host or run Phase B"; fi

  collect "C1 deployed smoke-test (reflection + RunAutomation + isolation)" \
          env HOST="$HOST" "$ROOT/scripts/smoke-test.sh"

  local r
  for r in screenshot_after_load dismiss_consent_and_screenshot scroll_to_load_lazy; do
    collect "C2 automation recipe: $r" \
            env HOST="$HOST" "$ROOT/scripts/recipe-run.sh" "recipes/$r.textproto"
  done
  # Flaky/bot-gated (BUG-001): report but do not treat as a real gate.
  warn "C2 note: Amazon/Google recipes are best-effort (headless bot walls) — not run by default"

  # Interactive tests target the unified interactive+pool service.
  local ihost="${INTERACTIVE_HOST:-$HOST}"
  collect "C3 loadtest (unified interactive+pool concurrency)" \
          env SERVICE="$INTERACTIVE_SERVICE" "$ROOT/loadtest/run.sh"

  # Built in the next steps of Phase 1 (need a live deployment to develop against).
  collect_optional "C4 chrome-proxy meta-test (local client → remote instance)" \
                   "$ROOT/scripts/metatest-chrome-proxy.sh" --host "$ihost"
  collect_optional "C5 multi-agent dynamic-navigation suite (headless claude agents)" \
                   "$ROOT/scripts/agentic-suite.sh" --host "$ihost"
}

# ============================== MAIN ===========================================
hdr "chromerpc refactor E2E — $(date '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo run)"
echo "  mode: $([ "$LOCAL_ONLY" = 1 ] && echo local-only || echo full)   root: $ROOT"

phase_a

if [ "$LOCAL_ONLY" = 1 ]; then
  hdr "RESULT"; ok "PHASE A passed (local-only mode). Phases B/C skipped."
  exit 0
fi

if [ "$SKIP_DEPLOY" = 1 ]; then
  warn "PHASE B skipped (--skip-deploy). Using HOST=${HOST:-<unset>} for Phase C."
  [ -n "$HOST" ] || fail_hard "--skip-deploy requires --host HOST (or HOST env)"
else
  phase_b
fi

phase_c

# ---- report -------------------------------------------------------------------
hdr "RESULT — Phase C summary"
fails=0; skips=0; passes=0
for row in "${RESULTS[@]:-}"; do
  [ -z "$row" ] && continue
  status="${row%%|*}"; name="${row#*|}"
  case "$status" in
    PASS) ok "$name"; passes=$((passes+1)) ;;
    FAIL) err "$name"; fails=$((fails+1)) ;;
    SKIP) warn "$name (skipped)"; skips=$((skips+1)) ;;
  esac
done
echo
echo "  ${passes} passed, ${fails} failed, ${skips} skipped"
if [ "$fails" -gt 0 ]; then
  echo "${RED}${BOLD}E2E finished with failures — fix the FAILs above, then re-run.${RST}"
  exit 1
fi
echo "${GREEN}${BOLD}E2E green.${RST}"
exit 0
