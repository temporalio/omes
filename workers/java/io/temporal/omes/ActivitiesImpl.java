package io.temporal.omes;

import io.temporal.activity.Activity;
import io.temporal.client.WorkflowClient;
import java.util.Random;

public class ActivitiesImpl implements Activities {

  private final WorkflowClient client;

  public ActivitiesImpl(WorkflowClient client) {
    this.client = client;
  }

  @Override
  public void noop() {}

  @Override
  public void delay(com.google.protobuf.Duration d) throws InterruptedException {
    Thread.sleep(1000 * d.getSeconds() + d.getNanos() / 1_000_000);
  }

  @Override
  public byte[] payload(byte[] inputData, int bytesToReturn) {
    byte[] output = new byte[bytesToReturn];
    new Random().nextBytes(output);
    return output;
  }

  @Override
  public void client(
      io.temporal.omes.KitchenSink.ExecuteActivityAction.ClientActivity clientActivity) {
    var activityInfo = Activity.getExecutionContext().getInfo();
    String workflowId = activityInfo.getWorkflowId();
    String taskQueue = activityInfo.getActivityTaskQueue();
    ClientActionExecutor executor = new ClientActionExecutor(client, workflowId, taskQueue);
    executor.executeClientSequence(clientActivity.getClientSequence());
  }
}
