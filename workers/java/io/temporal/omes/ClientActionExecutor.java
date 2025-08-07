package io.temporal.omes;

import io.temporal.client.WorkflowClient;
import io.temporal.failure.ApplicationFailure;

public class ClientActionExecutor {

  public ClientActionExecutor(WorkflowClient client, String workflowId, String taskQueue) {}

  public void executeClientSequence(KitchenSink.ClientSequence clientSeq) {
    throw ApplicationFailure.newNonRetryableFailure(
        "client actions activity is not supported", "UnsupportedOperation");
  }
}
