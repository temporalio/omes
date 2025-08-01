#!/bin/bash
set -euo pipefail

#
# Script to lint Python code
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Linting Python..."

if ! command -v python3 &> /dev/null && ! command -v python &> /dev/null; then
    echo "Error: Python is not installed. Please install Python first."
    exit 1
fi

if ! command -v uv &> /dev/null; then
    echo "Error: uv is not installed. Please install uv first."
    exit 1
fi

if ! command -v poe &> /dev/null; then
    echo "Error: poe (poethepoet) is not installed. Please install it with 'uv tool install poethepoet'."
    exit 1
fi

echo "Python version: $(python3 --version 2>/dev/null || python --version)"
echo "uv version: $(uv --version)"

cd "$REPO_ROOT/workers/python"

echo "Checking Python dependencies..."
uv sync

echo "Applying Python format..."
poe format

echo "Running Python linting..."
poe lint

echo "✅ Python linting completed successfully!"