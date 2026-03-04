GO ?= go
PKG ?= ./...
BIN ?= runlix-gitops

.PHONY: build test fmt fmt-check vet ci smoke

build:
	$(GO) build -o $(BIN) ./cmd/runlix-gitops

test:
	$(GO) test $(PKG)

fmt:
	gofmt -w $$(find . -name '*.go' -type f | grep -v '^./vendor/')

fmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -type f | grep -v '^./vendor/'))" || \
	(echo "Go files are not formatted. Run 'make fmt'." && exit 1)

vet:
	$(GO) vet $(PKG)

ci: fmt-check vet test

smoke:
	@IMAGE_TAG="$(IMAGE_TAG)" PLATFORM="$(PLATFORM)" FUNCTIONAL_SMOKE="$(FUNCTIONAL_SMOKE)" \
	SMOKE_REPO_URL="$(SMOKE_REPO_URL)" SMOKE_REPO_REF="$(SMOKE_REPO_REF)" \
	SMOKE_GITHUB_APP_ID="$(SMOKE_GITHUB_APP_ID)" \
	SMOKE_GITHUB_APP_INSTALLATION_ID="$(SMOKE_GITHUB_APP_INSTALLATION_ID)" \
	SMOKE_GITHUB_APP_PRIVATE_KEY="$(SMOKE_GITHUB_APP_PRIVATE_KEY)" \
	SMOKE_GITHUB_APP_PRIVATE_KEY_FILE="$(SMOKE_GITHUB_APP_PRIVATE_KEY_FILE)" \
	SMOKE_CONFIG_FILE="$(SMOKE_CONFIG_FILE)" \
	SMOKE_ALLOWLIST_FILES="$(SMOKE_ALLOWLIST_FILES)" \
	SMOKE_ALLOWLIST_DIRS="$(SMOKE_ALLOWLIST_DIRS)" \
	SMOKE_DENYLIST_PATHS="$(SMOKE_DENYLIST_PATHS)" \
	.ci/smoke-test.sh
