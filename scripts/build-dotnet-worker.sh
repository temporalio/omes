#!/bin/bash
set -euo pipefail

#
# Script to build .NET worker
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

if ! command -v dotnet &> /dev/null; then
    echo "Error: .NET is not installed. Please install .NET first."
    exit 1
fi

echo ".NET version: $(dotnet --version)"

"$SCRIPT_DIR/helpers/build-worker.sh" cs "$DOTNET_SDK_VERSION" "omes:dotnet-latest"