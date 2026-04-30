package io.temporal.omes.harness;

import io.grpc.BindableService;
import io.grpc.Server;
import io.grpc.ServerBuilder;
import io.grpc.Status;
import io.grpc.stub.StreamObserver;
import io.temporal.client.WorkflowClient;
import java.util.Objects;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import picocli.CommandLine;
import temporal.omes.projects.v1.Api;
import temporal.omes.projects.v1.ProjectServiceGrpc;

public final class ProjectHarness {
  private static final Logger logger = LoggerFactory.getLogger(ProjectHarness.class);

  private ProjectHarness() {}

  static void runProjectServerCli(
      ProjectHandlers handlers, HarnessClients.ClientFactory clientFactory, String... argv)
      throws Exception {
    Arguments args = new Arguments();
    new CommandLine(args).parseArgs(argv);
    Server server = ServerBuilder.forPort(args.port).addService(new ProjectServiceServer(handlers, clientFactory)).build();
    server.start();
    logger.info("Project server listening on port {}", server.getPort());
    server.awaitTermination();
  }

  public static final class ProjectRunMetadata {
    public final String runId;
    public final String executionId;

    public ProjectRunMetadata(String runId, String executionId) {
      this.runId = runId;
      this.executionId = executionId;
    }
  }

  public static final class ProjectInitContext {
    public final Logger logger;
    public final ProjectRunMetadata run;
    public final String taskQueue;
    public final byte[] configJson;

    public ProjectInitContext(
        Logger logger, ProjectRunMetadata run, String taskQueue, byte[] configJson) {
      this.logger = logger;
      this.run = run;
      this.taskQueue = taskQueue;
      this.configJson = configJson;
    }
  }

  public static final class ProjectExecuteContext {
    public final Logger logger;
    public final ProjectRunMetadata run;
    public final String taskQueue;
    public final long iteration;
    public final byte[] payload;

    public ProjectExecuteContext(
        Logger logger, ProjectRunMetadata run, String taskQueue, long iteration, byte[] payload) {
      this.logger = logger;
      this.run = run;
      this.taskQueue = taskQueue;
      this.iteration = iteration;
      this.payload = payload;
    }
  }

  @FunctionalInterface
  public interface ProjectInitHandler {
    void init(WorkflowClient client, ProjectInitContext context) throws Exception;
  }

  @FunctionalInterface
  public interface ProjectExecuteHandler {
    void execute(WorkflowClient client, ProjectExecuteContext context) throws Exception;
  }

  public static final class ProjectHandlers {
    public final ProjectExecuteHandler execute;
    public final ProjectInitHandler init;

    public ProjectHandlers(ProjectExecuteHandler execute) {
      this(execute, null);
    }

    public ProjectHandlers(ProjectExecuteHandler execute, ProjectInitHandler init) {
      this.execute = Objects.requireNonNull(execute);
      this.init = init;
    }
  }

  static final class ProjectServiceServer extends ProjectServiceGrpc.ProjectServiceImplBase {
    private static final Logger logger = LoggerFactory.getLogger(ProjectServiceServer.class);

    private final ProjectHandlers handlers;
    private final HarnessClients.ClientFactory clientFactory;
    private volatile WorkflowClient client;
    private volatile ProjectRunMetadata run;

    ProjectServiceServer(ProjectHandlers handlers, HarnessClients.ClientFactory clientFactory) {
      this.handlers = Objects.requireNonNull(handlers);
      this.clientFactory = Objects.requireNonNull(clientFactory);
    }

    @Override
    public void init(Api.InitRequest request, StreamObserver<Api.InitResponse> responseObserver) {
      if (request.getTaskQueue().isEmpty()) {
        abort(responseObserver, Status.INVALID_ARGUMENT, "task_queue required");
        return;
      }
      if (request.getExecutionId().isEmpty()) {
        abort(responseObserver, Status.INVALID_ARGUMENT, "execution_id required");
        return;
      }
      if (request.getRunId().isEmpty()) {
        abort(responseObserver, Status.INVALID_ARGUMENT, "run_id required");
        return;
      }

      Api.ConnectOptions conn = request.getConnectOptions();
      if (conn.getServerAddress().isEmpty()) {
        abort(responseObserver, Status.INVALID_ARGUMENT, "server_address required");
        return;
      }
      if (conn.getNamespace().isEmpty()) {
        abort(responseObserver, Status.INVALID_ARGUMENT, "namespace required");
        return;
      }

      HarnessClients.ClientConfig config;
      try {
        config =
            HarnessClients.buildClientConfig(
                conn.getServerAddress(),
                conn.getNamespace(),
                conn.getAuthHeader(),
                conn.getEnableTls(),
                conn.getTlsCertPath(),
                conn.getTlsKeyPath(),
                conn.getTlsServerName(),
                conn.getDisableHostVerification(),
                null,
                null);
      } catch (IllegalArgumentException e) {
        abort(responseObserver, Status.INVALID_ARGUMENT, messageOf(e));
        return;
      }

      WorkflowClient createdClient;
      try {
        createdClient = clientFactory.create(config);
      } catch (Exception e) {
        abort(responseObserver, Status.INTERNAL, "failed to create client: " + messageOf(e));
        return;
      }

      ProjectRunMetadata createdRun =
          new ProjectRunMetadata(request.getRunId(), request.getExecutionId());

      if (handlers.init != null) {
        try {
          handlers.init.init(
              createdClient,
              new ProjectInitContext(
                  logger, createdRun, request.getTaskQueue(), request.getConfigJson().toByteArray()));
        } catch (Exception e) {
          abort(responseObserver, Status.INTERNAL, "init handler failed: " + messageOf(e));
          return;
        }
      }

      client = createdClient;
      run = createdRun;
      responseObserver.onNext(Api.InitResponse.getDefaultInstance());
      responseObserver.onCompleted();
    }

    @Override
    public void execute(
        Api.ExecuteRequest request, StreamObserver<Api.ExecuteResponse> responseObserver) {
      if (request.getTaskQueue().isEmpty()) {
        abort(responseObserver, Status.INVALID_ARGUMENT, "task_queue required");
        return;
      }

      WorkflowClient cachedClient = client;
      ProjectRunMetadata cachedRun = run;
      if (cachedClient == null || cachedRun == null) {
        abort(responseObserver, Status.FAILED_PRECONDITION, "Init must be called before Execute");
        return;
      }

      try {
        handlers.execute.execute(
            cachedClient,
            new ProjectExecuteContext(
                logger,
                cachedRun,
                request.getTaskQueue(),
                request.getIteration(),
                request.getPayload().toByteArray()));
      } catch (Exception e) {
        abort(responseObserver, Status.INTERNAL, "execute handler failed: " + messageOf(e));
        return;
      }

      responseObserver.onNext(Api.ExecuteResponse.getDefaultInstance());
      responseObserver.onCompleted();
    }
  }

  private static void abort(
      StreamObserver<?> responseObserver, Status status, String description) {
    responseObserver.onError(status.withDescription(description).asRuntimeException());
  }

  private static String messageOf(Exception error) {
    return error.getMessage() == null ? error.toString() : error.getMessage();
  }

  private static final class Arguments {
    @CommandLine.Option(names = "--port", description = "gRPC listen port", defaultValue = "8080")
    private int port;
  }
}
