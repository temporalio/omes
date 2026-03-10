#!/usr/bin/env bash
set -euo pipefail

# Usage: fuzz-nexus.sh [--heavy] [duration] [timeout]

HEAVY=false
POSITIONAL=()
for arg in "$@"; do
    case $arg in
        --heavy) HEAVY=true ;;
        *) POSITIONAL+=("$arg") ;;
    esac
done

if $HEAVY; then
    DURATION="${POSITIONAL[0]:-120s}" TIMEOUT="${POSITIONAL[1]:-600s}"
else
    DURATION="${POSITIONAL[0]:-30s}" TIMEOUT="${POSITIONAL[1]:-180s}"
fi

RUN_ID="nexus-fuzz-$$"
TASK_QUEUE="omes-$RUN_ID"
ENDPOINT="nexus-endpoint-$RUN_ID"
OMES_DIR="$(cd "$(dirname "$0")" && pwd)"
WORKER_DIR="$OMES_DIR/workers/python"
CONFIG_FILE=$(mktemp)
WORKER_PID="" SERVER_PID=""

cleanup() {
    set +e
    echo ""
    echo "--- Cleaning up ---"
    [[ -n "$WORKER_PID" ]] && kill "$WORKER_PID" 2>/dev/null && echo "Stopped worker"
    temporal operator nexus endpoint delete --address localhost:7233 --name "$ENDPOINT" -y 2>/dev/null && echo "Deleted endpoint"
    rm -f "$CONFIG_FILE"
    [[ -n "$SERVER_PID" ]] && kill "$SERVER_PID" 2>/dev/null && echo "Stopped dev server"
}
trap cleanup EXIT

# Start dev server if not running
if ! lsof -i :7233 &>/dev/null; then
    echo "Starting temporal dev server..."
    temporal server start-dev --headless &>/dev/null &
    SERVER_PID=$!
    sleep 4
    lsof -i :7233 &>/dev/null || { echo "ERROR: Failed to start dev server"; exit 1; }
fi

# Install python deps if needed
[[ -d "$WORKER_DIR/.venv" ]] || (cd "$WORKER_DIR" && uv sync --no-install-project)

# Create nexus endpoint
echo "Creating nexus endpoint: $ENDPOINT -> $TASK_QUEUE"
temporal operator nexus endpoint create \
    --address localhost:7233 \
    --name "$ENDPOINT" \
    --target-namespace default \
    --target-task-queue "$TASK_QUEUE"

# Write fuzzer config — heavy mode uses larger limits and higher nexus weight
if $HEAVY; then
    MAX_SETS=30 MAX_PER_SET=5 MAX_ACTIONS=5 MAX_PAYLOAD=128 MAX_INITIAL=5
    CHANCES='"timer":10,"activity":10,"child_workflow":8,"patch_marker":2,"set_workflow_state":2,"await_workflow_state":2,"upsert_memo":2,"upsert_search_attributes":2,"nested_action_set":7,"nexus_operation":55'
else
    MAX_SETS=10 MAX_PER_SET=3 MAX_ACTIONS=3 MAX_PAYLOAD=64 MAX_INITIAL=3
    CHANCES='"timer":15,"activity":15,"child_workflow":10,"patch_marker":2.5,"set_workflow_state":2.5,"await_workflow_state":2.5,"upsert_memo":2.5,"upsert_search_attributes":2.5,"nested_action_set":7.5,"nexus_operation":40'
fi
cat > "$CONFIG_FILE" <<EOF
{
  "max_client_action_sets": $MAX_SETS,
  "max_client_actions_per_set": $MAX_PER_SET,
  "max_actions_per_set": $MAX_ACTIONS,
  "max_timer": {"secs": 0, "nanos": 100000000},
  "max_payload_size": $MAX_PAYLOAD,
  "max_client_action_set_wait": {"secs": 0, "nanos": 100000000},
  "action_chances": {$CHANCES},
  "max_initial_actions": $MAX_INITIAL,
  "nexus_endpoint": "$ENDPOINT"
}
EOF

# Start python worker
echo "Starting python worker on $TASK_QUEUE..."
(cd "$WORKER_DIR" && .venv/bin/python main.py -q "$TASK_QUEUE" --log-level WARN) &
WORKER_PID=$!
sleep 2
kill -0 "$WORKER_PID" 2>/dev/null || { echo "ERROR: Worker failed to start"; exit 1; }

# Run fuzzer
MODE="normal"; $HEAVY && MODE="heavy"
echo "Running nexus fuzzer ($MODE) for $DURATION (timeout $TIMEOUT)..."
echo ""
cd "$OMES_DIR"
set +o pipefail
go run ./cmd run-scenario \
    --scenario fuzzer \
    --duration "$DURATION" \
    --run-id "$RUN_ID" \
    --timeout "$TIMEOUT" \
    --option "config=$CONFIG_FILE" \
    2>&1 | grep -v "^warning\|^  -->\|^   |\|^   =\|^    Compil\|^    Finish\|^     Running"
FUZZER_EXIT=${PIPESTATUS[0]}
set -o pipefail

echo ""
[[ "$FUZZER_EXIT" -ne 0 ]] && { echo "FAILED (exit code $FUZZER_EXIT)"; exit "$FUZZER_EXIT"; }
echo "Done."
