package io.temporal.omes;

import io.temporal.api.common.v1.WorkflowExecution;
import io.temporal.api.workflowservice.v1.DescribeWorkflowExecutionRequest;
import io.temporal.api.workflowservice.v1.DescribeWorkflowExecutionResponse;
import io.temporal.client.WorkflowClient;
import io.temporal.failure.ApplicationFailure;
import io.temporal.serviceclient.WorkflowServiceStubs;
import java.util.List;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ClientActionExecutor {
  private static final Logger log = LoggerFactory.getLogger(ClientActionExecutor.class);

  private final WorkflowClient client;
  private final String workflowId;
  private final String taskQueue;
  private final WorkflowServiceStubs serviceStubs;

  public ClientActionExecutor(WorkflowClient client, String workflowId, String taskQueue) {
    this.client = client;
    this.workflowId = workflowId;
    this.taskQueue = taskQueue;
    this.serviceStubs = client.getWorkflowServiceStubs();
  }

  public void executeClientSequence(KitchenSink.ClientSequence clientSeq) {
    for (KitchenSink.ClientActionSet actionSet : clientSeq.getActionSetsList()) {
      executeClientActionSet(actionSet);
    }
  }

  private void executeClientActionSet(KitchenSink.ClientActionSet actionSet) {
    List<KitchenSink.ClientAction> actions = actionSet.getActionsList();
    if (actionSet.getConcurrent()) {
      // Execute actions concurrently
      for (KitchenSink.ClientAction action : actions) {
        executeClientAction(action);
      }
    } else {
      // Execute actions sequentially
      for (KitchenSink.ClientAction action : actions) {
        executeClientAction(action);
      }
    }
  }

  private void executeClientAction(KitchenSink.ClientAction action) {
    switch (action.getVariantCase()) {
      case DO_SELF_DESCRIBE:
        executeSelfDescribeAction(action.getDoSelfDescribe());
        break;
      default:
        throw ApplicationFailure.newNonRetryableFailure(
            "Unsupported client action: " + action.getVariantCase(), "UnsupportedAction");
    }
  }

  private void executeSelfDescribeAction(KitchenSink.DoSelfDescribe selfDescribe) {
    try {
      // Use current workflow ID if not specified
      String workflowId = selfDescribe.getWorkflowId();
      if (workflowId.isEmpty()) {
        workflowId = this.workflowId;
      }

      // Get the current workflow execution details
      DescribeWorkflowExecutionRequest request =
          DescribeWorkflowExecutionRequest.newBuilder()
              .setNamespace(selfDescribe.getNamespace())
              .setExecution(
                  WorkflowExecution.newBuilder()
                      .setWorkflowId(workflowId)
                      .setRunId(selfDescribe.getRunId())
                      .build())
              .build();

      DescribeWorkflowExecutionResponse response =
          serviceStubs.blockingStub().describeWorkflowExecution(request);

      // Log the workflow execution details
      log.info("Workflow Execution Details:");
      log.info(
          "  Workflow ID: {}", response.getWorkflowExecutionInfo().getExecution().getWorkflowId());
      log.info("  Run ID: {}", response.getWorkflowExecutionInfo().getExecution().getRunId());
      log.info("  Type: {}", response.getWorkflowExecutionInfo().getType().getName());
      log.info("  Status: {}", response.getWorkflowExecutionInfo().getStatus());
      log.info("  Start Time: {}", response.getWorkflowExecutionInfo().getStartTime());
      if (response.getWorkflowExecutionInfo().hasCloseTime()) {
        log.info("  Close Time: {}", response.getWorkflowExecutionInfo().getCloseTime());
      }
      log.info("  History Length: {}", response.getWorkflowExecutionInfo().getHistoryLength());
      log.info("  Task Queue: {}", response.getWorkflowExecutionInfo().getTaskQueue());
    } catch (Exception e) {
      throw ApplicationFailure.newNonRetryableFailure(
          "Failed to describe workflow execution: " + e.getMessage(),
          "DescribeWorkflowExecutionFailed");
    }
  }
}
