#!/bin/bash
set -euo pipefail

#
# Helper script to run multiple scripts and track results
#

if [[ $# -lt 2 ]]; then
    echo "Usage: $0 <description> <script1> [script2] [script3] ..."
    echo ""
    echo "Examples:"
    echo "  $0 \"Linting all workers\" lint-go-worker.sh lint-java-worker.sh"
    echo "  $0 \"Testing all workers\" test-go-worker.sh test-java-worker.sh"
    exit 1
fi

DESCRIPTION="$1"
shift

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PARENT_SCRIPT_DIR="$(dirname "$SCRIPT_DIR")"

echo "$DESCRIPTION..."

FAILED_SCRIPTS=()

for script in "$@"; do
    echo ""
    echo "===========================================" 
    echo "Running $script"
    echo "==========================================="
    
    if [[ -x "$PARENT_SCRIPT_DIR/$script" ]]; then
        if ! "$PARENT_SCRIPT_DIR/$script"; then
            echo "❌ $script failed"
            FAILED_SCRIPTS+=("$script")
        fi
    else
        echo "Error: Script $script not found or not executable"
        FAILED_SCRIPTS+=("$script")
    fi
done

echo ""
if [[ ${#FAILED_SCRIPTS[@]} -eq 0 ]]; then
    echo "✅ All scripts completed successfully!"
    exit 0
else
    echo "❌ ${#FAILED_SCRIPTS[@]} script(s) failed:"
    for failed_script in "${FAILED_SCRIPTS[@]}"; do
        echo "  - $failed_script"
    done
    exit 1
fi