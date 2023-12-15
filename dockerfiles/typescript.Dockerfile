# Build in a full featured container
FROM node:16 as build

RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler=3.12.4* git=1:2.34.1-1ubuntu1

# Get go compiler
ARG PLATFORM=amd64
RUN wget -q https://go.dev/dl/go1.20.4.linux-${PLATFORM}.tar.gz \
    && tar -C /usr/local -xzf go1.20.4.linux-${PLATFORM}.tar.gz

WORKDIR /app

# Copy CLI build dependencies
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY workers ./workers
COPY go.mod go.sum ./

# Build the CLI
RUN CGO_ENABLED=0 /usr/local/go/bin/go build -o temporal-omes ./cmd

ARG SDK_VERSION

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Build the worker
RUN CGO_ENABLED=0 ./temporal-omes prepare-worker --language ts --dir-name prepared --version "$SDK_VERSION"

# Copy the CLI and prepared feature to a "run" container. Distroless isn't used here since we run
# through Gradle and it's more annoying than it's worth to get its deps to line up
FROM gcr.io/distroless/nodejs:16

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/typescript /app/workers/typescript

# Node is installed here 👇 in distroless
ENV PATH="/nodejs/bin:$PATH"
# Use entrypoint instead of command to "bake" the default command options
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "typescript", "--dir-name", "prepared"]