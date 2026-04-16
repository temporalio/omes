# Nexus Attach-Callbacks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Nexus attach-callbacks action to the throughput stress scenario, matching bench-go's `createNexusOperationAttachCallbacks()` behavior.

**Architecture:** A new `ExecuteNexusOperationAttachCallbacks` proto action type (field 16 in `Action.oneof`) drives the feature. The kitchen sink workflow handler starts N concurrent Nexus operations targeting the same handler workflow (via a new `workflow_id_override` field on `NexusHandlerInput` + `USE_EXISTING` conflict policy), waits for all to start, signals the handler to unblock (via new `wait_for_signal` field), then waits for all to complete. Only Go and Python implement Nexus operations; other SDKs add a "not supported" stub.

**Tech Stack:** Protobuf, Go (Temporal SDK), Python (Temporal SDK), Rust (prost build for proto gen)

**Reference:** bench-go implementation at `workflows/configdriven/workflow.go:512-566` and `workflows/configdriven/prebuiltworkflows.go:412-424`.

---

### Task 1: Proto Changes

**Files:**
- Modify: `workers/proto/kitchen_sink/kitchen_sink.proto`

- [ ] **Step 1: Add `wait_for_signal` and `workflow_id_override` fields to `NexusHandlerInput`**

In `workers/proto/kitchen_sink/kitchen_sink.proto`, find the `NexusHandlerInput` message (lines 505-509) and add two fields:

```protobuf
// Input for the Nexus handler workflow that backs echo-sync and echo-async operations
message NexusHandlerInput {
  string input = 1;
  repeated ActionSet before_actions = 2;
  // If true, the handler workflow will wait for an "unblock" signal before returning.
  bool wait_for_signal = 3;
  // If set, the operation handler uses this as the workflow ID (instead of the Nexus request ID)
  // and sets WorkflowIDConflictPolicy to USE_EXISTING.
  string workflow_id_override = 4;
}
```

- [ ] **Step 2: Add `ExecuteNexusOperationAttachCallbacks` message**

Add the following message after the `NexusHandlerInput` message (after line 509):

```protobuf
// Starts multiple Nexus operations concurrently, all targeting the same handler workflow.
// Waits for all operations to start (attaching callbacks), signals the handler to unblock,
// then waits for all to complete.
message ExecuteNexusOperationAttachCallbacks {
  string endpoint = 1;
  // Number of concurrent Nexus operations to start. All target the same handler workflow.
  int32 num_operations = 2;
  // Input string echoed back by the handler.
  string input = 3;
}
```

- [ ] **Step 3: Add the new action to `Action.oneof variant`**

Find the `Action` message (line 153) and add field 16:

```protobuf
  oneof variant {
    TimerAction timer = 1;
    ExecuteActivityAction exec_activity = 2;
    ExecuteChildWorkflowAction exec_child_workflow = 3;
    AwaitWorkflowState await_workflow_state = 4;
    SendSignalAction send_signal = 5;
    CancelWorkflowAction cancel_workflow = 6;
    SetPatchMarkerAction set_patch_marker = 7;
    UpsertSearchAttributesAction upsert_search_attributes = 8;
    UpsertMemoAction upsert_memo = 9;
    WorkflowState set_workflow_state = 10;

    ReturnResultAction return_result = 11;
    ReturnErrorAction return_error = 12;
    ContinueAsNewAction continue_as_new = 13;

    ActionSet nested_action_set = 14;

    ExecuteNexusOperation nexus_operation = 15;
    ExecuteNexusOperationAttachCallbacks nexus_operation_attach_callbacks = 16;
  }
```

- [ ] **Step 4: Commit**

```bash
git add workers/proto/kitchen_sink/kitchen_sink.proto
git commit -m "proto: add ExecuteNexusOperationAttachCallbacks action and NexusHandlerInput fields"
```

---

### Task 2: Regenerate Proto Code

**Files:**
- Modified by codegen: `loadgen/kitchensink/kitchen_sink.pb.go`, `workers/python/protos/kitchen_sink_pb2.py`, `workers/python/protos/kitchen_sink_pb2.pyi`, `workers/java/io/temporal/omes/KitchenSink.java`, `workers/dotnet/Temporalio.Omes/protos/KitchenSink.cs`
- Manually regenerate: `workers/typescript/` (via npm), `workers/ruby/protos/kitchen_sink/kitchen_sink_pb.rb`

- [ ] **Step 1: Run proto generation**

