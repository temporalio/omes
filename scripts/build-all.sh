#!/bin/bash
set -euo pipefail

#
# Master script to build all workers and kitchen sink
# This is now consolidated - just runs the unified build-kitchensink.sh
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🏗️ Building all workers and kitchen sink..."
"$SCRIPT_DIR/build-kitchensink.sh"

echo "✅ All builds completed successfully!"