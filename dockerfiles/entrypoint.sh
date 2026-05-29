#!/bin/sh
set -eu

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
