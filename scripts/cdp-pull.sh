#!/usr/bin/env bash
#
# cdp-pull.sh — pull CDP protocol definitions and diff them against what this repo
# implements. This is the Phase-3 tracking / auto-pull mechanism.
#
# Two sources, because they answer different questions:
#   (1) THE RUNNING CHROME's exact protocol, via the DevTools `/json/protocol`
#       endpoint — ground truth for the Chrome build we actually drive. Use this to
#       decide what to add/remove for correctness.
#   (2) upstream `browser_protocol.json` + `js_protocol.json` from
#       ChromeDevTools/devtools-protocol@master — the latest (tip-of-tree) defs,
#       which may be AHEAD of stable Chrome. Use this to anticipate drift.
#
# Both are vendored under proto/cdp/_upstream/ (version-pinned by the Chrome build),
# and the script prints a diff of domains/commands vs our proto/cdp + server impls.
#
# Usage:  scripts/cdp-pull.sh [--chrome /path/to/chrome] [--no-master]
# Output: proto/cdp/_upstream/{chrome-protocol.json,browser_protocol.json,js_protocol.json,VERSION}
#         + a printed diff report.

set -uo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; cd "$ROOT"
OUT="$ROOT/proto/cdp/_upstream"; mkdir -p "$OUT"
CHROME=""; WANT_MASTER=1
while [ $# -gt 0 ]; do
  case "$1" in
    --chrome) CHROME="${2:-}"; shift 2 ;;
    --no-master) WANT_MASTER=0; shift ;;
    -h|--help) sed -n '2,22p' "$0"; exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

