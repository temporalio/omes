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
COPY workers ./workers/

# Build the CLI
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o temporal-omes ./cmd

# Build kitchen-sink-gen using official rust image
FROM --platform=linux/$TARGETARCH rust:1.83-bookworm AS rust-builder

WORKDIR /app

# Install protobuf compiler
RUN apt-get update \
  && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler libprotoc-dev \
  && rm -rf /var/lib/apt/lists/*

# Install protoc-gen-go (needed by kitchen-sink-gen build.rs)
COPY --from=go-builder /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:$PATH"
RUN --mount=type=cache,target=/go/pkg/mod \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
ENV PATH="/root/go/bin:$PATH"

# Add Rust musl target for static linking
RUN ARCH=$(uname -m) \
  && if [ "$TARGETARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then \
       rustup target add aarch64-unknown-linux-musl; \
     else \
       rustup target add x86_64-unknown-linux-musl; \
     fi

# Copy kitchen-sink-gen source
COPY loadgen/kitchen-sink-gen ./loadgen/kitchen-sink-gen
COPY workers/proto ./workers/proto

# Build kitchen-sink-gen (statically linked)
RUN --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    cd loadgen/kitchen-sink-gen && \
  ARCH=$(uname -m) && \
  if [ "$TARGETARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then \
    RUST_TARGET=aarch64-unknown-linux-musl; \
  else \
    RUST_TARGET=x86_64-unknown-linux-musl; \
  fi && \
  RUSTFLAGS='-C target-feature=+crt-static' cargo build --release --target $RUST_TARGET

# Copy the CLI to a distroless "run" container
FROM --platform=linux/$TARGETARCH gcr.io/distroless/static-debian11:nonroot

COPY --from=go-builder /app/temporal-omes /app/temporal-omes
COPY --from=rust-builder /app/loadgen/kitchen-sink-gen/target/*/release/kitchen-sink-gen /app/kitchen-sink-gen

# Default entrypoint for CLI usage
ENTRYPOINT ["/app/temporal-omes"]