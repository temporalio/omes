#!/bin/bash
set -euo pipefail

#
# Load tool versions from versions.env
#

repo_root="$(pwd)"
while [[ "$repo_root" != "/" ]]; do
    if [[ -f "$repo_root/versions.env" ]]; then
        break
    fi
    repo_root="$(dirname "$repo_root")"
done

versions_file="$repo_root/versions.env"

if [[ ! -f "$versions_file" ]]; then
    echo "Error: versions.env not found at $versions_file"
    exit 1
fi

# shellcheck source=/dev/null
source "$versions_file"

echo "Loaded versions from: $versions_file"
