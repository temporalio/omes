# Project test base image.
# This image contains omes CLI + Go toolchain + harness library only.
# Test projects are copied by a thin overlay image (see projecttest-go.Dockerfile).
#
# Build with:
#   docker build -f projecttests/dockerfiles/projecttest-go-base.Dockerfile -t omes-projecttest-go-base:latest .

ARG TARGETARCH
FROM --platform=linux/$TARGETARCH golang:1.25-bookworm AS build

# Install protobuf compiler
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler libprotobuf-dev \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

WORKDIR /app

# Copy CLI build dependencies
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY metrics ./metrics
COPY workers/*.go ./workers/
COPY internal ./internal
COPY projecttests/go/harness ./projecttests/go/harness
COPY go.mod go.sum ./

# Build the CLI
RUN CGO_ENABLED=0 go build -o omes ./cmd

# Runtime stage - need Go for SDK building
FROM --platform=linux/$TARGETARCH golang:1.25-bookworm

COPY --from=build /app/omes /app/omes

WORKDIR /app

# Copy harness library (maintains relative path structure for replace directives)
COPY projecttests/go/harness ./projecttests/go/harness

# Use omes as entrypoint; role and project are provided at runtime.
ENTRYPOINT ["/app/omes"]

# Example runner command:
#   project --language go --project-dir /app/projecttests/go/tests/helloworld ...
#
# Example worker command:
#   exec --language go --project-dir /app/projecttests/go/tests/helloworld -- worker ...
