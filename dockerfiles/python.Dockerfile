# Build CLI using official golang image
ARG TARGETARCH
FROM --platform=linux/$TARGETARCH golang:1.25 AS go-builder

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy CLI build dependencies
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY workers/*.go ./workers/

# Build the CLI
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o temporal-omes ./cmd

# Build worker using official rust image (for Python SDK core bridge compilation)
FROM --platform=linux/$TARGETARCH ghcr.io/astral-sh/uv:latest AS uv
FROM --platform=linux/$TARGETARCH rust:1.83-bookworm AS build

# Install Python 3.11
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
    python3.11 python3.11-dev python3.11-venv \
    protobuf-compiler libprotobuf-dev \
    && rm -rf /var/lib/apt/lists/*

# Set Python 3.11 as default
RUN update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.11 1

# Install uv
COPY --from=uv /uv /uvx /bin/

WORKDIR /app

# Copy the CLI from go-builder stage
COPY --from=go-builder /app/temporal-omes /app/temporal-omes

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

# Copy the CLI and built worker to a "run" container
FROM --platform=linux/$TARGETARCH python:3.11-slim-bookworm

COPY --from=uv /uv /uvx /bin/
COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/python /app/workers/python

ENV UV_NO_SYNC=1 UV_FROZEN=1 UV_OFFLINE=1

# Put the language and dir, but let other options (like required scenario and run-id) be given by user
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "python", "--dir-name", "prepared"]
