#!/bin/bash
set -euo pipefail

#
# Script to install Python using mise
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Installing Python..."

# shellcheck source=helpers/check-mise.sh
source "$SCRIPT_DIR/helpers/check-mise.sh"
# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

echo "Installing Python $PYTHON_VERSION..."
mise use python@"$PYTHON_VERSION"

echo "Installing uv $UV_VERSION..."
mise use uv@"$UV_VERSION"

echo "Installing poethepoet..."
mise exec uv@"$UV_VERSION" -- uv tool install poethepoet

echo "✅ Python $PYTHON_VERSION, uv $UV_VERSION, and poethepoet installed successfully!"