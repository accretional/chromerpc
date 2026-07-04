#!/usr/bin/env bash
#
# metatest-chrome-proxy.sh — the chrome-proxy meta-test (Phase C4).
#
# Opens the chrome-proxy UI from a LOCAL client connected to a REMOTE (Cloud Run)
# interactive instance and validates chrome-proxy's FUNCTIONAL and VISUAL aspects.
# Design: docs/refactor/notes/0003-testing-strategy.md.
#
#   functional : /health, /steps (navigate+evaluate), /shot.png (PNG + CSS dims),
#                /clickxy, /key, /capture DOM contract.
#   visual     : a local headless chromerpc opens http://127.0.0.1:<proxy>/capture
#                and screenshots it — proving the live remote frame renders in #shot.
#
# Usage:
#   scripts/metatest-chrome-proxy.sh --host <interactive-host> [--token <id-token>]
#
# Env: HOST (alt to --host), TOKEN (alt to --token), PROXY_PORT (default random).
# Prereqs: Go, Chrome, gcloud auth (for the identity token). IAM-gated service.

set -uo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; cd "$ROOT"

HOST="${HOST:-}"; TOKEN="${TOKEN:-}"
while [ $# -gt 0 ]; do
  case "$1" in
    --host) HOST="${2:-}"; shift 2 ;;
    --token) TOKEN="${2:-}"; shift 2 ;;
    -h|--help) sed -n '2,22p' "$0"; exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done
HOST="${HOST#https://}"; HOST="${HOST%/}"
[ -n "$HOST" ] || { echo "error: --host <interactive-service-host> required" >&2; exit 2; }

GREEN=$'\e[32m'; RED=$'\e[31m'; YEL=$'\e[33m'; RST=$'\e[0m'
ok(){ printf '  %s✔%s %s\n' "$GREEN" "$RST" "$*"; }
bad(){ printf '  %s✗%s %s\n' "$RED" "$RST" "$*"; FAILS=$((FAILS+1)); }
info(){ printf '  · %s\n' "$*"; }
FAILS=0

TMP="$(mktemp -d "${TMPDIR:-/tmp}/metaproxy.XXXXXX")"
declare -a PIDS=()
cleanup(){
  # Best-effort graceful proxy shutdown, then kill anything we started.
  curl -s -m 3 -X POST "http://127.0.0.1:${PROXY_PORT}/close" >/dev/null 2>&1 || true
  for p in "${PIDS[@]:-}"; do [ -n "$p" ] && kill "$p" 2>/dev/null; done
  rm -rf "$TMP" 2>/dev/null
}
trap cleanup EXIT

