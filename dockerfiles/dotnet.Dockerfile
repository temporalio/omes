# syntax=docker/dockerfile:1.7-labs
# Build in a full featured container
FROM mcr.microsoft.com/dotnet/sdk:8.0-jammy AS build

ARG TARGETARCH

# Install protobuf compiler and build tools
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler=3.12.4* libprotobuf-dev=3.12.4* build-essential=12.*

# Get go compiler
RUN wget -q https://go.dev/dl/go1.21.12.linux-${TARGETARCH}.tar.gz \
    && tar -C /usr/local -xzf go1.21.12.linux-${TARGETARCH}.tar.gz

# Install Rust for compiling the core bridge - only required for installation from a repo but is cheap enough to install
# in the "build" container (-y is for non-interactive install)
# hadolint ignore=DL4006
RUN wget -q -O - https://sh.rustup.rs | sh -s -- -y

ENV PATH="$PATH:/root/.cargo/bin:/usr/local/go/bin"

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
COPY workers/dotnet ./workers/dotnet

# Prepare the worker
RUN --mount=type=cache,target=/root/.nuget/packages \
    --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    CGO_ENABLED=0 ./temporal-omes prepare-worker --language cs --dir-name prepared --version "$SDK_VERSION"

# Copy the CLI and prepared feature to a distroless "run" container
FROM --platform=linux/$TARGETARCH mcr.microsoft.com/dotnet/sdk:8.0-jammy

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/dotnet /app/workers/dotnet
# # Use entrypoint instead of command to "bake" the default command options
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "cs", "--dir-name", "prepared"]
