#!/usr/bin/env bash
set -euo pipefail

# Usage: fuzz-nexus.sh [duration] [timeout] [--heavy]
# Default: 30s duration, 180s timeout
# --heavy: 120s duration, 600s timeout, more action sets and concurrency

HEAVY=false
POSITIONAL=()
for arg in "$@"; do
    case $arg in
        --heavy) HEAVY=true ;;
        *) POSITIONAL+=("$arg") ;;
    esac
done

if $HEAVY; then
    DURATION="${POSITIONAL[0]:-120s}"
    TIMEOUT="${POSITIONAL[1]:-600s}"
else
    DURATION="${POSITIONAL[0]:-30s}"
    TIMEOUT="${POSITIONAL[1]:-180s}"
fi

RUN_ID="nexus-fuzz-$$"
TASK_QUEUE="omes-$RUN_ID"
ENDPOINT="nexus-endpoint-$RUN_ID"
OMES_DIR="$(cd "$(dirname "$0")" && pwd)"
WORKER_DIR="$OMES_DIR/workers/python"
CONFIG_FILE=$(mktemp)
WORKER_PID=""
SERVER_PID=""

cleanup() {
    set +e
    echo ""
    echo "--- Cleaning up ---"
    if [ -n "$WORKER_PID" ]; then kill "$WORKER_PID" 2>/dev/null && echo "Stopped worker (pid $WORKER_PID)"; fi
    temporal operator nexus endpoint delete --address localhost:7233 --name "$ENDPOINT" -y 2>/dev/null && echo "Deleted endpoint $ENDPOINT"
    rm -f "$CONFIG_FILE"
    if [ -n "$SERVER_PID" ]; then kill "$SERVER_PID" 2>/dev/null && echo "Stopped dev server"; fi
}
trap cleanup EXIT

# Start dev server if not running
if ! lsof -i :7233 &>/dev/null; then
    echo "Starting temporal dev server..."
    temporal server start-dev --headless &>/dev/null &
    SERVER_PID=$!
    sleep 4
    if ! lsof -i :7233 &>/dev/null; then
        echo "ERROR: Failed to start dev server"
        exit 1
    fi
fi

# Install python deps if needed
if [ ! -d "$WORKER_DIR/.venv" ]; then
    echo "Installing python dependencies..."
    (cd "$WORKER_DIR" && uv sync --no-install-project)
fi

# Create nexus endpoint
echo "Creating nexus endpoint: $ENDPOINT -> $TASK_QUEUE"
temporal operator nexus endpoint create \
    --address localhost:7233 \
    --name "$ENDPOINT" \
    --target-namespace default \
    --target-task-queue "$TASK_QUEUE"

# Write fuzzer config
if $HEAVY; then
    cat > "$CONFIG_FILE" <<EOF
{
  "max_client_action_sets": 30,
  "max_client_actions_per_set": 5,
  "max_actions_per_set": 5,
  "max_timer": {"secs": 0, "nanos": 100000000},
  "max_payload_size": 128,
  "max_client_action_set_wait": {"secs": 0, "nanos": 100000000},
  "action_chances": {
    "timer": 10.0,
    "activity": 10.0,
    "child_workflow": 8.0,
    "patch_marker": 2.0,
    "set_workflow_state": 2.0,
    "await_workflow_state": 2.0,
    "upsert_memo": 2.0,
    "upsert_search_attributes": 2.0,
    "nested_action_set": 7.0,
    "nexus_operation": 55.0
  },
  "max_initial_actions": 5,
  "nexus_endpoint": "$ENDPOINT"
}
EOF
else
    cat > "$CONFIG_FILE" <<EOF
{
  "max_client_action_sets": 10,
  "max_client_actions_per_set": 3,
  "max_actions_per_set": 3,
  "max_timer": {"secs": 0, "nanos": 100000000},
  "max_payload_size": 64,
  "max_client_action_set_wait": {"secs": 0, "nanos": 100000000},
  "action_chances": {
    "timer": 15.0,
    "activity": 15.0,
    "child_workflow": 10.0,
    "patch_marker": 2.5,
    "set_workflow_state": 2.5,
    "await_workflow_state": 2.5,
    "upsert_memo": 2.5,
    "upsert_search_attributes": 2.5,
    "nested_action_set": 7.5,
    "nexus_operation": 40.0
  },
  "max_initial_actions": 3,
  "nexus_endpoint": "$ENDPOINT"
}
EOF
fi

# Start python worker
echo "Starting python worker on $TASK_QUEUE..."
(cd "$WORKER_DIR" && .venv/bin/python main.py -q "$TASK_QUEUE" --log-level WARN) &
WORKER_PID=$!
sleep 2

if ! kill -0 "$WORKER_PID" 2>/dev/null; then
    echo "ERROR: Worker failed to start"
    exit 1
fi

# Run fuzzer
MODE="normal"
$HEAVY && MODE="heavy"
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
if [ "$FUZZER_EXIT" -ne 0 ]; then
    echo "FAILED (exit code $FUZZER_EXIT)"
    exit "$FUZZER_EXIT"
fi
echo "Done."
