#!/bin/bash
set -euo pipefail

#
# Script to lint Go code
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Linting Go..."

if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go first."
    exit 1
fi

echo "Go version: $(go version)"

cd "$REPO_ROOT/workers/go"

echo "Cleaning Go workspace..."
go clean

go mod tidy

echo "Applying Go format..."
go fmt ./...

echo "Running Go tests..."
go test -race ./...

echo "Building Go worker..."
go build ./...

echo "✅ Go linting completed successfully!"