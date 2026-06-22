# Plan: Add schedule load to throughput_stress

## Goal

Extend `throughput_stress` so it exercises Temporal Schedule APIs while preserving the existing workflow throughput behavior.

The intended model is:

1. Keep exactly 3 stable schedules running for the active verification window.
2. Create additional short-lived schedules from the same per-iteration path that currently creates throughput workflows.
3. Ensure workflows started by schedules either complete on their own or can be released during cleanup and then complete.
4. Use `ListScheduleMatchingTimes` during the stable window to compute expected workflow starts for overlap policies that depend on workflow completion callbacks.
5. Keep the existing throughput verification meaningful and avoid false failures from expected long-running scheduled workflows.

This does make sense. The key is to model schedule load as two separate lanes:

- Stable schedule lane: exactly 3 fixed schedules that are not changed during a bounded verification window and keep firing.
- Iteration schedule lane: schedules created per iteration, under the same concurrency, retry, rate limit, and resume behavior as normal throughput workflow creation.

## Current State

`throughput_stress` currently starts kitchen-sink workflows through `loadgen.KitchenSinkExecutor`, which wraps `loadgen.GenericExecutor`. Each iteration starts one main workflow and the workflow itself starts child workflows and continue-as-new runs.

Relevant current behavior:

- `tpsExecutor.Run` configures state, then creates `KitchenSinkExecutor`.
- `KitchenSinkExecutor` owns the per-iteration loop.
- `OnCompletion` increments `CompletedIterations`.
- Visibility verification expects at least the known number of main, child, and continue-as-new workflows tagged with `OmesExecutionID`.
- `VerifyNoFailedWorkflows` checks all workflows tagged with `OmesExecutionID`.

There is already a `scheduler_stress` scenario, but it is not a good drop-in for this:

- It creates schedules with random UUID IDs, which is not ideal for resume/idempotency.
- It logs many schedule operation errors instead of returning them.
- It does not wait for scheduled workflows to complete.
- It is a standalone scenario, so it does not share the same per-iteration lifecycle as throughput workflows.
- Its `SleepScheduledWorkflow` and `NoopScheduledWorkflow` workflow types are Go-worker-only.

For `throughput_stress`, scheduled workflows should use the common `kitchenSink` workflow type so the feature works with the normal worker app across SDKs. The Go-only scheduler workflows can remain for `scheduler_stress`.

## High-Level Design

Add a schedule controller owned by `tpsExecutor`.

The controller manages:

1. Stable schedules:
   - Created once near the start of `Run`.
   - Deterministic IDs.
   - Exactly 3 by default.
   - Run through a mutation-free verification window.
   - Use `SKIP`, `BUFFER_ONE`, and `BUFFER_ALL` to verify callback-dependent overlap behavior.
   - Use `ListScheduleMatchingTimes` as the oracle for matching opportunities.
   - Are paused/deleted only after the stable verification window has ended.

2. Per-iteration schedules:
   - Created inside each throughput iteration.
   - Deterministic IDs derived from run ID, execution ID, iteration, and schedule index.
   - Respect the same `MaxConcurrent`, `MaxIterationsPerSecond`, retry, timeout, and resume model as normal workflow iterations.
   - Usually one-shot or bounded schedules so each iteration can finish deterministically.

The recommended implementation is to replace the direct `KitchenSinkExecutor` usage inside `throughput_stress` with an equivalent local `GenericExecutor`. That gives one per-iteration function that can do both:

1. Start and wait for the normal throughput kitchen-sink workflow.
2. Create, exercise, wait for, and clean up per-iteration schedules.

This preserves the current `GenericExecutor` semantics while allowing schedule work to participate in the same iteration lifecycle.

## Proposed Flags

Add these scenario options:

