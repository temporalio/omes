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

### Load patterns

Start a Go worker and send a repeating spike of iterations to a local embedded server:

```sh
go run ./cmd/omes run-scenario-with-worker \
  --scenario workflow_with_single_noop_activity --language go --embedded-server --run-id load-demo \
  --load spike-train peak=120/s every=5m --duration 12m --max-concurrent 240
```

This starts iterations at 48/s (the default base is `0.4` of peak), then at 120/s
for one minute starting at minutes five and ten. After twelve minutes it stops
starting iterations and waits for those already running to finish. Rates count
scenario iterations, which can each start multiple workflows or operations.
`run-scenario` accepts the same load flags when you already have a worker running.

At startup, the `Load pattern` log prints the canonical specification with every
default filled in, followed by the compiled phases. For this command, its fields
read as follows (formatting abbreviated):

```text
pattern: spike-train,peak=120/s,base=0.4,every=5m,for=1m
peak: 120
duration: 12m
phases:
  base [0s,5m0s) 48→48 iterations/s
  spike [5m0s,6m0s) 120→120 iterations/s
  base [6m0s,10m0s) 48→48 iterations/s
  spike [10m0s,11m0s) 120→120 iterations/s
  base [11m0s,12m0s) 48→48 iterations/s
```

Each interval includes its start and excludes its end. A `Load phase` log marks
each transition and its requested rate. Check these fields before interpreting
results: they show exactly which shape and timings the run uses.

To change the shape, replace `spike-train` and its parameters in the command above:

| Pattern | Shape | Parameters and defaults | Phases |
| --- | --- | --- | --- |
| `steady` | `████████` | `peak` required | `hold` at peak for the entire run. |
| `ramp-hold-ramp` | `▁▃▆████▆▃▁` | `peak` required, `up=0.1`, `down=0.1` | `up` from zero to peak, `hold` at peak, `down` to zero. |
| `sweep` | `▂▂▄▄▆▆██` | `peak` required, `from=0.25`, `to=1`, `steps=4` | Equal-duration `step-1` through `step-n`, including both endpoint levels. Ascending and descending sweeps are supported. |
| `spike-train` | `▃▃▃█▃▃█▃` | `peak` required, `base=0.4`, `every=10m`, `for=1m` | `base` until the first spike at `every`, then `spike` at peak for `for`; subsequent spike starts are `every` apart. The final phase is truncated to the run duration. |

Tune levels with fractions of peak. For example, `--load sweep peak=120/s
from=0.25 to=1 steps=4` requests 30, 60, 90, then 120 iterations/s. To descend,
swap `from` and `to`. `base=0` pauses starts between spikes while the run continues.

Tune timings with fractions of the **total run duration** or explicit Go durations.
For a twenty-minute run, `--load ramp-hold-ramp peak=120/s up=0.2 down=2m`
ramps up over four minutes, holds for fourteen, then ramps down for two.
`--load spike-train peak=120/s every=0.25 for=1m` starts a one-minute spike
every five minutes in that same run. Use `0.1` for one tenth; percentages such as
`10%` are rejected. `peak=120` and `peak=120/s` both mean 120 iterations/s.

Keep parameters immediately after the pattern: `--load spike-train peak=120/s
every=5m --duration 20m`. A flag, `--`, or a plain word ends the parameter group;
a later positional `key=value` is rejected with a corrected example.
A quoted value (`--load 'spike-train peak=120/s every=5m'`) works too.
The comma form, `--load=spike-train,peak=120/s,every=5m`, is useful for
CI, YAML, or Kubernetes args where the entire flag value can occupy one argument.

Under backpressure, the shape keeps following elapsed time. `--max-concurrent`
caps iterations in flight; a start blocked by that cap is dropped. Starts missed
because the scheduling loop is late are also dropped. **Dropped starts are never
caught up**, so achieved throughput can be lower than requested. Starts are spaced
using the integral of the requested rate; the first steady start is after one
rate interval, and no start occurs at or after the run duration.

