package io.temporal.omes.apps.helloworld;

public class HelloWorldWorkflowImpl implements HelloWorldWorkflow {
  @Override
  public String run(String name) {
    return String.format("Hello %s", name);
  }
}
