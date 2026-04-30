package io.temporal.omes.harness;

import static java.nio.charset.StandardCharsets.UTF_8;
import static org.junit.jupiter.api.Assertions.assertArrayEquals;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertSame;

import io.temporal.client.WorkflowClient;
import io.temporal.testing.TestWorkflowEnvironment;
import io.temporal.worker.Worker;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.atomic.AtomicReference;
import org.junit.jupiter.api.Test;
import temporal.omes.projects.v1.Api;

class ProjectHarnessTest {
  @Test
  void initRejectsInvalidTlsConfiguration() {
    ProjectHarness.ProjectServiceServer server =
        new ProjectHarness.ProjectServiceServer(
            new ProjectHarness.ProjectHandlers((client, context) -> {}),
            config -> HarnessTestSupport.fakeWorkflowClient());

    ProjectHarnessTestSupport.assertStatus(
        io.grpc.Status.Code.INVALID_ARGUMENT,
        "Client cert specified, but not client key!",
        ProjectHarnessTestSupport.init(
            server,
            ProjectHarnessTestSupport.makeInitRequest()
                .toBuilder()
                .setConnectOptions(
                    ProjectHarnessTestSupport.makeInitRequest()
                        .getConnectOptions()
                        .toBuilder()
                        .setEnableTls(true)
                        .setTlsCertPath("/tmp/cert.pem")
                        .setTlsKeyPath("")
                        .build())
                .build()));
  }

  @Test
  void initPassesRunMetadataToHandler() throws Exception {
    WorkflowClient client = HarnessTestSupport.fakeWorkflowClient();
    AtomicReference<HarnessClients.ClientConfig> configRef = new AtomicReference<>();
    AtomicReference<WorkflowClient> handlerClient = new AtomicReference<>();
    AtomicReference<ProjectHarness.ProjectInitContext> initContext = new AtomicReference<>();
    try (ProjectHarnessTestSupport.TempTlsFiles tlsFiles =
        ProjectHarnessTestSupport.writeValidTlsFiles()) {
      ProjectHarness.ProjectServiceServer server =
          new ProjectHarness.ProjectServiceServer(
              new ProjectHarness.ProjectHandlers(
                  (ignored, context) -> {},
                  (createdClient, context) -> {
                    handlerClient.set(createdClient);
                    initContext.set(context);
                  }),
              config -> {
                configRef.set(config);
                return client;
              });

      ProjectHarnessTestSupport.assertCompleted(
          ProjectHarnessTestSupport.init(
              server,
              ProjectHarnessTestSupport.makeInitRequest()
                  .toBuilder()
                  .setConnectOptions(
                      ProjectHarnessTestSupport.makeInitRequest()
                          .getConnectOptions()
                          .toBuilder()
                          .setAuthHeader("Bearer token")
                          .setEnableTls(true)
                          .setTlsCertPath(tlsFiles.certPath.toString())
                          .setTlsKeyPath(tlsFiles.keyPath.toString())
                          .setTlsServerName("server.local")
                          .setDisableHostVerification(true)
                          .build())
                  .build()));
    }

    HarnessClients.ClientConfig config = configRef.get();
    ProjectHarness.ProjectInitContext context = initContext.get();
    assertEquals("localhost:7233", config.targetHost);
    assertEquals("default", config.namespace);
    assertEquals("token", config.apiKey);
    assertEquals("server.local", config.tlsServerName);
    assertSame(client, handlerClient.get());
    assertEquals("run-id", context.run.runId);
    assertEquals("exec-id", context.run.executionId);
    assertEquals("task-queue", context.taskQueue);
    assertArrayEquals("{\"hello\":\"world\"}".getBytes(UTF_8), context.configJson);
  }

  @Test
  void executeRequiresInit() {
    ProjectHarness.ProjectServiceServer server =
        new ProjectHarness.ProjectServiceServer(
            new ProjectHarness.ProjectHandlers((client, context) -> {}),
            config -> HarnessTestSupport.fakeWorkflowClient());

    ProjectHarnessTestSupport.assertStatus(
        io.grpc.Status.Code.FAILED_PRECONDITION,
        "Init must be called before Execute",
        ProjectHarnessTestSupport.execute(server, ProjectHarnessTestSupport.makeExecuteRequest()));
  }

  @Test
  void executePassesIterationPayloadAndRunMetadata() throws Exception {
    WorkflowClient client = HarnessTestSupport.fakeWorkflowClient();
    AtomicReference<WorkflowClient> handlerClient = new AtomicReference<>();
    AtomicReference<ProjectHarness.ProjectExecuteContext> executeContext = new AtomicReference<>();

    ProjectHarness.ProjectServiceServer server =
        new ProjectHarness.ProjectServiceServer(
            new ProjectHarness.ProjectHandlers(
                (createdClient, context) -> {
                  handlerClient.set(createdClient);
                  executeContext.set(context);
                }),
            config -> client);
    ProjectHarnessTestSupport.assertCompleted(
        ProjectHarnessTestSupport.init(server, ProjectHarnessTestSupport.makeInitRequest()));
    ProjectHarnessTestSupport.assertCompleted(
        ProjectHarnessTestSupport.execute(server, ProjectHarnessTestSupport.makeExecuteRequest()));

    ProjectHarness.ProjectExecuteContext context = executeContext.get();
    assertSame(client, handlerClient.get());
    assertEquals(7L, context.iteration);
    assertArrayEquals("payload".getBytes(UTF_8), context.payload);
    assertEquals("task-queue", context.taskQueue);
    assertEquals("run-id", context.run.runId);
    assertEquals("exec-id", context.run.executionId);
  }

