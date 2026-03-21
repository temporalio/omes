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

### Why GenericExecutor for SAA

`KitchenSinkExecutor` always starts a kitchen-sink workflow. The SAA scenario must call
`client.ExecuteActivity` directly — no workflow. `GenericExecutor` gives us the `Execute` function
hook, plus all the concurrency/rate-limiting/duration infrastructure.

### SDK version

The current `go.temporal.io/sdk v1.40.0` already includes `client.ExecuteActivity` (added in
v1.40.0, commit `215920a6`). No upgrade needed.

### Activity configuration

Both scenarios use the `payload` activity type (already registered in the Go worker as `"payload"`).
Arguments: `inputData []byte` (256 bytes), `bytesToReturn int32` (256). No heartbeat. Retry policy
`MaximumAttempts: 1` (no retries). `ScheduleToCloseTimeout: 60s`.

## Implementation

### Step 1: Create `scenarios/saa_cogs_saw.go`

```go
package scenarios

func init() {
    loadgen.MustRegisterScenario(loadgen.Scenario{
        Description: "SAW baseline for COGS: single workflow executing one payload activity.",
        ExecutorFn: func() loadgen.Executor {
            return loadgen.KitchenSinkExecutor{
                TestInput: &kitchensink.TestInput{
                    WorkflowInput: &kitchensink.WorkflowInput{
                        InitialActions: []*kitchensink.ActionSet{
                            payloadActivityActionSet(),
                        },
                    },
                },
            }
        },
    })
}
```

Where `payloadActivityActionSet()` creates a `PayloadActivity(256, 256, ...)` plus
`ReturnResultAction`, with `MaximumAttempts: 1`, `ScheduleToCloseTimeout: 60s`.

### Step 2: Create `scenarios/saa_cogs_saa.go`

```go
package scenarios

func init() {
    loadgen.MustRegisterScenario(loadgen.Scenario{
        Description: "SAA for COGS: standalone activity with payload, no workflow.",
        ExecutorFn: func() loadgen.Executor {
            return &loadgen.GenericExecutor{
                Execute: executeSAA,
            }
        },
    })
}

func executeSAA(ctx context.Context, run *loadgen.Run) error {
    inputData := make([]byte, 256)
    handle, err := run.Client.ExecuteActivity(ctx, client.StartActivityOptions{
        ID:                     fmt.Sprintf("a-%s-%s-%d", run.RunID, run.ExecutionID, run.Iteration),
        TaskQueue:              run.TaskQueue(),
        ScheduleToCloseTimeout: 60 * time.Second,
        RetryPolicy:            &temporal.RetryPolicy{MaximumAttempts: 1},
    }, "payload", inputData, int32(256))
    if err != nil {
        return err
    }
    var result []byte
    return handle.Get(ctx, &result)
}
```

This calls `"payload"` by name (string) so the SDK dispatches it to the worker which has it
registered as `activity.RegisterOptions{Name: "payload"}`.

### Step 3: Create `commands.sh` — useful shell commands

A file with terse comments documenting how to run the scenarios locally and against the cloud cell.

### Step 4: Test locally

Run against local dev server using `run-scenario-with-worker`:

```sh
# SAW
go run ./cmd run-scenario-with-worker \
    --scenario saa_cogs_saw --language go \
    --iterations 5

# SAA
go run ./cmd run-scenario-with-worker \
    --scenario saa_cogs_saa --language go \
    --iterations 5
```

### Step 5: Connect to cloud cell

Use `omni admintools` to verify cell state, then obtain credentials. Run scenarios against
`s-saa-cogs-marathon.e2e.tmprl-test.cloud:7233` with TLS.

## Verification

1. **Build**: `go build ./...` succeeds.
2. **Lint/vet**: `go vet ./...` succeeds.
3. **Local test — SAW**: `go run ./cmd run-scenario-with-worker --scenario saa_cogs_saw --language go --iterations 5` completes successfully.
4. **Local test — SAA**: `go run ./cmd run-scenario-with-worker --scenario saa_cogs_saa --language go --iterations 5` completes successfully.
5. **List scenarios**: `go run ./cmd list-scenarios` includes both `saa_cogs_saw` and `saa_cogs_saa`.
6. **Cloud cell proof-of-concept**: Point dashboard at `s-saa-cogs`, run one scenario, observe metrics increase.
