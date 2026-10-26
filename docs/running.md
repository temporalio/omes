# Running omes

How to run omes and configure the load it generates. This is the reference for local runs (laptop or a
dev server) and applies equally whether you're self-hosting Temporal or running against a Temporal Cloud
namespace.

For authoring new scenarios, see the [README](../README.md#usage). For running against Temporal Cloud
cells inside Temporal's own infra (scaffold / VCR), see the internal runbooks.

## The commands

omes has three ways to run, plus cleanup. Each takes a **scenario** and a **run-id**.

> **`--run-id` is not a Temporal workflow Run ID.** A Temporal Run ID identifies a single workflow
> execution; omes's run-id is a label for the entire load run. omes uses it to name the task queue
> (`omes-<run-id>`) and to prefix the workflow IDs it starts
> (`w-<run-id>-<execution-id>-<iteration>`). Because the task queue is derived from it, a worker and the
> scenario driving it **must be given the same run-id**.

### Run a scenario with a worker (local all-in-one)

Easiest during development — starts the worker and the scenario together. Add `--embedded-server` to
also start an embedded Temporal server instead of connecting to one you're already running.

```sh
go run ./cmd/omes run-scenario-with-worker --scenario workflow_with_single_noop_activity --language go
```

### Run a worker on its own

```sh
go run ./cmd/omes run-worker --run-id local-test-run --language go
```

### Run a scenario against an already-running worker

```sh
go run ./cmd/omes run-scenario --scenario workflow_with_single_noop_activity --run-id local-test-run
```

### Clean up after a run

```sh
go run ./cmd/omes cleanup-scenario --scenario workflow_with_single_noop_activity --run-id local-test-run
```

Run any command with `--help` for the full, authoritative flag list.

## Configuring the load

Run configuration comes in **two separate channels**, split along one line:

> **Run flags shape the load. Scenario options decide what the load does.**
>
> How many iterations, how fast, how long, how many at once — that is the same question for every
> scenario, so it is a built-in flag. What each iteration actually executes is particular to one
> scenario, so it is an `--option`.

`list-scenarios` shows both for a given scenario: the run configuration it will use, with the flag that
overrides each value, and the options it accepts.

### 1. Built-in run flags (typed)

These apply to every scenario and override its defaults:

| Flag | Meaning |
| --- | --- |
| `--iterations` | Total iterations to run (mutually exclusive with `--duration`). |
| `--duration` | How long to keep starting new iterations (mutually exclusive with `--iterations`). |
| `--max-concurrent` | Max iterations running at once. |
| `--max-iterations-per-second` | Rate limit on starting iterations (0 = unlimited). |
| `--max-iteration-attempts` | Attempts per iteration (default 1). |
| `--timeout` | Hard stop; cancels in-flight iterations and exits non-zero. |

If you set neither `--iterations` nor `--duration`, the scenario's own default applies — and most
scenarios declare none, in which case omes's default does. `list-scenarios` states which is the case for
each scenario.

### 2. Per-scenario options (`--option key=value`)

Scenario-specific knobs are passed as repeated `--option key=value` pairs. Each scenario declares the
options it accepts; `list-scenarios` prints them with their types and defaults. A value may be loaded
from a file with `@`: `--option sleep-activity-json=@sleep.json`.

Some options are feature options: they turn on by default when the server reports support for the
underlying capability — some capabilities are reported per namespace, others by the server as a whole
— and off when it does not. Pass `=false` to force one off regardless; explicitly passing `=true`
where support is not reported fails the run. `list-scenarios` marks these with a dynamic default
instead of a fixed one.

```sh
go run ./cmd/omes run-scenario-with-worker --scenario throughput_stress --language go \
  --option nexus-endpoint=my-endpoint --run-id my-run
```

### One scenario per run

A single omes run executes exactly **one** scenario. To run two load shapes at the same time, start two
runs, each with its own `--run-id` (hence its own task queue). Composing load happens **within** a
scenario (its action tree / executor), not by combining scenarios on the command line.

## Worker configuration

`run-worker` (and `run-scenario-with-worker`) accept worker-tuning options:

- `--worker-profile <name>` selects a code-defined worker configuration profile in the language harness.
  It is forwarded to the worker via the `OMES_WORKER_PROFILE` env var (it is not a worker CLI flag), and
  when set it **takes precedence over** the individual tuning flags (poller counts, autoscale, slot
  counts, activity rate limits, versioning). Non-tuning flags (task queue, client connection, logging,
  metrics, `--worker-err-on-unimplemented`) still apply. Built-in profiles:
  - `resource-based-default` — the SDK resource-based worker tuner.
  - `throughput-stress-baseline` — a fixed config for `throughput_stress` runs (workflow cache 50, 8
    workflow-task slots, 32 activity/local-activity slots, 2 workflow-task pollers, 4 activity pollers).
  - `throughput-stress-baseline-x2`, `-x4`, `-x8`, `-x16`, `-x32`, `-x64` — `throughput-stress-baseline` with every
    limit multiplied by 2, 4, 8, 16, 32, or 64.
- `--task-queue-suffix-index-start` / `--task-queue-suffix-index-end` run the worker across an inclusive
  range of task queues (`<task-queue>-<start>` … `<task-queue>-<end>`), for multi-task-queue scenarios.
- `--embedded-server` starts an embedded localhost server (cannot be combined with TLS or a non-default
  `--server-address`).

```sh
go run ./cmd/omes run-scenario-with-worker --scenario throughput_stress --language go \
    --run-id local-profile-test --worker-profile resource-based-default
```

## Dynamic-partitioning load and drain tests

`phased_activity_load` produces an open-loop activity add rate independently of worker polling. It
starts standalone activities directly, so this scenario requires server dynamic config
`activity.enableStandalone=true`. Each `addRate` is the aggregate activities-per-second target for that
omes process; zero is valid. This is separate from `ebb_and_flow`, which remains a closed-loop controller
that adjusts workflow-batched production to follow a sinusoidal backlog target.

Create a plan such as `phase-plan.json`:

```json
{
  "phases": [
    {"name": "load", "duration": "70s", "addRate": 12000, "workerProfile": null},
    {"name": "drain", "duration": "7m", "addRate": 0, "workerProfile": "resource-based-default"}
  ]
}
```

The plan deliberately contains no start time, so it can be reused across runs. A `null`
`workerProfile` means no worker (`Pz`); a string names the OMES worker profile for that phase. Every
phase must specify both `addRate` and `workerProfile`.

Prepare the worker before scheduling the experiment so compilation cannot delay a phase transition:

```sh
go run ./cmd/omes prepare-worker --language go --dir-name dp-worker
```

Choose an RFC3339 timestamp far enough in the future for both processes to initialize. Then start the
generator and phased worker as separate processes using the same plan, start time, run ID, server, and
namespace. Both wait for `--start-at` and calculate all later boundaries from that absolute time:

```sh
go run ./cmd/omes run-scenario --scenario phased_activity_load --run-id dp-high \
  --start-at 2030-01-02T03:04:05Z \
  --option phase-plan-json=@phase-plan.json \
  --option start-concurrency=256 \
  --option visibility-verification-timeout=15m
```

```sh
go run ./cmd/omes run-phased-worker --run-id dp-high --language go --dir-name dp-worker \
  --start-at 2030-01-02T03:04:05Z \
  --phase-plan-file phase-plan.json \
  --worker-max-concurrent-activity-pollers=32 \
  --worker-max-concurrent-activities=1000
```

The load generator never waits for worker transitions. At a boundary the phased worker runner stops
the old worker when the profile changes, starts the new profile, and records scheduled, process-start,
and stop timestamps. Worker startup latency is therefore visible in the experiment rather than
shifting the load timeline. Consecutive phases with the same profile reuse the running worker.

Useful generator ranges for the production partition-scaler thresholds are:

| Experiment | Aggregate add rate | Expected use |
| --- | ---: | --- |
| Early scale-up | 1k–3k/s | Threshold crossing and initial partition growth. |
| Multi-step scale-up | 8k–16k/s | Roughly 10–20 partitions, depending on the active scaler window. |
| Maximum-scale qualification | 100k–160k/s | Approaching 200 partitions; use multiple qualified generators. |

Increase `start-concurrency` until achieved rate stops improving. If one process is insufficient, run
multiple generators against the same task queue and divide the desired aggregate rate among them.
Execution-specific activity IDs prevent collisions. Do not mix phased standalone generation with
workflow-batched generation in one measurement.

The useful sizing approximations are:

- To exceed a scaler target at `N` partitions, aggregate add rate must exceed `N × target rate`.
- With zero polling, `backlog = add rate × phase duration`.
- With active workers, backlog growth is approximately `(add rate − completion rate) × duration`.
- Drain time is approximately `backlog / (completion rate − continuing add rate)`.

Hold a downscale phase longer than the scaler's down window. Validate the offered load with server-side
`tasks_added` per partition; the scenario also reports requested and achieved start rates and fails a
sustained phase below `min-achieved-rate-ratio` (default 90% over phases of at least 10 seconds). Set the
ratio to zero to disable this check during generator qualification.

## Running a specific SDK version

`--version` accepts a released version (`v1.24.0`) or a local path to an SDK checkout — useful for
testing unreleased SDKs:

```sh
go run ./cmd/omes run-scenario-with-worker \
  --scenario workflow_with_single_noop_activity --language go --version /path/to/go-sdk
```

## Gotchas

- **A default run is quiet.** Per-iteration progress logs at debug level, so at the default `info` level
  you'll see a connect line, a long silence, then a completion line. Pass `--log-level debug` to watch
  iterations, and always sanity-check that work actually ran rather than trusting the absence of errors.
- **Not every worker can serve Nexus operations.** Scenarios with Nexus load (e.g. `throughput_stress`)
  enable it when the server supports Nexus, and route operations to the worker under test. The Rust
  worker can't handle them, so `run-scenario-with-worker --language rust` defaults
  `--option nexus-enabled=false`; with plain `run-scenario`, pass it yourself or point
  `--option nexus-endpoint` at an endpoint served by another worker.
- **Language names and aliases.** `--language` accepts `go`, `python` (`py`), `java`, `typescript`
  (`ts`), `dotnet` (`cs`), `ruby` (`rb`), `rust` (`rs`). If you pass an unknown value the error message lists the
  accepted set.
- **A scenario accepts exactly the options it declares.** An unknown name or a value of the wrong type
  is rejected before the run starts — before omes even connects, and before
  `run-scenario-with-worker` builds a worker — and every problem is reported at once.
  `list-scenarios` shows what each scenario accepts, including defaults. A scenario that declares no
  options accepts none.
- **Feature options are on by default against a capable namespace.** They resolve after omes connects,
  by probing the namespace's reported capabilities, so pass `=false` explicitly to force one off.
  Explicitly passing `=true` against a namespace that doesn't report the capability fails the run.
- **`--iterations` and `--duration` are mutually exclusive.** Setting both is rejected before the run
  starts.
- **Bad input prints as a message, not a stack trace.** An unknown scenario or option, a malformed
  `--option`, a feature the namespace doesn't support, or conflicting run flags print one line and exit
  non-zero. A stack trace means something failed *during* the run — so if you see one, read it as a real
  failure rather than a typo.
- **Some option names read like run flags but are not.** `throughput_stress` accepts
  `--option internal-iterations`, which sets how much work happens *inside* one iteration; `--iterations`
  sets how many iterations run. Setting the wrong one produces a valid run at the wrong load rather than
  an error, so check `list-scenarios` when a name looks familiar.
