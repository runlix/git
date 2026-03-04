# Git CI Configuration

This directory contains configuration and scripts for the CI/CD pipeline.

## Files

### docker-matrix.json

Defines the build matrix for multi-architecture Docker images. See the [schema documentation](https://github.com/runlix/build-workflow/blob/main/schema/docker-matrix-schema.json) for details.

**Variants:**
- `latest-amd64` - Stable build for AMD64
- `latest-arm64` - Stable build for ARM64
- `debug-amd64` - Debug build for AMD64
- `debug-arm64` - Debug build for ARM64

### smoke-test.sh

Automated smoke test script that validates built Docker images before they are released.

**What it tests:**
- Container starts successfully
- `runlix-gitops version` executes
- Correct architecture is used

**Environment Variables:**
- `IMAGE_TAG` (required) - The Docker image tag to test (set by workflow)
- `PLATFORM` (optional) - Platform to test, defaults to `linux/amd64`
