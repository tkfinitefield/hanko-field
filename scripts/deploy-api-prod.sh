#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_ID="${GCP_PROD_PROJECT:-hanko-field-prod}"
REGION="${GCP_PROD_REGION:-asia-northeast1}"
SERVICE_NAME="${HANKO_API_CLOUD_RUN_SERVICE:-hanko-field-api}"
API_CRATE_DIR="${ROOT_DIR}/api"
FIREBASE_SDK_REPO_DIR="${ROOT_DIR}/../firebase-sdk-rust"

if ! command -v gcloud >/dev/null 2>&1; then
  echo "gcloud is required but was not found in PATH." >&2
  exit 1
fi

if ! command -v rsync >/dev/null 2>&1; then
  echo "rsync is required but was not found in PATH." >&2
  exit 1
fi

if [[ ! -d "${FIREBASE_SDK_REPO_DIR}" ]]; then
  echo "firebase-sdk-rust repository was not found at ${FIREBASE_SDK_REPO_DIR}." >&2
  exit 1
fi

TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "${TMP_ROOT}"' EXIT

BUILD_CONTEXT="${TMP_ROOT}/cloud-build"
mkdir -p "${BUILD_CONTEXT}/hanko-field" "${BUILD_CONTEXT}/firebase-sdk-rust"

rsync -a --delete \
  --exclude '.git' \
  --exclude '.devbox' \
  --exclude '.gcloud' \
  --exclude 'target' \
  --exclude 'build' \
  --exclude '.dart_tool' \
  --exclude '.idea' \
  --exclude '.vscode' \
  --exclude '.DS_Store' \
  --exclude '*.log' \
  --exclude '.env*' \
  "${API_CRATE_DIR}/" \
  "${BUILD_CONTEXT}/hanko-field/"

rsync -a --delete \
  --exclude '.git' \
  --exclude '.devbox' \
  --exclude '.gcloud' \
  --exclude 'target' \
  --exclude 'build' \
  --exclude '.dart_tool' \
  --exclude '.idea' \
  --exclude '.vscode' \
  --exclude '.DS_Store' \
  --exclude '*.log' \
  "${FIREBASE_SDK_REPO_DIR}/" \
  "${BUILD_CONTEXT}/firebase-sdk-rust/"

cp "${ROOT_DIR}/api/Dockerfile.cloudrun" "${BUILD_CONTEXT}/Dockerfile"
cp "${ROOT_DIR}/.dockerignore" "${BUILD_CONTEXT}/.dockerignore"

IMAGE_REPO="${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-source-deploy/${SERVICE_NAME}"
IMAGE_TAG="api-$(date +%Y%m%d%H%M%S)"
IMAGE="${IMAGE_REPO}:${IMAGE_TAG}"

echo "Building ${IMAGE} with project=${PROJECT_ID} region=${REGION}..."
gcloud builds submit "${BUILD_CONTEXT}" \
  --project "${PROJECT_ID}" \
  --region "${REGION}" \
  --tag "${IMAGE}"

echo "Deploying ${SERVICE_NAME} with project=${PROJECT_ID} region=${REGION}..."
gcloud run deploy "${SERVICE_NAME}" \
  --image "${IMAGE}" \
  --project "${PROJECT_ID}" \
  --region "${REGION}" \
  --allow-unauthenticated