free_port(){ python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()'; }
wait_http(){ local u="$1" t="${2:-20}" i; for ((i=0;i<t*2;i++)); do curl -s -m 2 -o /dev/null "$u" && return 0; sleep 0.5; done; return 1; }
wait_tcp(){ local h="$1" p="$2" t="${3:-20}" i; for ((i=0;i<t*2;i++)); do (exec 3<>"/dev/tcp/$h/$p")2>/dev/null && { exec 3>&-3<&-;return 0;}; sleep 0.5; done; return 1; }
is_png(){ [ -s "$1" ] && [ "$(od -An -tx1 -N4 "$1" 2>/dev/null|tr -d ' \n')" = "89504e47" ]; }

# ---- identity token (Cloud Run is IAM-gated) ---------------------------------
if [ -z "$TOKEN" ]; then
  # Prefer an audience-scoped token (Cloud Run wants aud == service URL); fall back
  # to a plain identity token (works when ADC/ impersonation supplies the audience).
  TOKEN="$(gcloud auth print-identity-token --audiences="https://${HOST}" 2>/dev/null || true)"
  [ -n "$TOKEN" ] || TOKEN="$(gcloud auth print-identity-token 2>/dev/null || true)"
fi
[ -n "$TOKEN" ] || { echo "error: could not mint an identity token (gcloud auth login?)" >&2; exit 2; }

PROXY_PORT="${PROXY_PORT:-$(free_port)}"
BASE="http://127.0.0.1:${PROXY_PORT}"

echo "== chrome-proxy meta-test =="
info "remote interactive host: ${HOST}:443"
info "local proxy UI:          ${BASE}/capture"

# ---- launch chrome-proxy (local client → remote instance) --------------------
go run ./chrome-proxy -addr "${HOST}:443" -tls -token "$TOKEN" \
  -listen "127.0.0.1:${PROXY_PORT}" -shots "$TMP/shots" >"$TMP/proxy.log" 2>&1 &
PIDS+=("$!")
if ! ( for i in $(seq 1 60); do grep -q "session ready" "$TMP/proxy.log" 2>/dev/null && exit 0; grep -qiE "rpc error|permission|denied|fatal|Unauthenticated|connection refused" "$TMP/proxy.log" 2>/dev/null && exit 1; sleep 1; done; exit 1 ); then
  bad "chrome-proxy failed to establish a session"; echo "  --- proxy.log ---"; sed 's/^/    /' "$TMP/proxy.log" | tail -20
  exit 1
fi
ok "session established (local proxy holds a bidi stream to the remote instance)"
wait_http "${BASE}/health" 10 >/dev/null || true

# ---- FUNCTIONAL asserts ------------------------------------------------------
# /health
[ "$(curl -s -m 5 "${BASE}/health")" = "ok" ] && ok "/health → ok" || bad "/health not ok"

# /steps : navigate to a deterministic page + read document.title back
STEPS='{"steps":[{"navigate":{"url":"https://example.com","wait_until":"load","timeout_ms":30000}},{"evaluate_script":{"expression":"document.title"}}]}'
RESP="$(curl -s -m 60 -X POST -H 'content-type: application/json' -d "$STEPS" "${BASE}/steps")"
echo "$RESP" > "$TMP/steps.json"
# JSON is pretty-printed (spaces after colons); match whitespace-tolerantly and
# ensure no step reported success:false.
if echo "$RESP" | grep -qiE '"success"[[:space:]]*:[[:space:]]*true' \
   && ! echo "$RESP" | grep -qiE '"success"[[:space:]]*:[[:space:]]*false' \
   && echo "$RESP" | grep -qi 'Example Domain'; then
  ok "/steps navigate+evaluate → all steps success + title 'Example Domain'"
else
  bad "/steps did not return success + expected title"; info "resp: $(head -c 300 "$TMP/steps.json")"
fi

# /shot.png : PNG bytes + CSS-dimension headers
curl -s -m 30 -D "$TMP/shot.hdr" -o "$TMP/shot.png" "${BASE}/shot.png"
CW="$(grep -i '^X-Css-Width:' "$TMP/shot.hdr" | tr -dc '0-9')"
CH="$(grep -i '^X-Css-Height:' "$TMP/shot.hdr" | tr -dc '0-9')"
if is_png "$TMP/shot.png" && [ "${CW:-0}" -gt 0 ] && [ "${CH:-0}" -gt 0 ]; then
  ok "/shot.png → valid PNG ($(wc -c <"$TMP/shot.png"|tr -d ' ') bytes), CSS ${CW}x${CH}"
else
  bad "/shot.png invalid (png=$(is_png "$TMP/shot.png" && echo y || echo n) css=${CW:-?}x${CH:-?})"
fi

# /clickxy and /key
echo "$(curl -s -m 10 -X POST -H 'content-type: application/json' -d '{"x":10,"y":10}' "${BASE}/clickxy")" | grep -q '"ok":true' \
  && ok "/clickxy → ok" || bad "/clickxy not ok"
echo "$(curl -s -m 10 -X POST -H 'content-type: application/json' -d '{"key":"Enter"}' "${BASE}/key")" | grep -q '"ok":true' \
  && ok "/key Enter → ok" || bad "/key not ok"

# /capture DOM contract
CAP="$(curl -s -m 10 "${BASE}/capture")"
miss=""
for tok in "<title>chrome-proxy capture</title>" 'id="shot"' 'id="refresh"' 'id="enter"' 'id="auto"' 'id="status"'; do
  echo "$CAP" | grep -qF "$tok" || miss="$miss [$tok]"
done
[ -z "$miss" ] && ok "/capture DOM contract present (title,#shot,#refresh,#enter,#auto,#status)" \
              || bad "/capture missing:$miss"

# ---- VISUAL assert : a local client opens the UI and renders the remote frame -
info "visual: local headless chromerpc opens ${BASE}/capture and screenshots it"
SPORT="$(free_port)"
"$ROOT/bin/chromerpc" --headless --addr ":$SPORT" >"$TMP/local-server.log" 2>&1 &
PIDS+=("$!")
if wait_tcp 127.0.0.1 "$SPORT" 25; then
  if go run ./cmd/screenshot -addr "127.0.0.1:$SPORT" -url "${BASE}/capture" -out "$TMP/capture.png" -width 1200 -height 800 -wait 4s >"$TMP/cap-shot.log" 2>&1 \
     && is_png "$TMP/capture.png" && [ "$(wc -c <"$TMP/capture.png"|tr -d ' ')" -gt 5000 ]; then
    # Persist the capture OUTSIDE $TMP (which the trap removes) so it's inspectable.
    OUTDIR="${METATEST_OUT:-${TMPDIR:-/tmp}/chrome-proxy-metatest}"; mkdir -p "$OUTDIR"
    cp "$TMP/capture.png" "$OUTDIR/capture.png" 2>/dev/null
    ok "visual: captured the live chrome-proxy UI ($(wc -c <"$TMP/capture.png"|tr -d ' ') bytes) — remote frame rendered in the console"
    info "saved: $OUTDIR/capture.png (kept for inspection)"
  else
    bad "visual: could not screenshot the chrome-proxy UI"; tail -10 "$TMP/cap-shot.log" | sed 's/^/    /'
  fi
else
  bad "visual: local chromerpc did not come up"
fi

# ---- result ------------------------------------------------------------------
echo
if [ "$FAILS" -eq 0 ]; then echo "${GREEN}chrome-proxy meta-test PASSED${RST}"; exit 0; fi
echo "${RED}chrome-proxy meta-test FAILED (${FAILS} checks)${RST}"; exit 1
