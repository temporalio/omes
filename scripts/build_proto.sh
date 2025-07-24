#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
IMAGE_NAME="omes-ks-gen"

# Always load versions from versions.env
set -a && source "${PROJECT_ROOT}/versions.env" && set +a

echo "Building Docker image for kitchen-sink-gen..."
docker build -f "${PROJECT_ROOT}/dockerfiles/proto.Dockerfile" \
  --build-arg RUST_TOOLCHAIN="${RUST_TOOLCHAIN}" \
  --build-arg GO_VERSION="${GO_VERSION}" \
  --build-arg PROTOC_VERSION="${PROTOC_VERSION}" \
  --build-arg PROTOC_GEN_GO_VERSION="${PROTOC_GEN_GO_VERSION}" \
  --build-arg NODE_VERSION="${NODE_VERSION}" \
  -t "${IMAGE_NAME}" "${PROJECT_ROOT}"

echo "Running kitchen-sink-gen build..."
docker run --rm -v "${PROJECT_ROOT}:/workspace" -w /workspace "${IMAGE_NAME}" bash -c "
    set -e

    cd /workspace/loadgen/kitchen-sink-gen
    cargo build

    cd /workspace/workers/typescript
    npm install && npm run proto-gen
"

echo "✅ kitchen-sink-gen build complete!"