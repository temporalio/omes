# Workflow testing image for Go
# Build with: docker build -f dockerfiles/workflowtest-go.Dockerfile \
#   --build-arg TEST_PROJECT=workflowtests/go/tests/simple-test \
#   -t workflowtest-go:test .

ARG TARGETARCH
FROM --platform=linux/$TARGETARCH golang:1.25-bookworm AS build

# Install protobuf compiler
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler libprotobuf-dev \
 && rm -rf /var/lib/apt/lists/*

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

# Runtime stage - need Go for SDK building
FROM --platform=linux/$TARGETARCH golang:1.25-bookworm

COPY --from=build /app/omes /app/omes

WORKDIR /app

# Copy starter library (maintains relative path structure for replace directives)
COPY workflowtests/go/starter ./workflowtests/go/starter

# Copy test project (path like: workflowtests/go/tests/simpletest)
ARG TEST_PROJECT
COPY ${TEST_PROJECT} ./${TEST_PROJECT}

# Set the test project path for easy reference
ENV TEST_PROJECT_PATH=/app/${TEST_PROJECT}

# No fixed entrypoint - command provided at runtime
# Example orchestrator: /app/omes workflow --language go --project-dir $TEST_PROJECT_PATH ...
# Example worker: /app/omes exec --language go --project-dir $TEST_PROJECT_PATH --remote-worker 8081 ...
