#!/bin/bash
set -euo pipefail

#
# Master script to build all workers
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"$SCRIPT_DIR/helpers/run-scripts.sh" \
    "Building all workers" \
    build-go-worker.sh \
    build-java-worker.sh \
    build-python-worker.sh \
    build-typescript-worker.sh \
    build-dotnet-worker.sh