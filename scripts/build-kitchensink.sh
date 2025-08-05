#!/bin/bash
set -euo pipefail

#
# Script to build kitchen-sink proto
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

if ! command -v cargo &> /dev/null; then
    echo "Error: Rust/Cargo is not installed. Please install Rust first."
    exit 1
fi

if ! command -v node &> /dev/null; then
    echo "Error: Node.js is not installed. Please install Node.js first."
    exit 1
fi

echo "Cargo version: $(cargo --version)"
echo "Node.js version: $(node --version)"

echo "Building kitchen-sink proto..."

REPO_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Building kitchen-sink-gen..."
cd "$REPO_ROOT/loadgen/kitchen-sink-gen"
cargo build

echo "Generating TypeScript protos..."
cd "$REPO_ROOT/workers/typescript"
npm install
npm run proto-gen

echo "✅ Kitchen-sink proto build complete!"