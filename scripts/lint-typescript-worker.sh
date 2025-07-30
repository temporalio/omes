#!/bin/bash
set -euo pipefail

#
# Script to lint TypeScript code
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Linting TypeScript..."

if ! command -v node &> /dev/null; then
    echo "Error: Node.js is not installed. Please install Node.js first."
    exit 1
fi

if ! command -v npm &> /dev/null; then
    echo "Error: npm is not installed. Please install npm first."
    exit 1
fi

echo "Node.js version: $(node --version)"
echo "npm version: $(npm --version)"

cd "$REPO_ROOT/workers/typescript"

echo "Cleaning TypeScript workspace..."
npm run clean

npm ci
npm run build

echo "Applying TypeScript format..."
npm run format

echo "Running TypeScript linting..."
npm run lint

echo "✅ TypeScript linting completed successfully!"