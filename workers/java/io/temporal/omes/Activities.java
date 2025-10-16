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

  @ActivityMethod(name = "retryable_error")
  void retryableError(
      io.temporal.omes.KitchenSink.ExecuteActivityAction.RetryableErrorActivity config);

  @ActivityMethod(name = "timeout")
  void timeout(io.temporal.omes.KitchenSink.ExecuteActivityAction.TimeoutActivity config)
      throws InterruptedException;

  @ActivityMethod(name = "heartbeat")
  void heartbeat(io.temporal.omes.KitchenSink.ExecuteActivityAction.HeartbeatTimeoutActivity config)
      throws InterruptedException;
}
