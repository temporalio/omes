# SAA COGS Experiment: Research & Design

## 1. Current State of Omes

### Architecture
Omes is a load generation framework for Temporal. Scenarios are Go files in `scenarios/` that
register via `init()` → `loadgen.MustRegisterScenario()`. The scenario name comes from the
filename. Execution flows:

1. `run-scenario` command: dials Temporal, runs scenario executor
2. `run-worker` command: starts a worker (Go/Python/etc) polling a task queue
3. `run-scenario-with-worker`: runs both together (local development)

### Executor Types
- `GenericExecutor`: takes a `func(ctx, *Run) error` — most flexible
- `KitchenSinkExecutor`: wraps `GenericExecutor`, starts kitchen-sink workflows with configurable action sequences
- `FuzzExecutor`: random action generation

### Existing Standalone Activity Support
Branch `standalone-activity` (commit `efbbb7f`) adds SAA to the `throughput_stress` scenario as
an *optional extra activity within a workflow*. The implementation:

1. Proto: `StandaloneActivity` message in `kitchen_sink.proto`
2. Helper: `StandaloneActivity()` in `loadgen/kitchensink/helpers.go` creates an action
3. Worker: `ExecuteStandaloneActivity()` in `workers/go/kitchensink/kitchen_sink.go` — called as a
   *workflow activity* that internally calls `StartActivityExecution` + `PollActivityExecution`
4. Scenario: enabled via `--option enable-standalone-activity=true`

**Critical observation**: This existing support executes SAA *from within a workflow activity*.
That is useful for testing SAA functionality but **not** for the COGS experiment. For COGS, we need
to run SAA directly from the load generator (no workflow involved) so that the only server-side
work is the standalone activity execution itself.

## 2. What We Need for the COGS Experiment

### Two New Scenarios

**`saa_cogs_saw`** — Single Activity Workflow (the baseline):
- Each iteration: start a workflow that executes one activity (payload: 256B in, 256B out), then completes
- This is very close to `workflow_with_single_noop_activity` but with a payload activity

**`saa_cogs_saa`** — Standalone Activity:
- Each iteration: call `StartActivityExecution` directly from the load generator, then
  `PollActivityExecution` to wait for the result
- No workflow involved
- Same activity (payload: 256B in, 256B out) and task queue
- **Requires a `GenericExecutor`** since `KitchenSinkExecutor` always starts workflows

Both scenarios must use the same worker (the Go worker with `payload` activity registered).

### Key Design Decisions

1. **Activity type**: `payload` with 256B input, 256B output (matching the COGS analysis)
2. **No heartbeat, no retry** (matching the COGS analysis; retry max_attempts=1)
3. **Fixed start rate** (not fixed concurrency) — controls for latency differences
4. **Same task queue** for both scenarios — ensures same worker setup
5. **Sync match preferred** — the COGS analysis assumes sync match; verify via metrics

### SAA Load Generator Implementation

The SAA scenario needs to call gRPC APIs directly. Looking at the existing
`ExecuteStandaloneActivity` in the worker code (`workers/go/kitchensink/kitchen_sink.go:46-120`),
we have a working reference. The scenario version should:

1. Use `client.WorkflowService()` to get the gRPC client
2. Call `StartActivityExecution` with the activity config
3. Call `PollActivityExecution` to wait for completion
4. This is a `GenericExecutor` with a custom `Execute` function

## 3. Cloud Cell Operations

### Connecting to a Cloud Cell

From `bench-go.mdx`, the namespace format for test cells is `{cellId}-marathon.e2e` and the host
is `{cellId}-marathon.e2e.tmprl-test.cloud:7233`. For our cell `s-saa-cogs`:
- Namespace: `s-saa-cogs-marathon.e2e` (to be confirmed — Stephen may have set up differently)
- Host: `s-saa-cogs-marathon.e2e.tmprl-test.cloud:7233`

Omes connects via:
```
--server-address <host:port> --namespace <ns> --tls --tls-cert-path <cert> --tls-key-path <key>
```

Or with API key auth:
```
--server-address <host:port> --namespace <ns> --tls --auth-header "Bearer <api-key>"
```

### Running omes against a cloud cell

Two options:
1. **Local**: Run `go run ./cmd run-scenario` and `go run ./cmd run-worker` locally, connecting to
   the cloud cell via TLS. Simplest for proof-of-concept. Higher latency (network round trip to
   cloud) but the load generator itself isn't on the critical path for COGS measurement.
2. **K8s pod**: Deploy omes worker as a pod on the cell's k8s cluster. Lower latency, more
   realistic. The bench-go runbook shows this is the standard approach. Uses `omni scaffold` with
   `--benchgo-enabled` or manual deployment.

For initial proof-of-concept: run locally. For the actual experiment: deploy to k8s.

### Grafana Dashboard

The dashboard at `https://grafana.tmprl-internal.cloud/d/saacogs/saa-cogs` uses a `$cluster`
variable. Set `cluster=s-saa-cogs` to point at our cell.

### Cell Setup Verification

Use `ct` / `omni` to verify cell state:
```sh
# Check cell status
ct kubectl --context s-saa-cogs get pods -n temporal

# Check namespace exists
omni admintools --context s-saa-cogs -- temporal operator namespace describe s-saa-cogs-marathon.e2e
```

### Search Attributes

Cloud cells cannot register search attributes via the SDK — they must be registered via the
control plane. The `--do-not-register-search-attributes` flag exists for this. We should use it,
and register `OmesExecutionID` separately if needed. For the simple COGS scenarios, we may not
even need search attributes.

## 4. Implementation Plan

### Phase 1: Minimal Scenarios (omes code changes)

1. Create `scenarios/saa_cogs_saw.go` — SAW scenario using `KitchenSinkExecutor`
2. Create `scenarios/saa_cogs_saa.go` — SAA scenario using `GenericExecutor` with direct gRPC calls
3. Both share config: payload size, start rate, duration

### Phase 2: Local Proof-of-Concept

1. Test both scenarios against local Temporal server
2. Run `go run ./cmd run-scenario-with-worker` for SAW
3. For SAA: run worker separately, then scenario (since SAA doesn't use workflows but the
   worker still needs to poll for activity tasks)

### Phase 3: Cloud Cell Connection

1. Obtain credentials for s-saa-cogs cell
2. Verify dashboard shows idle state
3. Run a single SAW iteration and observe metrics
4. Run a single SAA iteration and observe metrics

### Phase 4: Full Experiment

1. Deploy omes worker to cloud cell k8s
2. Run SAW at target start rate for target duration
3. Wait for cool-down, collect metrics
4. Run SAA at same start rate for same duration
5. Collect and compare metrics

## 5. Open Questions

- What namespace(s) are configured on s-saa-cogs?
- How do we obtain TLS certs or API keys for the cell? (Check oncall or runbooks repos or search slack)
- Does the cell have CHASM standalone activities enabled? (Dynamic config flag)
- Worker deployment: should we use the existing bench-go infrastructure or deploy omes directly?
