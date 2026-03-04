#!/usr/bin/env bash
set -e
set -o pipefail

IMAGE="${IMAGE_TAG}"
PLATFORM="${PLATFORM:-linux/amd64}"
VERSION_CONTAINER_NAME="git-smoke-version-${RANDOM}"
PULL_CONTAINER_NAME="git-smoke-pull-${RANDOM}"
PUSH_CONTAINER_NAME="git-smoke-push-${RANDOM}"

if [ -z "${IMAGE}" ] || [ "${IMAGE}" = "null" ]; then
  echo "ERROR: IMAGE_TAG environment variable is not set"
  exit 1
fi

cleanup() {
  docker rm "${VERSION_CONTAINER_NAME}" 2>/dev/null || true
  docker rm "${PULL_CONTAINER_NAME}" 2>/dev/null || true
  docker rm "${PUSH_CONTAINER_NAME}" 2>/dev/null || true
  if [ -n "${TMP_DIR:-}" ] && [ -d "${TMP_DIR}" ]; then
    chmod -R 777 "${TMP_DIR}" 2>/dev/null || true
    rm -rf "${TMP_DIR}" 2>/dev/null || true
  fi
}
trap cleanup EXIT

echo "[smoke] running version check..."
docker run \
  --pull=never \
  --platform="${PLATFORM}" \
  --name "${VERSION_CONTAINER_NAME}" \
  "${IMAGE}" \
  version

IMAGE_ARCH=$(docker image inspect "${IMAGE}" | jq -r '.[0].Architecture')
EXPECTED_ARCH=$(echo "${PLATFORM}" | cut -d'/' -f2)

if [ "${IMAGE_ARCH}" != "${EXPECTED_ARCH}" ] && [ "${IMAGE_ARCH}" != "null" ]; then
  echo "ERROR: architecture mismatch expected=${EXPECTED_ARCH} got=${IMAGE_ARCH}"
  exit 1
fi

echo "[smoke] version and architecture checks passed"

if [ "${FUNCTIONAL_SMOKE:-false}" != "true" ]; then
  echo "[smoke] functional checks skipped (set FUNCTIONAL_SMOKE=true to enable)"
  exit 0
fi

REQUIRED_VARS=(
  SMOKE_REPO_URL
  SMOKE_GITHUB_APP_ID
  SMOKE_GITHUB_APP_INSTALLATION_ID
)
for var_name in "${REQUIRED_VARS[@]}"; do
  if [ -z "${!var_name:-}" ]; then
    echo "ERROR: FUNCTIONAL_SMOKE=true requires ${var_name}"
    exit 1
  fi
done

TMP_DIR=$(mktemp -d)
WORK_DIR="${TMP_DIR}/work"
CONFIG_DIR="${TMP_DIR}/config"
mkdir -p "${WORK_DIR}" "${CONFIG_DIR}"
chmod 777 "${WORK_DIR}" "${CONFIG_DIR}"

if [ -n "${SMOKE_GITHUB_APP_PRIVATE_KEY_FILE:-}" ]; then
  KEY_FILE="${SMOKE_GITHUB_APP_PRIVATE_KEY_FILE}"
elif [ -n "${SMOKE_GITHUB_APP_PRIVATE_KEY:-}" ]; then
  KEY_FILE="${TMP_DIR}/app.pem"
  printf "%s" "${SMOKE_GITHUB_APP_PRIVATE_KEY}" > "${KEY_FILE}"
  chmod 600 "${KEY_FILE}"
else
  echo "ERROR: FUNCTIONAL_SMOKE=true requires SMOKE_GITHUB_APP_PRIVATE_KEY or SMOKE_GITHUB_APP_PRIVATE_KEY_FILE"
  exit 1
fi

REPO_REF="${SMOKE_REPO_REF:-main}"
ALLOWLIST_FILES="${SMOKE_ALLOWLIST_FILES:-configuration.yaml,automations.yaml,scripts.yaml,scenes.yaml}"
ALLOWLIST_DIRS="${SMOKE_ALLOWLIST_DIRS:-packages,themes,www,blueprints,custom_components}"
DENYLIST_PATHS="${SMOKE_DENYLIST_PATHS:-secrets.yaml,.storage,home-assistant_v2.db,home-assistant.log,deps,tts,cloud,ssh}"
TARGET_CONFIG_FILE="${SMOKE_CONFIG_FILE:-configuration.yaml}"
UNIQUE_MARKER="# runlix-smoke $(date -u +%Y%m%dT%H%M%SZ)"

COMMON_ENV=(
  -e REPO_URL="${SMOKE_REPO_URL}"
  -e REPO_REF="${REPO_REF}"
  -e WORK_DIR=/work
  -e CONFIG_DIR=/config
  -e ALLOWLIST_FILES="${ALLOWLIST_FILES}"
  -e ALLOWLIST_DIRS="${ALLOWLIST_DIRS}"
  -e DENYLIST_PATHS="${DENYLIST_PATHS}"
  -e GITHUB_APP_ID="${SMOKE_GITHUB_APP_ID}"
  -e GITHUB_APP_INSTALLATION_ID="${SMOKE_GITHUB_APP_INSTALLATION_ID}"
  -e GITHUB_APP_PRIVATE_KEY_FILE=/secrets/app.pem
)

if [ -n "${GITHUB_API_URL:-}" ]; then
  COMMON_ENV+=( -e GITHUB_API_URL="${GITHUB_API_URL}" )
fi

echo "[smoke] running functional pull-init..."
docker run \
  --pull=never \
  --platform="${PLATFORM}" \
  --name "${PULL_CONTAINER_NAME}" \
  -v "${WORK_DIR}:/work" \
  -v "${CONFIG_DIR}:/config" \
  -v "${KEY_FILE}:/secrets/app.pem:ro" \
  "${COMMON_ENV[@]}" \
  "${IMAGE}" \
  pull-init

if [ ! -e "${CONFIG_DIR}/${TARGET_CONFIG_FILE}" ]; then
  echo "ERROR: pull-init did not materialize ${TARGET_CONFIG_FILE} in /config"
  exit 1
fi

printf "\n%s\n" "${UNIQUE_MARKER}" >> "${CONFIG_DIR}/${TARGET_CONFIG_FILE}"

mkdir -p "${CONFIG_DIR}/.storage"
printf '%s\n' '{"ignore": true}' > "${CONFIG_DIR}/.storage/should-not-sync"
printf '%s\n' 'token: test' > "${CONFIG_DIR}/secrets.yaml"

echo "[smoke] running functional sync-push..."
docker run \
  --pull=never \
  --platform="${PLATFORM}" \
  --name "${PUSH_CONTAINER_NAME}" \
  -v "${WORK_DIR}:/work" \
  -v "${CONFIG_DIR}:/config" \
  -v "${KEY_FILE}:/secrets/app.pem:ro" \
  "${COMMON_ENV[@]}" \
  -e GIT_AUTHOR_NAME="runlix-smoke" \
  -e GIT_AUTHOR_EMAIL="smoke@runlix.local" \
  -e COMMIT_MESSAGE_TEMPLATE="runlix smoke {{timestamp}}" \
  "${IMAGE}" \
  sync-push

echo "[smoke] functional checks passed (pull-init + sync-push)"
