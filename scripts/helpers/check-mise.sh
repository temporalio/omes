#!/bin/bash
set -euo pipefail

#
# Helper script to check if mise is installed
#

if ! command -v mise &> /dev/null; then
    echo "Error: mise is not installed. Please install mise first:"
    echo "  https://mise.jdx.dev/getting-started.html"
    exit 1
fi