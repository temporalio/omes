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
- **Disjunction CSA**: Multiple CSAs ORed (e.g., `(VS_Int_01 > 900 OR VS_Keyword_01 = 'alpha')`)

List queries fetch a randomized number of pages to exercise `search_after`
pagination; see [Pagination Depth](#pagination-depth).

Disjunctions exist as a separate class because they are the expensive shape and ANDs
do not cover them. A conjunction costs roughly its *rarest* clause — the engine leads
with the smallest posting list and skips — so adding terms to an AND usually makes a
query cheaper. A disjunction must evaluate and union every clause, so its cost tracks
the *sum* of the clauses' selectivities.

### Query Complexity Distribution

The cost of a CSA filter is driven by its selectivity: how much of the corpus it
matches. Two mechanisms control that, and both are tunable.

**Threshold predicates** (Int, Double, Datetime) sample a target selectivity and
invert it into a threshold, rather than sampling a threshold and accepting whatever
selectivity results. `log10(selectivity) ~ Normal(selMu, selSigma)`, so query cost is
log-normal by construction instead of an artifact of the write-side value ranges.

| Option     | Default   | Description                                          |
|------------|-----------|-------------------------------------------------------|
| `selMu`    | *derived* | log10 of the median selectivity (`-2` = 1% of corpus) |
| `selSigma` | `1.0`     | Spread in log10 space; larger means a heavier tail    |

#### Why `selMu` is derived, not fixed

Selectivity is a *fraction*, but cost is `fraction x corpus`, and the load presets
differ by two orders of magnitude in corpus size. A single fixed `selMu` would mean
wildly different absolute costs across presets.

So when a `loadPreset` is set, `selMu` defaults to whatever makes the median query
match about **1000 documents** (roughly one page: deep enough to exercise the sort,
short of a full scan). The corpus estimate combines running workflows
(`wfRPS x updatesPerWF x updateDelay`) with closed-but-undeleted ones
(`(wfRPS - deleteRPS) x duration / 2`, the midpoint of linear growth).

For a 1-hour run:

| Preset     | Est. corpus | Derived `selMu` | Median hits |
|------------|-------------|-----------------|-------------|
| `light`    | ~14k        | -1.16           | ~1000       |
| `moderate` | ~145k       | -2.16           | ~1000       |
| `heavy`    | ~1.45M      | -3.16           | ~1000       |

Each 10x step in load shifts `selMu` by -1. Deriving rather than hardcoding also
tracks `wfRPS` / `deleteRPS` / `--duration` overrides, which a static table cannot.
The actual values are logged at startup. In **read-only mode** there is no load
preset to derive from, so `selMu` falls back to `-2.0` — set it explicitly to match
whatever the prior write run left behind.

#### Choosing `selSigma`

`selSigma` sets how many decades of cost the run sweeps (`+/-2 sigma` spans `4 x sigma`
decades):

| `selSigma` | Span    | Use for                                    |
|------------|---------|--------------------------------------------|
| `0.5`      | 2 decades | Low-variance A/B comparison between builds |
| `1.0`      | 4 decades | **Default** — realistic spread             |
| `1.5`      | 6 decades | Deliberately tail-heavy stress             |

Selectivity is clamped at 1.0, so a fraction `P(Z > -selMu/selSigma)` of draws
saturate into full scans: 0.6% at `selMu=-2.5, selSigma=1`, but 16% at
`selSigma=1.5`. Past that point it stops being a distribution and becomes "one query
in six is a full scan."

#### Multi-clause corrections

Compound and disjunction filters sample per clause, so the target is redistributed
to keep the *query* on target:

- **AND**: selectivities multiply, so log10s add — each tunable clause takes
  `selMu / n`. Without this, three clauses at `10^-2.5` combine to `10^-7.5` and
  match nothing on any corpus the scenario builds, making the compound class
  degenerate.
- **OR**: selectivities roughly sum, so each clause is tightened by `log10(n)`.

The split covers only threshold clauses. Keyword, Text, and Bool clauses ignore
`selMu` entirely, so dividing by the full clause count when two of three are Keyword
would over-narrow the one tunable clause.

### Pagination Depth

After each page of a List result, the querier fetches one more with probability
`pageContinueProb`, so the page count is geometric, truncated at `maxQueryPages`.

| Option             | Default | Description                                     |
|--------------------|---------|--------------------------------------------------|
| `pageContinueProb` | `0.4`   | Chance of reading one more page (must be in `[0,1]`) |
| `maxQueryPages`    | `10`    | Hard cap on pages per List query (must be `>= 1`) |

At the defaults, ~60% of queries read a single page, ~24% read two, and the mean is
~1.66 pages. That roughly matches client behavior: most callers read the first page
of a UI or CLI listing and only some scroll on.

A fixed cap puts a hard edge on the cost distribution exactly where the
deep-pagination tail would be, which is why the count is randomized and the cap is
set high enough to rarely bind (~0.03% of queries reach 10 pages at the defaults).
Set `pageContinueProb=0` for single-page queries only, or `pageContinueProb=1` with
a low `maxQueryPages` to force a fixed depth.

Note that `search_after` is O(page size) regardless of depth — unlike `from`/`size`
offset paging, page 10 costs about what page 1 costs. Pagination depth therefore
adds request volume rather than a per-request cost tail; the dominant cost term stays
the initial match and sort.

**Vocabulary predicates** (Keyword, Text) get their selectivity from word popularity.
Write-side values are drawn Zipf, read-side filters draw uniformly, so most queries hit
a rare word and match almost nothing while the occasional one hits a hot word and scans
a large slice of the corpus. Drawing writes uniformly instead would pin every keyword
filter at exactly 1/100 selectivity and collapse that spread to a point.

| Option          | Default | Description                                        |
|-----------------|---------|-----------------------------------------------------|
| `vocabZipfSkew` | `1.2`   | Zipf exponent for write-side word draws (must be >1) |

Bool predicates are always ~50% selective and are not tunable.

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

| Option     | Type  | Description                                                |
|------------|-------|-------------------------------------------------------------|
| `countRPS` | float | CountWorkflowExecutions per second                         |
| `listRPS`  | float | ListWorkflowExecutions per second                          |
| `selMu`    | float | log10 of median filter selectivity (must be `<= 0`); derived from the load preset if unset |
| `selSigma` | float | Spread of log10(selectivity); must be positive             |
| `pageContinueProb` | float | Chance of fetching one more List page; must be in `[0,1]` |
| `maxQueryPages`    | int   | Hard cap on pages per List query; must be `>= 1`        |

### Namespace & Environment

| Option             | Type     | Default | Description                                              |
|--------------------|----------|---------|----------------------------------------------------------|
| `namespaceCount`   | int      | `1`     | Number of namespaces to spread load across               |
| `createNamespaces` | bool     | `false` | Auto-create namespaces (ignored when `namespaceCount=1`) |
| `retention`        | duration | `168h`  | Namespace retention period (minimum `24h`)               |
| `vocabZipfSkew`    | float    | `1.2`   | Zipf exponent for write-side vocabulary draws (must be >1) |

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

### Metrics

Every visibility read is timed and tagged, so each query class's latency distribution
can be fitted separately:

| Metric                          | Type    | Tags                              |
|---------------------------------|---------|-----------------------------------|
| `omes_visibility_query_latency` | Timer   | `operation`, `query_class`, `page` |
| `omes_visibility_query_errors`  | Counter | `operation`, `query_class`, `page` |

- `operation` — `list` or `count`
- `query_class` — `no_filter`, `open`, `closed_time_range`, `simple_csa`,
  `compound_csa`, `disjunction_csa`, `keyword`, `deleter_list`
- `page` — which page of a paginated List; page 1 and a deep page are different
  cost points and averaging them hides that. The counts per `page` tag also give
  you the realized pagination-depth distribution.

Always break out by `query_class`. An aggregate percentile over the whole mix is
dominated by whichever class carries the most weight, so it shifts whenever the
`queryPreset` weights change and runs stop being comparable. Per-class series plus
the known weights let any aggregate be reconstructed afterwards.

`deleter_list` covers the deleter's own List call. It is a real visibility read on
the same search path as the querier's, it scales with `deleteRPS`, and it will skew
querier latencies if you forget it is running. Set `deleteRPS=0` when fitting a
clean querier distribution.

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