- `include-schedules`: bool, default `false`.
- `stable-schedule-count`: int, default `3`.
- `stable-schedule-interval`: duration, default `5s`.
- `stable-schedule-workflow-duration`: duration, default `15s`.
- `stable-schedule-completion-mode`: string, default `release`, values `timer` or `release`.
- `stable-schedule-window`: duration, default derived from scenario duration or executor active time.
- `iteration-schedules-per-iteration`: int, default `1` when schedules are enabled, otherwise `0`.
- `iteration-schedule-workflow-duration`: duration, default `1s`.
- `schedule-visibility-timeout`: duration, default reuse `visibility-count-timeout` or `3m`.
- `schedule-api-operation-interval`: duration, default `50ms`.
- `schedule-overlap-policies`: string, default `skip,buffer_one,buffer_all`.

Configuration shape:

- Keep schedule settings in their own `tpsScheduleConfig` struct.
- Store it on `tpsConfig` as `Schedules tpsScheduleConfig`.
- Parse `include-schedules` into `tpsConfig.Schedules.Enabled`.
- Use `tpsConfig.Schedules.Enabled` as the only runtime gate for schedule behavior.
- Keep the default disabled so existing throughput stress runs do not create schedules unless explicitly enabled.

Validation:

- Counts must be non-negative.
- Stable schedule count must be exactly 3 by default, but allow override for tests.
- Durations must be positive.
- Completion mode must be one of the known values.
- Overlap policies should reject unknown strings.
- Avoid `cancel_other` and `terminate_other` in `throughput_stress` by default because the scenario asserts there are no failed or terminated workflows.
- The matching-times verification should require `skip`, `buffer_one`, and `buffer_all` when enabled. Other policies can be tested by per-iteration churn schedules, but they should not replace the stable verification set.

## Search Attributes

Scheduled workflows need their own tags in addition to `OmesExecutionID`.

Add schedule-specific keyword search attributes:

- `OmesScheduleKind`: `stable` or `iteration`.
- `OmesScheduleID`: the schedule ID that started the workflow.

Register them when schedules are enabled. Use `loadgen.InitSearchAttribute` for these optional attributes, or extend `RegisterDefaultSearchAttributes` if this should become common across scenarios.

Every `client.ScheduleWorkflowAction` should set:

- `TypedSearchAttributes` with `OmesExecutionID`.
- `TypedSearchAttributes` with `OmesScheduleKind`.
- `TypedSearchAttributes` with `OmesScheduleID`.

This makes cleanup and verification reliable without depending on workflow ID prefix queries or `ScheduleDescription.Info.RecentActions`, which only contains a bounded recent history.

## Scheduled Workflow Shape

Use `kitchenSink` as the scheduled workflow type.

Timer completion mode:

```text
Timer(schedule workflow duration)
ReturnResult
```

Release completion mode:

```text
Timer(minimum run duration)
AwaitWorkflowState("schedule_release", "true")
ReturnResult
```

The release mode gives the "can then complete" behavior. The workflow stays open after its minimum run duration until the controller sends a `do_actions_signal` that sets `schedule_release=true`.

For release:

- List or count open workflows with `OmesExecutionID`, `OmesScheduleKind`, and `ExecutionStatus='Running'`.
- Signal each open scheduled workflow using the kitchen-sink signal payload.
- Wait until no scheduled workflows for this execution remain running.

## Stable Schedule Lane

The 3 long-running schedules should be the stable matching-times verification schedules.

They are created once before the per-iteration executor starts and remain unchanged during a bounded stable window. In an iteration-count run, that stable window can cover the active executor window for the run. In a duration-based run, it can cover the configured scenario duration. For a future per-iteration variant, the same pattern can be nested inside an iteration, but the first implementation should use one scenario-level stable set to keep verification and cleanup simpler.

Default: 3 schedules.

IDs:

```text
sched-{runID}-{executionID}-stable-0
sched-{runID}-{executionID}-stable-1
sched-{runID}-{executionID}-stable-2
```

Workflow action IDs:

```text
w-{scheduleID}
```

The server may append a timestamp or unique suffix to scheduled workflow IDs. Do not rely on exact workflow IDs for cleanup; rely on search attributes.

