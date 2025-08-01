#!/bin/bash
set -euo pipefail

#
# Script to build Go worker
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go first."
    exit 1
fi

echo "Go version: $(go version)"

"$SCRIPT_DIR/helpers/build-worker.sh" go v"$GO_SDK_VERSION" "omes:go-latest"