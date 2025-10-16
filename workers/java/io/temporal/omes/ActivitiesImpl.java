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
    var durationMs = activityInfo.getStartToCloseTimeout().toMillis();
    if (activityInfo.getAttempt() <= config.getFailAttempts()) {
      // Failure case: run for double StartToCloseTimeout
      durationMs *= 2;
    } else {
      // Success case: run for half StartToCloseTimeout
      durationMs /= 2;
    }

    // Sleep for failure/success timeout duration.
    // In failure case, this will throw an InterruptedException.
    Thread.sleep(durationMs);
  }

  @Override
  public void heartbeat(
      io.temporal.omes.KitchenSink.ExecuteActivityAction.HeartbeatTimeoutActivity config)
      throws InterruptedException {
    var activityInfo = Activity.getExecutionContext().getInfo();
    boolean shouldSendHeartbeats = activityInfo.getAttempt() > config.getFailAttempts();

    // Run activity for 2x the heartbeat timeout
    // Ensures we miss enough heartbeat intervals (if not sending heartbeats).
    long durationMs = activityInfo.getHeartbeatTimeout().toMillis() * 2;

    // Unified loop with conditional heartbeat sending
    long elapsed = 0;
    long heartbeatInterval = 1000; // Send heartbeat every second
    while (elapsed < durationMs) {
      long sleepTime = Math.min(heartbeatInterval, durationMs - elapsed);
      Thread.sleep(sleepTime);
      elapsed += sleepTime;
      if (shouldSendHeartbeats && elapsed < durationMs) {
        Activity.getExecutionContext().heartbeat(null);
      }
    }
  }
}
