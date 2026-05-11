#!/bin/sh
set -eu

if [ "${1:-}" = "run-scenario" ]; then
  shift
  if [ -n "${OMES_PROJECT_NAME:-}" ]; then
    project_prebuilt_dir="/app/workers/${OMES_WORKER_LANGUAGE}/projects/tests/${OMES_PROJECT_NAME}/${OMES_PREPARED_DIR}"
    set -- \
      --option "language=${OMES_WORKER_LANGUAGE}" \
      --option "prebuilt-project-dir=${project_prebuilt_dir}" \
      "$@"
  fi
  exec /app/temporal-omes run-scenario "$@"
fi

if [ "${1:-}" = "run-worker" ]; then
  shift
fi

set -- \
  --language "${OMES_WORKER_LANGUAGE}" \
  --dir-name "${OMES_PREPARED_DIR}" \
  "$@"

if [ -n "${OMES_PROJECT_NAME:-}" ]; then
  set -- --project-name "${OMES_PROJECT_NAME}" "$@"
fi

exec /app/temporal-omes run-worker "$@"
