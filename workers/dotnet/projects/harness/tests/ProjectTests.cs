using System.Net;
using System.Text;
using Google.Protobuf;
using Grpc.Core;
using Grpc.Net.Client;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Server.Kestrel.Core;
using Microsoft.Extensions.DependencyInjection;
using Temporal.Omes.Projects.V1;
using Temporalio.Client;
using Temporalio.Omes.Projects.Harness;
using Temporalio.Testing;
using Temporalio.Worker;
using Temporalio.Workflows;
using Xunit;

namespace Temporalio.Omes.Projects.Tests.HarnessTests;

public class ProjectTests
{
    [Fact]
    public async Task ProjectServerExecutesWorkflowAgainstRealTemporalServer()
    {
        ProjectInitContext? initContext = null;
        ProjectExecuteContext? executeContext = null;
        ITemporalClient? initClient = null;
        ITemporalClient? executeClient = null;
        string? executeResult = null;
        var eventKinds = new List<string>();
        var taskQueue = $"project-harness-e2e-{Guid.NewGuid():N}";

        Task InitHandler(ITemporalClient client, ProjectInitContext context)
        {
            initClient = client;
            initContext = context;
            eventKinds.Add("init");
            return Task.CompletedTask;
        }

        async Task ExecuteHandler(ITemporalClient client, ProjectExecuteContext context)
        {
            executeClient = client;
            var handle = await client.StartWorkflowAsync(
                (ProjectHarnessEchoWorkflow workflow) => workflow.RunAsync(Encoding.UTF8.GetString(context.Payload)),
                new WorkflowOptions(
                    id: $"{context.Run.ExecutionId}-{context.Iteration}",
                    taskQueue: context.TaskQueue));
            executeContext = context;
            executeResult = await handle.GetResultAsync<string>();
            eventKinds.Add("execute");
        }

        await using var env = await WorkflowEnvironment.StartLocalAsync();
        using var worker = new TemporalWorker(
            env.Client,
            new TemporalWorkerOptions(taskQueue)
                .AddWorkflow<ProjectHarnessEchoWorkflow>());
        using var workerCancellation = new CancellationTokenSource();
        var workerTask = Task.Run(() => worker.ExecuteAsync(workerCancellation.Token));

        await using var host = await StartProjectServiceHostAsync(
            new ProjectServiceServer(
                new ProjectHandlers(Execute: ExecuteHandler, Init: InitHandler),
                ClientHelpers.DefaultClientFactory));

        try
        {
            await host.Client.InitAsync(
                MakeInitRequest(
                    taskQueue: taskQueue,
                    configJson: ByteString.CopyFromUtf8("{\"hello\":\"world\"}"),
                    serverAddress: env.Client.Connection.Options.TargetHost!));

            await host.Client.ExecuteAsync(
                MakeExecuteRequest(
                    taskQueue: taskQueue,
                    payload: ByteString.CopyFromUtf8("payload")));
        }
        finally
        {
            await workerCancellation.CancelAsync();
            try
            {
                await workerTask;
            }
            catch (OperationCanceledException)
            {
            }
        }

        Assert.Equal(["init", "execute"], eventKinds);
        Assert.NotNull(initClient);
        Assert.NotNull(initContext);
        Assert.Same(initClient, executeClient);
        Assert.Equal("run-id", initContext!.Run.RunId);
        Assert.Equal("exec-id", initContext.Run.ExecutionId);
        Assert.Equal(taskQueue, initContext.TaskQueue);
        Assert.Equal("{\"hello\":\"world\"}", Encoding.UTF8.GetString(initContext.ConfigJson));
        Assert.NotNull(executeContext);
        Assert.Equal("run-id", executeContext!.Run.RunId);
        Assert.Equal("exec-id", executeContext.Run.ExecutionId);
        Assert.Equal(taskQueue, executeContext.TaskQueue);
        Assert.Equal(7L, executeContext.Iteration);
        Assert.Equal("payload", Encoding.UTF8.GetString(executeContext.Payload));
        Assert.Equal("payload", executeResult);
    }

    [Fact]
    public async Task InitRejectsInvalidTlsConfiguration()
    {
        var service = CreateProjectService(
            clientFactory: _ => Task.FromResult(HarnessTestSupport.CreateStrictTemporalClientProbe()));

        var error = await Assert.ThrowsAsync<RpcException>(
            () =>
                service.InitAsync(
                    MakeInitRequest(
                        enableTls: true,
                        tlsCertPath: "/tmp/cert.pem")));

        Assert.Equal(StatusCode.InvalidArgument, error.StatusCode);
        Assert.Equal("Client cert specified, but not client key!", error.Status.Detail);
    }

