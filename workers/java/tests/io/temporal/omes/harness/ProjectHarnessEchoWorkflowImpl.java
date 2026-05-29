package io.temporal.omes.harness;

public final class ProjectHarnessEchoWorkflowImpl implements ProjectHarnessEchoWorkflow {
  public ProjectHarnessEchoWorkflowImpl() {}

  @Override
  public String run(String payload) {
    return payload;
  }
}
