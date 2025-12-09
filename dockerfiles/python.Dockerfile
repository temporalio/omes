# Build in a full featured container
ARG TARGETARCH
FROM --platform=linux/$TARGETARCH ghcr.io/astral-sh/uv:latest AS uv
FROM --platform=linux/$TARGETARCH python:3.11-bullseye AS build

# Install protobuf compiler
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
    protobuf-compiler=3.12.4-1+deb11u1 libprotobuf-dev=3.12.4-1+deb11u1

# Get go compiler
ARG TARGETARCH
RUN wget -q https://go.dev/dl/go1.21.12.linux-${TARGETARCH}.tar.gz \
    && tar -C /usr/local -xzf go1.21.12.linux-${TARGETARCH}.tar.gz
# Install Rust for compiling the core bridge - only required for installation from a repo but is cheap enough to install
# in the "build" container (-y is for non-interactive install)
# hadolint ignore=DL4006
RUN wget -q -O - https://sh.rustup.rs | sh -s -- -y

ENV PATH="$PATH:/root/.cargo/bin"

# Install uv
COPY --from=uv /uv /uvx /bin/

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
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 /usr/local/go/bin/go build -o temporal-omes ./cmd

ARG SDK_VERSION

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Copy the worker files
COPY workers/python ./workers/python

# Build the worker
RUN --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/root/.cache/uv \
    CGO_ENABLED=0 ./temporal-omes prepare-worker --language python --dir-name prepared --version "$SDK_VERSION"

# Copy the CLI and built worker to a distroless "run" container
FROM --platform=linux/$TARGETARCH python:3.11-slim-bullseye

COPY --from=uv /uv /uvx /bin/
COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/python /app/workers/python

ENV UV_NO_SYNC=1 UV_FROZEN=1 UV_OFFLINE=1

# Put the language and dir, but let other options (like required scenario and run-id) be given by user
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "python", "--dir-name", "prepared"]
