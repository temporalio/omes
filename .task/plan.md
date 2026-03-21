# Implementation Plan: SAA COGS Load Generation

## Goal

Create two omes scenarios to generate SAW and SAA workloads against cloud cell `s-saa-cogs`, then
observe metrics on the Grafana dashboard.

## Design

### Scenarios

**`saa_cogs_saw`** — Single Activity Workflow baseline. Uses `KitchenSinkExecutor` with a single
payload activity (256B in, 256B out), no retry, no heartbeat. Very close to
`workflow_with_single_noop_activity` but with payload instead of noop.

**`saa_cogs_saa`** — Standalone Activity. Uses `GenericExecutor`. Each iteration calls
`client.ExecuteActivity()` (the SDK's standalone activity API) with the same payload activity, then
`handle.Get()` to wait for the result. No workflow involved.

Both use the same task queue (derived from run-id) and the same Go worker (which already registers
the `payload` activity).

### Why different executor types

`KitchenSinkExecutor` always starts a kitchen-sink workflow — this is inherently what SAW needs.
SAA must call `client.ExecuteActivity` directly (no workflow). `GenericExecutor` gives us the
`Execute` function hook for this.

Both executor types share the same iteration-driving machinery: `KitchenSinkExecutor` wraps
`GenericExecutor`, so concurrency control, rate limiting, and duration handling are identical.
The only difference between the scenarios is what each iteration *does*, which is exactly the
variable under test.

The activity configuration (256B payload, no retry, 60s timeout) is specified independently in each
scenario. These are simple literal values; sharing them via an abstraction would add indirection
without meaningful deduplication.

### SDK version

The current `go.temporal.io/sdk v1.40.0` already includes `client.ExecuteActivity` (added in
v1.40.0, commit `215920a6`). No upgrade needed.

### Activity configuration

Both scenarios use the `payload` activity type (already registered in the Go worker as `"payload"`).
Arguments: `inputData []byte` (256 bytes), `bytesToReturn int32` (256). No heartbeat. Retry policy
`MaximumAttempts: 1` (no retries). `ScheduleToCloseTimeout: 60s`.

## Implementation steps

### Step 1: Create `scenarios/saa_cogs_saw.go`

`KitchenSinkExecutor` with a single `ActionSet` containing a `PayloadActivity(256, 256)` action
(with `MaximumAttempts: 1`, `ScheduleToCloseTimeout: 60s`) followed by a `ReturnResultAction`.

### Step 2: Create `scenarios/saa_cogs_saa.go`

`GenericExecutor` whose `Execute` function:
1. Calls `run.Client.ExecuteActivity()` with `StartActivityOptions` (ID derived from
   run/execution/iteration, task queue from `run.TaskQueue()`, same timeout and retry policy as SAW).
2. Passes activity type `"payload"` by name with `[]byte` (256 zeros) and `int32(256)` as args.
3. Calls `handle.Get()` to wait for the result.

### Step 3: Create `commands.sh`

Useful shell commands with terse comments for:
- Local testing with `--embedded-server`
- Cloud cell verification via `ct`
- Running scenarios against `s-saa-cogs`

### Step 4: Test locally

- `go build ./...` and `go vet ./...`
- `go run ./cmd list-scenarios` shows both new scenarios
- SAW: `go run ./cmd run-scenario-with-worker --scenario saa_cogs_saw --language go --iterations 5 --embedded-server`
- SAA: same command with `saa_cogs_saa` — will get "Standalone activity is disabled" from the dev
  server (v1.30.1 doesn't have the feature flag), confirming the code path reaches
  `StartActivityExecution`. Will succeed on the cloud cell.

### Step 5: Connect to cloud cell

1. Verify cell: `ct kubectl --context s-saa-cogs get pods -n temporal`
2. Check namespace: `ct admintools --context s-saa-cogs -- temporal operator namespace describe s-saa-cogs-marathon.e2e`
3. Obtain operator TLS certs (from k8s secrets via `ct`, or ask Stephen)
4. Point Grafana dashboard at `s-saa-cogs`, observe idle state
5. Run worker + SAW scenario against the cell, observe activity in dashboard
6. Run worker + SAA scenario, observe activity

## Verification

1. **Build**: `go build ./...` succeeds.
2. **Lint/vet**: `go vet ./...` clean on our files.
3. **List scenarios**: `go run ./cmd list-scenarios` includes both `saa_cogs_saw` and `saa_cogs_saa`.
4. **Local test — SAW**: `run-scenario-with-worker --embedded-server --iterations 5` completes.
5. **Local test — SAA**: Same command hits `StartActivityExecution` on the server (expected to fail
   on dev server with "disabled" error; succeeds on cloud cell with CHASM enabled).
6. **Cloud cell proof-of-concept**: Dashboard shows idle → run scenario → dashboard shows activity.
