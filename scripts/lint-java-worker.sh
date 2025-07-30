#!/bin/bash
set -euo pipefail

#
# Script to lint Java code
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Linting Java..."

if ! command -v java &> /dev/null; then
    echo "Error: Java is not installed. Please install Java first."
    exit 1
fi

echo "Java version: $(java -version 2>&1 | head -n 1)"

cd "$REPO_ROOT/workers/java"

echo "Cleaning Java workspace..."
./gradlew --no-daemon clean

echo "Applying Java format..."
./gradlew --no-daemon spotlessApply

echo "Building Java worker..."
./gradlew --no-daemon build -x spotlessCheck

echo "✅ Java linting completed successfully!"