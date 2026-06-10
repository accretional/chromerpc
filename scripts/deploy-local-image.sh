#!/usr/bin/env bash
#
# Build the chromerpc image LOCALLY, push to Artifact Registry, and deploy to
# Cloud Run by image URI (skips Cloud Build).
#
# IMPORTANT (Apple Silicon / arm64 hosts): Cloud Run needs linux/amd64, so this
# builds with --platform linux/amd64. On an arm64 host that means qemu emulation,
# which makes the Chrome apt-install layer SLOW on the first build (cached after).
# On such hosts, Cloud Build (scripts/deploy-cloudrun.sh) is usually faster
# because it builds natively on amd64. Use this script when you specifically want
# a local build, or when running on an amd64 host.
#
# Usage:
#   ./scripts/deploy-local-image.sh
# Env overrides: PROJECT REGION REPO SERVICE CONCURRENCY CPU MEMORY MAX_INSTANCES INVOKER_AUTH PLATFORM
set -euo pipefail

PROJECT="${PROJECT:-$(gcloud config get-value project 2>/dev/null)}"
REGION="${REGION:-us-central1}"
REPO="${REPO:-chromerpc}"
SERVICE="${SERVICE:-chromerpc}"
CONCURRENCY="${CONCURRENCY:-8}"
CPU="${CPU:-4}"
MEMORY="${MEMORY:-4Gi}"
MAX_INSTANCES="${MAX_INSTANCES:-10}"
INVOKER_AUTH="${INVOKER_AUTH:-require}"
PLATFORM="${PLATFORM:-linux/amd64}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${REGION}-docker.pkg.dev/${PROJECT}/${REPO}/chromerpc"

echo ">> Building ${IMAGE} for ${PLATFORM} (local docker)..."
docker build --platform "${PLATFORM}" -t "${IMAGE}:build" "${ROOT}"

echo ">> Deriving Chrome version from the built image..."
VER="$(docker run --rm --platform "${PLATFORM}" --entrypoint google-chrome-stable "${IMAGE}:build" --version \
        | grep -oE '[0-9]+(\.[0-9]+){2,3}' | head -n1)"
[ -n "${VER}" ] || { echo "could not determine Chrome version" >&2; exit 1; }
echo "   Chrome version: ${VER}  ->  ${IMAGE}:${VER}"
docker tag "${IMAGE}:build" "${IMAGE}:${VER}"
docker tag "${IMAGE}:build" "${IMAGE}:latest"

echo ">> Pushing to Artifact Registry..."
gcloud auth configure-docker "${REGION}-docker.pkg.dev" --quiet >/dev/null 2>&1
docker push "${IMAGE}:${VER}"
docker push "${IMAGE}:latest"

if [ "${INVOKER_AUTH}" = "allow" ]; then AUTH="--allow-unauthenticated"; else AUTH="--no-allow-unauthenticated"; fi
echo ">> Deploying ${IMAGE}:${VER} to Cloud Run..."
gcloud run deploy "${SERVICE}" \
  --image "${IMAGE}:${VER}" \
  --region "${REGION}" \
  --project "${PROJECT}" \
  --port 8080 --use-http2 \
  --concurrency "${CONCURRENCY}" \
  --min-instances 0 --max-instances "${MAX_INSTANCES}" \
  --cpu "${CPU}" --memory "${MEMORY}" --cpu-boost --timeout 300 \
  ${AUTH} \
  --labels "chrome-version=${VER//./-}"

URL="$(gcloud run services describe "${SERVICE}" --region "${REGION}" --project "${PROJECT}" --format='value(status.url)')"
echo ">> Deployed: ${URL}  (gRPC: ${URL#https://}:443)"
