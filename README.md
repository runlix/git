# git

Unified distroless Docker image for Home Assistant config pull/push Git operations.

## Purpose

Provides one hardened image (`runlix-gitops`) that replaces the split runtime of:
- `git-sync` init for pull
- separate git cron image for commit/push

## Commands

- `pull-init`: sync allowlisted content from repo to `/config`
- `sync-push`: sync allowlisted content from `/config` to repo, commit, and push when changed
- `version`: print binary version

## Environment

Required:
- `REPO_URL`
- `GITHUB_APP_ID`
- `GITHUB_APP_INSTALLATION_ID`
- `GITHUB_APP_PRIVATE_KEY_FILE`

Optional with defaults:
- `REPO_REF` = `main`
- `WORK_DIR` = `/work`
- `CONFIG_DIR` = `/config`
- `ALLOWLIST_FILES` = `configuration.yaml,automations.yaml,scripts.yaml,scenes.yaml`
- `ALLOWLIST_DIRS` = `packages,themes,www,blueprints,custom_components`
- `DENYLIST_PATHS` = `secrets.yaml,.storage,home-assistant_v2.db,home-assistant.log,deps,tts,cloud,ssh`
- `GIT_AUTHOR_NAME` = `runlix-gitops`
- `GIT_AUTHOR_EMAIL` = `gitops@runlix.local`
- `COMMIT_MESSAGE_TEMPLATE` = `Home Assistant config sync ({{ref}} @ {{timestamp}})`
- `GITHUB_API_URL` = `https://api.github.com` (override for tests/mocks)

## Logging Contract

All logs are key-value lines.

Success logs include:
- `operation`
- `repo`
- `ref`
- `changed_paths_count`
- `duration_ms`
- `commit_sha` (only when a commit is pushed)

Error logs include:
- `operation`
- `error_code`
- `msg`
- `error`
- `duration_ms`

Error code values:
- `CONFIG_MISSING`
- `AUTH_GITHUB_APP`
- `REPO_PREPARE`
- `COPY_ALLOWLIST`
- `DENYLIST_ENFORCE`
- `GIT_STATUS`
- `GIT_COMMIT`
- `GIT_PUSH`
- `UNKNOWN_SUBCOMMAND`

## Exit Codes

- `0`: success, including no-op `sync-push` when nothing changed
- non-zero: hard failure

## Build and Test

- `make build`
- `make test`
- `make ci`
- `make smoke IMAGE_TAG=ghcr.io/runlix/git:tag`
