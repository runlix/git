ARG BUILDER_IMAGE=docker.io/library/debian
ARG BUILDER_TAG=bookworm-slim
ARG BASE_IMAGE=ghcr.io/runlix/distroless-runtime
ARG BASE_TAG=stable
ARG BUILDER_DIGEST=""
ARG BASE_DIGEST=""
ARG APP_VERSION=0.1.0

FROM ${BUILDER_IMAGE}:${BUILDER_TAG}@${BUILDER_DIGEST} AS builder

ARG APP_VERSION

WORKDIR /src

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    golang-go \
 && rm -rf /var/lib/apt/lists/*

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${APP_VERSION}" \
    -o /out/runlix-gitops \
    ./cmd/runlix-gitops

FROM ${BUILDER_IMAGE}:${BUILDER_TAG}@${BUILDER_DIGEST} AS git-deps

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    openssh-client \
 && rm -rf /var/lib/apt/lists/*

FROM ${BASE_IMAGE}:${BASE_TAG}@${BASE_DIGEST}

ARG LIB_DIR=aarch64-linux-gnu

COPY --from=builder /out/runlix-gitops /app/runlix-gitops
COPY --from=git-deps /usr/bin/git /usr/bin/git
COPY --from=git-deps /usr/bin/ssh /usr/bin/ssh
COPY --from=git-deps /usr/lib/git-core /usr/lib/git-core
COPY --from=git-deps /etc/ssl/certs /etc/ssl/certs
COPY --from=git-deps /usr/lib/${LIB_DIR}/ /usr/lib/${LIB_DIR}/
COPY --from=git-deps /lib/${LIB_DIR}/ /lib/${LIB_DIR}/

ENV HOME=/tmp
ENV GIT_EXEC_PATH=/usr/lib/git-core

WORKDIR /work
USER 20020:20020
ENTRYPOINT ["/app/runlix-gitops"]
