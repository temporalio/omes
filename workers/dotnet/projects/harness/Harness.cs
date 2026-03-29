using System.CommandLine;
using System.CommandLine.Invocation;
using Google.Protobuf;
using Grpc.Core;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Server.Kestrel.Core;
using Microsoft.Extensions.DependencyInjection;
using Temporalio.Client;
using Temporal.Omes.Projects.V1;

namespace Temporalio.Omes.Projects.Harness;

/// <summary>
/// Minimal config passed to ClientFunc for client creation.
/// </summary>
public record ClientConfig(
    string TaskQueue
);

/// <summary>
/// Run-level config passed to InitFunc for project-specific setup.
/// </summary>
public record InitConfig(
    byte[]? ConfigJson,
    string TaskQueue,
    string RunId,
    string ExecutionId
);

/// <summary>
/// Per-iteration data passed to ExecuteFunc.
/// </summary>
public record ExecuteInfo(
    string TaskQueue,
    string ExecutionId,
    string RunId,
    long Iteration,
    byte[]? Payload
);

/// <summary>
/// Worker configuration passed to WorkerFunc.
/// </summary>
public record WorkerConfig(
    string TaskQueue,
    string? PromListenAddress = null
);

/// <summary>
/// Bridges the omes CLI with .NET project test code via gRPC.
/// </summary>
public partial class ProjectHarness
{
    public const string OmesSearchAttributeKey = "OmesExecutionID";

    public delegate Task<ITemporalClient> ClientFunc(
        TemporalClientConnectOptions opts, ClientConfig config);

    public delegate Task InitFunc(
        ITemporalClient client, InitConfig config);

    public delegate Task ExecuteFunc(
        ITemporalClient client, ExecuteInfo info);

    public delegate Task WorkerFunc(
        ITemporalClient client, WorkerConfig config);

    private ClientFunc? _clientFn;
    private InitFunc? _initFn;
    private WorkerFunc? _workerFn;
    private ExecuteFunc? _executeFn;
    private ITemporalClient? _client;
    private RunContext? _runCtx;

    public void RegisterClient(ClientFunc fn)
    {
        if (_clientFn != null)
            throw new InvalidOperationException("Client factory already registered");
        _clientFn = fn;
    }

    public void OnInit(InitFunc fn)
    {
        if (_initFn != null)
            throw new InvalidOperationException("Init handler already registered");
        _initFn = fn;
    }

    public void RegisterWorker(WorkerFunc fn)
    {
        if (_workerFn != null)
            throw new InvalidOperationException("Worker already registered");
        _workerFn = fn;
    }

    public void OnExecute(ExecuteFunc fn)
    {
        if (_executeFn != null)
            throw new InvalidOperationException("Execute handler already registered");
        _executeFn = fn;
    }

    public async Task<int> RunAsync(string[] args)
    {
        if (_clientFn == null)
            throw new InvalidOperationException("No client factory registered; call RegisterClient before RunAsync");

        var root = new RootCommand("Project test harness");
        root.Add(BuildWorkerCommand());

        // Project-server subcommand
        var portOption = new Option<int>("--port", () => 8080, "gRPC server port");
        var serverCmd = new Command("project-server", "Run the project gRPC server");
        serverCmd.Add(portOption);
        serverCmd.SetHandler(async (InvocationContext ctx) =>
        {
            if (_executeFn == null)
                throw new InvalidOperationException("No execute handler registered");
            if (_workerFn == null)
                throw new InvalidOperationException("No worker handler registered");

            var port = ctx.ParseResult.GetValueForOption(portOption);
            await StartGrpcServerAsync(port);
        });
        root.Add(serverCmd);

        return await root.InvokeAsync(args);
    }

    private async Task StartGrpcServerAsync(int port)
    {
        var builder = WebApplication.CreateBuilder();
        builder.WebHost.ConfigureKestrel(options =>
        {
            options.ListenAnyIP(port, listenOptions =>
            {
                listenOptions.Protocols = HttpProtocols.Http2;
            });
        });
        builder.Services.AddGrpc();
        builder.Services.AddSingleton(this);

        var app = builder.Build();
        app.MapGrpcService<ProjectServiceImpl>();

        await app.RunAsync();
    }

    internal async Task<InitResponse> HandleInit(InitRequest req)
    {
        ValidateInit(req);

        var co = req.ConnectOptions;
        var opts = await BuildClientConnectOptions(
            co.ServerAddress,
            co.Namespace,
            co.AuthHeader,
            co.EnableTls,
            co.TlsCertPath,
            co.TlsKeyPath,
            co.TlsServerName);

        _client = await _clientFn!(opts, new ClientConfig(req.TaskQueue));

        if (_initFn != null)
        {
            await _initFn(_client, new InitConfig(
                req.ConfigJson?.ToByteArray(),
                req.TaskQueue,
                req.RunId,
                req.ExecutionId
            ));
        }

        if (req.RegisterSearchAttributes)
        {
            await RegisterSearchAttributes(co.Namespace);
        }

        _runCtx = new RunContext(req.TaskQueue, req.ExecutionId, req.RunId);
        return new InitResponse();
    }

