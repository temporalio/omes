#!/bin/bash
set -euo pipefail

#
# Helper script to build worker with Docker
#

if [[ $# -lt 3 ]]; then
    echo "Usage: $0 <language> <sdk-version> <image-name>"
    echo ""
    echo "Examples:"
    echo "  $0 go v1.35.0 omes:go-latest"
    echo "  $0 python 1.15.0 omes:python-latest"
    echo ""
    exit 1
fi

LANGUAGE="$1"
SDK_VERSION="$2"
IMAGE_NAME="$3"
SCENARIO="workflow_with_single_noop_activity"
ITERATIONS="5"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

cd "$REPO_ROOT"

if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go first."
    exit 1
fi

echo "Building $LANGUAGE worker (SDK $SDK_VERSION)..."

echo "Building temporal-omes..."
go build -o temporal-omes ./cmd

RUN_ID="worker-test-$$"

echo "Running local scenario with worker..."
./temporal-omes run-scenario-with-worker \
    --scenario "$SCENARIO" \
    --log-level debug \
    --language "$LANGUAGE" \
    --embedded-server \
    --iterations "$ITERATIONS" \
    --version "$SDK_VERSION"

echo "Building worker image..."
./temporal-omes build-worker-image \
    --language "$LANGUAGE" \
    --version "$SDK_VERSION" \
    --tag-as-latest

echo "Starting worker image..."
CONTAINER_ID=$(docker run --rm --detach -i -p 10233:10233 \
    "$IMAGE_NAME" \
    --scenario "$SCENARIO" \
    --log-level debug \
    --language "$LANGUAGE" \
    --run-id "$RUN_ID" \
    --embedded-server-address 0.0.0.0:10233)
docker logs -f "$CONTAINER_ID" &
LOGS_PID=$!

cleanup() {
    if [[ -n "${LOGS_PID:-}" ]]; then
        echo "Stopping Docker logs..."
        kill "$LOGS_PID" 2>/dev/null || true
    fi
    if [[ -n "${CONTAINER_ID:-}" ]]; then
        echo "Stopping Docker container..."
        docker stop "$CONTAINER_ID" >/dev/null 2>&1 || true
    fi
}
trap cleanup EXIT

echo "Running scenario against worker image..."
./temporal-omes run-scenario \
    --scenario "$SCENARIO" \
    --log-level debug \
    --server-address 127.0.0.1:10233 \
    --run-id "$RUN_ID" \
    --connect-timeout 1m \
    --iterations "$ITERATIONS"

echo "✅ $LANGUAGE worker build completed successfully!"