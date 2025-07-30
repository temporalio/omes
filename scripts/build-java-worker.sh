#!/bin/bash
set -euo pipefail

#
# Script to build Java worker
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

if ! command -v java &> /dev/null; then
    echo "Error: Java is not installed. Please install Java first."
    exit 1
fi

echo "Java version: $(java -version 2>&1 | head -n 1)"

"$SCRIPT_DIR/helpers/build-worker.sh" java "$JAVA_SDK_VERSION" "omes:java-latest"