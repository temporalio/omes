#!/bin/bash
set -euo pipefail

#
# Script to install Node.js using mise
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Installing Node.js..."

# shellcheck source=helpers/check-mise.sh
source "$SCRIPT_DIR/helpers/check-mise.sh"
# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

echo "Installing Node.js $NODE_VERSION..."
mise use node@"$NODE_VERSION"

echo "✅ Node.js $NODE_VERSION installed successfully!"