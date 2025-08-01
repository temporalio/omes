#!/bin/bash
set -euo pipefail

#
# Script to build Python worker
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

if ! command -v python3 &> /dev/null && ! command -v python &> /dev/null; then
    echo "Error: Python is not installed. Please install Python first."
    exit 1
fi

echo "Python version: $(python3 --version 2>/dev/null || python --version)"

"$SCRIPT_DIR/helpers/build-worker.sh" python "$PYTHON_SDK_VERSION" "omes:python-latest"