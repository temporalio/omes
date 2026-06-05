package io.temporal.omes.apps.helloworld;

import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowOptions;
import io.temporal.omes.harness.Harness;
import io.temporal.omes.harness.HarnessClients;
import io.temporal.omes.harness.ProjectHarness;
import io.temporal.omes.harness.WorkerHarness;
import io.temporal.worker.Worker;

public final class HelloWorldApp {
  public static final Harness.App APP =
      new Harness.App(
          HelloWorldApp::configureWorker,
          HarnessClients::defaultClientFactory,
          new ProjectHarness.ProjectHandlers(HelloWorldApp::executeProjectIteration));

  private HelloWorldApp() {}

  private static void configureWorker(
      WorkflowClient client, Worker worker, WorkerHarness.WorkerContext context) {
    worker.registerWorkflowImplementationTypes(HelloWorldWorkflowImpl.class);
  }

  private static void executeProjectIteration(
      WorkflowClient client, ProjectHarness.ProjectExecuteContext context) {
    HelloWorldWorkflow workflow =
        client.newWorkflowStub(
            HelloWorldWorkflow.class,
            WorkflowOptions.newBuilder()
                .setWorkflowId(String.format("%s-%d", context.run.executionId, context.iteration))
                .setTaskQueue(context.taskQueue)
                .build());
    String result = workflow.run("World");
    System.out.println(result);
  }
}