```bash
go run ./cmd/dev build-proto
```

This requires `cargo`, `protoc`, and `node` to be installed. It generates code for Go, Python, Java, and .NET via the Rust build script, and TypeScript via npm.

- [ ] **Step 2: Regenerate Ruby proto**

Ruby protos are not part of the automated build. Regenerate manually:

```bash
cd workers/ruby
grpc_tools_ruby_protoc \
  -I ../proto/api_upstream \
  -I ../proto/kitchen_sink \
  --ruby_out=protos \
  ../proto/kitchen_sink/kitchen_sink.proto
```

If `grpc_tools_ruby_protoc` is not available, use `protoc` with the Ruby plugin, or copy the pattern from the existing Ruby proto file to add the new message/fields manually.

- [ ] **Step 3: Verify Go code compiles**

```bash
go build ./...
```

- [ ] **Step 4: Commit all generated code**

```bash
git add loadgen/kitchensink/kitchen_sink.pb.go
git add workers/python/protos/kitchen_sink_pb2.py workers/python/protos/kitchen_sink_pb2.pyi
git add workers/java/io/temporal/omes/KitchenSink.java
git add workers/dotnet/Temporalio.Omes/protos/KitchenSink.cs
git add workers/typescript/
git add workers/ruby/protos/
git commit -m "codegen: regenerate proto code for all SDKs"
```

---

### Task 3: Go — Modify EchoAsyncOperation and NexusHandlerWorkflow

**Files:**
- Modify: `workers/go/kitchensink/kitchen_sink.go`

- [ ] **Step 1: Modify `NexusHandlerWorkflow` to support `wait_for_signal`**

In `workers/go/kitchensink/kitchen_sink.go`, find `NexusHandlerWorkflow` (line 593) and add signal waiting:

```go
func NexusHandlerWorkflow(ctx workflow.Context, input *kitchensink.NexusHandlerInput) (string, error) {
	state := KSWorkflowState{
		workflowState: &kitchensink.WorkflowState{},
	}
	for _, actionSet := range input.BeforeActions {
		if _, err := state.handleActionSet(ctx, actionSet); err != nil {
			return "", err
		}
	}
	if input.WaitForSignal {
		workflow.GetSignalChannel(ctx, "unblock").Receive(ctx, nil)
	}
	return input.Input, nil
}
```

- [ ] **Step 2: Modify `EchoAsyncOperation` to support `workflow_id_override`**

Find `EchoAsyncOperation` (line 613) and update it to use `workflow_id_override` when set:

```go
var EchoAsyncOperation = temporalnexus.NewWorkflowRunOperation("echo-async", NexusHandlerWorkflow, func(ctx context.Context, input *kitchensink.NexusHandlerInput, opts nexus.StartOperationOptions) (client.StartWorkflowOptions, error) {
	startOpts := client.StartWorkflowOptions{
		ID: opts.RequestID,
	}
	if input.WorkflowIdOverride != "" {
		startOpts.ID = input.WorkflowIdOverride
		startOpts.WorkflowIDConflictPolicy = enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING
	}
	return startOpts, nil
})
```

- [ ] **Step 3: Add `enumspb` import**

Add the import for `enumspb` to the import block at the top of the file (line 3):

```go
import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/temporalnexus"
	"go.temporal.io/sdk/workflow"
)
```

Note: also add `"github.com/google/uuid"` — needed in Task 4 for generating unique workflow IDs. If this dependency doesn't exist, use `workflow.SideEffect` with `rand` instead (see Task 4 step 1 for alternative).

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

If `github.com/google/uuid` is not already a dependency, run:
```bash
go get github.com/google/uuid
```

Or skip the uuid import and use a simpler approach in Task 4 (generate ID from workflow info + side effect).

- [ ] **Step 5: Commit**

```bash
git add workers/go/kitchensink/kitchen_sink.go
git commit -m "go: support workflow_id_override and wait_for_signal in NexusHandlerWorkflow"
```

---

### Task 4: Go — Implement `handleNexusOperationAttachCallbacks`

**Files:**
- Modify: `workers/go/kitchensink/kitchen_sink.go`

- [ ] **Step 1: Add the handler function**

Add the following function after `handleNexusOperation` (after line 504):

