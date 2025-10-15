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

  @Override
  public void createScheduleActivity(io.temporal.omes.KitchenSink.CreateScheduleAction action) {
    throw new UnsupportedOperationException("Schedule operations are not yet implemented for Java SDK");
  }

  @Override
  public Object describeScheduleActivity(
      io.temporal.omes.KitchenSink.DescribeScheduleAction action) {
    throw new UnsupportedOperationException("Schedule operations are not yet implemented for Java SDK");
  }

  @Override
  public void updateScheduleActivity(io.temporal.omes.KitchenSink.UpdateScheduleAction action) {
    throw new UnsupportedOperationException("Schedule operations are not yet implemented for Java SDK");
  }

  @Override
  public void deleteScheduleActivity(io.temporal.omes.KitchenSink.DeleteScheduleAction action) {
    throw new UnsupportedOperationException("Schedule operations are not yet implemented for Java SDK");
  }
}
