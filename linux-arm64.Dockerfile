ARG BUILDER_REF="docker.io/library/debian:bookworm-slim@sha256:818bdcdbfc5364abea04a4f3038db34a0fdc26a164ac95f54eb626b32bc4dda4"
ARG BASE_REF="ghcr.io/runlix/distroless-runtime-v2-canary:stable@sha256:111187cf8b6644f9779a9b654dce1d33bc194a1a8f5434b583d1c036420a28fa"
ARG APP_VERSION="0.1.0"

FROM ${BUILDER_REF} AS builder

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

FROM ${BUILDER_REF} AS git-deps

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    openssh-client \
 && rm -rf /var/lib/apt/lists/*

FROM ${BASE_REF}

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
