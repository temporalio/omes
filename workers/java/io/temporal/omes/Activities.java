package io.temporal.omes;

import io.temporal.activity.ActivityInterface;
import io.temporal.activity.ActivityMethod;

@ActivityInterface
public interface Activities {
  @ActivityMethod(name = "noop")
  void noop();

  @ActivityMethod(name = "delay")
  void delay(com.google.protobuf.Duration d) throws InterruptedException;

  @ActivityMethod(name = "payload")
  byte[] payload(byte[] inputData, int bytesToReturn);

  @ActivityMethod(name = "client")
  void client(io.temporal.omes.KitchenSink.ExecuteActivityAction.ClientActivity clientActivity);

  @ActivityMethod(name = "CreateScheduleActivity")
  void createScheduleActivity(io.temporal.omes.KitchenSink.CreateScheduleAction action);

  @ActivityMethod(name = "DescribeScheduleActivity")
  Object describeScheduleActivity(io.temporal.omes.KitchenSink.DescribeScheduleAction action);

  @ActivityMethod(name = "UpdateScheduleActivity")
  void updateScheduleActivity(io.temporal.omes.KitchenSink.UpdateScheduleAction action);

  @ActivityMethod(name = "DeleteScheduleActivity")
  void deleteScheduleActivity(io.temporal.omes.KitchenSink.DeleteScheduleAction action);
}
