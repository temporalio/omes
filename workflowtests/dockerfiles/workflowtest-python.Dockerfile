# Workflow testing base image for Python.
# This image contains omes + Python runtime deps + starter library only.
# Test projects are copied by a thin overlay image (see workflowtest-project.Dockerfile).
#
# Build with:
#   docker build -f workflowtests/dockerfiles/workflowtest-python.Dockerfile -t omes-workflowtest-python-base:latest .

ARG TARGETARCH
FROM --platform=linux/$TARGETARCH ghcr.io/astral-sh/uv:latest AS uv
FROM --platform=linux/$TARGETARCH python:3.11-bullseye AS build

# Install protobuf compiler
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
    protobuf-compiler=3.12.4-1+deb11u1 libprotobuf-dev=3.12.4-1+deb11u1 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

# Get go compiler
ARG TARGETARCH
RUN wget -q https://go.dev/dl/go1.21.12.linux-${TARGETARCH}.tar.gz \
    && tar -C /usr/local -xzf go1.21.12.linux-${TARGETARCH}.tar.gz

# Install Rust for compiling the core bridge (needed for local SDK builds)
# hadolint ignore=DL4006
RUN wget -q -O - https://sh.rustup.rs | sh -s -- -y

ENV PATH="$PATH:/root/.cargo/bin:/usr/local/go/bin"

# Install uv
COPY --from=uv /uv /uvx /bin/

WORKDIR /app

# Copy CLI build dependencies
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY metrics ./metrics
COPY workers/*.go ./workers/
COPY internal ./internal
COPY go.mod go.sum ./

# Build the CLI
RUN CGO_ENABLED=0 go build -o omes ./cmd

# Runtime stage
FROM --platform=linux/$TARGETARCH python:3.11-bullseye

# Install Rust in runtime for SDK building (core-bridge compilation)
# hadolint ignore=DL4006
RUN wget -q -O - https://sh.rustup.rs | sh -s -- -y
ENV PATH="$PATH:/root/.cargo/bin"

COPY --from=uv /uv /uvx /bin/
COPY --from=build /app/omes /app/omes

WORKDIR /app

# Copy omes_starter library and its dependencies (maintains relative path structure)
COPY workflowtests/python/omes_starter ./workflowtests/python/omes_starter
COPY workflowtests/python/pyproject.toml ./workflowtests/python/pyproject.toml
COPY workflowtests/python/uv.lock ./workflowtests/python/uv.lock

# Use omes as entrypoint; role and project are provided at runtime.
ENTRYPOINT ["/app/omes"]

# Example runner command:
#   workflow --language python --project-dir /app/workflowtests/python/tests/simple_test ...
#
# Example worker command:
#   exec --language python --project-dir /app/workflowtests/python/tests/simple_test -- worker ...
