#!/bin/bash
set -euo pipefail

#
# Script to install Go using mise
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Installing Go..."

# shellcheck source=helpers/check-mise.sh
source "$SCRIPT_DIR/helpers/check-mise.sh"
# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

echo "Installing Go $GO_VERSION..."
mise use go@"$GO_VERSION"

echo "✅ Go $GO_VERSION installed successfully!"