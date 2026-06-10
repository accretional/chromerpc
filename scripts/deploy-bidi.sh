#!/usr/bin/env bash
#
# Deploy the interactive bidi-streaming service (InteractiveSessionService) to a
# dedicated Cloud Run service with concurrency=1, reusing the chromerpc image.
#
# Prereqs: build & push the image first (e.g. `make deploy`, which pushes
# :latest). This script just deploys that image as a second service running with
# the --interactive flag.
#
# Usage:
#   ./scripts/deploy-bidi.sh
# Env overrides: PROJECT REGION REPO SERVICE IMAGE_TAG INVOKER_AUTH CPU MEMORY MAX_INSTANCES
set -euo pipefail

PROJECT="${PROJECT:-$(gcloud config get-value project 2>/dev/null)}"
REGION="${REGION:-us-central1}"
REPO="${REPO:-chromerpc}"
SERVICE="${SERVICE:-chromerpc-bidi-interactive}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
INVOKER_AUTH="${INVOKER_AUTH:-require}"
IMAGE="${REGION}-docker.pkg.dev/${PROJECT}/${REPO}/chromerpc:${IMAGE_TAG}"

if [ "${INVOKER_AUTH}" = "allow" ]; then AUTH="--allow-unauthenticated"; else AUTH="--no-allow-unauthenticated"; fi

echo ">> Deploying ${SERVICE} from ${IMAGE}"
echo "   concurrency=1, min-instances=0, --interactive (recycles Chrome per session)"
gcloud run deploy "${SERVICE}" \
  --image "${IMAGE}" \
  --region "${REGION}" \
  --project "${PROJECT}" \
  --port 8080 \
  --use-http2 \
  --concurrency 1 \
  --min-instances 0 \
  --max-instances "${MAX_INSTANCES:-10}" \
  --cpu "${CPU:-2}" \
  --memory "${MEMORY:-2Gi}" \
  --cpu-boost \
  --timeout 3600 \
  --command=chromerpc \
  --args="^@^--headless@--no-sandbox@--disable-dev-shm-usage@--interactive" \
  ${AUTH} \
  --labels "mode=interactive"

URL="$(gcloud run services describe "${SERVICE}" --region "${REGION}" --project "${PROJECT}" --format='value(status.url)')"
HOST="${URL#https://}"
echo
echo "=================================================================="
echo "Deployed: ${URL}"
echo "Interactive bidi endpoint (TLS/HTTP2): ${HOST}:443"
echo
echo "Drive a session (TLS + identity token):"
echo "  go run ./cmd/isession -addr ${HOST}:443 -tls -audience ${URL} < session.jsonl"
echo "=================================================================="
