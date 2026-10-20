# Build Go separately so the Rust compiler and runtime use matching Debian ABIs.
ARG TARGETARCH
FROM --platform=linux/$TARGETARCH golang:1.26 AS cli
WORKDIR /app
COPY go.mod go.sum ./
COPY workers/go/harness/api ./workers/go/harness/api
RUN go mod download
COPY cmd ./cmd
COPY clioptions ./clioptions
COPY loadgen ./loadgen
COPY metrics ./metrics
COPY scenarios ./scenarios
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o temporal-omes ./cmd/omes

FROM --platform=linux/$TARGETARCH rust:1.93.1-bookworm AS build
RUN apt-get update \
    && apt-get install --no-install-recommends --assume-yes protobuf-compiler libprotobuf-dev clang libssl-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=cli /app/temporal-omes ./temporal-omes

ARG SDK_VERSION
# Optional sdk-rust checkout (including its initialized submodules).
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo
COPY workers/rust ./workers/rust
COPY workers/proto ./workers/proto
RUN ./temporal-omes prepare-worker --language rust --dir-name prepared --version "$SDK_VERSION" --log-level debug

FROM --platform=linux/$TARGETARCH debian:bookworm-slim
RUN apt-get update \
    && apt-get install --no-install-recommends --assume-yes ca-certificates libssl3 libstdc++6 \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=cli /app/temporal-omes /app/temporal-omes
# The worker and helloworld project-server are one native executable.
COPY --from=build /app/workers/rust/prepared/program /app/workers/rust/prepared/program
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "rust", "--dir-name", "prepared"]
