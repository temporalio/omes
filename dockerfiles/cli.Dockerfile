# Build in a full featured container
ARG TARGETARCH

# Source stage: prepare source code and install Antithesis SDK
FROM --platform=linux/$TARGETARCH golang:1.25 AS source

WORKDIR /app

# Install protobuf compiler and git
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
    protobuf-compiler=3.21.12-11 libprotoc-dev=3.21.12-11 \
    && rm -rf /var/lib/apt/lists/*

# Copy all source code
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY workers ./workers/
COPY go.mod go.sum ./

# Install Antithesis SDK and instrumentor
RUN go get github.com/antithesishq/antithesis-sdk-go@feature-assertion-wrappers && \
    go install github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor@feature-assertion-wrappers

# Instrumented stage: instrument the code with Antithesis
FROM --platform=linux/$TARGETARCH golang:1.25 AS instrumented

# Copy source and instrumentor
COPY --from=source /app /app
COPY --from=source /go/bin/antithesis-go-instrumentor /go/bin/antithesis-go-instrumentor
COPY --from=source /go/pkg/mod /go/pkg/mod

WORKDIR /app

RUN mkdir /app_transformed && \
    antithesis-go-instrumentor /app /app_transformed

# Build stage: compile the instrumented code
FROM --platform=linux/$TARGETARCH golang:1.25 AS build

ARG TARGETARCH

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

# Copy entire instrumented structure
COPY --from=instrumented /app_transformed /app_transformed

# Set working directory to the customer code
WORKDIR /app_transformed/customer

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

# Copy the CLI to a distroless "run" container
FROM --platform=linux/$TARGETARCH gcr.io/distroless/static-debian11:nonroot

COPY --from=build /app_transformed/customer/temporal-omes /app/temporal-omes
COPY --from=build /app_transformed/customer/loadgen/kitchen-sink-gen/target/*/release/kitchen-sink-gen /app/kitchen-sink-gen

# Copy instrumentation metadata
COPY --from=instrumented /app_transformed/notifier /notifier
COPY --from=instrumented /app_transformed/symbols /symbols

# Default entrypoint for CLI usage
ENTRYPOINT ["/app/temporal-omes"]
