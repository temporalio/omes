#!/bin/sh
set -eu

# This image normally behaves like the other worker images: callers pass worker
# flags and we add run-worker plus the prepared Python package defaults. Explicit
# omes subcommands pass through so the same image can run project scenarios.
case "${1:-}" in
  run-scenario|cleanup-scenario|list-scenarios|prepare-worker|run-scenario-with-worker|completion|help)
    exec /app/temporal-omes "$@"
    ;;
  run-worker)
    shift
    ;;
esac

exec /app/temporal-omes run-worker \
  --language "${OMES_WORKER_LANGUAGE}" \
  --dir-name "${OMES_PREPARED_DIR}" \
  "$@"