Schedule spec:

- Interval schedule, not cron, because interval is easier to control in short tests.
- `TriggerImmediately: true` so each stable schedule starts work right away.
- `EndAt` set to a conservative upper bound: now plus scenario duration or timeout plus cleanup grace.
- Delete the schedule explicitly during finalization even if `EndAt` is set.

Overlap policies:

- Use three callback-dependent policies by default: `SKIP`, `BUFFER_ONE`, and `BUFFER_ALL`.
- With workflow duration longer than interval, these exercise paths where the scheduler must learn that the previous workflow completed before it can start or drain the next workflow.
- Do not use `ALLOW_ALL` for this stable verification lane. It creates useful schedule load, but it does not prove workflow-completion callbacks are delivered because starts do not depend on prior workflow completion.

API operations to exercise:

1. Create schedule.
2. Describe schedule.
3. List matching times for the stable window.
4. Observe started workflows and completion-driven follow-up starts.
5. Pause schedule after the stable window.
6. Delete schedule at finalization.

The stable lane should not block per-iteration workflow creation. It runs alongside the normal throughput load and is finalized after normal throughput iterations stop.

Do not update, pause, unpause, backfill, or delete these 3 schedules during the stable window. Any API churn on schedules should happen in the per-iteration schedule lane or after stable verification is complete. This is what makes `ListScheduleMatchingTimes` usable as an oracle.

## Matching-Times Verification

Use `ListScheduleMatchingTimes` for the 3 stable schedules to turn schedule specs into expected matching opportunities.

The request window should be recorded explicitly:

```text
stableWindowStart = time.Now() after all 3 stable schedules are created and described
stableWindowEnd = time.Now() when the executor stops starting throughput iterations
```

For each stable schedule:

1. Call `ListScheduleMatchingTimes(namespace, scheduleID, stableWindowStart, stableWindowEnd)`.
2. List actual scheduled workflows with:
   `OmesExecutionID = executionID AND OmesScheduleKind = 'stable' AND OmesScheduleID = scheduleID`.
3. Read actual start and close times from visibility or workflow histories.
4. Simulate the overlap policy using matching times and observed workflow intervals.
5. Assert that the actual workflow starts match the expected starts within a tolerance.

The verifier should not use matching times as a raw workflow count. Matching times are opportunities, and overlap policies decide which opportunities produce workflow starts.

Expected behavior by policy:

1. `SKIP`
   - If a matching time occurs while a scheduled workflow is running, no workflow should start for that match.
   - After the running workflow completes, the next matching time should start a workflow.
   - If no workflow starts after the next post-completion matching time, the scheduler may not have received or processed the workflow completion callback.

2. `BUFFER_ONE`
   - If one or more matching times occur while a workflow is running, exactly one buffered workflow should start soon after the running workflow completes.
   - The next workflow start should be driven by completion, not by waiting for the next wall-clock matching time.
   - If that buffered start does not happen after completion, completion callback delivery is suspect.

3. `BUFFER_ALL`
   - Every matching time that occurs while a workflow is running should eventually drain as a sequential workflow start.
   - After each workflow completes, the next buffered workflow should start within a tolerance until the buffer is empty.
   - If draining stops while buffered matches remain, completion callback delivery is suspect.

Use release-mode scheduled workflows for the strongest callback check:

```text
Timer(minimum run duration)
AwaitWorkflowState("schedule_release", "true")
ReturnResult
```

The controller can release one running workflow at a time, wait for the expected next start, then release the next workflow. This provides a deterministic check that callback-dependent policies actually drain after workflow completion.

Recommended tolerance:

- Start with `5s` locally and make it configurable if flakes appear.
- Use the schedule interval plus a small grace period for cloud or loaded environments.

## Per-Iteration Schedule Lane

Per-iteration schedules are created from the same per-iteration function that starts throughput workflows.

IDs:

```text
sched-{runID}-{executionID}-iter-{iteration}-{index}
```

Recommended default:

