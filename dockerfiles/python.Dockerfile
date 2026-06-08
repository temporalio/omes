# Build in a full featured container
ARG TARGETARCH
FROM --platform=linux/$TARGETARCH ghcr.io/astral-sh/uv:latest AS uv
FROM --platform=linux/$TARGETARCH python:3.11-bullseye AS build

# Install protobuf compiler
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
    protobuf-compiler=3.12.4-1+deb11u1 libprotobuf-dev=3.12.4-1+deb11u1

# Get go compiler
ARG TARGETARCH
RUN wget -q https://go.dev/dl/go1.21.12.linux-${TARGETARCH}.tar.gz \
    && tar -C /usr/local -xzf go1.21.12.linux-${TARGETARCH}.tar.gz
# Install Rust for compiling the core bridge - only required for installation from a repo but is cheap enough to install
# in the "build" container (-y is for non-interactive install)
# hadolint ignore=DL4006
RUN wget -q -O - https://sh.rustup.rs | sh -s -- -y

ENV PATH="$PATH:/root/.cargo/bin"

# Install uv
COPY --from=uv /uv /uvx /bin/

WORKDIR /app

# Copy CLI build dependencies
COPY cmd ./cmd
COPY clioptions ./clioptions
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY metrics ./metrics
COPY devserver ./devserver
COPY versions ./versions
COPY internal ./internal
COPY workers/go/harness/api ./workers/go/harness/api
COPY go.mod go.sum ./

# Build the CLI
RUN CGO_ENABLED=0 /usr/local/go/bin/go build -o temporal-omes ./cmd/omes

ARG SDK_VERSION

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Copy the worker files
COPY workers/python ./workers/python

# Build one prepared package that can run any Python app via --app.
RUN CGO_ENABLED=0 ./temporal-omes prepare-worker --language python --dir-name prepared --version "$SDK_VERSION"

# Copy the CLI and built worker to a run container
FROM --platform=linux/$TARGETARCH python:3.11-slim-bullseye

COPY --from=uv /uv /uvx /bin/
COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/python /app/workers/python
COPY dockerfiles/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

ENV UV_NO_SYNC=1 UV_FROZEN=1 UV_OFFLINE=1
ENV OMES_WORKER_LANGUAGE=python
ENV OMES_PREPARED_DIR=prepared

ENTRYPOINT ["/app/entrypoint.sh"]
