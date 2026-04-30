package io.temporal.omes.harness;

import static java.nio.charset.StandardCharsets.UTF_8;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.google.protobuf.ByteString;
import io.grpc.ManagedChannel;
import io.grpc.Server;
import io.grpc.Status;
import io.grpc.inprocess.InProcessChannelBuilder;
import io.grpc.inprocess.InProcessServerBuilder;
import io.grpc.stub.StreamObserver;
import io.temporal.client.WorkflowClient;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import temporal.omes.projects.v1.Api;
import temporal.omes.projects.v1.ProjectServiceGrpc;

final class ProjectHarnessTestSupport {
  private ProjectHarnessTestSupport() {}

  static Api.InitRequest makeInitRequest() {
    return makeInitRequest(
        "exec-id",
        "run-id",
        "task-queue",
        ByteString.copyFrom("{\"hello\":\"world\"}".getBytes(UTF_8)),
        "",
        false,
        "",
        "",
        "",
        false,
        "localhost:7233",
        "default");
  }

  static Api.InitRequest makeInitRequest(
      String executionId,
      String runId,
      String taskQueue,
      ByteString configJson,
      String authHeader,
      boolean enableTls,
      String tlsCertPath,
      String tlsKeyPath,
      String tlsServerName,
      boolean disableHostVerification,
      String serverAddress,
      String namespace) {
    return Api.InitRequest.newBuilder()
        .setExecutionId(executionId)
        .setRunId(runId)
        .setTaskQueue(taskQueue)
        .setConnectOptions(
            Api.ConnectOptions.newBuilder()
                .setNamespace(namespace)
                .setServerAddress(serverAddress)
                .setAuthHeader(authHeader)
                .setEnableTls(enableTls)
                .setTlsCertPath(tlsCertPath)
                .setTlsKeyPath(tlsKeyPath)
                .setTlsServerName(tlsServerName)
                .setDisableHostVerification(disableHostVerification)
                .build())
        .setConfigJson(configJson)
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

  static <T> T assertCompleted(RecordingObserver<T> observer) {
    assertNull(observer.error);
    assertTrue(observer.completed);
    return observer.value;
  }

  static void assertStatus(
      Status.Code statusCode, String details, RecordingObserver<?> observer) {
    Status status = Status.fromThrowable(observer.error);
    assertEquals(statusCode, status.getCode());
    assertEquals(details, status.getDescription());
  }

  static RecordingObserver<Api.InitResponse> init(
      ProjectHarness.ProjectServiceServer server, Api.InitRequest request) {
    RecordingObserver<Api.InitResponse> observer = new RecordingObserver<>();
    server.init(request, observer);
    return observer;
  }

  static RecordingObserver<Api.ExecuteResponse> execute(
      ProjectHarness.ProjectServiceServer server, Api.ExecuteRequest request) {
    RecordingObserver<Api.ExecuteResponse> observer = new RecordingObserver<>();
    server.execute(request, observer);
    return observer;
  }

  static TempTlsFiles writeValidTlsFiles() throws IOException {
    Path certPath = Files.createTempFile("harness-cert", ".pem");
    Path keyPath = Files.createTempFile("harness-key", ".pem");
    Files.writeString(certPath, VALID_CERT_PEM);
    Files.writeString(keyPath, VALID_KEY_PEM);
    return new TempTlsFiles(certPath, keyPath);
  }

  static final class TestServer implements AutoCloseable {
    final Server server;
    final ManagedChannel channel;
    final ProjectServiceGrpc.ProjectServiceBlockingStub stub;

    TestServer(
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

  static final class RecordingObserver<T> implements StreamObserver<T> {
    T value;
    Throwable error;
    boolean completed;

    @Override
    public void onNext(T value) {
      this.value = value;
    }

    @Override
    public void onError(Throwable error) {
      this.error = error;
    }

    @Override
    public void onCompleted() {
      completed = true;
    }
  }

  static final class InitEvent {
    final WorkflowClient client;
    final ProjectHarness.ProjectInitContext context;

    InitEvent(WorkflowClient client, ProjectHarness.ProjectInitContext context) {
      this.client = client;
      this.context = context;
    }
  }

  static final class ExecuteEvent {
    final WorkflowClient client;
    final ProjectHarness.ProjectExecuteContext context;
    final String result;

    ExecuteEvent(
        WorkflowClient client, ProjectHarness.ProjectExecuteContext context, String result) {
      this.client = client;
      this.context = context;
      this.result = result;
    }
  }

  static final class TempTlsFiles implements AutoCloseable {
    final Path certPath;
    final Path keyPath;

    TempTlsFiles(Path certPath, Path keyPath) {
      this.certPath = certPath;
      this.keyPath = keyPath;
    }

    @Override
    public void close() throws IOException {
      Files.deleteIfExists(certPath);
      Files.deleteIfExists(keyPath);
    }
  }

  private static final String VALID_CERT_PEM =
      "-----BEGIN CERTIFICATE-----\n"
          + "MIIDDzCCAfegAwIBAgIUPDx5jOYfGVFoqqpOWHo7GZubTCYwDQYJKoZIhvcNAQEL\n"
          + "BQAwFzEVMBMGA1UEAwwMc2VydmVyLmxvY2FsMB4XDTI2MDQyMzAzMTQxOFoXDTI2\n"
          + "MDQyNDAzMTQxOFowFzEVMBMGA1UEAwwMc2VydmVyLmxvY2FsMIIBIjANBgkqhkiG\n"
          + "9w0BAQEFAAOCAQ8AMIIBCgKCAQEA04ZGLQ/SlwWY4XADZgA4ZUCGOMxGnu2RmVjt\n"
          + "5kbmYdYStrdPjc7dJe8ZM8sprMnLoHcKk6bYRBvCVB3ArZhcSAGU+3Nxbk/nU7OH\n"
          + "rETy1SeyWT4t57nW7EqRF8TjNEENoX62sl+d4QT+0LYEvnJcUj8UlXmq39/RhlZX\n"
          + "4xrsJef3xO9ULhs/Jj7bYZizOEK+uaVRvfzkvhv9AoXeBKsclsRtz+2SJzkc5vAk\n"
          + "9ZFSR0ctXIJw3uyXZaeTB14PaYWPM2xt//0urBOEtkpxz6Y5gFxT2Ek0Ugu/mPxx\n"
          + "zvyds+ZAcbCxw1c7NP3URz2o5j04nik1dyITD5mUNT+853Wx+QIDAQABo1MwUTAd\n"
          + "BgNVHQ4EFgQUG2Vpum9Yz49VGehBzCRMkLPTItowHwYDVR0jBBgwFoAUG2Vpum9Y\n"
          + "z49VGehBzCRMkLPTItowDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOC\n"
          + "AQEAIWf3AQMZTApaGQwmcGrM77Fdj+hzjf5sEEhLP6261sLzgIX4SvZhGtEutNGL\n"
          + "uO/vJiZHtRDLZZ3KAvdxVAaSSbO6q+eHm3jhdbT0Kx7dsPWv73y1BI53gMQTGPm3\n"
          + "04Z1lROhFyPD5Fl5E8+CICT//ZS6g8+XXZA10PxztDJIl/u24268PaKbsY4qXyB3\n"
          + "8MdUMJ+k0Jeq+sSllKeqFUUwgWGZylBWSKyiebzLYcn4UVscv2ZsKjW1J/PvvBd2\n"
          + "7cPCk7gncyLYNEa/I47SV3AKm0j0xXt6i8YEkpFfLl7mJM5D5/IvNFBFF0tEDvRJ\n"
          + "i5xxQFGiNmh2flLO8stdozstHw==\n"
          + "-----END CERTIFICATE-----\n";

  private static final String VALID_KEY_PEM =
      "-----BEGIN PRIVATE KEY-----\n"
          + "MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDThkYtD9KXBZjh\n"
          + "cANmADhlQIY4zEae7ZGZWO3mRuZh1hK2t0+Nzt0l7xkzyymsycugdwqTpthEG8JU\n"
          + "HcCtmFxIAZT7c3FuT+dTs4esRPLVJ7JZPi3nudbsSpEXxOM0QQ2hfrayX53hBP7Q\n"
          + "tgS+clxSPxSVearf39GGVlfjGuwl5/fE71QuGz8mPtthmLM4Qr65pVG9/OS+G/0C\n"
          + "hd4EqxyWxG3P7ZInORzm8CT1kVJHRy1cgnDe7Jdlp5MHXg9phY8zbG3//S6sE4S2\n"
          + "SnHPpjmAXFPYSTRSC7+Y/HHO/J2z5kBxsLHDVzs0/dRHPajmPTieKTV3IhMPmZQ1\n"
          + "P7zndbH5AgMBAAECggEASVxc81z0+zbQPoO0UgiKhqdZxdInPhCD+kzK+Z4mYdE2\n"
          + "nVM3TqXrsi/aLEnucsRsEIOo0evAPuLnw3esLyjT/I875fe0Y/9nafKuf9NL6xyA\n"
          + "8Q2tKxybi0kTSEybRjC3swZ5A6VA4t1yKN2wCIMuPMIu9+aCGnIMP4yrn5LjSwOm\n"
          + "W03Rl4ZAoy2Ep3PqZdGBnl7Nz+1zr/yt4wqVHu1CL5AD+BIvYGUmJE0ToOG0V7xj\n"
          + "gbZLkPE2aOTE+oHNJ/c28SKn3mqvY3+Z3xCbkjxvwMXaKVhcaaxVxxVzaxoVqI6/\n"
          + "eU3kXWjsmElz/XyUcuNGfMhgDyXi5+Uy8rdlD6diNwKBgQD7J4hs/j1WylKZr6oo\n"
          + "OxN0UVhEG5h46xorvxIGNyOu0SMW54Lo102q9SeKf4/8z/CGqPm41F3HnfaqEv5n\n"
          + "wFxntoT9ugWT7qFPz0kO0n7f9i8UeDoc6hFrtkO4d/hJc2C8X/iGjXLT+Rhp5O+d\n"
          + "GOFK2CE9TiejHTyjo2u/iOazSwKBgQDXmwGyI/gl/BuQC7OyK4QHG30pIU4kUotD\n"
          + "/wIbwebEmBM/FfwOAGdDaZFTa7OYRdAbhkmgbQgndHHpk5mFl9hzdTRroGV56Bfb\n"
          + "bwWjCvB9AofnQfsrizjnpm3Aoz6BMe0G9dtEiB/+xKql9bOcKLoQxEgDEWOE/nD6\n"
          + "ahPUkDehSwKBgHLt5koqFZuvvhjCABWk4wQpbUDNd/pta251YyQg+102Kt6CVq+C\n"
          + "RvJieROxyAwig6i7jnr8A2YjbQrq4ixMJHz5UuZgx8ioPH0vF/mGbbTDDUxKsB0n\n"
          + "J42ovFif3aiO+cd6C1pXRCKoLHnY36V+CyqauKs7JnxIFsWzNM1TMm79AoGBAJrz\n"
          + "b9iTOTgzY6u2fULDO3PQMbdplDtOh4AquV0xkaQgl1RzfF6js5MjP6pwcPYy1kmx\n"
          + "zSBau81/Ro7T4TW913XC+hWPhN6ECwFNXQO8TPHK69kr9lNpD1CMr7wOllFLjEnA\n"
          + "UAGEw1naBbqYRqkoK/D437g0uw1Nv+x4aCAQNarZAoGAUe4BM/RGbTmdxCi6hmZO\n"
          + "QVzEQlowVRjkGsDJzR8RcvoAVnw+K+r+DizeY5JlQD+NRV5IkmBTU9af/6bbYMdX\n"
          + "4mE/vUnbeCpLnnywqJCde4tWcx/UiZISkNWB2hwnPPLq3tR4hjFTHyLKqWkB5CU3\n"
          + "A4EHER3nXPSyBjuz01X42MI=\n"
          + "-----END PRIVATE KEY-----\n";
}
