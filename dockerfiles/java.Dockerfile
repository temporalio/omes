# Build in a full featured container
ARG TARGETARCH
FROM --platform=linux/$TARGETARCH eclipse-temurin:11-jammy AS build

# Install protobuf compiler and dependencies
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler=3.12.4* git=1:2.34.1-1ubuntu1

# Get go compiler
ARG TARGETARCH
RUN wget -q https://go.dev/dl/go1.21.12.linux-${TARGETARCH}.tar.gz \
    && tar -C /usr/local -xzf go1.21.12.linux-${TARGETARCH}.tar.gz

WORKDIR /app

# Copy CLI build dependencies
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY workers/*.go ./workers/
COPY go.mod go.sum ./

# Build the CLI
RUN CGO_ENABLED=0 /usr/local/go/bin/go build -o temporal-omes ./cmd

ARG SDK_VERSION

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Copy the worker files
COPY workers/java ./workers/java

# Download Gradle using wrapper to cache it in build layer
ENV GRADLE_USER_HOME="/gradle"
RUN /app/workers/java/gradlew --version

# Build the worker
WORKDIR /app
RUN CGO_ENABLED=0 ./temporal-omes prepare-worker --language java --dir-name prepared --version "$SDK_VERSION"

# Copy the CLI and prepared feature to a "run" container. Distroless isn't used here since we run
# through Gradle and it's more annoying than it's worth to get its deps to line up
FROM --platform=linux/$TARGETARCH eclipse-temurin:11
ENV GRADLE_USER_HOME="/gradle"

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/java /app/workers/java
COPY --from=build /gradle /gradle

# Use entrypoint instead of command to "bake" the default command options
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "java", "--dir-name", "prepared"]
