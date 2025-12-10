# syntax=docker/dockerfile:1.7-labs
# Build in a full featured container
FROM node:20-bullseye-slim AS build

ARG TARGETARCH

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Install protobuf compiler and build tools
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler=3.12.4-1+deb11u1 libprotobuf-dev=3.12.4-1+deb11u1 \
      build-essential wget ca-certificates

# Get go compiler
RUN wget -q https://go.dev/dl/go1.21.12.linux-${TARGETARCH}.tar.gz \
    && tar -C /usr/local -xzf go1.21.12.linux-${TARGETARCH}.tar.gz

# Need Rust to compile core if not already built
RUN wget -q -O - https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    /usr/local/go/bin/go mod download

# Copy CLI build dependencies
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY workers/*.go ./workers/

# Build the CLI
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,id=go-build-${TARGETARCH},target=/root/.cache/go-build \
    CGO_ENABLED=0 /usr/local/go/bin/go build -o temporal-omes ./cmd

ARG SDK_VERSION

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Copy the worker files
COPY workers/proto ./workers/proto
COPY workers/typescript ./workers/typescript

# Build typescript proto files
# hadolint ignore=DL3003
RUN --mount=type=cache,id=npm-cache-${TARGETARCH},target=/root/.npm \
    cd workers/typescript && npm install && npm run proto-gen

# Build the worker
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,id=go-build-${TARGETARCH},target=/root/.cache/go-build \
    --mount=type=cache,id=cargo-registry,target=/root/.cargo/registry \
    --mount=type=cache,id=cargo-git,target=/root/.cargo/git \
    --mount=type=cache,id=npm-cache-${TARGETARCH},target=/root/.npm \
    CGO_ENABLED=0 ./temporal-omes prepare-worker --language ts --dir-name prepared --version "$SDK_VERSION"

# Copy the CLI and prepared feature to a "run" container.
# hadolint ignore=DL3006
FROM --platform=linux/$TARGETARCH gcr.io/distroless/nodejs20-debian11

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/typescript /app/workers/typescript

# Node is installed here 👇 in distroless
ENV PATH="/nodejs/bin:$PATH"
# Use entrypoint instead of command to "bake" the default command options
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "typescript", "--dir-name", "prepared"]
