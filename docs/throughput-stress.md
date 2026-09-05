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

The following opt-in options exercise actions from the Nexus handler:

- `include-nexus-standalone-activity`
- `include-nexus-signal`
- `include-nexus-signal-with-start`
- `include-nexus-update`

The standalone activity action is driven two ways each iteration: as an in-workflow Nexus operation
and, when standalone Nexus is part of the run, as a standalone Nexus operation. It also needs server
support for standalone activities and activity completion callbacks (dynamic config
`activity.enableStandalone` and `activity.enableCallbacks`) and a Nexus callback URL; if those are
off, the operation fails clearly rather than being skipped.

The workflow actions use one target kitchenSink workflow per iteration for all selected actions.
With signal-with-start enabled, exactly one signal-with-start request is made: it either creates the
target or messages a target created by a regular Nexus workflow start.

All four options are off by default, require `nexus-enabled`, and are currently supported by Go
workers.
