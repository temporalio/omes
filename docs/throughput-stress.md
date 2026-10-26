# The throughput_stress scenario

`throughput_stress` is the general-purpose load scenario: it drives workflows with child workflows,
activities, signals, queries, updates, continue-as-new, and optionally Nexus operations. It is the
scenario used for release validation and for standing load.

For the commands that run it, the built-in run flags, and how `--option` works in general, see
[Running omes](./running.md). This page covers the options specific to this scenario.

Run `omes list-scenarios` for the authoritative list of options with their types and defaults.

## Retry and heartbeat timing

`include-retry-scenarios=true` adds probes that deliberately fail their first
activity attempt and must succeed on their second. `retry-scenario-timeout`
(default `1s`) sets the retryable-error and timeout probes' start-to-close
timeouts, and the heartbeat probe's heartbeat timeout. The deliberate
start-to-close failure runs for twice that duration;
the heartbeat probe's start-to-close timeout stays longer than its heartbeat
timeout. Changing the option preserves both attempts and the expected first
failure. It does not affect the separately configured healthy heartbeat actions.

Healthy heartbeat activities retain a single attempt. Their local heartbeat
interval does not determine RPC cadence: the Go SDK batches calls. The Go
worker's `--worker-max-heartbeat-throttle-interval` can cap that batching window
without changing the activity's heartbeat timeout or retry policy. Zero keeps
the SDK default. Worker profiles take precedence over this individual tuning
flag, as they do for poller and slot settings.

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

`nexus-enabled` decides whether the run generates Nexus load. It is a feature option gated on the
server supporting Nexus, and **on by default wherever it does** — omes is a testing tool, so the
useful default is to exercise what the server can do. Against a server without Nexus it turns itself
off, so the default is safe anywhere; asking for Nexus explicitly there fails instead, since the run
cannot do what was asked.

Nexus needs an endpoint targeting the run's task queue. The scenario creates one for the run when
`nexus-endpoint` is empty, and logs that it did.

To use an endpoint you manage instead, create it against the run's task queue — which is
`omes-<run-id>` — and name it with `--option nexus-endpoint`:

```sh
temporal operator nexus endpoint create \
  --name my-nexus-endpoint \
  --target-namespace default \
  --target-task-queue omes-my-run

go run ./cmd/omes run-scenario-with-worker --scenario throughput_stress --language go \
  --option nexus-endpoint=my-nexus-endpoint --run-id my-run
```

To generate no Nexus load at all, pass `--option nexus-enabled=false`. Naming an endpoint while Nexus
is off is rejected, rather than accepting an endpoint the run would never use.

## Standalone activities

The scenario can generate standalone-activity load — activities started outside any workflow context
via `StartActivityExecution`. This is a feature option, described under per-scenario options in
[Running omes](./running.md): it turns on by itself when the namespace reports support for
standalone activities (dynamic config `activity.enableStandalone`), and
`--option include-standalone-activity=false` forces it off. Requesting it against a namespace that
does not report support fails the run.

Implemented for the Go, Python, TypeScript, .NET, Java, and Ruby workers.

## Standalone activity operator commands

`include-standalone-activity-operator-commands` enables an independent driver loop that rotates through
pause/unpause, reset, and update. It is enabled by default when the namespace reports the
`StandaloneActivityOperatorCommands` capability; pass
`--option include-standalone-activity-operator-commands=false` to disable it.

Commands run directly in the Go driver, while heartbeat activities run on the selected worker.
Baseline workflow iterations do not wait for these sequences. Operator rate and concurrency are
controlled separately; see [running an independent load](./running.md#running-an-independent-load) for
configuration and shutdown behavior.

## Standalone Nexus operations

`include-standalone-nexus` is a feature option in the same way, but it only adds to a run that is
already generating Nexus load. `nexus-enabled` governs that, and a capability probe never turns it
on: the probe decides whether standalone operations are *part of* the Nexus load, not whether there
is any.

| `nexus-enabled` | Standalone Nexus reported | Result |
| --- | --- | --- |
| on (default) | yes | Nexus load including standalone operations |
| on (default) | no | Nexus load without standalone operations |
| `=false` | either | no Nexus load; standalone Nexus not included |

Asking for `include-standalone-nexus=true` while Nexus is off is a contradiction, and fails the run.

## Nexus operation actions

### `include-nexus-workflow-actions`

The workflow actions start their target workflow with signal-with-start, update it, then send an
ordinary signal that completes it. This requires Nexus update callback support.

### `include-nexus-standalone-activity`

The standalone activity action is driven two ways each iteration: as an in-workflow Nexus operation
and, when standalone Nexus is part of the run, as a standalone Nexus operation. This requires
standalone activity and callback support.
