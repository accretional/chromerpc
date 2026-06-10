#!/usr/bin/env bash
#
# Deploy the multi-process interactive pool service to Cloud Run: each instance
# pre-warms POOL_SIZE independent Chrome processes and serves up to POOL_SIZE
# concurrent isolated sessions, recycling each Chrome asynchronously on release.
#
# concurrency is set equal to POOL_SIZE so Cloud Run admits exactly as many
# clients per instance as there are Chromes; the in-process admission control
# then makes excess clients pending (during recycle churn) or rejects them.
#
# Prereq: build & push the image first (e.g. `make deploy`).
#
# Usage:  ./scripts/deploy-pool.sh
# Env: PROJECT REGION REPO SERVICE IMAGE_TAG POOL_SIZE CPU MEMORY MAX_INSTANCES INVOKER_AUTH
set -euo pipefail

PROJECT="${PROJECT:-$(gcloud config get-value project 2>/dev/null)}"
REGION="${REGION:-us-central1}"
REPO="${REPO:-chromerpc}"
SERVICE="${SERVICE:-chromerpc-bidi-pool}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
POOL_SIZE="${POOL_SIZE:-4}"
INVOKER_AUTH="${INVOKER_AUTH:-require}"
IMAGE="${REGION}-docker.pkg.dev/${PROJECT}/${REPO}/chromerpc:${IMAGE_TAG}"

# Size resources generously: ~2 vCPU + ~2Gi per Chrome so a (re)launching Chrome
# isn't CPU-starved while a peer is actively rendering/screenshotting — that
# starvation is what blew past the 30s launch timeout under load. Clamp to the
# Cloud Run maxima (8 vCPU / 32Gi).
_cpu=$(( POOL_SIZE * 2 )); [ "$_cpu" -gt 8 ] && _cpu=8
_mem=$(( POOL_SIZE * 2 )); [ "$_mem" -gt 32 ] && _mem=32
CPU="${CPU:-$_cpu}"
MEMORY="${MEMORY:-${_mem}Gi}"

if [ "${INVOKER_AUTH}" = "allow" ]; then AUTH="--allow-unauthenticated"; else AUTH="--no-allow-unauthenticated"; fi

echo ">> Deploying ${SERVICE} from ${IMAGE}"
echo "   pool-size=${POOL_SIZE}, concurrency=${POOL_SIZE}, cpu=${CPU}, memory=${MEMORY}"
gcloud run deploy "${SERVICE}" \
  --image "${IMAGE}" \
  --region "${REGION}" \
  --project "${PROJECT}" \
  --port 8080 \
  --use-http2 \
  --concurrency "${POOL_SIZE}" \
  --min-instances 0 \
  --max-instances "${MAX_INSTANCES:-10}" \
  --cpu "${CPU}" \
  --memory "${MEMORY}" \
  --cpu-boost \
  --timeout 3600 \
  --command=chromerpc \
  --args="^@^--headless@--no-sandbox@--disable-dev-shm-usage@--interactive@--pool-size=${POOL_SIZE}" \
  ${AUTH} \
  --labels "mode=interactive-pool,pool-size=${POOL_SIZE}"

URL="$(gcloud run services describe "${SERVICE}" --region "${REGION}" --project "${PROJECT}" --format='value(status.url)')"
HOST="${URL#https://}"
echo
echo "=================================================================="
echo "Deployed: ${URL}"
echo "Interactive pool endpoint (TLS/HTTP2): ${HOST}:443  (pool-size ${POOL_SIZE})"
echo "=================================================================="
