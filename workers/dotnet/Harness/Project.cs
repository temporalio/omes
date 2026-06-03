using System.CommandLine;
using System.CommandLine.Invocation;
using Google.Protobuf;
using Grpc.Core;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Server.Kestrel.Core;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Logging.Abstractions;
using Temporalio.Client;
using Temporal.Omes.Projects.V1;

namespace Temporalio.Omes.Projects.Harness;

public sealed record ProjectRunMetadata(string RunId, string ExecutionId);

public sealed record ProjectInitContext(
    ILogger Logger,
    ProjectRunMetadata Run,
    string TaskQueue,
    byte[] ConfigJson);

public sealed record ProjectExecuteContext(
    ILogger Logger,
    ProjectRunMetadata Run,
    string TaskQueue,
    long Iteration,
    byte[] Payload);

public delegate Task ProjectExecuteHandler(ITemporalClient client, ProjectExecuteContext context);

public delegate Task ProjectInitHandler(ITemporalClient client, ProjectInitContext context);

public sealed record ProjectHandlers(ProjectExecuteHandler Execute, ProjectInitHandler? Init = null);

internal static class ProjectHarness
{
    private static readonly Option<int> PortOption = new(
        name: "--port",
        description: "gRPC listen port",
        getDefaultValue: () => 8080);

    public static async Task<int> RunAsync(App app, string[] args)
    {
        var command = CreateProjectCommandDefinition();
        command.SetHandler(
            async (InvocationContext context) =>
            {
                await ServeAsync(
                    app.Project!,
                    app.ClientFactory,
                    context.ParseResult.GetValueForOption(PortOption));
                context.ExitCode = 0;
            });
        return await command.InvokeAsync(args);
    }

    private static async Task ServeAsync(ProjectHandlers handlers, ClientFactory clientFactory, int port)
    {
        var builder = WebApplication.CreateBuilder();
        builder.WebHost.ConfigureKestrel(
            options =>
            {
                options.ListenAnyIP(
                    port,
                    listenOptions =>
                    {
                        listenOptions.Protocols = HttpProtocols.Http2;
                    });
            });
        builder.Services.AddGrpc();
        builder.Services.AddSingleton(new ProjectServiceServer(handlers, clientFactory));

        var app = builder.Build();
        app.MapGrpcService<ProjectServiceServer>();
        app.Lifetime.ApplicationStarted.Register(
            () => app.Logger.LogInformation("Project server listening on port {Port}", port));
        await app.RunAsync();
    }

    private static RootCommand CreateProjectCommandDefinition()
    {
        var command = new RootCommand(".NET project harness server");
        command.AddOption(PortOption);
        return command;
    }
}

public sealed class ProjectServiceServer : ProjectService.ProjectServiceBase
{
    private readonly ProjectHandlers _handlers;
    private readonly ClientFactory _clientFactory;
    private ITemporalClient? _client;
    private ProjectRunMetadata? _run;
    private ILogger _logger;

    public ProjectServiceServer(ProjectHandlers handlers, ClientFactory clientFactory)
    {
        _handlers = handlers;
        _clientFactory = clientFactory;
        _logger = NullLoggerFactory.Instance.CreateLogger<ProjectServiceServer>();
    }

    public async Task<InitResponse> InitAsync(InitRequest request)
    {
        ValidateInit(request);

        ClientConfig clientConfig;
        try
        {
            clientConfig = ClientHelpers.BuildClientConfig(
                serverAddress: request.ConnectOptions.ServerAddress,
                @namespace: request.ConnectOptions.Namespace,
                authHeader: request.ConnectOptions.AuthHeader,
                tls: request.ConnectOptions.EnableTls,
                tlsCertPath: request.ConnectOptions.TlsCertPath,
                tlsKeyPath: request.ConnectOptions.TlsKeyPath,
                tlsServerName: request.ConnectOptions.TlsServerName,
                disableHostVerification: request.ConnectOptions.DisableHostVerification);
        }
        catch (ArgumentException err)
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, err.Message));
        }

        ITemporalClient client;
        try
        {
            client = await _clientFactory(clientConfig);
        }
        catch (Exception err)
        {
            throw new RpcException(new Status(StatusCode.Internal, $"failed to create client: {err.Message}"));
        }

        _logger = clientConfig.LoggerFactory.CreateLogger<ProjectServiceServer>();
        var run = new ProjectRunMetadata(request.RunId, request.ExecutionId);

        if (_handlers.Init is not null)
        {
            try
            {
                await _handlers.Init(
                    client,
                    new ProjectInitContext(
                        Logger: _logger,
                        Run: run,
                        TaskQueue: request.TaskQueue,
                        ConfigJson: request.ConfigJson.ToByteArray()));
            }
            catch (Exception err)
            {
                throw new RpcException(new Status(StatusCode.Internal, $"init handler failed: {err.Message}"));
            }
        }

        _client = client;
        _run = run;
        return new InitResponse();
    }

    public async Task<ExecuteResponse> ExecuteAsync(ExecuteRequest request)
    {
        if (string.IsNullOrEmpty(request.TaskQueue))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "task_queue required"));
        }

        if (_client is null || _run is null)
        {
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "Init must be called before Execute"));
        }

        try
        {
            await _handlers.Execute(
                _client,
                new ProjectExecuteContext(
                    Logger: _logger,
                    Run: _run,
                    TaskQueue: request.TaskQueue,
                    Iteration: request.Iteration,
                    Payload: request.Payload.ToByteArray()));
        }
        catch (Exception err)
        {
            throw new RpcException(new Status(StatusCode.Internal, $"execute handler failed: {err.Message}"));
        }

        return new ExecuteResponse();
    }

    public override Task<InitResponse> Init(InitRequest request, ServerCallContext context) => InitAsync(request);

    public override Task<ExecuteResponse> Execute(ExecuteRequest request, ServerCallContext context) => ExecuteAsync(request);

    private static void ValidateInit(InitRequest request)
    {
        if (string.IsNullOrEmpty(request.TaskQueue))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "task_queue required"));
        }

        if (string.IsNullOrEmpty(request.ExecutionId))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "execution_id required"));
        }

        if (string.IsNullOrEmpty(request.RunId))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "run_id required"));
        }

        if (string.IsNullOrEmpty(request.ConnectOptions?.ServerAddress))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "server_address required"));
        }

        if (string.IsNullOrEmpty(request.ConnectOptions?.Namespace))
        {
            throw new RpcException(new Status(StatusCode.InvalidArgument, "namespace required"));
        }
    }
}
