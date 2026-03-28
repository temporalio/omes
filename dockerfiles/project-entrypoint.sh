#!/bin/sh
case "$1" in
  worker|project-server)
    exec ${OMES_PROJECT_BINARY} "$@" ;;
  run-scenario)
    exec /app/temporal-omes "$@" \
      --option "language=${OMES_PROJECT_LANGUAGE}" \
      --option "prebuilt-project-dir=/app/prebuilt-project" ;;
  *)
    exec /app/temporal-omes "$@" ;;
esac
