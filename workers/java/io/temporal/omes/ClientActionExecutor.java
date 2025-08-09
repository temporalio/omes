package io.temporal.omes;

import io.temporal.client.WorkflowClient;
import io.temporal.failure.ApplicationFailure;

public class ClientActionExecutor {

  private final WorkflowClient client;
  private String workflowId;
  private final String taskQueue;
  private String runId = "";
  private final String workflowType = "kitchenSink";
  private Object workflowInput = null;

  public ClientActionExecutor(WorkflowClient client, String workflowId, String taskQueue) {
    this.client = client;
    this.workflowId = workflowId;
    this.taskQueue = taskQueue;
  }

  public void executeClientSequence(KitchenSink.ClientSequence clientSeq) {
    throw ApplicationFailure.newNonRetryableFailure(
        "client actions are not supported", "UnsupportedOperation");
  }
}
