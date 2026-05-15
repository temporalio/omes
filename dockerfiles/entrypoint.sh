#!/bin/sh
set -eu

if [ "${1:-}" = "run-scenario" ]; then
  shift
  if [ -n "${OMES_PROJECT_NAME:-}" ] && [ -n "${OMES_PROJECT_PREBUILT_DIR:-}" ]; then
    exec /app/temporal-omes run-scenario "$@" \
      --option "language=${OMES_WORKER_LANGUAGE}" \
      --option "prebuilt-project-dir=${OMES_PROJECT_PREBUILT_DIR}"
  fi
  exec /app/temporal-omes run-scenario "$@"
fi

if [ "${1:-}" = "run-worker" ]; then
  shift
fi

if [ -n "${OMES_PROJECT_NAME:-}" ]; then
  exec /app/temporal-omes run-worker \
    --language "${OMES_WORKER_LANGUAGE}" \
    --project-name "${OMES_PROJECT_NAME}" \
    --dir-name "${OMES_PROJECT_PREPARED_DIR}" \
    "$@"
fi

exec /app/temporal-omes run-worker \
  --language "${OMES_WORKER_LANGUAGE}" \
  --dir-name "${OMES_PREPARED_DIR}" \
  "$@"
