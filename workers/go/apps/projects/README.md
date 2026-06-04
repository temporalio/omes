# Go project apps

Future Go project harness apps should live under this directory, one package per project.

Each project app should expose a `harness.App` entrypoint and may import shared worker-side code from `workers/go/workerlib`.