  @Test
  void clientFactoryFailureMapsToInternalError() {
    ProjectHarness.ProjectServiceServer server =
        new ProjectHarness.ProjectServiceServer(
            new ProjectHarness.ProjectHandlers((client, context) -> {}),
            config -> {
              throw new RuntimeException("boom");
            });

    ProjectHarnessTestSupport.assertStatus(
        io.grpc.Status.Code.INTERNAL,
        "failed to create client: boom",
        ProjectHarnessTestSupport.init(server, ProjectHarnessTestSupport.makeInitRequest()));
  }

  @Test
  void initHandlerFailureDoesNotLeaveServerInitialized() throws Exception {
    WorkflowClient client = HarnessTestSupport.fakeWorkflowClient();

    ProjectHarness.ProjectServiceServer server =
        new ProjectHarness.ProjectServiceServer(
            new ProjectHarness.ProjectHandlers(
                (createdClient, context) -> {},
                (createdClient, context) -> {
                  throw new RuntimeException("bad init");
                }),
            config -> client);

    ProjectHarnessTestSupport.assertStatus(
        io.grpc.Status.Code.INTERNAL,
        "init handler failed: bad init",
        ProjectHarnessTestSupport.init(server, ProjectHarnessTestSupport.makeInitRequest()));
    ProjectHarnessTestSupport.assertStatus(
        io.grpc.Status.Code.FAILED_PRECONDITION,
        "Init must be called before Execute",
        ProjectHarnessTestSupport.execute(server, ProjectHarnessTestSupport.makeExecuteRequest()));
  }

  @Test
  void executeHandlerFailureMapsToInternalError() throws Exception {
    WorkflowClient client = HarnessTestSupport.fakeWorkflowClient();

    ProjectHarness.ProjectServiceServer server =
        new ProjectHarness.ProjectServiceServer(
            new ProjectHarness.ProjectHandlers(
                (createdClient, context) -> {
                  throw new RuntimeException("bad execute");
                }),
            config -> client);
    ProjectHarnessTestSupport.assertCompleted(
        ProjectHarnessTestSupport.init(server, ProjectHarnessTestSupport.makeInitRequest()));
    ProjectHarnessTestSupport.assertStatus(
        io.grpc.Status.Code.INTERNAL,
        "execute handler failed: bad execute",
        ProjectHarnessTestSupport.execute(server, ProjectHarnessTestSupport.makeExecuteRequest()));
  }

  @Test
  void projectServerExecutesWorkflowAgainstRealTemporalServer() throws Exception {
    List<Object> events = new ArrayList<>();
    String taskQueue = "project-harness-e2e";

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
                                io.temporal.client.WorkflowOptions.newBuilder()
                                    .setWorkflowId(String.format("%s-%d", context.run.executionId, context.iteration))
                                    .setTaskQueue(context.taskQueue)
                                    .build());
                        String result = workflow.run(new String(context.payload, UTF_8));
                        events.add(new ProjectHarnessTestSupport.ExecuteEvent(client, context, result));
                      },
                      (client, context) -> events.add(new ProjectHarnessTestSupport.InitEvent(client, context))),
                  config -> environment.getWorkflowClient()))) {
        server.stub.init(
            ProjectHarnessTestSupport.makeInitRequest().toBuilder().setTaskQueue(taskQueue).build());
        server.stub.execute(
            ProjectHarnessTestSupport.makeExecuteRequest().toBuilder().setTaskQueue(taskQueue).build());
      }
    }

    ProjectHarnessTestSupport.InitEvent initEvent = (ProjectHarnessTestSupport.InitEvent) events.get(0);
    ProjectHarnessTestSupport.ExecuteEvent executeEvent = (ProjectHarnessTestSupport.ExecuteEvent) events.get(1);
    assertEquals("run-id", initEvent.context.run.runId);
    assertEquals("exec-id", initEvent.context.run.executionId);
    assertEquals(taskQueue, initEvent.context.taskQueue);
    assertArrayEquals("{\"hello\":\"world\"}".getBytes(UTF_8), initEvent.context.configJson);
    assertSame(initEvent.client, executeEvent.client);
    assertEquals("run-id", executeEvent.context.run.runId);
    assertEquals("exec-id", executeEvent.context.run.executionId);
    assertEquals(taskQueue, executeEvent.context.taskQueue);
    assertEquals(7L, executeEvent.context.iteration);
    assertArrayEquals("payload".getBytes(UTF_8), executeEvent.context.payload);
    assertEquals("payload", executeEvent.result);
  }

  public static class ProjectHarnessEchoWorkflowImpl implements ProjectHarnessEchoWorkflow {
    public ProjectHarnessEchoWorkflowImpl() {}

    @Override
    public String run(String payload) {
      return payload;
    }
  }
}
