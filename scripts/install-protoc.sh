#!/bin/bash
set -euo pipefail

#
# Script to install protoc using mise
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Installing protoc..."

# shellcheck source=helpers/check-mise.sh
source "$SCRIPT_DIR/helpers/check-mise.sh"
# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

echo "Installing protoc $PROTOC_VERSION..."
mise use protoc@"$PROTOC_VERSION"

echo "Installing protoc-gen-go $PROTOC_GEN_GO_VERSION..."
mise exec go@"$GO_VERSION" -- go install google.golang.org/protobuf/cmd/protoc-gen-go@"$PROTOC_GEN_GO_VERSION"

echo "✅ protoc $PROTOC_VERSION and protoc-gen-go $PROTOC_GEN_GO_VERSION installed successfully!"