    internal async Task<ExecuteResponse> HandleExecute(ExecuteRequest req)
    {
        if (req == null)
            throw new RpcException(new Status(StatusCode.InvalidArgument, "execute request is nil"));
        if (_executeFn == null)
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "no execute handler"));
        if (_runCtx == null)
            throw new RpcException(new Status(StatusCode.FailedPrecondition, "not initialized"));

        await _executeFn(_client!, new ExecuteInfo(
            _runCtx.TaskQueue,
            _runCtx.ExecutionId,
            _runCtx.RunId,
            req.Iteration,
            req.Payload?.ToByteArray()
        ));

        return new ExecuteResponse();
    }

    // --- Internal helpers ---

    private static void ValidateInit(InitRequest req)
    {
        if (req == null)
            throw new RpcException(new Status(StatusCode.InvalidArgument, "request: required"));
        if (string.IsNullOrEmpty(req.ExecutionId))
            throw new RpcException(new Status(StatusCode.InvalidArgument, "execution_id: required"));
        if (string.IsNullOrEmpty(req.RunId))
            throw new RpcException(new Status(StatusCode.InvalidArgument, "run_id: required"));
        if (string.IsNullOrEmpty(req.TaskQueue))
            throw new RpcException(new Status(StatusCode.InvalidArgument, "task_queue: required"));
        var co = req.ConnectOptions;
        if (co == null)
            throw new RpcException(new Status(StatusCode.InvalidArgument, "connect_options: required"));
        if (string.IsNullOrEmpty(co.Namespace))
            throw new RpcException(new Status(StatusCode.InvalidArgument, "connect_options.namespace: required"));
        if (string.IsNullOrEmpty(co.ServerAddress))
            throw new RpcException(new Status(StatusCode.InvalidArgument, "connect_options.server_address: required"));
    }

    internal static async Task<TemporalClientConnectOptions> BuildClientConnectOptions(
        string serverAddress,
        string ns,
        string? authHeader,
        bool enableTls,
        string? tlsCertPath,
        string? tlsKeyPath,
        string? tlsServerName)
    {
        var opts = new TemporalClientConnectOptions(serverAddress) { Namespace = ns };

        if (!string.IsNullOrEmpty(authHeader))
        {
            opts.RpcMetadata = new Dictionary<string, string>
            {
                ["Authorization"] = authHeader
            };
        }

        if (enableTls || !string.IsNullOrEmpty(tlsCertPath))
        {
            opts.Tls = new TlsOptions();
            if (!string.IsNullOrEmpty(tlsServerName))
            {
                opts.Tls.Domain = tlsServerName;
            }
            if (!string.IsNullOrEmpty(tlsCertPath) && !string.IsNullOrEmpty(tlsKeyPath))
            {
                opts.Tls.ClientCert = await File.ReadAllBytesAsync(tlsCertPath);
                opts.Tls.ClientPrivateKey = await File.ReadAllBytesAsync(tlsKeyPath);
            }
        }

        return opts;
    }

    private async Task RegisterSearchAttributes(string ns)
    {
        try
        {
            await _client!.Connection.OperatorService.AddSearchAttributesAsync(
                new Temporalio.Api.OperatorService.V1.AddSearchAttributesRequest
                {
                    Namespace = ns,
                    SearchAttributes =
                    {
                        { "KS_Keyword", Temporalio.Api.Enums.V1.IndexedValueType.Keyword },
                        { "KS_Int", Temporalio.Api.Enums.V1.IndexedValueType.Int },
                        { OmesSearchAttributeKey, Temporalio.Api.Enums.V1.IndexedValueType.Keyword },
                    }
                });
        }
        catch (RpcException ex) when (ex.StatusCode == StatusCode.AlreadyExists)
        {
            // Already registered, that's fine
        }
        catch (RpcException ex) when (ex.Message.Contains("attributes mapping unavailable"))
        {
            // Also fine
        }
    }

    private record RunContext(string TaskQueue, string ExecutionId, string RunId);
}

/// <summary>
/// gRPC service implementation that delegates to the harness.
/// </summary>
public class ProjectServiceImpl : ProjectService.ProjectServiceBase
{
    private readonly ProjectHarness _harness;

    public ProjectServiceImpl(ProjectHarness harness)
    {
        _harness = harness;
    }

    public override async Task<InitResponse> Init(InitRequest request, ServerCallContext context)
    {
        return await _harness.HandleInit(request);
    }

    public override async Task<ExecuteResponse> Execute(ExecuteRequest request, ServerCallContext context)
    {
        return await _harness.HandleExecute(request);
    }
}
