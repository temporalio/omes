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

Run configuration comes in **two separate channels**:

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

If you set neither `--iterations` nor `--duration`, the scenario's own default applies.

### 2. Per-scenario options (`--option key=value`)

Scenario-specific knobs are passed as repeated `--option key=value` pairs. The valid keys are defined by
each scenario (read the scenario source or its `list-scenarios` description). A value may be loaded from
a file with `@`: `--option sleep-activity-json=@sleep.json`.

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
- `--task-queue-suffix-index-start` / `--task-queue-suffix-index-end` run the worker across an inclusive
  range of task queues (`<task-queue>-<start>` … `<task-queue>-<end>`), for multi-task-queue scenarios.
- `--embedded-server` starts an embedded localhost server (cannot be combined with TLS or a non-default
  `--server-address`).

```sh
go run ./cmd/omes run-scenario-with-worker --scenario throughput_stress --language go \
    --run-id local-profile-test --worker-profile resource-based-default
```

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
- **Language names and aliases.** `--language` accepts `go`, `python` (`py`), `java`, `typescript`
  (`ts`), `dotnet` (`cs`), `ruby` (`rb`). If you pass an unknown value the error message lists the
  accepted set.
- **Option validation depends on the scenario declaring its options.** When a scenario declares them
  (most do), an unknown name or a malformed value is rejected before the run starts, and
  `list-scenarios` shows you what it accepts. A scenario that declares none accepts any option name and
  parses values on first read, so a typo there is silently ignored and the default is used.
- **`--iterations` and `--duration` are mutually exclusive.** Setting both is rejected at run time.
