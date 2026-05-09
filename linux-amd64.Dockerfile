ARG BUILDER_REF="docker.io/library/debian:bookworm-slim@sha256:2749ca60ffb3c42de053229d7967d292d7dad1067936b38995da0bbfb96c4c23"
ARG BASE_REF="ghcr.io/runlix/distroless-runtime-v2-canary:stable@sha256:1a89d10a17987e68616af43e078b60480fc6e889f16a5ab249568a17a1e8a8a8"
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

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
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

ARG LIB_DIR=x86_64-linux-gnu

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
