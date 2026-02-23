# Workflow testing base image for TypeScript.
# This image contains omes + TypeScript runtime deps + starter library only.
# Test projects are copied by a thin overlay image (see workflowtest-project.Dockerfile).
#
# Build with:
#   docker build -f workflowtests/dockerfiles/workflowtest-typescript.Dockerfile -t omes-workflowtest-typescript-base:latest .

ARG TARGETARCH
FROM --platform=linux/$TARGETARCH node:20-bullseye AS build
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

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

# Build omes-starter TypeScript library so runtime does not depend on host-built lib/
FROM --platform=linux/$TARGETARCH node:20-bullseye AS ts-starter-build

WORKDIR /app/workflowtests/typescript
COPY workflowtests/typescript/src ./src
COPY workflowtests/typescript/package.json ./package.json
COPY workflowtests/typescript/package-lock.json ./package-lock.json
COPY workflowtests/typescript/tsconfig.json ./tsconfig.json
RUN npm ci && npm run build

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
COPY --from=ts-starter-build /app/workflowtests/typescript/lib ./workflowtests/typescript/lib
COPY workflowtests/typescript/package.json ./workflowtests/typescript/package.json
COPY workflowtests/typescript/package-lock.json ./workflowtests/typescript/package-lock.json
COPY workflowtests/typescript/tsconfig.json ./workflowtests/typescript/tsconfig.json

# Use omes as entrypoint; role and project are provided at runtime.
ENTRYPOINT ["/app/omes"]

# Example runner command:
#   workflow --language typescript --project-dir /app/workflowtests/typescript/tests/simple-test ...
#
# Example worker command:
#   exec --language typescript --project-dir /app/workflowtests/typescript/tests/simple-test -- worker ...
