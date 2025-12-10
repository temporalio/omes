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

# Build worker using official rust image (for TypeScript SDK core bridge compilation)
FROM --platform=linux/$TARGETARCH rust:1.83-bookworm AS build

# Install Node.js 20
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      curl gnupg \
      protobuf-compiler libprotobuf-dev \
 && mkdir -p /etc/apt/keyrings \
 && curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg \
 && echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_20.x nodistro main" | tee /etc/apt/sources.list.d/nodesource.list \
 && apt-get update \
 && apt-get install --no-install-recommends --assume-yes nodejs \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the CLI from go-builder stage
COPY --from=go-builder /app/temporal-omes /app/temporal-omes

ARG SDK_VERSION

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Copy the worker files
COPY workers/proto ./workers/proto
COPY workers/typescript ./workers/typescript

# Build typescript proto files
# hadolint ignore=DL3003
RUN --mount=type=cache,target=/root/.npm \
    cd workers/typescript && npm install && npm run proto-gen

# Build the worker
RUN --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/root/.npm \
    CGO_ENABLED=0 ./temporal-omes prepare-worker --language ts --dir-name prepared --version "$SDK_VERSION"

# Copy the CLI and prepared feature to a "run" container.
# hadolint ignore=DL3006
FROM --platform=linux/$TARGETARCH gcr.io/distroless/nodejs20-debian12

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/typescript /app/workers/typescript

# Node is installed here 👇 in distroless
ENV PATH="/nodejs/bin:$PATH"
# Use entrypoint instead of command to "bake" the default command options
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "typescript", "--dir-name", "prepared"]