```go
func handleNexusOperationAttachCallbacks(
	ctx workflow.Context,
	action *kitchensink.ExecuteNexusOperationAttachCallbacks,
) error {
	numOps := int(action.NumOperations)
	if numOps <= 0 {
		numOps = 3
	}

	// Generate a unique workflow ID for the handler workflow using SideEffect.
	var handlerWfID string
	if err := workflow.SideEffect(ctx, func(_ workflow.Context) interface{} {
		return fmt.Sprintf("nexus-handler-attach-callbacks-%s", uuid.NewString())
	}).Get(&handlerWfID); err != nil {
		return err
	}

	client := workflow.NewNexusClient(action.Endpoint, KitchenSinkServiceName)
	input := &kitchensink.NexusHandlerInput{
		Input:              action.Input,
		WaitForSignal:      true,
		WorkflowIdOverride: handlerWfID,
	}

	// Start all Nexus operations concurrently.
	opFutures := make([]workflow.NexusOperationFuture, 0, numOps)
	for i := 0; i < numOps; i++ {
		fut := client.ExecuteOperation(ctx, "echo-async", input, workflow.NexusOperationOptions{})
		opFutures = append(opFutures, fut)
	}

	// Wait for all operations to start (callbacks are now attached to the handler workflow).
	for _, fut := range opFutures {
		if err := fut.GetNexusOperationExecution().Get(ctx, nil); err != nil {
			return fmt.Errorf("unexpected error starting Nexus operation: %w", err)
		}
	}

	// Signal the handler workflow to unblock.
	signalFut := workflow.SignalExternalWorkflow(ctx, handlerWfID, "", "unblock", nil)
	if err := signalFut.Get(ctx, nil); err != nil {
		return fmt.Errorf("failed to signal handler workflow %s: %w", handlerWfID, err)
	}

	// Wait for all operations to complete.
	for _, fut := range opFutures {
		if err := fut.Get(ctx, nil); err != nil {
			return fmt.Errorf("Nexus operation failed: %w", err)
		}
	}

	return nil
}
```

If `uuid` is not available as a dependency, replace the SideEffect body with:
```go
return fmt.Sprintf("nexus-handler-attach-callbacks-%d", rand.Int63())
```
(The `rand` package is already imported.)

- [ ] **Step 2: Add dispatch case in `handleAction`**

In the `handleAction` function (line 261), find the existing nexus operation case (line 340):

```go
	} else if nexusOp := action.GetNexusOperation(); nexusOp != nil {
		return nil, handleNexusOperation(ctx, nexusOp, ws)
	} else {
```

Add the new case before the `else`:

```go
	} else if nexusOp := action.GetNexusOperation(); nexusOp != nil {
		return nil, handleNexusOperation(ctx, nexusOp, ws)
	} else if attachCb := action.GetNexusOperationAttachCallbacks(); attachCb != nil {
		return nil, handleNexusOperationAttachCallbacks(ctx, attachCb)
	} else {
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add workers/go/kitchensink/kitchen_sink.go
git commit -m "go: implement handleNexusOperationAttachCallbacks in kitchen sink workflow"
```

---

### Task 5: Python — Modify Operation and Handler Workflow

**Files:**
- Modify: `workers/python/nexus_service.py`
- Modify: `workers/python/kitchen_sink.py`

- [ ] **Step 1: Modify `echo_async` operation in `nexus_service.py`**

Update the `echo_async` method to use `workflow_id_override` when set:

```python
@nexus.workflow_run_operation
async def echo_async(
    self, ctx: nexus.WorkflowRunOperationContext, input: NexusHandlerInput
) -> nexus.WorkflowHandle[str]:
    wf_id = ctx.request_id
    id_conflict_policy = (
        temporalio.common.WorkflowIDConflictPolicy.FAIL
    )
    if input.workflow_id_override:
        wf_id = input.workflow_id_override
        id_conflict_policy = (
            temporalio.common.WorkflowIDConflictPolicy.USE_EXISTING
        )
    return await ctx.start_workflow(
        NexusHandlerWorkflow.run,
        input,
        id=wf_id,
        id_conflict_policy=id_conflict_policy,
    )
```

Add the required import at the top of `nexus_service.py`:

```python
import temporalio.common
```

- [ ] **Step 2: Modify `NexusHandlerWorkflow` in `kitchen_sink.py`**

Find `NexusHandlerWorkflow` (line 401) and add signal waiting:

```python
@workflow.defn
class NexusHandlerWorkflow:
    @workflow.run
    async def run(self, input: NexusHandlerInput) -> str:
        state = KitchenSinkWorkflow()
        for action_set in input.before_actions:
            await state.handle_action_set(action_set)
        if input.wait_for_signal:
            await workflow.wait_condition(
                lambda: self._unblocked
            )
        return input.input

    _unblocked: bool = False

    @workflow.signal(name="unblock")
    async def unblock(self) -> None:
        self._unblocked = True
```

