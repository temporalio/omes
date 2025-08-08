#!/bin/bash
set -euo pipefail

#
# Script to install protobuf tools using mise
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Installing protobuf tools..."

# shellcheck source=helpers/check-mise.sh
source "$SCRIPT_DIR/helpers/check-mise.sh"
# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

echo "Installing Buf $BUF_VERSION..."
mise use buf@"$BUF_VERSION"

echo "Installing protoc $PROTOC_VERSION..."
mise use protoc@"$PROTOC_VERSION"

echo "Installing Go protobuf plugins..."
mise exec go@"$GO_VERSION" -- go install google.golang.org/protobuf/cmd/protoc-gen-go@"$PROTOC_GEN_GO_VERSION"

echo "✅ protobuf tools installed successfully!"
