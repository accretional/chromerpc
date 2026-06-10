#!/usr/bin/env bash
#
# One-shot deploy of chromerpc to Google Cloud Run.
#
# Builds the image with Cloud Build (no local Docker needed), tags it with the
# bundled Chrome version, pushes to Artifact Registry, and deploys to Cloud Run
# with HTTP/2 (gRPC), scale-to-zero, and concurrency 1.
#
# Usage:
#   gcloud auth login                 # sign in
#   gcloud config set project <PROJECT>
#   ./scripts/deploy-cloudrun.sh
#
# Overridable via env:
#   PROJECT, REGION, REPO, SERVICE, MAX_INSTANCES, INVOKER_AUTH (allow|require)
set -euo pipefail

PROJECT="${PROJECT:-$(gcloud config get-value project 2>/dev/null)}"
REGION="${REGION:-us-central1}"
REPO="${REPO:-chromerpc}"
SERVICE="${SERVICE:-chromerpc}"
MAX_INSTANCES="${MAX_INSTANCES:-10}"
INVOKER_AUTH="${INVOKER_AUTH:-allow}"   # "allow" => public, else IAM-gated

if [ -z "${PROJECT}" ] || [ "${PROJECT}" = "(unset)" ]; then
  echo "ERROR: no project set. Run: gcloud config set project <PROJECT>" >&2
  exit 1
fi

echo ">> Project: ${PROJECT}   Region: ${REGION}   Service: ${SERVICE}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo ">> Enabling required APIs..."
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  --project "${PROJECT}"

echo ">> Ensuring Artifact Registry repo '${REPO}' exists in ${REGION}..."
if ! gcloud artifacts repositories describe "${REPO}" \
      --location "${REGION}" --project "${PROJECT}" >/dev/null 2>&1; then
  gcloud artifacts repositories create "${REPO}" \
    --repository-format=docker \
    --location "${REGION}" \
    --description "chromerpc images" \
    --project "${PROJECT}"
fi

echo ">> Granting the Cloud Build service account(s) permission to deploy to Cloud Run..."
PROJECT_NUMBER="$(gcloud projects describe "${PROJECT}" --format='value(projectNumber)')"
# Depending on project age, Cloud Build runs as either the legacy Cloud Build SA
# or the Compute Engine default SA. Grant both so the deploy step works either way.
# run.admin: deploy services. iam.serviceAccountUser: act as the runtime SA.
for CB_SA in \
  "${PROJECT_NUMBER}@cloudbuild.gserviceaccount.com" \
  "${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"; do
  for ROLE in roles/run.admin roles/iam.serviceAccountUser roles/artifactregistry.writer; do
    gcloud projects add-iam-policy-binding "${PROJECT}" \
      --member "serviceAccount:${CB_SA}" \
      --role "${ROLE}" \
      --condition=None --quiet >/dev/null 2>&1 || true
  done
done

echo ">> Submitting Cloud Build (build -> version-tag -> push -> deploy)..."
gcloud builds submit "${ROOT}" \
  --config "${ROOT}/cloudbuild.yaml" \
  --project "${PROJECT}" \
  --substitutions "_REGION=${REGION},_REPO=${REPO},_SERVICE=${SERVICE},_MAX_INSTANCES=${MAX_INSTANCES},_INVOKER_AUTH=${INVOKER_AUTH}"

URL="$(gcloud run services describe "${SERVICE}" --region "${REGION}" \
        --project "${PROJECT}" --format='value(status.url)')"
HOST="${URL#https://}"

echo
echo "=================================================================="
echo "Deployed: ${URL}"
echo "gRPC endpoint (TLS, HTTP/2): ${HOST}:443"
echo
echo "Smoke test with grpcurl (reflection is enabled):"
echo "  grpcurl ${HOST}:443 list"
echo "  grpcurl ${HOST}:443 describe cdp.headlessbrowser.HeadlessBrowserService"
echo "=================================================================="
