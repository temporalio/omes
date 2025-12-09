# Build in a full featured container
ARG TARGETARCH

FROM --platform=linux/$TARGETARCH golang:1.25 AS build

WORKDIR /app

# Install protobuf compiler and git
RUN apt-get update \
  && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler=3.21.12-11 libprotoc-dev=3.21.12-11 \
  && rm -rf /var/lib/apt/lists/*

# Install Rust for kitchen-sink-gen
RUN wget -q -O - https://sh.rustup.rs | sh -s -- -y \
  && . $HOME/.cargo/env \
  && echo "TARGETARCH: $TARGETARCH" \
  && ARCH=$(uname -m) \
  && echo "uname -m: $ARCH" \
  && if [ "$TARGETARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then \
       rustup target add aarch64-unknown-linux-musl; \
     else \
       rustup target add x86_64-unknown-linux-musl; \
     fi
ENV PATH="$PATH:/root/.cargo/bin"

# Copy CLI build dependencies
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY workers ./workers/
COPY go.mod go.sum ./

# Build the CLI
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o temporal-omes ./cmd

# Install protoc-gen-go for kitchen-sink-gen build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0

# Build kitchen-sink-gen (statically linked)
RUN --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/app/loadgen/kitchen-sink-gen/target \
    cd loadgen/kitchen-sink-gen && \
  echo "TARGETARCH: $TARGETARCH" && \
  ARCH=$(uname -m) && \
  echo "uname -m: $ARCH" && \
  if [ "$TARGETARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then \
    RUST_TARGET=aarch64-unknown-linux-musl; \
  else \
    RUST_TARGET=x86_64-unknown-linux-musl; \
  fi && \
  echo "Building for rust target: $RUST_TARGET" && \
  RUSTFLAGS='-C target-feature=+crt-static' cargo build --release --target $RUST_TARGET

# Copy the CLI to a distroless "run" container
FROM --platform=linux/$TARGETARCH gcr.io/distroless/static-debian11:nonroot

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/loadgen/kitchen-sink-gen/target/*/release/kitchen-sink-gen /app/kitchen-sink-gen

# Default entrypoint for CLI usage
ENTRYPOINT ["/app/temporal-omes"]