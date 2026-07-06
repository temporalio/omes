package io.temporal.omes.apps.helloworld;

import io.temporal.workflow.WorkflowInterface;
import io.temporal.workflow.WorkflowMethod;

@WorkflowInterface
public interface HelloWorldWorkflow {
  @WorkflowMethod(name = "HelloWorldWorkflow")
  String run(String name);
}
