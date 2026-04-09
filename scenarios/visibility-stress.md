# Visibility Stress Test — User Guide

## What It Does

The `visibility_stress` scenario generates controlled read and write traffic against Temporal's
visibility store (Elasticsearch or SQL). It creates short-lived workflows that perform custom
search attribute (CSA) updates, while separate goroutines issue List/Count queries and explicit
deletes.

Note : This scenario focuses on benchmarking the visibility store - not testing the correctness or 
any other portion of the temporal workflow engine.

### Operations Exercised

| Visibility Operation  | How it's generated                                                        |
|-----------------------|---------------------------------------------------------------------------|
| **Insert**            | Workflow starts (at `wfRPS`)                                              |
| **CSA Update**        | `UpsertSearchAttributes` inside each workflow (at `wfRPS * updatesPerWF`) |
| **Close (success)**   | Workflow completes normally (~85% by default)                             |
| **Close (failed)**    | Workflow returns intentional error (`failPercent`)                        |
| **Close (timed out)** | Workflow exceeds execution timeout (`timeoutPercent`)                     |
| **Explicit Delete**   | Deleter goroutine lists terminal WFs and deletes them (`deleteRPS`)       |
| **Retention Delete**  | Server-side GC after namespace retention period                           |
| **List/Count Query**  | Querier goroutine with varying filter complexity                          |

---

## Quick Start

```sh
# Simplest possible run (single namespace, write + read, 5 minutes)
go run ./cmd run-scenario-with-worker \
  --scenario visibility_stress --language go \
  --duration 5m \
  --option loadPreset=light --option queryPreset=light \
  --option csaPreset=small
```

This starts a Go worker, registers 6 CSAs on the default namespace, then runs for 5 minutes at
10 workflow starts/sec with light query traffic.

---

## Presets

The scenario is controlled by three independent presets. Each can be selected via `--option` and
individually overridden.

### Load Presets (`--option loadPreset=...`)

Controls write traffic. If omitted, no workflows are created (read-only mode).

| Preset        | wfRPS | updatesPerWF | deleteRPS | failPercent | timeoutPercent |
|---------------|-------|--------------|-----------|-------------|----------------|
| `light`       | 10    | 5            | 2         | 10%         | 5%             |
| `moderate`    | 100   | 10           | 20        | 10%         | 5%             |
| `heavy`       | 1000  | 20           | 200       | 10%         | 5%             |
| `no-failures` | 100   | 10           | 20        | 0%          | 0%             |

Override individual values:
```sh
--option loadPreset=moderate --option wfRPS=500 --option deleteRPS=0
```

### Query Presets (`--option queryPreset=...`)

Controls read traffic. If omitted, no queries are issued (write-only mode).

| Preset     | countRPS | listRPS | Filter distribution       |
|------------|----------|---------|---------------------------|
| `light`    | 1        | 2       | Balanced                  |
| `moderate` | 5        | 10      | Balanced                  |
| `heavy`    | 10       | 25      | Biased toward CSA filters |

Query types (weighted distribution):
- **No filter**: `WorkflowType = 'visibilityStressWorker'`
- **Open**: `ExecutionStatus = 'Running'`
- **Closed + time range**: `ExecutionStatus != 'Running' AND CloseTime > ...`
- **Simple CSA**: Single CSA filter (e.g., `VS_Int_01 > 500`)
- **Compound CSA**: Multiple CSAs ANDed (e.g., `VS_Int_01 > 200 AND VS_Keyword_01 = 'alpha'`)

List queries fetch up to 3 pages to exercise pagination.

### CSA Presets (`--option csaPreset=...`)

Defines which custom search attributes to register and use. Shared by both writer and querier.

| Preset   | CSAs                                                            |
|----------|-----------------------------------------------------------------|
| `small`  | 6 CSAs: 1 each of Int, Keyword, Bool, Double, Text, Datetime    |
| `medium` | 10 CSAs: 2 Int, 2 Keyword, 1 Bool, 2 Double, 1 Text, 2 Datetime |
| `heavy`  | 20 CSAs: 5 Int, 5 Keyword, 2 Bool, 3 Double, 3 Text, 2 Datetime |

