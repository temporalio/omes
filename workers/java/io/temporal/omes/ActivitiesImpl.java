package io.temporal.omes;

import io.temporal.activity.Activity;
import io.temporal.client.WorkflowClient;
import java.util.Random;

public class ActivitiesImpl implements Activities {

  private final WorkflowClient client;
  private final boolean errOnUnimplemented;

  public ActivitiesImpl(WorkflowClient client) {
    this(client, false);
  }

  public ActivitiesImpl(WorkflowClient client, boolean errOnUnimplemented) {
    this.client = client;
    this.errOnUnimplemented = errOnUnimplemented;
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
    ClientActionExecutor executor =
        new ClientActionExecutor(client, workflowId, taskQueue, errOnUnimplemented);
    executor.executeClientSequence(clientActivity.getClientSequence());
  }

  @Override
  public void retryableError(
      io.temporal.omes.KitchenSink.ExecuteActivityAction.RetryableErrorActivity config) {
    var activityInfo = Activity.getExecutionContext().getInfo();
    if (activityInfo.getAttempt() <= config.getFailAttempts()) {
      throw Activity.wrap(new RuntimeException("retryable error"));
    }
  }

  @Override
  public void timeout(io.temporal.omes.KitchenSink.ExecuteActivityAction.TimeoutActivity config)
      throws InterruptedException {
    var activityInfo = Activity.getExecutionContext().getInfo();
    var duration = config.getSuccessDuration();
    if (activityInfo.getAttempt() <= config.getFailAttempts()) {
      // Failure case: run failure duration (exceeds activity timeout)
      duration = config.getFailureDuration();
    }

    // Sleep for failure/success timeout duration.
    // In failure case, this will throw an InterruptedException.
    delay(duration);
  }

  @Override
  public void heartbeat(
      io.temporal.omes.KitchenSink.ExecuteActivityAction.HeartbeatTimeoutActivity config)
      throws InterruptedException {
    var activityInfo = Activity.getExecutionContext().getInfo();
    boolean shouldSendHeartbeats = activityInfo.getAttempt() > config.getFailAttempts();
    var duration = config.getSuccessDuration();
    if (!shouldSendHeartbeats) {
      // Failure case: run failure duration (exceeds heartbeat timeout)
      duration = config.getFailureDuration();
    }
    // Sleep for failure/success timeout duration.
    // In failure case, this will throw an InterruptedException.
    delay(duration);
    // On success, heartbeat
    Activity.getExecutionContext().heartbeat(null);
  }
}
