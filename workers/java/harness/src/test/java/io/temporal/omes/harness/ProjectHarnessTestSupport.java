package io.temporal.omes.harness;

import static java.nio.charset.StandardCharsets.UTF_8;

import com.google.protobuf.ByteString;
import io.grpc.ManagedChannel;
import io.grpc.Server;
import io.grpc.inprocess.InProcessChannelBuilder;
import io.grpc.inprocess.InProcessServerBuilder;
import java.io.IOException;
import temporal.omes.projects.v1.Api;
import temporal.omes.projects.v1.ProjectServiceGrpc;

final class ProjectHarnessTestSupport {
  private ProjectHarnessTestSupport() {}

  static Api.InitRequest makeInitRequest() {
    return Api.InitRequest.newBuilder()
        .setExecutionId("exec-id")
        .setRunId("run-id")
        .setTaskQueue("task-queue")
        .setConnectOptions(
            Api.ConnectOptions.newBuilder()
                .setNamespace("default")
                .setServerAddress("localhost:7233")
                .build())
        .setConfigJson(ByteString.copyFrom("{\"hello\":\"world\"}".getBytes(UTF_8)))
        .build();
  }

  static Api.ExecuteRequest makeExecuteRequest() {
    return Api.ExecuteRequest.newBuilder()
        .setIteration(7)
        .setTaskQueue("task-queue")
        .setPayload(ByteString.copyFrom("payload".getBytes(UTF_8)))
        .build();
  }

  static TestServer startServer(ProjectHarness.ProjectServiceServer service) throws IOException {
    String serverName = InProcessServerBuilder.generateName();
    Server server =
        InProcessServerBuilder.forName(serverName).directExecutor().addService(service).build();
    server.start();
    ManagedChannel channel = InProcessChannelBuilder.forName(serverName).directExecutor().build();
    return new TestServer(server, channel, ProjectServiceGrpc.newBlockingStub(channel));
  }

  static final class TestServer implements AutoCloseable {
    final Server server;
    final ManagedChannel channel;
    final ProjectServiceGrpc.ProjectServiceBlockingStub stub;

    private TestServer(
        Server server, ManagedChannel channel, ProjectServiceGrpc.ProjectServiceBlockingStub stub) {
      this.server = server;
      this.channel = channel;
      this.stub = stub;
    }

    @Override
    public void close() throws Exception {
      channel.shutdownNow();
      server.shutdownNow();
      channel.awaitTermination(5, java.util.concurrent.TimeUnit.SECONDS);
      server.awaitTermination(5, java.util.concurrent.TimeUnit.SECONDS);
    }
  }
}
