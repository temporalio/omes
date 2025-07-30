#!/bin/bash
set -euo pipefail

#
# Master script to lint all worker SDKs
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"$SCRIPT_DIR/helpers/run-scripts.sh" \
    "Linting all workers" \
    lint-go-worker.sh \
    lint-java-worker.sh \
    lint-python-worker.sh \
    lint-typescript-worker.sh \
    lint-dotnet-worker.sh