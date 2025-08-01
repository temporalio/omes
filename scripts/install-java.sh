#!/bin/bash
set -euo pipefail

#
# Script to install Java using mise
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Installing Java..."

# shellcheck source=helpers/check-mise.sh
source "$SCRIPT_DIR/helpers/check-mise.sh"
# shellcheck source=helpers/load-versions.sh
source "$SCRIPT_DIR/helpers/load-versions.sh"

echo "Installing Java $JAVA_VERSION..."
mise use java@temurin-"$JAVA_VERSION"

echo "Installing Gradle $GRADLE_VERSION..."
mise use gradle@"$GRADLE_VERSION"

echo "✅ Java $JAVA_VERSION and Gradle $GRADLE_VERSION installed successfully!"