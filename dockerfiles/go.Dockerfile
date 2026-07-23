# Build in a full featured container
ARG TARGETARCH
FROM --platform=linux/$TARGETARCH golang:1.26 AS build

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
RUN CGO_ENABLED=0 go build -o temporal-omes ./cmd/omes

ARG SDK_VERSION

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Copy the worker files
COPY workers/go ./workers/go

# The worker's `go mod tidy` (run by prepare-worker below) resolves the omes
# module's full test-import graph, which reaches devserver via
# loadgen.test -> internal/workertest -> devserver. devserver lives at the repo
# root rather than under internal/, so it isn't picked up by `COPY internal`.
COPY devserver ./devserver

# Build the worker
RUN CGO_ENABLED=0 ./temporal-omes prepare-worker --language go --dir-name prepared --version "$SDK_VERSION"

# Copy the CLI and built worker to a distroless "run" container
FROM --platform=linux/$TARGETARCH gcr.io/distroless/static-debian11:nonroot

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/go/prepared /app/workers/go/prepared

# Put the language and dir, but let other options (like required scenario and run-id) be given by user
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "go", "--dir-name", "prepared"]
