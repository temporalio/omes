#!/bin/sh
set -eu

if [ "${1:-}" = "run-scenario" ]; then
  shift
  if [ -n "${OMES_PROJECT_NAME:-}" ]; then
    exec /app/temporal-omes run-scenario "$@" \
      --option "language=python" \
      --option "prebuilt-project-dir=/app/workers/python/projects/tests/${OMES_PROJECT_NAME}/project-build-runner-${OMES_PROJECT_NAME}"
  fi
  exec /app/temporal-omes run-scenario "$@"
fi

if [ "${1:-}" = "run-worker" ]; then
  shift
fi

if [ -n "${OMES_PROJECT_NAME:-}" ]; then
  exec /app/temporal-omes run-worker \
    --language python \
    --project-name "${OMES_PROJECT_NAME}" \
    --dir-name "project-build-runner-${OMES_PROJECT_NAME}" \
    "$@"
fi

exec /app/temporal-omes run-worker --language python --dir-name prepared "$@"
