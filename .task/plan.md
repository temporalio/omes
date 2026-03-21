# Implementation Plan: SAA Load Generation

## Goal

Create two omes scenarios to generate SAW and SAA workloads against cloud cell `s-saa-cogs`, then
observe metrics on the Grafana dashboard.

## Design

### Scenarios

Both scenarios use `GenericExecutor` with a simple `Execute` function. This keeps the
implementations symmetric — the only difference is what each iteration does, which is exactly the
variable under test.

**`workflow_with_single_activity`** — Each iteration calls `client.ExecuteWorkflow` with a dedicated
minimal workflow that executes one `payload` activity and returns. Then `handle.Get()`.

**`standalone_activity`** — Each iteration calls `client.ExecuteActivity` with the same `payload`
activity. Then `handle.Get()`. No workflow involved.

Both use the same task queue (derived from run-id) and the same Go worker.

### Worker code

Reuse the existing `payload` activity at [kitchen_sink.go:511-516](workers/go/kitchensink/kitchen_sink.go#L511-L516),
already registered as `"payload"` at [worker.go:105](workers/go/worker/worker.go#L105).

Add one new workflow: a minimal function that executes the `payload` activity with its input and
returns the result. Register it on the existing Go worker at [worker.go:102](workers/go/worker/worker.go#L102)
alongside the existing registrations. No new worker binary needed.

### Activity configuration

Both scenarios: `inputData []byte` (256 bytes), `bytesToReturn int32` (256). No heartbeat.
`MaximumAttempts: 1` (no retries). `ScheduleToCloseTimeout: 60s`.

### SDK version

`go.temporal.io/sdk v1.40.0` already includes `client.ExecuteActivity`. No upgrade needed.

## Implementation steps

IMPORTANT: Rather than doing the implementation yourself, please "teach" the user to do the
implementation themselves. Take a "painting by numbers" approach: Decide on the first component they
should write, and insert a comment in the code indicating what they should do. Then pause and give
them a clickable links to the comment, and to any existing prior art in the codebase they might want
to refer to. Don't output code directly to them. Work with them to complete the stage; review their
work carefully. Do not consider the stage complete until the work is done to an equal or greater
standard than you yourself would have achieved. When that stage is completed by them, or with
further assistance from you, move on to the next component to be implemented and repeat this
procedure.

Regarding names: we will not use "cogs" anywhere in omes code itself. Conceptually, the omes code is
defining SAW and SAA workloads. What those are used for (to run an experiment) and why (COGS
investigation) is not the concern of the omes code.

### Step 1: Add workflow to worker

Add a small file under `workers/go/` with the minimal workflow function. Register it in
[worker.go](workers/go/worker/worker.go) alongside existing registrations.

### Step 2: Create `scenarios/workflow_with_single_activity.go`

`GenericExecutor` whose `Execute` function:
1. Calls `run.Client.ExecuteWorkflow()` starting the new workflow with the payload input.
2. Calls `handle.Get()` to wait for result.

### Step 3: Create `scenarios/standalone_activity.go`

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
- SAW: `go run ./cmd run-scenario-with-worker --scenario workflow_with_single_activity --language go --iterations 5 --embedded-server`
- SAA: same command with `standalone_activity` — will get "Standalone activity is disabled" from the
  embedded dev server (v1.30.1 doesn't have the feature flag), confirming the code path reaches
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
3. **List scenarios**: `go run ./cmd list-scenarios` includes both names.
4. **Local test — SAW**: `run-scenario-with-worker --embedded-server --iterations 5` completes.
5. **Local test — SAA**: Same command hits `StartActivityExecution` on the server (expected to fail
   on dev server with "disabled" error; succeeds on cloud cell with CHASM enabled).
6. **Cloud cell proof-of-concept**: Dashboard shows idle -> run scenario -> dashboard shows activity.
