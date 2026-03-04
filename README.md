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

## Required Environment

- `REPO_URL`
- `REPO_REF` (default `main`)
- `WORK_DIR` (default `/work`)
- `GITHUB_APP_ID`
- `GITHUB_APP_INSTALLATION_ID`
- `GITHUB_APP_PRIVATE_KEY_FILE`

## Optional Environment

- `CONFIG_DIR` (default `/config`)
- `ALLOWLIST_FILES`
- `ALLOWLIST_DIRS`
- `DENYLIST_PATHS`
- `GIT_AUTHOR_NAME`
- `GIT_AUTHOR_EMAIL`
- `COMMIT_MESSAGE_TEMPLATE`
