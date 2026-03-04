#!/usr/bin/env bash
set -e
set -o pipefail

IMAGE="${IMAGE_TAG}"
PLATFORM="${PLATFORM:-linux/amd64}"
CONTAINER_NAME="git-smoke-test-${RANDOM}"

if [ -z "${IMAGE}" ] || [ "${IMAGE}" = "null" ]; then
  echo "ERROR: IMAGE_TAG environment variable is not set"
  exit 1
fi

cleanup() {
  docker rm "${CONTAINER_NAME}" 2>/dev/null || true
}
trap cleanup EXIT

echo "Running version check..."
docker run \
  --pull=never \
  --platform="${PLATFORM}" \
  --name "${CONTAINER_NAME}" \
  "${IMAGE}" \
  version

IMAGE_ARCH=$(docker image inspect "${IMAGE}" | jq -r '.[0].Architecture')
EXPECTED_ARCH=$(echo "${PLATFORM}" | cut -d'/' -f2)

if [ "${IMAGE_ARCH}" != "${EXPECTED_ARCH}" ] && [ "${IMAGE_ARCH}" != "null" ]; then
  echo "ERROR: architecture mismatch expected=${EXPECTED_ARCH} got=${IMAGE_ARCH}"
  exit 1
fi

echo "Smoke test passed"