If omitted when `loadPreset` is set, defaults to `medium`. Required in read-only mode.

---

## Modes

| `loadPreset` | `queryPreset` | What happens                                                    |
|--------------|---------------|-----------------------------------------------------------------|
| Set          | Set           | Full benchmark: writes + reads simultaneously                   |
| Set          | Omitted       | Write-only: workflows created/updated/deleted, no queries       |
| Omitted      | Set           | Read-only: queries against existing data from a prior write run |
| Omitted      | Omitted       | Error (unless `cleanup=true`)                                   |

---

## Common Usage Patterns

### Write-Only (Benchmark Inserts/Updates)

```sh
go run ./cmd run-scenario-with-worker \
  --scenario visibility_stress --language go \
  --duration 30m \
  --option loadPreset=moderate
```

### Read-Only (Benchmark Queries Against Existing Data)

Run this after a write run that used `csaPreset=medium`:

```sh
go run ./cmd run-scenario-with-worker \
  --scenario visibility_stress --language go \
  --duration 10m \
  --option queryPreset=heavy --option csaPreset=medium
```

### Full Benchmark (Writes + Reads)

```sh
go run ./cmd run-scenario-with-worker \
  --scenario visibility_stress --language go \
  --duration 1h \
  --option loadPreset=moderate --option queryPreset=light --option csaPreset=medium
```

### No Intentional Failures

```sh
go run ./cmd run-scenario-with-worker \
  --scenario visibility_stress --language go \
  --duration 30m \
  --option loadPreset=no-failures --option csaPreset=heavy
```

### Custom Failure Rate

```sh
go run ./cmd run-scenario-with-worker \
  --scenario visibility_stress --language go \
  --duration 30m \
  --option loadPreset=moderate --option failPercent=0.20 --option timeoutPercent=0.10
```

### High Throughput (No Deletes, Retention Only)

```sh
go run ./cmd run-scenario-with-worker \
  --scenario visibility_stress --language go \
  --duration 2h \
  --option loadPreset=heavy --option deleteRPS=0 --option retention=24h
```

---

## Multi-Namespace

For `namespaceCount > 1`, you must run the workers and scenario separately (
`run-scenario-with-worker` is not supported for multi-namespace).

```sh
# 1. Start a worker per namespace
for i in $(seq 0 4); do
  go run ./cmd run-worker --language go --run-id my-test \
    --namespace "vs-stress-my-test-$i" \
    --server-address localhost:7233 &
done

# 2. Run the scenario
go run ./cmd run-scenario --scenario visibility_stress --run-id my-test \
  --server-address localhost:7233 \
  --duration 30m \
  --option loadPreset=moderate --option queryPreset=light --option csaPreset=medium \
  --option namespaceCount=5 --option createNamespaces=true

# 3. Stop workers when done
kill $(jobs -p)
```

For single namespace (`namespaceCount=1`, the default), the scenario uses the CLI's `--namespace`
flag (default: `default`). No custom namespaces are created.

---

## Cleanup

After a run, workflows may remain on the server. Clean them up:

```sh
# Single namespace
go run ./cmd run-scenario-with-worker \
  --scenario visibility_stress --language go \
  --run-id my-test \
  --option cleanup=true

# Multi-namespace
go run ./cmd run-scenario --scenario visibility_stress --run-id my-test \
  --option cleanup=true --option namespaceCount=5

# Also delete the namespaces
go run ./cmd run-scenario --scenario visibility_stress --run-id my-test \
  --option cleanup=true --option deleteNamespaces=true --option namespaceCount=5
```

Cleanup terminates all running workflows and deletes all workflows with
`WorkflowType = 'visibilityStressWorker'` on the scenario's task queue.

---

## All Options Reference

### Preset Selection

