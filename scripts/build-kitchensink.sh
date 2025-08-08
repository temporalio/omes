#!/bin/bash
set -euo pipefail

# This script builds the Kitchen Sink proto source files.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
WORKERS_PROTO_DIR="$PROJECT_ROOT/workers/proto"
VERSION_FILE="$WORKERS_PROTO_DIR/.temporal_api_version"

# Load version from versions.env
if [ -f "$PROJECT_ROOT/versions.env" ]; then
    source "$PROJECT_ROOT/versions.env"
fi

echo "Building Kitchen Sink with Temporal API ${TEMPORAL_API_VERSION}..."

if [ ! -f "$VERSION_FILE" ] || [ "$(cat "$VERSION_FILE" 2>/dev/null)" != "$TEMPORAL_API_VERSION" ]; then
    echo "Downloading Temporal API ${TEMPORAL_API_VERSION} proto files..."
    buf export buf.build/temporalio/api:${TEMPORAL_API_VERSION} --output "$WORKERS_PROTO_DIR"
    echo "$TEMPORAL_API_VERSION" > "$VERSION_FILE"
fi

echo "Generating proto source files..."
cd "$PROJECT_ROOT"

buf generate --template buf.gen.yaml "$WORKERS_PROTO_DIR"

echo "✅ Kitchen-sink proto build complete!"
