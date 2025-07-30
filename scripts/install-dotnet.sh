#!/bin/bash
set -euo pipefail

#
# Script to install .NET using mise
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Installing .NET..."

# shellcheck source=helpers/check-mise.sh
source "$SCRIPT_DIR/helpers/check-mise.sh"
# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

echo "Installing .NET $DOTNET_VERSION..."
mise use dotnet-core@"$DOTNET_VERSION"

echo "✅ .NET $DOTNET_VERSION installed successfully!"