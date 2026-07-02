#!/usr/bin/env bash
#
# chrome-proxy/test.sh — end-to-end test of the ChromeMan monitoring dashboard,
# which ALSO (re)generates the showcase screenshots referenced by the README, so
# those images always reflect the real, tested implementation.
#
# Dogfoods chromerpc to test its own UI. It stands up three processes:
#   1. an interactive chromerpc  — the session the proxy steers;
#   2. chrome-proxy              — holds that session, records history, serves the dashboard;
#   3. a plain chromerpc         — used as a headless browser to LOAD the dashboard,
#                                  assert its DOM, and capture the screenshots.
# It drives two navigate+screenshot pairs through the proxy, then loads the
# dashboard and asserts the rendered DOM reflects the recorded history — crucially,
# one inline capture PER screenshot call, i.e. captures are viewable CHRONOLOGICALLY
# rather than only the latest. Finally it writes chrome-proxy/screenshots/{dashboard,
# history,modal}.png.
#
# Requires: go, Google Chrome/Chromium (launched by chromerpc), grpcurl, curl, and
# network access to the two sample sites (override SITE1/SITE2 to run offline).
# Usage:    chrome-proxy/test.sh

set -euo pipefail
cd "$(dirname "$0")/.."   # repo root

ISESS_ADDR=${ISESS_ADDR:-127.0.0.1:50161}    # interactive server (proxy steers this)
PLAIN_ADDR=${PLAIN_ADDR:-127.0.0.1:50162}    # plain server (loads/asserts/screenshots the dashboard)
PROXY_LISTEN=${PROXY_LISTEN:-127.0.0.1:8192} # dashboard HTTP listen
SITE1=${SITE1:-https://example.com}
SITE2=${SITE2:-https://accretional.com}
DASH_URL="http://${PROXY_LISTEN}/"
SHOTS_OUT="$(pwd)/chrome-proxy/screenshots"  # committed, README-referenced (absolute: the plain server writes here)
WORK=$(mktemp -d)
PIDS=()

log()  { printf '\033[36m[test]\033[0m %s\n' "$*"; }
fail() { printf '\033[31m[FAIL]\033[0m %s\n' "$*" >&2; exit 1; }

cleanup() {
  code=$?
  if [ ${#PIDS[@]} -gt 0 ]; then kill "${PIDS[@]}" 2>/dev/null || true; fi
  rm -rf "$WORK"
  if [ $code -eq 0 ]; then printf '\033[32m[PASS]\033[0m dashboard test passed; screenshots -> %s\n' "chrome-proxy/screenshots";
  else printf '\033[31m[test failed]\033[0m\n'; fi
}
trap cleanup EXIT

# Wait until host:port accepts a TCP connection (bash /dev/tcp), up to ~30s.
wait_listen() {
  host=${1%%:*}; port=${1##*:}
  for _ in $(seq 1 60); do
    if (exec 3<>"/dev/tcp/$host/$port") 2>/dev/null; then exec 3>&- 3<&-; return 0; fi
    sleep 0.5
  done
  return 1
}

# Run an AutomationSequence (JSON on stdin) against the plain server; fail on any
# step error. Writes the response to $2.
runauto() { # <req-file> <resp-file>
  grpcurl -plaintext -d @ "$PLAIN_ADDR" cdp.headlessbrowser.HeadlessBrowserService/RunAutomation \
    <"$1" >"$2" 2>&1 || fail "RunAutomation failed: $(cat "$2")"
  if grep -q '"success": false' "$2"; then fail "a RunAutomation step failed: $(cat "$2")"; fi
}

# ---- dependencies ----
for bin in go grpcurl curl; do command -v "$bin" >/dev/null 2>&1 || fail "missing dependency: $bin"; done

# ---- build ----
log "building chromerpc + chrome-proxy..."
go build -o "$WORK/chromerpc" ./cmd/chromerpc
go build -o "$WORK/chrome-proxy" ./chrome-proxy

# ---- start the three processes ----
log "starting interactive chromerpc ($ISESS_ADDR)"
"$WORK/chromerpc" -interactive -headless -addr "$ISESS_ADDR" >"$WORK/isrv.log" 2>&1 & PIDS+=($!)
log "starting plain chromerpc / dashboard viewer ($PLAIN_ADDR)"
"$WORK/chromerpc" -headless -addr "$PLAIN_ADDR" >"$WORK/plain.log" 2>&1 & PIDS+=($!)
wait_listen "$ISESS_ADDR" || fail "interactive server never listened (see $WORK/isrv.log)"
wait_listen "$PLAIN_ADDR" || fail "plain server never listened (see $WORK/plain.log)"

log "starting chrome-proxy ($PROXY_LISTEN -> $ISESS_ADDR)"
"$WORK/chrome-proxy" -addr "$ISESS_ADDR" -tls=false -listen "$PROXY_LISTEN" -shots "$WORK/shots" >"$WORK/proxy.log" 2>&1 & PIDS+=($!)
wait_listen "$PROXY_LISTEN" || fail "proxy never listened (see $WORK/proxy.log)"
curl -fsS "http://$PROXY_LISTEN/health" >/dev/null || fail "proxy /health failed"

# ---- static HTML smoke check ----
log "checking served dashboard HTML..."
HTML=$(curl -fsS "$DASH_URL")
printf '%s' "$HTML" | grep -q 'bidi proxy monitor' || fail "dashboard HTML not served"
printf '%s' "$HTML" | grep -q 'call-shot'          || fail "dashboard HTML missing per-call capture support (.call-shot)"

# ---- drive two navigate+screenshot pairs through the proxy ----
log "driving two navigate+screenshot steps through the proxy ($SITE1, $SITE2)..."
cat >"$WORK/steps.json" <<JSON
{"steps":[
  {"label":"go-one","navigate":{"url":"$SITE1","wait_until":"load"}},
  {"label":"shot-one","screenshot":{"format":"webp"}},
  {"label":"go-two","navigate":{"url":"$SITE2","wait_until":"networkidle"}},
  {"label":"shot-two","screenshot":{"format":"webp"}}
]}
JSON
curl -fsS "http://$PROXY_LISTEN/steps" -d @"$WORK/steps.json" >"$WORK/steps-resp.json" || fail "/steps request failed"
if grep -q '"error"' "$WORK/steps-resp.json"; then fail "a step errored: $(cat "$WORK/steps-resp.json")"; fi

mkdir -p "$SHOTS_OUT"

# ---- (1) overview screenshot + DOM assertions ----
# The evaluate step returns a single "RESULT ..." line we assert on. Expectations:
#   windows=1  (one connection window)   calls=4  (2 navigate + 2 screenshot)
#   shots=2    (one inline capture per screenshot call — chronological)
#   main=1     (window thumbnail set)    state=READY
log "loading dashboard, asserting DOM, writing dashboard.png..."
cat >"$WORK/overview.json" <<JSON
{"name":"overview","steps":[
  {"setViewport":{"width":1200,"height":760,"deviceScaleFactor":1}},
  {"navigate":{"url":"$DASH_URL","waitUntil":"load"}},
  {"wait":{"milliseconds":6000}},
  {"evaluateScript":{"expression":"var w=document.querySelectorAll('.window').length; var c=document.querySelectorAll('ol.calls li').length; var s=document.querySelectorAll('ol.calls img.call-shot').length; var m=(document.querySelector('.shot')&&document.querySelector('.shot').getAttribute('src'))?1:0; var st=(document.querySelector('.state')||{}).textContent||''; 'RESULT windows='+w+' calls='+c+' shots='+s+' main='+m+' state='+st;"}},
  {"wait":{"milliseconds":300}},
  {"screenshot":{"outputPath":"$SHOTS_OUT/dashboard.png","format":"png"}}
]}
JSON
runauto "$WORK/overview.json" "$WORK/overview-resp.json"
RES=$(grep -o 'RESULT[^"]*' "$WORK/overview-resp.json" | head -1 | tr -d '\\')
[ -n "$RES" ] || fail "no RESULT from dashboard eval: $(cat "$WORK/overview-resp.json")"
log "assertion => $RES"
grep -q 'windows=1'    "$WORK/overview-resp.json" || fail "expected 1 connection window ($RES)"
grep -q 'calls=4'      "$WORK/overview-resp.json" || fail "expected 4 recorded calls ($RES)"
grep -q 'shots=2'      "$WORK/overview-resp.json" || fail "expected 2 inline chronological captures ($RES)"
grep -q 'main=1'       "$WORK/overview-resp.json" || fail "expected window thumbnail to be set ($RES)"
grep -q 'state=READY'  "$WORK/overview-resp.json" || fail "expected connection state READY ($RES)"

# ---- (2) history screenshot: expand everything, uncap the calls list ----
log "writing history.png (expanded, chronological captures)..."
cat >"$WORK/history.json" <<JSON
{"name":"history","steps":[
  {"setViewport":{"width":780,"height":1600,"deviceScaleFactor":1}},
  {"navigate":{"url":"$DASH_URL","waitUntil":"load"}},
  {"wait":{"milliseconds":6000}},
  {"evaluateScript":{"expression":"document.querySelectorAll('details').forEach(function(d){d.open=true;}); document.querySelectorAll('ol.calls').forEach(function(o){o.style.maxHeight='none';o.style.overflow='visible';}); 'ok';"}},
  {"wait":{"milliseconds":600}},
  {"screenshot":{"outputPath":"$SHOTS_OUT/history.png","format":"png","fullPage":true}}
]}
JSON
runauto "$WORK/history.json" "$WORK/history-resp.json"

# ---- (3) modal screenshot: :target fullscreen viewer ----
log "writing modal.png (:target fullscreen viewer)..."
cat >"$WORK/modal.json" <<JSON
{"name":"modal","steps":[
  {"setViewport":{"width":1200,"height":820,"deviceScaleFactor":1}},
  {"navigate":{"url":"$DASH_URL","waitUntil":"load"}},
  {"wait":{"milliseconds":6000}},
  {"evaluateScript":{"expression":"location.hash='#conn-1'; 'ok';"}},
  {"wait":{"milliseconds":500}},
  {"screenshot":{"outputPath":"$SHOTS_OUT/modal.png","format":"png"}}
]}
JSON
runauto "$WORK/modal.json" "$WORK/modal-resp.json"

log "dashboard reflected the full history with one inline capture per screenshot call."
ls -1 "$SHOTS_OUT"/*.png | sed 's#.*/#  screenshots/#'
