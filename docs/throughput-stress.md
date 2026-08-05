# The throughput_stress scenario

`throughput_stress` is the general-purpose load scenario: it drives workflows with child workflows,
activities, signals, queries, updates, continue-as-new, and optionally Nexus operations. It is the
scenario used for release validation and for standing load.

For the commands that run it, the built-in run flags, and how `--option` works in general, see
[Running omes](./running.md). This page covers the options specific to this scenario.

Run `omes list-scenarios` for the authoritative list of options with their types and defaults.

## Sleep activities

The scenario can run "sleep" activities with different configurations, described by a JSON document
passed with `--option sleep-activity-json`. Use `@<file>` to read it from a file:

```sh
echo '{"count":{"type":"fixed","value":5},"groups":{"high":{"weight":2,"sleepDuration":{"type":"uniform","min":"2s","max":"4s"}},"low":{"weight":3,"sleepDuration":{"type":"discrete","weights":{"5s":3,"10s":1}}}}}' > sleep.json

go run ./cmd/omes run-scenario-with-worker --scenario throughput_stress --language go \
  --option sleep-activity-json=@sleep.json --run-id my-run
```

That runs 5 sleep activities per iteration: `high` has weight 2 and sleeps a random 2–4s, `low` has
weight 3 and sleeps either 5s or 10s. See `DistributionField` for the kinds of distribution
available.

## Nexus

`nexus-enabled` decides whether the run generates Nexus load. Nexus needs an endpoint targeting the
run's task queue, and the scenario creates one for the run when `nexus-endpoint` is empty.

To use an endpoint you manage instead, create it against the run's task queue — which is
`omes-<run-id>` — and name it with `--option nexus-endpoint`:

```sh
temporal operator nexus endpoint create \
  --name my-nexus-endpoint \
  --target-namespace default \
  --target-task-queue omes-my-run

go run ./cmd/omes run-scenario-with-worker --scenario throughput_stress --language go \
  --option nexus-enabled=true --option nexus-endpoint=my-nexus-endpoint --run-id my-run
```

Naming an endpoint while Nexus is off is rejected, rather than accepting an endpoint the run would
never use.

## Standalone activities

The scenario can generate standalone-activity load — activities started outside any workflow context
via `StartActivityExecution`. This is a feature option, described under per-scenario options in
[Running omes](./running.md): it turns on by itself when the namespace reports support for
standalone activities (dynamic config `activity.enableStandalone`), and
`--option include-standalone-activity=false` forces it off. Requesting it against a namespace that
does not report support fails the run.

Implemented for the Go, Python, TypeScript, .NET, Java, and Ruby workers.

## Standalone Nexus operations

`include-standalone-nexus` is a feature option in the same way, but standalone operations are Nexus
operations, so they also need `nexus-enabled`. Asking for standalone Nexus while Nexus is off fails
the run.
