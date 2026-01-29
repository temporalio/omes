# Workflow testing image for TypeScript
# Build with: docker build -f dockerfiles/workflowtest-typescript.Dockerfile \
#   --build-arg TEST_PROJECT=workflowtests/typescript/tests/simple-test \
#   -t workflowtest-typescript:test .

ARG TARGETARCH
FROM --platform=linux/$TARGETARCH node:20-bullseye AS build
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Install protobuf compiler
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler=3.12.4-1+deb11u1 libprotobuf-dev=3.12.4-1+deb11u1

# Get go compiler
ARG TARGETARCH
RUN wget -q https://go.dev/dl/go1.21.12.linux-${TARGETARCH}.tar.gz \
    && tar -C /usr/local -xzf go1.21.12.linux-${TARGETARCH}.tar.gz

# Need Rust to compile core if not already built
RUN wget -q -O - https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:/usr/local/go/bin:${PATH}"

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

# Runtime stage - need full Node for SDK building
FROM --platform=linux/$TARGETARCH node:20-bullseye

# Need Rust in runtime for SDK building (core-bridge compilation)
RUN wget -q -O - https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

# Install pnpm (needed for local TypeScript SDK builds)
RUN npm install -g pnpm

COPY --from=build /app/omes /app/omes

WORKDIR /app

# Copy omes-starter library (maintains relative path structure for file:../.. references)
COPY workflowtests/typescript/src ./workflowtests/typescript/src
COPY workflowtests/typescript/lib ./workflowtests/typescript/lib
COPY workflowtests/typescript/package.json ./workflowtests/typescript/package.json
COPY workflowtests/typescript/package-lock.json ./workflowtests/typescript/package-lock.json
COPY workflowtests/typescript/tsconfig.json ./workflowtests/typescript/tsconfig.json

# Copy test project (path like: workflowtests/typescript/tests/simple-test)
ARG TEST_PROJECT
COPY ${TEST_PROJECT} ./${TEST_PROJECT}

# Set the test project path for easy reference
ENV TEST_PROJECT_PATH=/app/${TEST_PROJECT}

# No fixed entrypoint - command provided at runtime
# Example orchestrator: /app/omes workflow --language ts --project-dir $TEST_PROJECT_PATH ...
# Example worker: /app/omes exec --language ts --project-dir $TEST_PROJECT_PATH --remote-worker 8081 ...
