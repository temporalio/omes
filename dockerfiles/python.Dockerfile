# Build in a full featured container
FROM python:3.11-bullseye as build

# Install protobuf compiler
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive \
    apt-get install --no-install-recommends --assume-yes \
      protobuf-compiler=3.12.4-1 libprotobuf-dev=3.12.4-1

# Get go compiler
ARG PLATFORM=amd64
RUN wget -q https://go.dev/dl/go1.20.4.linux-${PLATFORM}.tar.gz \
    && tar -C /usr/local -xzf go1.20.4.linux-${PLATFORM}.tar.gz
# Install Rust for compiling the core bridge - only required for installation from a repo but is cheap enough to install
# in the "build" container (-y is for non-interactive install)
# hadolint ignore=DL4006
RUN wget -q -O - https://sh.rustup.rs | sh -s -- -y

ENV PATH="$PATH:/root/.cargo/bin"

# Install poetry
RUN pip install --no-cache-dir "poetry==1.2.2"

WORKDIR /app

# Copy CLI build dependencies
COPY cmd ./cmd
COPY loadgen ./loadgen
COPY scenarios ./scenarios
COPY workers ./workers
COPY go.mod go.sum ./

# Build the CLI
RUN CGO_ENABLED=0 /usr/local/go/bin/go build -o temporal-omes ./cmd

# Optional SDK dir to copy, defaults to unimportant file
ARG SDK_DIR=.gitignore
COPY ${SDK_DIR} ./repo

# Build the worker
ENV POETRY_VIRTUALENVS_IN_PROJECT=true
RUN CGO_ENABLED=0 ./temporal-omes prepare-worker --language python --dir-name prepared --version "$SDK_VERSION"

# Copy the CLI and built worker to a distroless "run" container
FROM python:3.11-slim-bullseye

# Poetry needed for running python tests
RUN pip install --no-cache-dir "poetry==1.2.2"

COPY --from=build /app/temporal-omes /app/temporal-omes
COPY --from=build /app/workers/python /app/workers/python

# Put the language and dir, but let other options (like required scenario and run-id) be given by user
ENV POETRY_VIRTUALENVS_IN_PROJECT=true
ENTRYPOINT ["/app/temporal-omes", "run-worker", "--language", "python", "--dir-name", "prepared"]
