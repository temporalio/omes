# Workflow testing image for Python
# Build with: docker build -f dockerfiles/workflowtest-python.Dockerfile \
#   --build-arg TEST_PROJECT=workflowtests/python/tests/simple_test \
#   -t workflowtest-python:test .

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

# Copy test project (path like: workflowtests/python/tests/simple_test)
ARG TEST_PROJECT
COPY ${TEST_PROJECT} ./${TEST_PROJECT}

# Set the test project path for easy reference
ENV TEST_PROJECT_PATH=/app/${TEST_PROJECT}

# No fixed entrypoint - command provided at runtime
# Example orchestrator: /app/omes workflow --language python --project-dir $TEST_PROJECT_PATH ...
# Example worker: /app/omes exec --language python --project-dir $TEST_PROJECT_PATH --remote-worker 8081 ...
