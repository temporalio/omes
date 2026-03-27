using System.CommandLine;
using System.CommandLine.Invocation;
using Temporalio.Client;
using Temporalio.Runtime;

namespace Temporalio.Omes.Projects.Harness;

public partial class ProjectHarness
{
    private static readonly Option<string> ServerAddressOption = new(
        "--server-address", () => "localhost:7233", "Temporal server address");
    private static readonly Option<string> NamespaceOption = new(
        "--namespace", () => "default", "Temporal namespace");
    private static readonly Option<string> TaskQueueOption = new(
        "--task-queue", "Task queue name (required)");
    private static readonly Option<string> AuthHeaderOption = new(
        "--auth-header", "Authorization header value");
    private static readonly Option<bool> TlsOption = new(
        "--tls", "Enable TLS");
    private static readonly Option<string> TlsCertPathOption = new(
        "--tls-cert-path", "Path to client TLS certificate");
    private static readonly Option<string> TlsKeyPathOption = new(
        "--tls-key-path", "Path to client TLS private key");
    private static readonly Option<string> TlsServerNameOption = new(
        "--tls-server-name", "TLS target server name");
    private static readonly Option<bool> DisableHostVerificationOption = new(
        "--disable-tls-host-verification", "Disable TLS host verification (not supported in .NET SDK)");
    private static readonly Option<string> PromListenAddressOption = new(
        "--prom-listen-address", "Prometheus metrics address");

    private Command BuildWorkerCommand()
    {
        var cmd = new Command("worker", "Run the project worker");
        cmd.Add(TaskQueueOption);
        cmd.Add(ServerAddressOption);
        cmd.Add(NamespaceOption);
        cmd.Add(AuthHeaderOption);
        cmd.Add(TlsOption);
        cmd.Add(TlsCertPathOption);
        cmd.Add(TlsKeyPathOption);
        cmd.Add(TlsServerNameOption);
        cmd.Add(DisableHostVerificationOption);
        cmd.Add(PromListenAddressOption);
        cmd.SetHandler(StartWorkerAsync);
        return cmd;
    }

    private async Task StartWorkerAsync(InvocationContext ctx)
    {
        if (_workerFn == null)
            throw new InvalidOperationException("No worker registered");

        var taskQueue = ctx.ParseResult.GetValueForOption(TaskQueueOption)!;
        if (string.IsNullOrEmpty(taskQueue))
        {
            Console.Error.WriteLine("error: --task-queue is required");
            Environment.Exit(1);
        }

        var opts = await BuildClientConnectOptions(
            ctx.ParseResult.GetValueForOption(ServerAddressOption)!,
            ctx.ParseResult.GetValueForOption(NamespaceOption)!,
            ctx.ParseResult.GetValueForOption(AuthHeaderOption),
            ctx.ParseResult.GetValueForOption(TlsOption),
            ctx.ParseResult.GetValueForOption(TlsCertPathOption),
            ctx.ParseResult.GetValueForOption(TlsKeyPathOption),
            ctx.ParseResult.GetValueForOption(TlsServerNameOption));

        // Set up Prometheus metrics if address provided
        var promListenAddress = ctx.ParseResult.GetValueForOption(PromListenAddressOption);
        if (!string.IsNullOrEmpty(promListenAddress))
        {
            var runtime = new TemporalRuntime(new()
            {
                Telemetry = new TelemetryOptions
                {
                    Metrics = new MetricsOptions
                    {
                        Prometheus = new PrometheusOptions(promListenAddress)
                        {
                            UseSecondsForDuration = true
                        }
                    }
                }
            });
            opts.Runtime = runtime;
        }

        var client = await _clientFn!(opts, new ClientConfig(taskQueue));

        Console.WriteLine($"Worker starting on task queue: {taskQueue}");
        await _workerFn(client, new WorkerConfig(taskQueue, promListenAddress));
    }
}
