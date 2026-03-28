# Build the CLI in a Go container
ARG TARGETARCH
FROM --platform=linux/$TARGETARCH golang:1.25 AS cli-build

WORKDIR /app
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY metrics ./metrics
COPY workers/*.go ./workers/
COPY workers/go/projects/api ./workers/go/projects/api
COPY go.mod go.sum ./
RUN CGO_ENABLED=0 go build -o temporal-omes ./cmd

# Build the .NET project
FROM --platform=linux/$TARGETARCH mcr.microsoft.com/dotnet/sdk:8.0-jammy AS build

WORKDIR /app
COPY --from=cli-build /app/temporal-omes ./temporal-omes
COPY workers/dotnet ./workers/dotnet
COPY workers/proto ./workers/proto

ARG PROJECT_DIR
RUN ./temporal-omes prepare-worker --language cs --dir-name project-prepared --project-dir "$PROJECT_DIR"

# Runtime container with ASP.NET for gRPC server support
FROM --platform=linux/$TARGETARCH mcr.microsoft.com/dotnet/aspnet:8.0-jammy

ENV OMES_PROJECT_LANGUAGE=cs
ENV OMES_PROJECT_BINARY=/app/prebuilt-project/build/program

COPY --from=cli-build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/dotnet/projects/tests/*/build /app/prebuilt-project/build
COPY --from=build /app/workers/dotnet/projects/tests/*/program.csproj /app/prebuilt-project/
COPY dockerfiles/project-entrypoint.sh /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