| Option        | Values                                      | Default                      | Description             |
|---------------|---------------------------------------------|------------------------------|-------------------------|
| `loadPreset`  | `light`, `moderate`, `heavy`, `no-failures` | (none)                       | Write traffic profile   |
| `queryPreset` | `light`, `moderate`, `heavy`                | (none)                       | Read traffic profile    |
| `csaPreset`   | `small`, `medium`, `heavy`                  | `medium` (if loadPreset set) | CSA set to register/use |

### Load Overrides (apply on top of `loadPreset`)

| Option           | Type  | Description                                           |
|------------------|-------|-------------------------------------------------------|
| `wfRPS`          | float | Workflow starts per second                            |
| `updatesPerWF`   | float | CSA updates per workflow (fractional OK, e.g., `0.5`) |
| `deleteRPS`      | float | Explicit deletes per second (`0` = retention only)    |
| `failPercent`    | float | Fraction of WFs that fail (e.g., `0.10` = 10%)        |
| `timeoutPercent` | float | Fraction of WFs that timeout (e.g., `0.05` = 5%)      |

### Query Overrides (apply on top of `queryPreset`)

| Option     | Type  | Description                        |
|------------|-------|------------------------------------|
| `countRPS` | float | CountWorkflowExecutions per second |
| `listRPS`  | float | ListWorkflowExecutions per second  |

### Namespace & Environment

| Option             | Type     | Default | Description                                              |
|--------------------|----------|---------|----------------------------------------------------------|
| `namespaceCount`   | int      | `1`     | Number of namespaces to spread load across               |
| `createNamespaces` | bool     | `false` | Auto-create namespaces (ignored when `namespaceCount=1`) |
| `retention`        | duration | `168h`  | Namespace retention period (minimum `24h`)               |

### Modes

| Option             | Type | Default | Description                                      |
|--------------------|------|---------|--------------------------------------------------|
| `cleanup`          | bool | `false` | Cleanup mode: terminate + delete all workflows   |
| `deleteNamespaces` | bool | `false` | Delete namespaces during cleanup (multi-NS only) |

### CLI Flags (not `--option`)

| Flag            | Required               | Description                                      |
|-----------------|------------------------|--------------------------------------------------|
| `--duration`    | Yes                    | How long to run the steady-state phase           |
| `--language go` | Yes                    | Must be `go` (Go-only scenario)                  |
| `--run-id`      | Yes for `run-scenario` | Links worker and scenario to the same task queue |

`--iterations` is **not supported** by this scenario.

---

## How It Works (Brief)

1. **Setup**: Register CSAs on each namespace, poll until propagated (up to 30s).
2. **Writer goroutine**: Rate-limited loop starts workflows at `wfRPS`. Each workflow receives
   instructions (CSA update groups, delay, fail/timeout flags) baked into its input. Fire-and-forget.
3. **Workflow**: Executes CSA updates with sleeps between them, then reaches a terminal state
   (completed / failed / timed out).
4. **Deleter goroutines**: One per namespace. Each periodically lists terminal workflows and
   deletes them. The list query itself is a visibility read (realistic!).
5. **Querier goroutine**: Rate-limited loop issues List/Count queries with varying filter
   complexity, fetching up to 3 pages per query.
6. **Teardown**: Log final stats (total created, deleted, queried, errors).

### Derived Rates (Logged at Startup)

- **Effective CSA update RPS** = `wfRPS * updatesPerWF`
- **Effective close RPS** ≈ `wfRPS` (at steady state)
- **Workflow lifetime** ≈ `updatesPerWF * updateDelay` (then completes/fails/times out)

### Example Startup Log

```
Mode: write+read
Namespaces: [default]
Task queue: omes-my-test
CSAs: 10 total
Write: wfRPS=100.0, updatesPerWF=10.0, effective CSA update RPS≈1000, deleteRPS=20.0
       failPercent=0.10, timeoutPercent=0.05, updateDelay=1s
Read: countRPS=5.0, listRPS=10.0
```

