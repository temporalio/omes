#!/bin/bash
set -euo pipefail

#
# Master script to install all tools using mise
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"$SCRIPT_DIR/helpers/run-scripts.sh" \
    "Installing all tools" \
    install-dotnet.sh \
    install-go.sh \
    install-java.sh \
    install-python.sh \
    install-node.sh \
    install-rust.sh \
    install-proto.sh