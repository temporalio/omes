package io.temporal.omes;

public class ActivitiesImpl implements Activities {

  @Override
  public void noop() {}

  @Override
  public void delay(com.google.protobuf.Duration d) throws InterruptedException {
    Thread.sleep(1000 * d.getSeconds() + d.getNanos() / 1_000_000);
  }
}