- [ ] **Step 3: Verify Python syntax**

```bash
cd workers/python && python -c "import kitchen_sink; import nexus_service" && cd ../..
```

- [ ] **Step 4: Commit**

```bash
git add workers/python/nexus_service.py workers/python/kitchen_sink.py
git commit -m "python: support workflow_id_override and wait_for_signal in NexusHandlerWorkflow"
```

---

### Task 6: Python — Implement Attach-Callbacks Handler

**Files:**
- Modify: `workers/python/kitchen_sink.py`

- [ ] **Step 1: Add the handler function**

Add the following function after `handle_nexus_operation` (after line 334 in `kitchen_sink.py`):

```python
async def handle_nexus_operation_attach_callbacks(
    action: ExecuteNexusOperationAttachCallbacks,
):
    num_ops = action.num_operations if action.num_operations > 0 else 3

    # Generate a unique handler workflow ID
    handler_wf_id = f"nexus-handler-attach-callbacks-{workflow.uuid4()}"

    client = workflow.create_nexus_client(
        endpoint=action.endpoint,
        service=KITCHEN_SINK_SERVICE_NAME,
    )
    op_input = NexusHandlerInput(
        input=action.input,
        wait_for_signal=True,
        workflow_id_override=handler_wf_id,
    )

    # Start all Nexus operations concurrently
    handles = []
    for _ in range(num_ops):
        handle = await client.start_operation(
            "echo-async",
            op_input,
            output_type=str,
        )
        handles.append(handle)

    # Signal the handler workflow to unblock
    ext_handle = workflow.get_external_workflow_handle(handler_wf_id)
    await ext_handle.signal("unblock")

    # Wait for all operations to complete
    for handle in handles:
        await handle
```

- [ ] **Step 2: Add the import for `ExecuteNexusOperationAttachCallbacks`**

At the top of `kitchen_sink.py`, find the existing proto imports and add `ExecuteNexusOperationAttachCallbacks`:

```python
from protos.kitchen_sink_pb2 import (
    ...
    ExecuteNexusOperationAttachCallbacks,
    ...
)
```

- [ ] **Step 3: Add dispatch case in `handle_action`**

Find the existing nexus operation dispatch (line 195-196):

```python
        elif action.HasField("nexus_operation"):
            await handle_nexus_operation(action.nexus_operation)
```

Add the new case after it:

```python
        elif action.HasField("nexus_operation"):
            await handle_nexus_operation(action.nexus_operation)
        elif action.HasField("nexus_operation_attach_callbacks"):
            await handle_nexus_operation_attach_callbacks(
                action.nexus_operation_attach_callbacks
            )
```

- [ ] **Step 4: Commit**

```bash
git add workers/python/kitchen_sink.py
git commit -m "python: implement handle_nexus_operation_attach_callbacks"
```

---

### Task 7: Other SDKs — Add "Not Supported" Stubs

**Files:**
- Modify: `workers/typescript/src/workflows/kitchen_sink.ts`
- Modify: `workers/java/io/temporal/omes/KitchenSinkWorkflowImpl.java`
- Modify: `workers/dotnet/Temporalio.Omes/KitchenSinkWorkflow.cs`
- Modify: `workers/ruby/kitchen_sink.rb`

These SDKs don't implement Nexus operations. Add a "not supported" case for the new action, following the same pattern as the existing `nexus_operation` stub.

- [ ] **Step 1: TypeScript**

In `workers/typescript/src/workflows/kitchen_sink.ts`, find line 202-203:

```typescript
    } else if (action.nexusOperation) {
      throw ApplicationFailure.nonRetryable('ExecuteNexusOperation is not supported');
```

Add after it:

```typescript
    } else if (action.nexusOperationAttachCallbacks) {
      throw ApplicationFailure.nonRetryable('ExecuteNexusOperationAttachCallbacks is not supported');
```

- [ ] **Step 2: Java**

In `workers/java/io/temporal/omes/KitchenSinkWorkflowImpl.java`, find line 233-234:

```java
    } else if (action.hasNexusOperation()) {
      throw ApplicationFailure.newNonRetryableFailure("ExecuteNexusOperation is not supported", "");
```

Add after it:

