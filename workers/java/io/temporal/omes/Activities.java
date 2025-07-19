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
}
