#!/usr/bin/env bash
set -uo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
REPO=$(cd "$HERE/.." && pwd)

FIXED_TIMEOUT=${FILEWORKER_FIXED_TIMEOUT:-120}
VALIDATION_TIMEOUT=${FILEWORKER_VALIDATION_TIMEOUT:-240}
DYNAMIC_UNIT_TIMEOUT=${FILEWORKER_DYNAMIC_UNIT_TIMEOUT:-60}
DYNAMIC_TIMEOUT=${FILEWORKER_DYNAMIC_TIMEOUT:-540}
WITH_CLAUDE=0
DYNAMIC_ARGS=()

usage() {
  cat <<'EOF'
Usage: fileworker-tests/run_all.sh [--with-claude -- DYNAMIC_ARGS...]

Runs the fixed UI, cross-cutting validation, and dynamic-harness unit suites
sequentially. --with-claude additionally runs the live interactive test; pass
its required --proxy and --app arguments after --.

Timeouts (seconds) may be overridden with:
  FILEWORKER_FIXED_TIMEOUT       default 120
  FILEWORKER_VALIDATION_TIMEOUT  default 240
  FILEWORKER_DYNAMIC_UNIT_TIMEOUT default 60
  FILEWORKER_DYNAMIC_TIMEOUT     default 540
EOF
}

while (($#)); do
  case "$1" in
    --with-claude) WITH_CLAUDE=1; shift ;;
    --help|-h) usage; exit 0 ;;
    --) shift; DYNAMIC_ARGS=("$@"); break ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if ((WITH_CLAUDE)) && ((${#DYNAMIC_ARGS[@]} == 0)); then
  echo "--with-claude requires dynamic runner arguments after -- (at least --proxy and --app)" >&2
  exit 2
fi

# Execute a command in its own process group. On timeout, terminate the complete
# group so Chrome, HTTP servers, and Claude cannot leak into a later suite.
bounded() {
  local seconds=$1
  shift
  python3 - "$seconds" "$@" <<'PY'
import os
import signal
import subprocess
import sys

seconds = float(sys.argv[1])
command = sys.argv[2:]
child = subprocess.Popen(command, start_new_session=True)
try:
    raise SystemExit(child.wait(timeout=seconds))
except subprocess.TimeoutExpired:
    print(f"TIMEOUT after {seconds:g}s: {' '.join(command)}", file=sys.stderr)
    os.killpg(child.pid, signal.SIGTERM)
    try:
        child.wait(timeout=5)
    except subprocess.TimeoutExpired:
        os.killpg(child.pid, signal.SIGKILL)
        child.wait()
    raise SystemExit(124)
except KeyboardInterrupt:
    os.killpg(child.pid, signal.SIGTERM)
    child.wait()
    raise SystemExit(130)
PY
}

failures=()
run_stage() {
  local name=$1
  local seconds=$2
  shift 2
  printf '\n==> %s (deadline: %ss)\n' "$name" "$seconds"
  if bounded "$seconds" "$@"; then
    printf '<== PASS: %s\n' "$name"
  else
    local status=$?
    printf '<== FAIL: %s (exit %s)\n' "$name" "$status" >&2
    failures+=("$name:$status")
  fi
}

cd "$REPO"
run_stage "fixed UI" "$FIXED_TIMEOUT" \
  go test ./fileworker-tests/fixed -v -count=1 -timeout="${FIXED_TIMEOUT}s"
run_stage "cross-cutting validation" "$VALIDATION_TIMEOUT" \
  python3 fileworker-tests/validation/run_validation.py
run_stage "dynamic harness unit tests" "$DYNAMIC_UNIT_TIMEOUT" \
  python3 -m unittest fileworker-tests/dynamic/test_runner.py

if ((WITH_CLAUDE)); then
  run_stage "live Claude navigation" "$DYNAMIC_TIMEOUT" \
    "$HERE/dynamic/run.sh" "${DYNAMIC_ARGS[@]}"
fi

if ((${#failures[@]})); then
  printf '\nFileworker test failures: %s\n' "${failures[*]}" >&2
  exit 1
fi

printf '\nAll requested Fileworker suites passed.\n'
