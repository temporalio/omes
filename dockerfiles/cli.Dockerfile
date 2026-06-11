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

# Download Go dependencies first so this layer caches independently of source
# changes. The harness/api replace target must be present for `go mod download`
# to resolve the module graph.
COPY go.mod go.sum ./
COPY workers/go/harness/api ./workers/go/harness/api
RUN go mod download

# Copy CLI source and build the CLI.
COPY cmd ./cmd
COPY clioptions ./clioptions
COPY loadgen ./loadgen
COPY metrics ./metrics
COPY scenarios ./scenarios
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o temporal-omes ./cmd/omes

# Install protoc-gen-go and build kitchen-sink-gen, which needs the proto tree.
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
COPY workers/proto ./workers/proto

# Build kitchen-sink-gen (statically linked)
RUN cd loadgen/kitchen-sink-gen && \
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
