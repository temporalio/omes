# Build in a full featured container
ARG TARGETARCH
FROM --platform=linux/$TARGETARCH ruby:3.3-bullseye AS build

# Get go compiler
ARG TARGETARCH
RUN wget -q https://go.dev/dl/go1.21.12.linux-${TARGETARCH}.tar.gz \
    && tar -C /usr/local -xzf go1.21.12.linux-${TARGETARCH}.tar.gz

WORKDIR /app

# Copy CLI build dependencies
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY metrics ./metrics
COPY workers/*.go ./workers/
COPY workers/go/projects/api ./workers/go/projects/api
COPY go.mod go.sum ./

# Build the CLI
RUN CGO_ENABLED=0 /usr/local/go/bin/go build -o temporal-omes ./cmd

ARG SDK_VERSION

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Copy the worker files
COPY workers/ruby ./workers/ruby

# Override BUNDLE_APP_CONFIG so bundler reads .bundle/config from the prepared dir
# (the Ruby Docker image sets this to /usr/local/bundle which is lost in multi-stage builds)
ENV BUNDLE_APP_CONFIG=.bundle

# Build the worker
RUN CGO_ENABLED=0 ./temporal-omes prepare-worker --language ruby --dir-name prepared --version "$SDK_VERSION"

# Copy the CLI and built worker to a slim "run" container
FROM --platform=linux/$TARGETARCH ruby:3.3-slim-bullseye

# Override BUNDLE_APP_CONFIG so bundler reads .bundle/config from the prepared dir
ENV BUNDLE_APP_CONFIG=.bundle

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/ruby /app/workers/ruby

# Put the language and dir, but let other options (like required scenario and run-id) be given by user
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "ruby", "--dir-name", "prepared"]
