# CLI image with Prometheus for ECS scenarios that need worker metric scraping
# Use this instead of cli.Dockerfile when you need to scrape remote worker metrics
#
# Build: docker build -f dockerfiles/cli-prometheus.Dockerfile -t omes-cli-prometheus .
# Run:   docker run -e WORKER_METRICS_HOST=<worker-ip> omes-cli-prometheus run-scenario ...

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
RUN CGO_ENABLED=0 go build -o temporal-omes ./cmd

# Install protoc-gen-go for kitchen-sink-gen build
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0

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

# Runtime with Prometheus for remote worker scraping
FROM --platform=linux/$TARGETARCH alpine:3.20

RUN apk add --no-cache ca-certificates bash

# Install Prometheus
ARG PROMETHEUS_VERSION=3.0.1
RUN wget -q https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/prometheus-${PROMETHEUS_VERSION}.linux-amd64.tar.gz \
    && tar xzf prometheus-*.tar.gz \
    && mv prometheus-*/prometheus /usr/local/bin/ \
    && rm -rf prometheus-*

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/loadgen/kitchen-sink-gen/target/*/release/kitchen-sink-gen /app/kitchen-sink-gen

# Copy entrypoint script for runtime prom-config.yml generation
COPY dockerfiles/cli-prometheus-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]