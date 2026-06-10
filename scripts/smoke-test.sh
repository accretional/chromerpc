#!/usr/bin/env bash
#
# Smoke test for a deployed (IAM-gated) chromerpc Cloud Run service.
#
# Verifies:
#   1. reflection / liveness        (grpcurl list)
#   2. a basic RunAutomation call    (navigate + evaluate)
#   3. per-call isolation            (call A writes cookie+localStorage on
#                                     example.com; call B sees neither)
#
# Usage:
#   ./scripts/smoke-test.sh                # auto-discovers the service URL
#   HOST=chromerpc-xxxx-uc.a.run.app ./scripts/smoke-test.sh
set -euo pipefail

REGION="${REGION:-us-central1}"
SERVICE="${SERVICE:-chromerpc}"
PROJECT="${PROJECT:-$(gcloud config get-value project 2>/dev/null)}"

if [ -z "${HOST:-}" ]; then
  URL="$(gcloud run services describe "${SERVICE}" --region "${REGION}" \
          --project "${PROJECT}" --format='value(status.url)')"
  HOST="${URL#https://}"
fi
TOKEN="$(gcloud auth print-identity-token)"
AUTH=(-H "authorization: Bearer ${TOKEN}")
ADDR="${HOST}:443"
SVC="cdp.headlessbrowser.HeadlessBrowserService"

echo ">> Target: ${ADDR}"

echo ">> [1] reflection / list services"
grpcurl "${AUTH[@]}" "${ADDR}" list >/dev/null && echo "    OK"

echo ">> [2] basic RunAutomation (navigate + evaluate)"
grpcurl "${AUTH[@]}" -d '{
  "name": "basic",
  "steps": [
    { "navigate": { "url": "https://example.com" } },
    { "evaluate_script": { "expression": "document.title" } }
  ]
}' "${ADDR}" "${SVC}/RunAutomation" | grep -q '"success": true' && echo "    OK"

echo ">> [3a] call A: write cookie + localStorage on example.com"
grpcurl "${AUTH[@]}" -d '{
  "name": "writer",
  "steps": [
    { "navigate": { "url": "https://example.com" } },
    { "evaluate_script": { "expression": "document.cookie=\"iso=1\"; localStorage.setItem(\"k\",\"v\"); \"wrote\"" } }
  ]
}' "${ADDR}" "${SVC}/RunAutomation" | grep -q '"success": true' && echo "    OK"

echo ">> [3b] call B: read them back (expect empty cookie + null localStorage)"
OUT="$(grpcurl "${AUTH[@]}" -d '{
  "name": "reader",
  "steps": [
    { "navigate": { "url": "https://example.com" } },
    { "evaluate_script": { "expression": "JSON.stringify({cookie: document.cookie, k: localStorage.getItem(\"k\")})" } }
  ]
}' "${ADDR}" "${SVC}/RunAutomation")"

echo "    reader returned: $(echo "$OUT" | grep -o '"scriptResult": "[^"]*"' | tail -n1)"
# The scriptResult is JSON-escaped inside the response; check it shows no leak.
if echo "$OUT" | grep -q 'iso=1'; then
  echo "    ISOLATION FAILED: cookie leaked across calls" >&2
  exit 1
fi
if echo "$OUT" | grep -q '\\"k\\":\\"v\\"'; then
  echo "    ISOLATION FAILED: localStorage leaked across calls" >&2
  exit 1
fi
echo "    OK — no cookie/localStorage leak between calls"

echo
echo "All smoke tests passed."
