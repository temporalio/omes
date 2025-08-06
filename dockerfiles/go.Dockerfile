# Build in a full featured container
ARG TARGETARCH
FROM --platform=linux/$TARGETARCH golang:1.24 AS build

WORKDIR /app

# Copy CLI build dependencies
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY workers/*.go ./workers/
COPY go.mod go.sum ./

# Build the CLI
RUN CGO_ENABLED=0 go build -o temporal-omes ./cmd

ARG SDK_VERSION

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Copy the worker files
COPY workers/go ./workers/go

# Build the worker
RUN CGO_ENABLED=0 ./temporal-omes prepare-worker --language go --dir-name prepared --version "$SDK_VERSION"

# Copy the CLI and built worker to a distroless "run" container
FROM --platform=linux/$TARGETARCH gcr.io/distroless/static-debian11:nonroot

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/go/prepared /app/workers/go/prepared

# Put the language and dir, but let other options (like required scenario and run-id) be given by user
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "go", "--dir-name", "prepared"]
