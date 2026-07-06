package io.temporal.omes.harness;

import io.temporal.workflow.WorkflowInterface;
import io.temporal.workflow.WorkflowMethod;

@WorkflowInterface
public interface ProjectHarnessEchoWorkflow {
  @WorkflowMethod
  String run(String payload);
}
