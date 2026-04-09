# git

Distroless Docker image for `runlix-gitops`, the Git pull/push helper used to sync Home Assistant configuration repositories.

## Image

- Image: `ghcr.io/runlix/git`
- Example pull: `docker pull ghcr.io/runlix/git:<version>-stable`
- The authoritative published tags, digests, and source revision live in `release.json`.

## Commands

- `pull-init`: sync allowlisted repository content into `/config`
- `sync-push`: sync allowlisted `/config` content back to the repository, commit, and push when changed
- `version`: print the embedded application version

## Environment

Required:
- `REPO_URL`
- `GITHUB_APP_ID`
- `GITHUB_APP_INSTALLATION_ID`
- `GITHUB_APP_PRIVATE_KEY_FILE`

Optional:
- `REPO_REF` defaults to `main`
- `WORK_DIR` defaults to `/work`
- `CONFIG_DIR` defaults to `/config`
- `ALLOWLIST_FILES` defaults to `configuration.yaml,automations.yaml,scripts.yaml,scenes.yaml`
- `ALLOWLIST_DIRS` defaults to `packages,themes,www,blueprints,custom_components`
- `DENYLIST_PATHS` defaults to `secrets.yaml,.storage,home-assistant_v2.db,home-assistant.log,deps,tts,cloud,ssh`
- `GIT_AUTHOR_NAME` defaults to `runlix-gitops`
- `GIT_AUTHOR_EMAIL` defaults to `gitops@runlix.local`
- `COMMIT_MESSAGE_TEMPLATE` defaults to `Home Assistant config sync ({{ref}} @ {{timestamp}})`
- `GITHUB_API_URL` defaults to `https://api.github.com`

## Branch Layout

- `main` owns docs, release metadata, and Renovate configuration.
- `release` owns the Go source, Dockerfiles, smoke tests, and release automation.
- Successful `release` publishes sync `release.json` back to `main`.

## CI

- `Validate Release Metadata` runs on `main` pull requests.
- `Validate Build`, `Publish Release`, and `Go Quality` run from `release`.
