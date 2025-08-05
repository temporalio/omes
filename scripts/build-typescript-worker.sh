#!/bin/bash
set -euo pipefail

#
# Script to build TypeScript worker
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

if ! command -v node &> /dev/null; then
    echo "Error: Node.js is not installed. Please install Node.js first."
    exit 1
fi

echo "Node.js version: $(node --version)"

"$SCRIPT_DIR/helpers/build-worker.sh" ts "$TYPESCRIPT_SDK_VERSION" "omes:typescript-latest"