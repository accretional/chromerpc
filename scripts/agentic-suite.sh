#!/usr/bin/env bash
#
# agentic-suite.sh — multi-agent dynamic-navigation suite (Phase C5).
#
# Spawns N headless `claude` agents, each holding its OWN chrome-proxy session to
# the remote (Cloud Run) unified interactive+pool service, and each driving genuine
# *dynamic* navigation (read the page → decide the next step) via chrome-proxy's
# HTTP API. This exercises the unified pool under concurrent agentic load and proves
# the agent-in-the-loop path end to end. Design: docs/refactor/notes/0003 + 0006.
#
# Concurrency is bounded by the remote pool size (one session per agent). Each agent
# is time-boxed and told to be efficient (cost/determinism guards). Task targets are
# deterministic, non-bot-gated pages (example.com → IANA) — see BUG-001.
#
# Usage:
#   scripts/agentic-suite.sh --host <interactive-host> [--agents N] [--model sonnet] [--token T]
#
# Env: HOST, TOKEN, AGENTS (default 2), MODEL (default sonnet), AGENT_TIMEOUT (default 240s).
# Prereqs: Go, gcloud auth, and the `claude` CLI (agentic driver).

set -uo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; cd "$ROOT"

HOST="${HOST:-}"; TOKEN="${TOKEN:-}"; AGENTS="${AGENTS:-2}"; MODEL="${MODEL:-sonnet}"
AGENT_TIMEOUT="${AGENT_TIMEOUT:-240}"
while [ $# -gt 0 ]; do
  case "$1" in
    --host) HOST="${2:-}"; shift 2 ;;
    --agents) AGENTS="${2:-}"; shift 2 ;;
    --model) MODEL="${2:-}"; shift 2 ;;
    --token) TOKEN="${2:-}"; shift 2 ;;
    -h|--help) sed -n '2,22p' "$0"; exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done
HOST="${HOST#https://}"; HOST="${HOST%/}"
[ -n "$HOST" ] || { echo "error: --host <interactive-service-host> required" >&2; exit 2; }
command -v claude >/dev/null 2>&1 || { echo "error: claude CLI not found (needed for the agentic suite)" >&2; exit 2; }

GREEN=$'\e[32m'; RED=$'\e[31m'; YEL=$'\e[33m'; RST=$'\e[0m'
ok(){ printf '  %s✔%s %s\n' "$GREEN" "$RST" "$*"; }
bad(){ printf '  %s✗%s %s\n' "$RED" "$RST" "$*"; }
info(){ printf '  · %s\n' "$*"; }

TMP="$(mktemp -d "${TMPDIR:-/tmp}/agentic.XXXXXX")"
declare -a PIDS=() PORTS=()
cleanup(){
  for i in "${!PORTS[@]}"; do curl -s -m 3 -X POST "http://127.0.0.1:${PORTS[$i]}/close" >/dev/null 2>&1 || true; done
  for p in "${PIDS[@]:-}"; do [ -n "$p" ] && kill "$p" 2>/dev/null; done
  rm -rf "$TMP" 2>/dev/null
}
trap cleanup EXIT

