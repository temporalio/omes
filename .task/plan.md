# Implementation Plan: SAA COGS Load Generation

## Goal

Create two omes scenarios to generate SAW and SAA workloads against cloud cell `s-saa-cogs`, then
observe metrics on the Grafana dashboard.

## Design

### Scenarios

Both scenarios use `GenericExecutor` with a simple `Execute` function. This keeps the
implementations symmetric — the only difference is what each iteration does, which is exactly the
variable under test.

**`saw`** — Single Activity Workflow baseline. Each iteration calls `client.ExecuteWorkflow` with a
dedicated minimal workflow (`saw`) that executes one payload activity and returns. Then
`handle.Get()` to wait for the result.

**`saa`** — Standalone Activity. Each iteration calls `client.ExecuteActivity` with the same payload
activity. Then `handle.Get()` to wait for the result. No workflow involved.

Both use the same task queue (derived from run-id) and the same Go worker.

### Worker code

A dedicated activity (`payload`) and a dedicated workflow (`saw`), both minimal:

- **`payload` activity**: Takes `[]byte` input and `int32` output size, returns `[]byte` of
  requested size. (This activity already exists in the kitchen-sink worker as `"payload"` with
  exactly this signature. We write our own to avoid depending on the kitchen-sink worker.)
- **`saw` workflow**: Executes the `payload` activity with the input it receives, returns the
  result. No signals, queries, updates, or other machinery.

Both are registered on the Go worker alongside the existing kitchen-sink registrations.

### Activity configuration

Both scenarios use: `inputData []byte` (256 bytes), `bytesToReturn int32` (256). No heartbeat.
Retry policy `MaximumAttempts: 1` (no retries). `ScheduleToCloseTimeout: 60s`.

### SDK version

`go.temporal.io/sdk v1.40.0` already includes `client.ExecuteActivity`. No upgrade needed.

## Implementation steps

### Step 1: Add worker code

In `workers/go/`, add a small file registering:
- Activity `"payload"` — takes `(ctx, []byte, int32)`, returns `([]byte, error)`
- Workflow `"saw"` — executes `"payload"` activity with its input, returns result

These are registered on the worker alongside existing kitchen-sink registrations.

**Wait — the existing worker already registers `"payload"` with the same signature.** We should
reuse that registration rather than duplicate it. The question is whether we also need a separate
worker binary or can share the existing one. The existing Go worker registers the kitchen-sink
workflow plus all activities including `"payload"`. For SAW we just need to also register our `saw`
workflow. For SAA we need no workflow at all — just the `"payload"` activity, which is already
registered.

Decision: add `saw` workflow registration to the existing Go worker. No new worker binary needed.

### Step 2: Create `scenarios/saw.go`

`GenericExecutor` whose `Execute` function:
1. Calls `run.Client.ExecuteWorkflow()` starting workflow `"saw"` with the payload input.
2. Calls `handle.Get()` to wait for result.

### Step 3: Create `scenarios/saa.go`

`GenericExecutor` whose `Execute` function:
1. Calls `run.Client.ExecuteActivity()` with `StartActivityOptions` (ID derived from
   run/execution/iteration, task queue from `run.TaskQueue()`, same timeout and retry policy).
2. Passes activity type `"payload"` by name with `[]byte` (256 zeros) and `int32(256)`.
3. Calls `handle.Get()` to wait for result.

### Step 4: Create `commands.sh`

Useful shell commands with terse comments for:
- Local testing with `--embedded-server`
- Cloud cell verification via `ct`
- Running scenarios against `s-saa-cogs`

### Step 5: Test locally

- `go build ./...` and `go vet ./...`
- `go run ./cmd list-scenarios` shows both new scenarios
- SAW: `go run ./cmd run-scenario-with-worker --scenario saw --language go --iterations 5 --embedded-server`
- SAA: same command with `saa` — will get "Standalone activity is disabled" from the embedded dev
  server (v1.30.1 doesn't have the feature flag), confirming the code path reaches
  `StartActivityExecution`. Will succeed on the cloud cell.

### Step 6: Connect to cloud cell

1. Verify cell: `ct kubectl --context s-saa-cogs get pods -n temporal`
2. Check namespace: `ct admintools --context s-saa-cogs -- temporal operator namespace describe s-saa-cogs-marathon.e2e`
3. Obtain operator TLS certs (from k8s secrets via `ct`, or ask Stephen)
4. Point Grafana dashboard at `s-saa-cogs`, observe idle state
5. Run worker + SAW scenario against the cell, observe activity in dashboard
6. Run worker + SAA scenario, observe activity

## Verification

1. **Build**: `go build ./...` succeeds.
2. **Lint/vet**: `go vet ./...` clean on our files.
3. **List scenarios**: `go run ./cmd list-scenarios` includes both `saw` and `saa`.
4. **Local test — SAW**: `run-scenario-with-worker --embedded-server --iterations 5` completes.
5. **Local test — SAA**: Same command hits `StartActivityExecution` on the server (expected to fail
   on dev server with "disabled" error; succeeds on cloud cell with CHASM enabled).
6. **Cloud cell proof-of-concept**: Dashboard shows idle → run scenario → dashboard shows activity.
