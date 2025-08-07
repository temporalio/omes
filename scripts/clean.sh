#!/bin/bash
set -euo pipefail

#
# Script to clean up temporary directories and build artifacts
#

# Get the root directory (parent of scripts)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "Removing omes-temp-* directories..."
find "$ROOT_DIR/workers" -name "omes-temp-*" -type d -exec rm -rf {} + 2>/dev/null || true

echo "Cleaning Go..."
cd "$ROOT_DIR/workers/go"
go clean 2>/dev/null || true

echo "Cleaning Java..."
cd "$ROOT_DIR/workers/java"
./gradlew clean 2>/dev/null || true

echo "Cleaning Python..."
cd "$ROOT_DIR/workers/python"
uv clean 2>/dev/null || true

echo "Cleaning TypeScript..."
cd "$ROOT_DIR/workers/typescript"
npm run clean 2>/dev/null || true

echo "Cleaning .NET..."
cd "$ROOT_DIR/workers/dotnet"
dotnet clean 2>/dev/null || true

echo "✅ Clean completed"