- `iteration-schedules-per-iteration=1` when `include-schedules=true`.
- Each per-iteration schedule is bounded and deterministic.
- Use `RemainingActions: 1` or create paused and manually `Trigger`.
- The scheduled workflow should complete during the same iteration.

Per-iteration schedule API sequence:

1. Create schedule with deterministic ID and `Paused: true`.
2. Describe it.
3. Update action payload, memo, or overlap policy.
4. Unpause or trigger it.
5. Wait for the scheduled workflow to complete.
6. Describe again to observe action metadata.
7. Delete the schedule.

This gives schedule API coverage without leaving per-iteration schedules around after an iteration succeeds.

The per-iteration schedule should return errors to the iteration. It should not only log errors. This is important because `GenericExecutor` retry and failure behavior depends on returned errors.

## Refactor Throughput Execution

Current flow:

```text
tpsExecutor.Run
  -> KitchenSinkExecutor.Run
    -> GenericExecutor.Run
      -> run.ExecuteKitchenSinkWorkflow
```

Proposed flow:

```text
tpsExecutor.Run
  -> configure
  -> maybe create stable schedules
  -> GenericExecutor.Run
    -> t.executeThroughputIteration(ctx, run)
      -> start per-iteration schedule work
      -> execute normal kitchen-sink workflow
      -> wait per-iteration schedule work
  -> finalize stable schedules
  -> existing throughput visibility checks
  -> schedule-specific checks
  -> no failed workflow check
```

Move the current `KitchenSinkExecutor.UpdateWorkflowOptions` logic into a helper:

```text
t.buildKitchenSinkWorkflowOptions(ctx, run, isResuming)
```

That helper keeps the existing behavior:

- Default start options.
- Resume behavior for already-started workflows.
- `OmesExecutionID` search attribute.
- 50 percent update-with-start path.
- `createActions(run)`.

Then `executeThroughputIteration` can call `run.ExecuteKitchenSinkWorkflow(ctx, opts)` directly.

Per-iteration schedule work can run concurrently with the normal kitchen-sink workflow using an errgroup, or sequentially if determinism is more important. The safer initial implementation is:

1. Create and trigger per-iteration schedules.
2. Start the normal throughput workflow.
3. Wait for both normal workflow and scheduled workflows.
4. Delete per-iteration schedules.

## State and Resume

Extend `tpsState` with schedule state:

```go
type tpsState struct {
    CompletedIterations int
    LastCompletedIterationAt time.Time
    AccumulatedDuration time.Duration

    StableSchedulesCreated bool
    StableSchedulesFinalized bool
    StableWindowStart time.Time
    StableWindowEnd time.Time
    MatchingTimesVerified bool
    CompletedIterationScheduledWorkflows int
    CompletedStableScheduledWorkflows int
}
```

Keep state compact. Do not store every scheduled workflow ID unless required. Search attributes should be the source of truth for cleanup and verification.

Resume behavior:

- Stable schedule IDs are deterministic, so creation should handle already-exists by adopting the existing schedule handle.
- Per-iteration schedule IDs are deterministic, so retries can adopt and finish or delete/recreate existing schedules.
- If resuming from a completed iteration, do not recreate its per-iteration schedules because `StartFromIteration` already skips that iteration.
- If resuming after a crash during an iteration, retrying that iteration should be idempotent:
  - Describe existing schedule.
  - If schedule exists and has already triggered work, wait/release/cleanup.
  - If schedule exists but has not triggered, finish the API sequence.
  - If schedule is missing, create it.
- Finalization must be idempotent:
  - Pause/delete stable schedules if present.
  - Ignore not-found on delete.
  - Release running scheduled workflows if any.
  - Wait for no running scheduled workflows.
- If resuming after the stable window has started, the controller should set `StableWindowEnd` to the current finalization time and still run matching-times verification for the portion of the window that can be observed.

## Verification

Keep existing throughput verification mostly unchanged.