    [Fact]
    public async Task InitPassesRunMetadataToHandler()
    {
        var certBytes = Encoding.UTF8.GetBytes("cert");
        var keyBytes = Encoding.UTF8.GetBytes("key");
        var certPath = Path.GetTempFileName();
        var keyPath = Path.GetTempFileName();
        ClientConfig? capturedConfig = null;
        ITemporalClient? initClient = null;

        try
        {
            File.WriteAllBytes(certPath, certBytes);
            File.WriteAllBytes(keyPath, keyBytes);

            var service = CreateProjectService(
                clientFactory: config =>
                {
                    capturedConfig = config;
                    return Task.FromResult(HarnessTestSupport.CreateStrictTemporalClientProbe());
                },
                initHandler: (handlerClient, context) =>
                {
                    initClient = handlerClient;
                    Assert.Equal("run-id", context.Run.RunId);
                    Assert.Equal("exec-id", context.Run.ExecutionId);
                    Assert.Equal("task-queue", context.TaskQueue);
                    Assert.Equal("{\"hello\":\"world\"}", Encoding.UTF8.GetString(context.ConfigJson));
                    return Task.CompletedTask;
                });

            await service.InitAsync(
                MakeInitRequest(
                    authHeader: "Bearer token",
                    enableTls: true,
                    tlsCertPath: certPath,
                    tlsKeyPath: keyPath,
                    tlsServerName: "server.local",
                    disableHostVerification: true,
                    configJson: ByteString.CopyFromUtf8("{\"hello\":\"world\"}")));

            Assert.NotNull(initClient);
            Assert.NotNull(capturedConfig);
            Assert.Equal("localhost:7233", capturedConfig!.ServerAddress);
            Assert.Equal("default", capturedConfig.Namespace);
            Assert.Equal("token", capturedConfig.ApiKey);
            Assert.NotNull(capturedConfig.Tls);
            Assert.Equal("server.local", capturedConfig.Tls!.Domain);
            Assert.Equal(certBytes, capturedConfig.Tls.ClientCert);
            Assert.Equal(keyBytes, capturedConfig.Tls.ClientPrivateKey);
        }
        finally
        {
            File.Delete(certPath);
            File.Delete(keyPath);
        }
    }

    [Fact]
    public async Task ExecuteRequiresInit()
    {
        var service = CreateProjectService(
            clientFactory: _ => Task.FromResult(HarnessTestSupport.CreateStrictTemporalClientProbe()));

        var error = await Assert.ThrowsAsync<RpcException>(() => service.ExecuteAsync(MakeExecuteRequest()));

        Assert.Equal(StatusCode.FailedPrecondition, error.StatusCode);
        Assert.Equal("Init must be called before Execute", error.Status.Detail);
    }

    [Fact]
    public async Task ExecutePassesIterationPayloadAndRunMetadata()
    {
        var sharedClient = HarnessTestSupport.CreateStrictTemporalClientProbe();
        ITemporalClient? executeClient = null;

        var service = CreateProjectService(
            clientFactory: _ => Task.FromResult(sharedClient),
            executeHandler: (handlerClient, context) =>
            {
                executeClient = handlerClient;
                Assert.Equal(7L, context.Iteration);
                Assert.Equal("payload", Encoding.UTF8.GetString(context.Payload));
                Assert.Equal("task-queue", context.TaskQueue);
                Assert.Equal("run-id", context.Run.RunId);
                Assert.Equal("exec-id", context.Run.ExecutionId);
                return Task.CompletedTask;
            });

        await service.InitAsync(MakeInitRequest());
        await service.ExecuteAsync(MakeExecuteRequest(payload: ByteString.CopyFromUtf8("payload")));

        Assert.Same(sharedClient, executeClient);
    }

    [Fact]
    public async Task ClientFactoryFailureMapsToInternalError()
    {
        var service = CreateProjectService(
            clientFactory: _ => throw new InvalidOperationException("boom"));

        var error = await Assert.ThrowsAsync<RpcException>(() => service.InitAsync(MakeInitRequest()));

        Assert.Equal(StatusCode.Internal, error.StatusCode);
        Assert.Equal("failed to create client: boom", error.Status.Detail);
    }