free_port(){ python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()'; }
# Portable timeout (macOS has neither `timeout` nor `gtimeout` by default).
run_timeout(){ local secs="$1"; shift
  if command -v timeout >/dev/null 2>&1; then timeout "$secs" "$@"; return $?; fi
  if command -v gtimeout >/dev/null 2>&1; then gtimeout "$secs" "$@"; return $?; fi
  "$@" & local cmdpid=$!
  ( sleep "$secs"; kill -TERM "$cmdpid" 2>/dev/null; sleep 3; kill -KILL "$cmdpid" 2>/dev/null ) & local wpid=$!
  wait "$cmdpid" 2>/dev/null; local rc=$?
  kill "$wpid" 2>/dev/null; wait "$wpid" 2>/dev/null; return $rc
}

# ---- identity token ----------------------------------------------------------
if [ -z "$TOKEN" ]; then
  TOKEN="$(gcloud auth print-identity-token --audiences="https://${HOST}" 2>/dev/null || true)"
  [ -n "$TOKEN" ] || TOKEN="$(gcloud auth print-identity-token 2>/dev/null || true)"
fi
[ -n "$TOKEN" ] || { echo "error: could not mint an identity token (gcloud auth login?)" >&2; exit 2; }

echo "== multi-agent dynamic-navigation suite =="
info "remote interactive host: ${HOST}:443   agents: ${AGENTS}   model: ${MODEL}"

# ---- start one chrome-proxy per agent (own session) --------------------------
for ((i=1;i<=AGENTS;i++)); do
  port="$(free_port)"; PORTS+=("$port")
  go run ./chrome-proxy -addr "${HOST}:443" -tls -token "$TOKEN" \
    -listen "127.0.0.1:${port}" -shots "$TMP/shots-$i" >"$TMP/proxy-$i.log" 2>&1 &
  PIDS+=("$!")
done
# wait (up to ~60s) for each proxy to report a ready session
for ((i=1;i<=AGENTS;i++)); do
  for _ in $(seq 1 60); do
    grep -q "session ready" "$TMP/proxy-$i.log" 2>/dev/null && break
    grep -qiE "rpc error|permission|denied|fatal|Unauthenticated|connection refused" "$TMP/proxy-$i.log" 2>/dev/null && break
    sleep 1
  done
done
UP=0; for ((i=1;i<=AGENTS;i++)); do grep -q "session ready" "$TMP/proxy-$i.log" 2>/dev/null && UP=$((UP+1)); done
[ "$UP" -eq "$AGENTS" ] && ok "$UP/$AGENTS proxy sessions established to the remote pool" \
                        || { bad "only $UP/$AGENTS proxy sessions came up"; for i in $(seq 1 $AGENTS); do tail -3 "$TMP/proxy-$i.log" 2>/dev/null | sed "s/^/    [proxy-$i] /"; done; }

# ---- the agent goal (genuine dynamic navigation) -----------------------------
agent_prompt() { # $1 = base url, $2 = agent id
  cat <<EOF
You are agent $2 driving a headless Chrome browser through an HTTP proxy at $1 .
Use ONLY the shell (curl). The proxy API:
  POST $1/steps  -H 'content-type: application/json'  -d '{"steps":[ <step>, ... ]}'
  where a <step> is one of:
    {"navigate":{"url":"URL","wait_until":"load","timeout_ms":30000}}
    {"evaluate_script":{"expression":"<JS expression returning a string>"}}
  The JSON response has results[].success (bool) and results[].script_result (string).

TASK (dynamic — you must READ the page to decide the next step, not hardcode):
  1) Navigate to https://example.com .
  2) evaluate_script to read the href of the first link on the page
     (expression: document.querySelector('a').href) — do NOT assume the value.
  3) Navigate to whatever URL you found in step 2.
  4) evaluate_script to read document.title of that page.
  5) Decide: does the final page belong to IANA (title or URL contains 'IANA' or
     'iana.org')?

Be efficient — batch steps where sensible, keep total calls small.
Print EXACTLY one final line and nothing after it:
  RESULT: SUCCESS   (if the final page is confirmed IANA)
  RESULT: FAILURE <short reason>   (otherwise)
EOF
}

# ---- launch agents concurrently ----------------------------------------------
declare -a APIDS=()
for ((i=1;i<=AGENTS;i++)); do
  base="http://127.0.0.1:${PORTS[$((i-1))]}"
  info "agent $i → $base"
  ( run_timeout "$AGENT_TIMEOUT" claude -p "$(agent_prompt "$base" "$i")" \
      --model "$MODEL" --dangerously-skip-permissions --allowedTools Bash \
      --output-format text >"$TMP/agent-$i.out" 2>"$TMP/agent-$i.err"; echo $? >"$TMP/agent-$i.rc" ) &
  APIDS+=("$!")
done
for p in "${APIDS[@]}"; do wait "$p" 2>/dev/null; done

# ---- collect verdicts --------------------------------------------------------
echo; echo "== agent results =="
FAILS=0
for ((i=1;i<=AGENTS;i++)); do
  verdict="$(grep -aoE 'RESULT: (SUCCESS|FAILURE.*)' "$TMP/agent-$i.out" 2>/dev/null | tail -1)"
  rc="$(cat "$TMP/agent-$i.rc" 2>/dev/null || echo '?')"
  if echo "$verdict" | grep -q 'RESULT: SUCCESS'; then
    ok "agent $i: SUCCESS"
  else
    FAILS=$((FAILS+1))
    bad "agent $i: ${verdict:-<no verdict>} (claude rc=$rc)"
    tail -6 "$TMP/agent-$i.out" 2>/dev/null | sed 's/^/      /'
  fi
done

echo
if [ "$FAILS" -eq 0 ] && [ "$UP" -eq "$AGENTS" ]; then
  echo "${GREEN}agentic suite PASSED — $AGENTS/$AGENTS agents completed dynamic navigation${RST}"; exit 0
fi
echo "${RED}agentic suite FAILED (${FAILS} agent failures, ${UP}/${AGENTS} sessions up)${RST}"; exit 1
