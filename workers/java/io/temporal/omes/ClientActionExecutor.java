package io.temporal.omes;

import io.temporal.api.enums.v1.WorkflowIdConflictPolicy;
import io.temporal.client.UpdateOptions;
import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowOptions;
import io.temporal.client.WorkflowStub;
import io.temporal.failure.ApplicationFailure;

public class ClientActionExecutor {

  private final WorkflowClient client;
  private final String workflowId;
  private final String workflowType = "kitchenSink";
  private final Object workflowInput = null;
  private final String taskQueue;
  private final boolean errOnUnimplemented;

  public ClientActionExecutor(WorkflowClient client, String workflowId, String taskQueue) {
    this(client, workflowId, taskQueue, false);
  }

  public ClientActionExecutor(
      WorkflowClient client, String workflowId, String taskQueue, boolean errOnUnimplemented) {
    this.client = client;
    this.workflowId = workflowId;
    this.taskQueue = taskQueue;
    this.errOnUnimplemented = errOnUnimplemented;
  }

  public void executeClientSequence(KitchenSink.ClientSequence clientSeq) {
    for (KitchenSink.ClientActionSet actionSet : clientSeq.getActionSetsList()) {
      executeClientActionSet(actionSet);
    }
  }

  private void executeClientActionSet(KitchenSink.ClientActionSet actionSet) {
    if (actionSet.getConcurrent()) {
      if (errOnUnimplemented) {
        throw ApplicationFailure.newNonRetryableFailure(
            "concurrent client actions are not supported", "UnsupportedOperation");
      }
      // Skip concurrent actions when not erroring on unimplemented
      System.out.println("Skipping concurrent client actions (not implemented)");
      return;
    }

    for (KitchenSink.ClientAction action : actionSet.getActionsList()) {
      executeClientAction(action);
    }
  }

  private void executeClientAction(KitchenSink.ClientAction action) {
    if (action.hasDoSignal()) {
      executeSignalAction(action.getDoSignal());
    } else if (action.hasDoUpdate()) {
      executeUpdateAction(action.getDoUpdate());
    } else if (action.hasDoQuery()) {
      executeQueryAction(action.getDoQuery());
    } else if (action.hasNestedActions()) {
      executeClientActionSet(action.getNestedActions());
    } else {
      throw new IllegalArgumentException("Client action must have a recognized variant");
    }
  }

  private void executeSignalAction(KitchenSink.DoSignal signal) {
    String signalName;
    Object signalArgs;

    if (signal.hasDoSignalActions()) {
      signalName = "do_actions_signal";
      signalArgs = signal.getDoSignalActions();
    } else if (signal.hasCustom()) {
      signalName = signal.getCustom().getName();
      signalArgs = signal.getCustom().getArgsList().toArray();
    } else {
      throw new IllegalArgumentException("DoSignal must have a recognizable variant");
    }

    if (signal.getWithStart()) {
      WorkflowOptions options =
          WorkflowOptions.newBuilder()
              .setWorkflowId(workflowId)
              .setTaskQueue(taskQueue)
              .setWorkflowIdConflictPolicy(
                  WorkflowIdConflictPolicy.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING)
              .build();
      WorkflowStub stub = client.newUntypedWorkflowStub(workflowType, options);
      stub.signalWithStart(signalName, new Object[] {signalArgs}, new Object[] {workflowInput});
    } else {
      WorkflowStub stub = client.newUntypedWorkflowStub(workflowId);
      stub.signal(signalName, signalArgs);
    }
  }

  private void executeUpdateAction(KitchenSink.DoUpdate update) {
    String updateName;
    Object updateArgs;

    if (update.hasDoActions()) {
      updateName = "do_actions_update";
      updateArgs = update.getDoActions();
    } else if (update.hasCustom()) {
      updateName = update.getCustom().getName();
      updateArgs = update.getCustom().getArgsList().toArray();
    } else {
      throw new IllegalArgumentException("DoUpdate must have a recognizable variant");
    }

    try {
      if (update.getWithStart()) {
        WorkflowOptions workflowOptions =
            WorkflowOptions.newBuilder()
                .setWorkflowId(workflowId)
                .setTaskQueue(taskQueue)
                .setWorkflowIdConflictPolicy(
                    WorkflowIdConflictPolicy.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING)
                .build();
        WorkflowStub stub = client.newUntypedWorkflowStub(workflowType, workflowOptions);

        UpdateOptions<KitchenSink.WorkflowState> updateOptions =
            UpdateOptions.newBuilder(KitchenSink.WorkflowState.class)
                .setUpdateName(updateName)
                .build();

        stub.executeUpdateWithStart(
            updateOptions, new Object[] {updateArgs}, new Object[] {workflowInput});
      } else {
        WorkflowStub stub = client.newUntypedWorkflowStub(workflowId);
        stub.update(updateName, KitchenSink.WorkflowState.class, updateArgs);
      }
    } catch (Exception e) {
      if (!update.getFailureExpected()) {
        throw e;
      }
      // If failure was expected, swallow the exception
    }
  }

  private void executeQueryAction(KitchenSink.DoQuery query) {
    try {
      WorkflowStub stub = client.newUntypedWorkflowStub(workflowId);

      if (query.hasReportState()) {
        stub.query("report_state", KitchenSink.WorkflowState.class, (Object) null);
      } else if (query.hasCustom()) {
        String queryName = query.getCustom().getName();
        Object[] queryArgs = query.getCustom().getArgsList().toArray();
        stub.query(queryName, Object.class, queryArgs);
      } else {
        throw new IllegalArgumentException("DoQuery must have a recognizable variant");
      }
    } catch (Exception e) {
      if (!query.getFailureExpected()) {
        throw e;
      }
      // If failure was expected, swallow the exception
    }
  }
}