Do not add scheduled workflows to the main throughput denominator by default. The existing `completedWorkflows` value is used for throughput accounting and min throughput checks. Scheduled workflows are additional traffic and should be reported separately.

Schedule-specific verification:

1. Per-iteration schedules:
   - Expected completed count is deterministic:
     `CompletedIterations * iteration-schedules-per-iteration`.
   - Count completed workflows with:
     `OmesExecutionID = executionID AND OmesScheduleKind = 'iteration' AND ExecutionStatus = 'Completed'`.
   - Wait until count reaches expected.

2. Stable schedules:
   - Call `ListScheduleMatchingTimes` for each stable schedule over `StableWindowStart..StableWindowEnd`.
   - Compute expected workflow starts from matching times, actual workflow intervals, and overlap policy.
   - Assert callback-dependent starts happen:
     `BUFFER_ONE` starts one buffered workflow after completion, and `BUFFER_ALL` drains buffered matches after each completion.
   - Assert at least one completed or running workflow was started per stable schedule.
   - During finalization, release any running stable scheduled workflows.
   - Wait until no workflows with `OmesScheduleKind = 'stable'` are running.
   - Count completed stable scheduled workflows for summary output.

3. Global failure check:
   - Keep `VerifyNoFailedWorkflows` on `OmesExecutionID`.
   - This covers scheduled workflows too, because they will carry the same execution ID.

4. Open workflow check:
   - Add a schedule-specific no-open-workflows check after finalization:
     `OmesExecutionID = executionID AND OmesScheduleKind IN (...) AND ExecutionStatus = 'Running'`.

## Cleanup Strategy

Cleanup order at the end of `Run`:

1. Stop schedule production:
   - Record `StableWindowEnd`.
   - Pause stable schedules.
   - Delete stable schedules.
   - Delete any known per-iteration schedules that remain.

2. Complete scheduled workflows:
   - If completion mode is `release`, signal all running scheduled workflows to set `schedule_release=true`.
   - If completion mode is `timer`, simply wait for timers to finish.

3. Wait for visibility:
   - Wait for no scheduled workflows to remain running.
   - Wait for expected per-iteration scheduled workflows to be completed.
   - Verify matching-times expected starts for stable schedules.
   - Count stable completed workflows for the scenario summary.

4. Run existing failure verification.

This order avoids creating new scheduled workflows while trying to drain old ones.

## Error Handling

Schedule API errors should return from the iteration or scenario finalization, except:

- Delete not-found is harmless.
- Describe not-found is harmless only during cleanup/adoption paths.
- Already-exists on deterministic schedule create should be handled by adopting the schedule, not by failing immediately.

Do not mirror the current `scheduler_stress` pattern of logging schedule operation errors and returning nil. Throughput stress should fail if schedule API calls fail unexpectedly.

## Metrics and Logging

Add summary fields to the existing completion log:

- Stable schedules created.
- Per-iteration schedules created.
- Iteration scheduled workflows completed.
- Stable matching times observed.
- Stable expected starts.
- Stable actual starts.
- Stable scheduled workflows completed.
- Stable scheduled workflows released.
- Schedule API errors, if any.

Optional metrics:

- `omes_schedule_create_count`
- `omes_schedule_update_count`
- `omes_schedule_describe_count`
- `omes_schedule_delete_count`
- `omes_schedule_matching_times_count`
- `omes_schedule_expected_start_count`
- `omes_schedule_workflow_completed_count`

Metrics can be deferred if this needs to stay small.

## Implementation Steps

1. Add schedule config and flags to `scenarios/throughput_stress.go`.
2. Add schedule-specific search attribute constants.
3. Add config parsing and validation.
4. Add helper to build scheduled `kitchenSink` workflow inputs.
5. Add helper to create a `client.ScheduleWorkflowAction` with typed search attributes.
6. Add `tpsScheduleController` with:
   - `StartStableSchedules`
   - `ExecuteIterationSchedules`
   - `FinalizeStableSchedules`
   - `ListMatchingTimes`
   - `VerifyStableScheduleExpectedStarts`
   - `ReleaseRunningScheduledWorkflows`
   - `WaitForScheduledWorkflowsClosed`
