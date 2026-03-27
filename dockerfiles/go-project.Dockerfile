# Build in a full featured container
ARG TARGETARCH
FROM --platform=linux/$TARGETARCH golang:1.25 AS build

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
RUN CGO_ENABLED=0 go build -o temporal-omes ./cmd

# Copy the worker + project files
COPY workers/go ./workers/go
COPY workers/proto ./workers/proto

# Build the project
ARG PROJECT_DIR
RUN CGO_ENABLED=0 ./temporal-omes prepare-worker --language go --dir-name project-prepared --project-dir "$PROJECT_DIR"

# Runtime container
FROM --platform=linux/$TARGETARCH alpine:3

ENV OMES_PROJECT_LANGUAGE=go

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/go/projects/tests/project-build-*/. /app/prebuilt-project/
COPY dockerfiles/project-entrypoint.sh /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
