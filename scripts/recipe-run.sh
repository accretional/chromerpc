#!/usr/bin/env bash
#
# Run a textproto recipe against the deployed (IAM-gated) chromerpc Cloud Run
# service, then save and open any screenshot bytes from the response.
#
# Usage:
#   ./scripts/recipe-run.sh recipes/screenshot_after_load.textproto
#   HOST=chromerpc-xxxx-uc.a.run.app ./scripts/recipe-run.sh recipes/foo.textproto
set -euo pipefail

RECIPE="${1:?usage: recipe-run.sh <recipe.textproto>}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RECIPE_ABS="$(cd "$(dirname "$RECIPE")" && pwd)/$(basename "$RECIPE")"
REGION="${REGION:-us-central1}"
SERVICE="${SERVICE:-chromerpc}"
PROJECT="${PROJECT:-$(gcloud config get-value project 2>/dev/null)}"
SVC=cdp.headlessbrowser.HeadlessBrowserService

if [ -z "${HOST:-}" ]; then
  URL="$(gcloud run services describe "${SERVICE}" --region "${REGION}" \
          --project "${PROJECT}" --format='value(status.url)')"
  HOST="${URL#https://}"
fi
TOKEN="$(gcloud auth print-identity-token)"

echo ">> recipe: ${RECIPE}  ->  ${HOST}:443"
JSON="$(cd "$ROOT" && go run ./cmd/recipe2json "$RECIPE_ABS")"

resp="$(grpcurl -max-time 180 -H "authorization: Bearer ${TOKEN}" -d "$JSON" \
         "${HOST}:443" "${SVC}/RunAutomation")"

echo "$resp" | jq -r '"success=\(.success)  \(if .error then "error=\(.error)" else "" end)",
  (.stepResults[]? | "  - \(.label): success=\(.success) \(if .error then "ERR=\(.error)" elif .scriptResult then "=> \(.scriptResult)" else "" end)")'

# Save (and, unless NO_OPEN=1, open) any screenshots in the response.
i=0
while IFS= read -r b64; do
  [ -n "$b64" ] || continue
  out="$(mktemp -t "$(basename "${RECIPE%.textproto}").XXXX").png"
  echo "$b64" | base64 -d > "$out"
  echo ">> screenshot -> $out ($(wc -c <"$out" | tr -d ' ') bytes)"
  [ "${NO_OPEN:-0}" = 1 ] || { command -v open >/dev/null && open "$out" || true; }
  i=$((i+1))
done < <(echo "$resp" | jq -r '.stepResults[]? | select(.screenshotData != null) | .screenshotData')
[ "$i" -eq 0 ] && echo ">> (no screenshot bytes in response)"

# Gate: fail if the automation itself did not succeed (so this is usable as a test).
ok="$(echo "$resp" | jq -r '.success')"
if [ "$ok" != "true" ]; then echo ">> recipe FAILED (success=$ok)"; exit 1; fi
exit 0