7. Refactor the current `KitchenSinkExecutor` usage into a local `GenericExecutor`.
8. Wire per-iteration schedules into the new iteration function.
9. Wire stable schedule start/finalize around the executor run.
10. Extend `tpsState` and snapshot/resume handling.
11. Add the matching-times expected-start simulator.
12. Add schedule-specific verification after the existing visibility min-count check.
13. Update README option docs.
14. Add tests.

## Test Plan

Unit/config tests:

- Invalid negative schedule counts fail.
- Invalid zero/negative durations fail.
- Unknown completion mode fails.
- Unknown overlap policy fails.
- Defaults produce 3 stable schedules when schedules are enabled.
- Stable matching-times verification requires `SKIP`, `BUFFER_ONE`, and `BUFFER_ALL`.

Functional tests:

1. `throughput_stress` without schedules still passes unchanged.
2. `throughput_stress` with schedules enabled:
   - 2 iterations.
   - 3 stable schedules.
   - 1 per-iteration schedule.
   - short intervals and short workflow durations.
   - verify matching-times expected starts for `SKIP`, `BUFFER_ONE`, and `BUFFER_ALL`.
   - verify no failed workflows.
   - verify no running scheduled workflows after finalization.
3. Release-mode test:
   - stable scheduled workflows wait on `schedule_release=true`.
   - finalization signals release.
   - `BUFFER_ONE` and `BUFFER_ALL` start buffered workflows after release.
   - workflows complete.
4. Resume from middle:
   - load state with one completed iteration.
   - ensure skipped iteration schedules are not duplicated.
   - ensure remaining iteration schedules complete.
5. Resume with existing stable schedules:
   - create/adopt deterministic schedule IDs.
   - preserve or reconstruct the observable stable verification window.
   - finalize idempotently.

Recommended initial test command:

```sh
go test ./scenarios -run TestThroughputStress -count=1
```

If the new tests need a dev server and worker, keep durations short:

- `stable-schedule-interval=500ms`
- `stable-schedule-workflow-duration=1s`
- `iteration-schedule-workflow-duration=100ms`
- `visibility-count-timeout=10s`

## Risks and Mitigations

Risk: stable `BUFFER_ALL` schedules in release mode can build a large backlog if the stable window is long.

Mitigation: keep interval conservative by default, cap the stable verification window, and release/drain workflows during finalization.

Risk: schedule recent action history is bounded.

Mitigation: rely on typed search attributes for cleanup and verification.

Risk: matching-times verification can be flaky if the stable window boundaries are ambiguous.

Mitigation: record `StableWindowStart` only after all 3 schedules are created and described, record `StableWindowEnd` before pausing/deleting them, and use a configurable start tolerance.

Risk: adding schedules changes throughput counts.

Mitigation: keep scheduled workflows out of the main throughput denominator by default and report them separately.

Risk: schedule create happens before default search attributes are registered.

Mitigation: explicitly register required search attributes before creating schedules when `include-schedules=true`.

Risk: cross-SDK compatibility.

Mitigation: schedule `kitchenSink` workflows, not Go-only scheduler workflow types.

Risk: resume duplicates per-iteration schedules.

Mitigation: deterministic schedule IDs plus adopt/delete/recreate logic.

## Open Decisions

1. Should scheduled workflows count toward `min-throughput-per-hour`?
   - Recommendation: no. Report them separately.

2. Should stable schedules default to `timer` or `release` completion mode?
   - Recommendation: `release`, because the matching-times callback verification is strongest when the test controls workflow completion.

3. Should per-iteration schedule work run concurrently with the main workflow?
   - Recommendation: start sequential for determinism, then make it concurrent after the first stable implementation.

4. Should schedule helpers be shared with `scheduler_stress`?
   - Recommendation: only extract small shared helpers after this lands. Keep the first change scoped to `throughput_stress`.