find_chrome(){
  local c
  for c in "$CHROME" google-chrome google-chrome-stable chromium chromium-browser \
           "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
           "/Applications/Chromium.app/Contents/MacOS/Chromium"; do
    [ -n "$c" ] && command -v "$c" >/dev/null 2>&1 && { command -v "$c"; return 0; }
    [ -n "$c" ] && [ -x "$c" ] && { echo "$c"; return 0; }
  done
  return 1
}
free_port(){ python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()'; }

CHROME_BIN="$(find_chrome)" || { echo "error: no Chrome found (set --chrome)"; exit 1; }
PORT="$(free_port)"; PROFILE="$(mktemp -d)"
echo ">> launching Chrome for protocol dump: $CHROME_BIN (port $PORT)"
"$CHROME_BIN" --headless=new --remote-debugging-port="$PORT" --no-first-run \
  --no-default-browser-check --user-data-dir="$PROFILE" about:blank >/dev/null 2>&1 &
CPID=$!
cleanup(){ kill "$CPID" 2>/dev/null; rm -rf "$PROFILE" 2>/dev/null; }
trap cleanup EXIT

# wait for the DevTools endpoint
VER=""
for _ in $(seq 1 40); do
  VER="$(curl -s "http://127.0.0.1:$PORT/json/version" 2>/dev/null | python3 -c 'import sys,json;print(json.load(sys.stdin).get("Browser",""))' 2>/dev/null)"
  [ -n "$VER" ] && break; sleep 0.25
done
[ -n "$VER" ] || { echo "error: Chrome DevTools endpoint never came up"; exit 1; }
echo ">> Chrome: $VER"
echo "$VER" > "$OUT/VERSION"

curl -s "http://127.0.0.1:$PORT/json/protocol" -o "$OUT/chrome-protocol.json"
python3 -c "import json,sys; json.load(open('$OUT/chrome-protocol.json'))" || { echo "error: bad /json/protocol payload"; exit 1; }
echo ">> saved $OUT/chrome-protocol.json ($(wc -c <"$OUT/chrome-protocol.json"|tr -d ' ') bytes)"

if [ "$WANT_MASTER" = 1 ]; then
  base="https://raw.githubusercontent.com/ChromeDevTools/devtools-protocol/master/json"
  curl -sL "$base/browser_protocol.json" -o "$OUT/browser_protocol.json" || true
  curl -sL "$base/js_protocol.json" -o "$OUT/js_protocol.json" || true
  echo ">> saved upstream master browser_protocol.json + js_protocol.json"
fi

echo
echo ">> diffing repo vs the running Chrome's protocol ..."
python3 - "$ROOT" "$OUT/chrome-protocol.json" <<'PY'
import json, os, re, sys, subprocess
root, proto_path = sys.argv[1], sys.argv[2]

# --- our domains (proto/cdp/* dirs, excluding non-CDP + _upstream) ---
cdp_dir = os.path.join(root, "proto", "cdp")
skip = {"_upstream", "headlessbrowser"}  # headlessbrowser = custom API, not CDP
ours = {d for d in os.listdir(cdp_dir)
        if os.path.isdir(os.path.join(cdp_dir, d)) and d not in skip}

# --- CDP methods/events we reference in server impls: "Domain.method" string literals ---
impl = set()
try:
    out = subprocess.run(["grep","-rhoE",r'"[A-Z][A-Za-z]+\.[a-zA-Z]+"',
                          os.path.join(root,"internal","server")],
                         capture_output=True, text=True).stdout
    for m in re.findall(r'"([A-Z][A-Za-z]+\.[a-zA-Z]+)"', out):
        impl.add(m)
except Exception:
    pass

# --- the running Chrome's protocol ---
proto = json.load(open(proto_path))
chrome_domains = {}
for d in proto.get("domains", []):
    name = d["domain"]
    cmds = {c["name"] for c in d.get("commands", [])}
    evs  = {e["name"] for e in d.get("events", [])}
    chrome_domains[name] = {"commands": cmds, "events": evs,
                            "experimental": d.get("experimental", False)}

def low(s): return s[:1].lower()+s[1:]
ours_canon = {d.lower(): d for d in ours}

print(f"repo domains: {len(ours)}   chrome domains: {len(chrome_domains)}   "
      f"impl method-strings: {len(impl)}")
print()

# domains in Chrome missing from us
missing_dom = [n for n in chrome_domains if n.lower() not in ours_canon]
print(f"[+] domains in THIS Chrome but not in repo ({len(missing_dom)}):")
for n in sorted(missing_dom):
    exp = " (experimental)" if chrome_domains[n]["experimental"] else ""
    print(f"      {n}{exp}  cmds={len(chrome_domains[n]['commands'])}")

# domains we have that Chrome lacks (dead surface)
chrome_lower = {n.lower() for n in chrome_domains}
dead_dom = [ours_canon[k] for k in ours_canon if k not in chrome_lower]
print(f"\n[-] repo domains NOT in this Chrome (dead surface) ({len(dead_dom)}):")
for n in sorted(dead_dom): print(f"      {n}")

# methods/events we call that this Chrome does NOT have (stale for our runtime)
chrome_all = set()
for n,d in chrome_domains.items():
    for c in d["commands"]: chrome_all.add(f"{n}.{c}")
    for e in d["events"]:   chrome_all.add(f"{n}.{e}")
stale = sorted(m for m in impl if m.split(".")[0] in chrome_domains and m not in chrome_all)
print(f"\n[!] method/event strings we reference that THIS Chrome lacks ({len(stale)}):")
for m in stale: print(f"      {m}")

# coverage summary: for domains we have, how many of Chrome's commands we implement
print("\n[=] command coverage (this Chrome) for domains with gaps:")
rows=[]
for n,d in chrome_domains.items():
    if n.lower() not in ours_canon: continue
    have = {c for c in d["commands"] if f"{n}.{c}" in impl}
    miss = len(d["commands"]) - len(have)
    if miss>0 and d["commands"]:
        rows.append((n, len(d["commands"]), len(have), miss))
for n,up,got,miss in sorted(rows, key=lambda r:-r[3])[:15]:
    print(f"      {n:20s} {got:3d}/{up:<3d}  missing {miss}")
PY
echo
echo ">> vendored under proto/cdp/_upstream/ (commit to pin; re-run to refresh & re-diff)."
