# Build in a full featured container
# Use BUILDPLATFORM so the build stage runs natively (avoids QEMU networking issues
# with apt-get when cross-building for arm64 on an amd64 host).
FROM --platform=$BUILDPLATFORM eclipse-temurin:21-jammy AS build

# Install protobuf compiler and dependencies (runs on native build platform, no QEMU)
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler=3.12.4* git=1:2.34.1-1ubuntu1

# Get go compiler for the build platform
ARG BUILDARCH
ARG TARGETARCH
RUN wget -q https://go.dev/dl/go1.21.12.linux-${BUILDARCH}.tar.gz \
    && tar -C /usr/local -xzf go1.21.12.linux-${BUILDARCH}.tar.gz
ENV PATH="$PATH:/usr/local/go/bin"

WORKDIR /app

# Download Go dependencies first so this layer caches independently of source
# changes. The harness/api replace target must be present for `go mod download`
# to resolve the module graph.
COPY go.mod go.sum ./
COPY workers/go/harness/api ./workers/go/harness/api
RUN go mod download

# Copy CLI source and build the CLI.
COPY cmd ./cmd
COPY clioptions ./clioptions
COPY loadgen ./loadgen
COPY metrics ./metrics
COPY scenarios ./scenarios
COPY internal ./internal
# Build a native binary to use during Docker build (for the prepare-worker step below)
RUN CGO_ENABLED=0 go build -o temporal-omes-build ./cmd/omes
# Cross-compile the binary for the target platform (included in the final image)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o temporal-omes ./cmd/omes

ARG SDK_VERSION

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Copy the worker files
COPY workers/proto ./workers/proto
COPY workers/java ./workers/java

# Download Gradle using wrapper to cache it in build layer
ENV GRADLE_USER_HOME="/gradle"
RUN /app/workers/java/gradlew --version

# Build the worker using the native binary (avoids running a TARGETARCH binary under QEMU)
WORKDIR /app
RUN CGO_ENABLED=0 ./temporal-omes-build prepare-worker --language java --dir-name prepared --version "$SDK_VERSION"

# Copy the CLI and prepared feature to a "run" container. Use Alpine to avoid apt-get
# networking issues with ports.ubuntu.com in QEMU-emulated arm64 containers.
FROM --platform=linux/$TARGETARCH eclipse-temurin:21-alpine
RUN apk add --no-cache git
ENV GRADLE_USER_HOME="/gradle"

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/proto/harness /app/workers/proto/harness
COPY --from=build /app/workers/java /app/workers/java
COPY --from=build /app/repo /app/repo
COPY --from=build /gradle /gradle

# Use entrypoint instead of command to "bake" the default command options
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "java", "--dir-name", "prepared"]
