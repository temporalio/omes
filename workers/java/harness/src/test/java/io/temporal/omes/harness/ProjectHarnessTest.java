package io.temporal.omes.harness;

import static java.nio.charset.StandardCharsets.UTF_8;
import static org.junit.jupiter.api.Assertions.assertArrayEquals;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertSame;

import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowOptions;
import io.temporal.testing.TestWorkflowEnvironment;
import io.temporal.worker.Worker;
import java.util.ArrayList;
import java.util.List;
import org.junit.jupiter.api.Test;

class ProjectHarnessTest {
  @Test
  void projectServerExecutesWorkflowAgainstRealTemporalServer() throws Exception {
    String taskQueue = "project-harness-e2e";
    List<String> events = new ArrayList<>();
    CapturedProjectCall captured = new CapturedProjectCall();

    try (TestWorkflowEnvironment environment = TestWorkflowEnvironment.newInstance()) {
      Worker worker = environment.newWorker(taskQueue);
      worker.registerWorkflowImplementationTypes(ProjectHarnessEchoWorkflowImpl.class);
      environment.start();

      try (ProjectHarnessTestSupport.TestServer server =
          ProjectHarnessTestSupport.startServer(
              new ProjectHarness.ProjectServiceServer(
                  new ProjectHarness.ProjectHandlers(
                      (client, context) -> {
                        ProjectHarnessEchoWorkflow workflow =
                            client.newWorkflowStub(
                                ProjectHarnessEchoWorkflow.class,
                                WorkflowOptions.newBuilder()
                                    .setWorkflowId(
                                        String.format(
                                            "%s-%d", context.run.executionId, context.iteration))
                                    .setTaskQueue(context.taskQueue)
                                    .build());
                        captured.executeClient = client;
                        captured.executeContext = context;
                        captured.executeResult = workflow.run(new String(context.payload, UTF_8));
                        events.add("execute");
                      },
                      (client, context) -> {
                        captured.initClient = client;
                        captured.initContext = context;
                        events.add("init");
                      }),
                  config -> environment.getWorkflowClient()))) {
        server.stub.init(
            ProjectHarnessTestSupport.makeInitRequest().toBuilder()
                .setTaskQueue(taskQueue)
                .build());
        server.stub.execute(
            ProjectHarnessTestSupport.makeExecuteRequest().toBuilder()
                .setTaskQueue(taskQueue)
                .build());
      }
    }

    assertEquals(List.of("init", "execute"), events);
    assertSame(captured.initClient, captured.executeClient);
    assertEquals("run-id", captured.initContext.run.runId);
    assertEquals("exec-id", captured.initContext.run.executionId);
    assertEquals(taskQueue, captured.initContext.taskQueue);
    assertArrayEquals("{\"hello\":\"world\"}".getBytes(UTF_8), captured.initContext.configJson);
    assertEquals("run-id", captured.executeContext.run.runId);
    assertEquals("exec-id", captured.executeContext.run.executionId);
    assertEquals(taskQueue, captured.executeContext.taskQueue);
    assertEquals(7L, captured.executeContext.iteration);
    assertArrayEquals("payload".getBytes(UTF_8), captured.executeContext.payload);
    assertEquals("payload", captured.executeResult);
  }

  private static final class CapturedProjectCall {
    private WorkflowClient initClient;
    private WorkflowClient executeClient;
    private ProjectHarness.ProjectInitContext initContext;
    private ProjectHarness.ProjectExecuteContext executeContext;
    private String executeResult;
  }
}