A per-phase warning reports the number dropped for each cause. A warning saying
`raise --max-concurrent or lower peak= in --load` means the concurrency cap blocked
starts. A warning saying `scheduling loop was late; lower peak= in --load` means
the generator missed scheduled slots; check generator capacity and lower the rate.
If scheduling stops early because of failure, cancellation, or an explicit stop,
drop counts cover only the time scheduling actually ran.

Size `--max-concurrent` from peak rate × iteration latency, with headroom. At
120 iterations/s and a two-second iteration latency, 240 slots cover the expected
in-flight work; a cap of 300 gives room for variation. Measure latency for your
scenario and account for retries and slower iterations. At startup, omes warns
once if the cap is below peak × **an assumed one-second iteration latency**.
This is a simple sizing hint, not a measurement or an error; short iterations may
need fewer slots, and slow iterations may need many more.

Load pattern reference:

- `--load` requires a positive `--duration`; scenario default iteration limits do
  not apply. `--iterations` and `--max-iterations-per-second` are rejected even
  when explicitly zero. Set `peak=` to control the rate.
- `peak` is required, positive, finite, and at most 10,000 iterations/s. This
  ceiling bounds scheduling overhead from accidental rates.
- Levels (`from`, `to`, `base`) are fractions from zero to one. Timings (`up`,
  `down`, `every`, `for`) accept a fraction from zero to one or a Go duration.
  `up + down` must fit the run; zero-duration ramp or hold phases are omitted.
  `every` and `for` must be positive, with `for <= every`.
- `steps` must be an integer from 2 to 1024, with at least 1ns per step. Every
  pattern is limited to 1024 phases. Unknown, duplicate, missing, or malformed
  parameters and empty comma-separated segments are rejected.
- Executors must support patterns through `GenericExecutor`. Scenarios with
  their own scheduling, such as `ebb_and_flow` and `state_transitions_steady`,
  reject `--load`.

Without `--load`, `--max-iterations-per-second` keeps its ticker behavior: it waits
for a free concurrency slot and buffers at most one tick.

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

A single omes run executes exactly **one** scenario, which can compose multiple load shapes.
To run different scenarios at the same time, start separate runs, each with its own `--run-id`
(hence its own task queue).

### Running an independent load

A separate scheduler gives additional load its own rate and concurrency limits, so slow tests do not
occupy the main load's scheduling slots. The loads still share server and machine resources.
A scenario can implement this using separate `GenericExecutor` instances for the main and additional
loads. See [client-side load without a workflow](./authoring-scenarios.md#client-side-load-without-a-workflow).

#### Example: operator commands in throughput_stress

`throughput_stress` runs standalone-activity operator checks in a separate driver loop when the
namespace reports support. The existing `include-standalone-activity-operator-commands` option controls
this load. No additional CLI invocation is needed, and baseline workflow iterations do not wait for
operator tests. Both loops use the same client, worker, and task queue.

For example:

```sh
go run ./cmd/omes run-scenario-with-worker \
  --scenario throughput_stress --language go \
  --run-id mixed-load --server-address 127.0.0.1:17233 \
  --duration 30s --max-concurrent 2 --max-iterations-per-second 2 --timeout 2m \
  --option standalone-activity-operator-max-concurrent=3 \
  --option standalone-activity-operator-max-iterations-per-second=1
```

The standard flags control the main load. The `throughput_stress`-specific operator options control
operator tests, defaulting to concurrency 3 with no rate cap.

When baseline completes, the operator loop stops starting new tests and waits up to
`standalone-activity-operator-drain-timeout` (default 30s) for ongoing work. Failure, timeout, or
cancellation stops both loops. Operator work and final drain time are excluded from baseline
completion counts and throughput.

To run operator load on its own, use the standard rate and concurrency flags:

```sh
go run ./cmd/omes run-scenario-with-worker \
  --scenario standalone_activity_operator_commands --language go \
  --run-id operator-only --server-address 127.0.0.1:17233 \
  --duration 30s --max-concurrent 3 --max-iterations-per-second 1 --timeout 1m
```

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
- `--worker-max-heartbeat-throttle-interval` caps heartbeat RPC batching for Go
  workers. Zero keeps the SDK default; a worker profile takes precedence.
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