    [Fact]
    public async Task InitHandlerFailureDoesNotLeaveServerInitialized()
    {
        var service = CreateProjectService(
            clientFactory: _ => Task.FromResult(HarnessTestSupport.CreateStrictTemporalClientProbe()),
            initHandler: (_, _) => throw new InvalidOperationException("bad init"));

        var initError = await Assert.ThrowsAsync<RpcException>(() => service.InitAsync(MakeInitRequest()));
        Assert.Equal(StatusCode.Internal, initError.StatusCode);
        Assert.Equal("init handler failed: bad init", initError.Status.Detail);

        var executeError = await Assert.ThrowsAsync<RpcException>(() => service.ExecuteAsync(MakeExecuteRequest()));
        Assert.Equal(StatusCode.FailedPrecondition, executeError.StatusCode);
        Assert.Equal("Init must be called before Execute", executeError.Status.Detail);
    }

    [Fact]
    public async Task ExecuteHandlerFailureMapsToInternalError()
    {
        var service = CreateProjectService(
            clientFactory: _ => Task.FromResult(HarnessTestSupport.CreateStrictTemporalClientProbe()),
            executeHandler: (_, _) => throw new InvalidOperationException("bad execute"));

        await service.InitAsync(MakeInitRequest());

        var error = await Assert.ThrowsAsync<RpcException>(() => service.ExecuteAsync(MakeExecuteRequest()));
        Assert.Equal(StatusCode.Internal, error.StatusCode);
        Assert.Equal("execute handler failed: bad execute", error.Status.Detail);
    }

    private static ProjectServiceServer CreateProjectService(
        ClientFactory clientFactory,
        ProjectExecuteHandler? executeHandler = null,
        ProjectInitHandler? initHandler = null) =>
        new(
            new ProjectHandlers(
                Execute: executeHandler ?? ((_, _) => Task.CompletedTask),
                Init: initHandler),
            clientFactory);

    private static InitRequest MakeInitRequest(
        string executionId = "exec-id",
        string runId = "run-id",
        string taskQueue = "task-queue",
        ByteString? configJson = null,
        string authHeader = "",
        bool enableTls = false,
        string tlsCertPath = "",
        string tlsKeyPath = "",
        string tlsServerName = "",
        bool disableHostVerification = false,
        string serverAddress = "localhost:7233",
        string @namespace = "default") =>
        new()
        {
            ExecutionId = executionId,
            RunId = runId,
            TaskQueue = taskQueue,
            ConfigJson = configJson ?? ByteString.Empty,
            ConnectOptions = new ConnectOptions
            {
                Namespace = @namespace,
                ServerAddress = serverAddress,
                AuthHeader = authHeader,
                EnableTls = enableTls,
                TlsCertPath = tlsCertPath,
                TlsKeyPath = tlsKeyPath,
                TlsServerName = tlsServerName,
                DisableHostVerification = disableHostVerification,
            },
        };

    private static ExecuteRequest MakeExecuteRequest(
        long iteration = 7,
        string taskQueue = "task-queue",
        ByteString? payload = null) =>
        new()
        {
            Iteration = iteration,
            TaskQueue = taskQueue,
            Payload = payload ?? ByteString.Empty,
        };

    private static async Task<ProjectServiceHost> StartProjectServiceHostAsync(ProjectServiceServer service)
    {
        var builder = WebApplication.CreateBuilder();
        builder.WebHost.ConfigureKestrel(
            options =>
            {
                options.Listen(
                    IPAddress.Loopback,
                    0,
                    listenOptions =>
                    {
                        listenOptions.Protocols = HttpProtocols.Http2;
                    });
            });
        builder.Services.AddGrpc();
        builder.Services.AddSingleton(service);

        var app = builder.Build();
        app.MapGrpcService<ProjectServiceServer>();
        await app.StartAsync();

        var address = app.Urls.Single();
        var channel = GrpcChannel.ForAddress(address);
        return new ProjectServiceHost(app, channel);
    }
}

internal sealed class ProjectServiceHost(WebApplication app, GrpcChannel channel) : IAsyncDisposable
{
    public ProjectService.ProjectServiceClient Client { get; } = new(channel);

    public async ValueTask DisposeAsync()
    {
        channel.Dispose();
        await app.StopAsync();
        await app.DisposeAsync();
    }
}

[Workflow]
public class ProjectHarnessEchoWorkflow
{
    [WorkflowRun]
    public Task<string> RunAsync(string payload) => Task.FromResult(payload);
}
