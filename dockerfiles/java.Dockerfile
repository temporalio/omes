# syntax=docker/dockerfile:1.7-labs
# Build in a full featured container
FROM eclipse-temurin:11-jammy AS build

ARG TARGETARCH

# Install protobuf compiler and dependencies
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler=3.12.4* git=1:2.34.1-1ubuntu1

# Get go compiler
RUN wget -q https://go.dev/dl/go1.21.12.linux-${TARGETARCH}.tar.gz \
    && tar -C /usr/local -xzf go1.21.12.linux-${TARGETARCH}.tar.gz

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    /usr/local/go/bin/go mod download

# Copy CLI build dependencies
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY workers/*.go ./workers/

# Build the CLI
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,id=go-build-${TARGETARCH},target=/root/.cache/go-build \
    CGO_ENABLED=0 /usr/local/go/bin/go build -o temporal-omes ./cmd

ARG SDK_VERSION

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Copy the worker files
COPY workers/java ./workers/java

# Download Gradle using wrapper and cache it
ENV GRADLE_USER_HOME="/gradle"
RUN --mount=type=cache,id=gradle-cache-${TARGETARCH},target=/gradle-cache \
    GRADLE_USER_HOME=/gradle-cache /app/workers/java/gradlew --version && \
    cp -r /gradle-cache /gradle

# Build the worker
WORKDIR /app
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,id=go-build-${TARGETARCH},target=/root/.cache/go-build \
    --mount=type=cache,id=gradle-cache-${TARGETARCH},target=/gradle-cache \
    GRADLE_USER_HOME=/gradle-cache CGO_ENABLED=0 ./temporal-omes prepare-worker --language java --dir-name prepared --version "$SDK_VERSION" && \
    cp -r /gradle-cache /gradle

# Copy the CLI and prepared feature to a "run" container. Distroless isn't used here since we run
# through Gradle and it's more annoying than it's worth to get its deps to line up
FROM --platform=linux/$TARGETARCH eclipse-temurin:11
ENV GRADLE_USER_HOME="/gradle"

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/java /app/workers/java
COPY --from=build /gradle /gradle

# Use entrypoint instead of command to "bake" the default command options
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "java", "--dir-name", "prepared"]
