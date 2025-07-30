#!/bin/bash
set -euo pipefail

#
# Script to install Rust using mise
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Installing Rust..."

# shellcheck source=helpers/check-mise.sh
source "$SCRIPT_DIR/helpers/check-mise.sh"
# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

echo "Installing Rust $RUST_TOOLCHAIN..."
mise use rust@"$RUST_TOOLCHAIN"

echo "✅ Rust $RUST_TOOLCHAIN installed successfully!"