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

# Build worker using official .NET SDK image
FROM --platform=linux/$TARGETARCH mcr.microsoft.com/dotnet/sdk:8.0-jammy AS build

# Install protobuf compiler and build tools
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler=3.12.4* libprotobuf-dev=3.12.4* build-essential=12.* \
      curl \
 && rm -rf /var/lib/apt/lists/*

# Install Rust using official rustup installer
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain 1.83.0
ENV PATH="/root/.cargo/bin:${PATH}"

WORKDIR /app

# Copy the CLI from go-builder stage
COPY --from=go-builder /app/temporal-omes /app/temporal-omes

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
