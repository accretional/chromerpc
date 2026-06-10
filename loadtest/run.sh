#!/usr/bin/env bash
#
# Orchestrate the interactive-pool load test: resolve the service, mint a token,
# run the concurrent-pulse workload, then query Cloud Run metrics for health.
#
# Usage: ./loadtest/run.sh
# Env: SERVICE (default chromerpc-bidi-pool), REGION, PROJECT, CONCURRENCY (20),
#      PULSES (3), GAP (30s)
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
SERVICE="${SERVICE:-chromerpc-bidi-pool}"
REGION="${REGION:-us-central1}"
PROJECT="${PROJECT:-$(gcloud config get-value project 2>/dev/null)}"
CONCURRENCY="${CONCURRENCY:-20}"
PULSES="${PULSES:-3}"
GAP="${GAP:-30s}"

URL="$(gcloud run services describe "$SERVICE" --region "$REGION" --project "$PROJECT" --format='value(status.url)')"
HOST="${URL#https://}"
TOKEN="$(gcloud auth print-identity-token)"

echo ">> load test ${SERVICE} @ ${HOST}:443  (${CONCURRENCY} x ${PULSES} pulses, ${GAP} gap)"
set +e
go run ./loadtest -addr "${HOST}:443" -tls -token "$TOKEN" \
  -concurrency "$CONCURRENCY" -pulses "$PULSES" -gap "$GAP"
rc=$?
set -e

echo
echo ">> letting metrics settle (60s) before querying..."
sleep 60
SERVICE="$SERVICE" PROJECT="$PROJECT" ./loadtest/check-metrics.sh "$SERVICE" 30m

exit $rc
