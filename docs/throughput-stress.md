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

`include-standalone-activity-operator-commands` adds standalone-activity operator-command load,
rotating through pause, reset, and update-options across Continue-As-New. It is independently
selectable from plain standalone-activity load and turns on by itself when the namespace reports the
`StandaloneActivityOperatorCommands` capability. Passing
`--option include-standalone-activity-operator-commands=false` forces it off.

The workload currently sends the commands through WorkflowService RPCs in the Go worker while the
SDK command APIs are being added. Other language workers skip the action in the default non-strict
mode and report it as unsupported with `--worker-err-on-unimplemented`.

## Standalone activity batch operations

`include-standalone-activity-batch-operations` adds standalone-activity batch load, rotating through
cancel, terminate, and delete across Continue-As-New. It is independently selectable and turns on
by itself when the namespace reports the `StandaloneActivityBatchOperations` capability. Passing
`--option include-standalone-activity-batch-operations=false` forces it off.

Each batch targets five newly started, running standalone activities by default. Use
`--option standalone-activity-batch-size=<count>` to change the target count. Omes waits for the
batch job to complete, validates its aggregate counts, and verifies one target's resulting state.

Batch operations are an operator server API rather than an SDK feature. The Go worker sends
the WorkflowService RPCs directly; other language workers skip the action in the default non-strict
mode and report it as unsupported with `--worker-err-on-unimplemented`.

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
