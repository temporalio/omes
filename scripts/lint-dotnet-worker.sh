#!/bin/bash
set -euo pipefail

#
# Script to lint .NET code
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Linting .NET..."

if ! command -v dotnet &> /dev/null; then
    echo "Error: .NET is not installed. Please install .NET first."
    exit 1
fi

echo ".NET version: $(dotnet --version)"

cd "$REPO_ROOT/workers/dotnet"

echo "Applying .NET format..."
dotnet format

echo "Verifying .NET format..."
dotnet format --verify-no-changes

echo "Building .NET worker..."
dotnet build --no-restore -c Library

echo "✅ .NET linting completed successfully!"