```java
    } else if (action.hasNexusOperationAttachCallbacks()) {
      throw ApplicationFailure.newNonRetryableFailure("ExecuteNexusOperationAttachCallbacks is not supported", "");
```

- [ ] **Step 3: .NET**

In `workers/dotnet/Temporalio.Omes/KitchenSinkWorkflow.cs`, find line 275-278:

```csharp
        else if (action.NexusOperation is { })
        {
            throw new ApplicationFailureException("ExecuteNexusOperation is not supported", nonRetryable: true);
        }
```

Add after it:

```csharp
        else if (action.NexusOperationAttachCallbacks is { })
        {
            throw new ApplicationFailureException("ExecuteNexusOperationAttachCallbacks is not supported", nonRetryable: true);
        }
```

- [ ] **Step 4: Ruby**

In `workers/ruby/kitchen_sink.rb`, find line 181-185:

```ruby
    when :nexus_operation
      raise Temporalio::Error::ApplicationError.new(
        'ExecuteNexusOperation is not supported',
        non_retryable: true
      )
```

Add after it:

```ruby
    when :nexus_operation_attach_callbacks
      raise Temporalio::Error::ApplicationError.new(
        'ExecuteNexusOperationAttachCallbacks is not supported',
        non_retryable: true
      )
```

- [ ] **Step 5: Commit**

```bash
git add workers/typescript/src/workflows/kitchen_sink.ts
git add workers/java/io/temporal/omes/KitchenSinkWorkflowImpl.java
git add workers/dotnet/Temporalio.Omes/KitchenSinkWorkflow.cs
git add workers/ruby/kitchen_sink.rb
git commit -m "all sdks: add not-supported stub for ExecuteNexusOperationAttachCallbacks"
```

---

### Task 8: Wire Into Throughput Stress Scenario

**Files:**
- Modify: `scenarios/throughput_stress.go`

- [ ] **Step 1: Add `createNexusAttachCallbacksAction` helper**

Add the following method after `createNexusWaitForCancelAction` (after line 633):

```go
func (t *tpsExecutor) createNexusAttachCallbacksAction() *Action {
	return &Action{
		Variant: &Action_NexusOperationAttachCallbacks{
			NexusOperationAttachCallbacks: &ExecuteNexusOperationAttachCallbacks{
				Endpoint:      t.config.NexusEndpoint,
				NumOperations: 3,
				Input:         "hello",
			},
		},
	}
}
```

- [ ] **Step 2: Wire into async actions**

In `createActionsChunk`, find the Nexus operations block (around line 421):

```go
		// Add Nexus operations, if configured.
		if t.config.NexusEndpoint != "" {
			asyncActions = append(asyncActions, t.createNexusEchoSyncAction())
			asyncActions = append(asyncActions, t.createNexusEchoAsyncAction())
		}
```

Add the attach-callbacks action:

```go
		// Add Nexus operations, if configured.
		if t.config.NexusEndpoint != "" {
			asyncActions = append(asyncActions, t.createNexusEchoSyncAction())
			asyncActions = append(asyncActions, t.createNexusEchoAsyncAction())
			asyncActions = append(asyncActions, t.createNexusAttachCallbacksAction())
		}
```

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add scenarios/throughput_stress.go
git commit -m "scenario: add Nexus attach-callbacks action to throughput stress"
```

---

### Task 9: Register NexusHandlerWorkflow for Signal Support (Go Worker)

**Files:**
- Modify: `workers/go/worker/worker.go`

- [ ] **Step 1: Verify NexusHandlerWorkflow registration**

Check `workers/go/worker/worker.go` (around line 103) to confirm `NexusHandlerWorkflow` is already registered:

```go
w.RegisterWorkflow(kitchensink.NexusHandlerWorkflow)
```

This should already be there. No changes needed unless it's missing. The `wait_for_signal` change in Task 3 is backward compatible — existing callers don't set `WaitForSignal`, so the signal receive is skipped.

- [ ] **Step 2: Verify Python worker registration**

Check that the Python worker registers `NexusHandlerWorkflow`. It should already be registered. The `unblock` signal handler added in Task 5 is part of the workflow class definition so it's automatically available.

---

### Task 10: Final Verification

- [ ] **Step 1: Full Go build**

```bash
go build ./...
```

- [ ] **Step 2: Run Go tests**

```bash
go test ./...
```

- [ ] **Step 3: Verify the scenario can be listed**

```bash
go run ./cmd list-scenarios
```

Confirm `throughput_stress` appears with its description.
