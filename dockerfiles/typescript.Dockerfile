# Build in a full featured container
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
ENV PATH="/root/.cargo/bin:${PATH}"

WORKDIR /app

# Copy CLI build dependencies
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY metrics ./metrics
COPY workers/*.go ./workers/
COPY workers/go/harness/api ./workers/go/harness/api
COPY go.mod go.sum ./

# Build the CLI
RUN CGO_ENABLED=0 /usr/local/go/bin/go build -o temporal-omes ./cmd

ARG SDK_VERSION
ARG PROJECT_NAME=""

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Read BUILD_CORE_RELEASE env var. This builds TS worker with core bridge in release mode.
# This is only relevant if building from source (if using a published version, the worker is already
# built in release mode).
ARG BUILD_CORE_RELEASE=false
ENV BUILD_CORE_RELEASE=${BUILD_CORE_RELEASE}

# Copy the worker files
COPY workers/proto ./workers/proto
COPY workers/typescript ./workers/typescript

# Install pnpm (sdkbuild uses pnpm to build typescript programs)
RUN npm install -g pnpm

# prepare-worker builds the TypeScript workspace itself: it installs npm deps,
# runs the relevant build, and generates the prepared sdkbuild package.
RUN if [ -n "$PROJECT_NAME" ]; then \
      CGO_ENABLED=0 ./temporal-omes prepare-worker --language ts --project-name "$PROJECT_NAME" --dir-name "project-build-runner-$PROJECT_NAME" --version "$SDK_VERSION" ; \
    else \
      CGO_ENABLED=0 ./temporal-omes prepare-worker --language ts --dir-name prepared --version "$SDK_VERSION" ; \
    fi

# Copy the CLI and prepared feature to a "run" container.
FROM --platform=linux/$TARGETARCH node:20-bullseye-slim

ARG PROJECT_NAME=""

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/typescript /app/workers/typescript
COPY dockerfiles/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

ENV OMES_WORKER_LANGUAGE=typescript
ENV OMES_PREPARED_DIR=prepared
ENV OMES_PROJECT_NAME=$PROJECT_NAME
ENV OMES_PROJECT_PREPARED_DIR=project-build-runner-${PROJECT_NAME}
ENV OMES_PROJECT_PREBUILT_DIR=/app/workers/typescript/projects/tests/${PROJECT_NAME}/project-build-runner-${PROJECT_NAME}

ENTRYPOINT ["/app/entrypoint.sh"]
