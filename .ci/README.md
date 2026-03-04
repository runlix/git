# Git CI Configuration

This directory contains configuration and scripts for the CI/CD pipeline.

## Files

### docker-matrix.json

Defines the build matrix for multi-architecture Docker images. See the [schema documentation](https://github.com/runlix/build-workflow/blob/main/schema/docker-matrix-schema.json) for details.

Variants:
- `latest-amd64` - Stable build for AMD64
- `latest-arm64` - Stable build for ARM64
- `debug-amd64` - Debug build for AMD64
- `debug-arm64` - Debug build for ARM64

### smoke-test.sh

Automated smoke test script used by CI to validate built images.

What it tests:
- Container runs and `runlix-gitops version` executes.
- Image architecture matches requested platform.
- Optional functional flow for both subcommands:
  - `pull-init`
  - `sync-push`

Environment variables:
- `IMAGE_TAG` (required)
- `PLATFORM` (optional, default `linux/amd64`)
- `FUNCTIONAL_SMOKE` (optional, `true` to enable functional flow)
- `SMOKE_REPO_URL` (required when functional smoke enabled)
- `SMOKE_REPO_REF` (optional, default `main`)
- `SMOKE_GITHUB_APP_ID` (required when functional smoke enabled)
- `SMOKE_GITHUB_APP_INSTALLATION_ID` (required when functional smoke enabled)
- `SMOKE_GITHUB_APP_PRIVATE_KEY` or `SMOKE_GITHUB_APP_PRIVATE_KEY_FILE` (required when functional smoke enabled)
- `SMOKE_ALLOWLIST_FILES` (optional override)
- `SMOKE_ALLOWLIST_DIRS` (optional override)
- `SMOKE_DENYLIST_PATHS` (optional override)
- `SMOKE_CONFIG_FILE` (optional, file expected after `pull-init`, default `configuration.yaml`)
- `GITHUB_API_URL` (optional override for mock API endpoint)
