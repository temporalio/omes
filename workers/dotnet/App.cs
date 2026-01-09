using System.CommandLine;
using System.CommandLine.Invocation;
using Temporalio.Client;
using Temporalio.Runtime;
using Temporalio.Worker;
using Temporalio.Worker.Tuning;

namespace Temporalio.Omes;

/// <summary>
/// Main application that can parse args and run command.
/// </summary>
public static class App
{
    private static readonly Option<string> serverOption = new(
        aliases: new[] { "-a", "--server-address" },
        description: "The host:port of the server",
        getDefaultValue: () => "localhost:7233");

    private static readonly Option<string> taskQueueOption = new(
        aliases: new[] { "-q", "--task-queue" },
        description: "The task queue to use",
        getDefaultValue: () => "omes");

    private static readonly Option<string> namespaceOption = new(
        aliases: new[] { "-n", "--namespace" },
        description: "The namespace to use",
        getDefaultValue: () => "default");

    private static readonly Option<uint> taskQSuffixStartOption = new(
        name: "--task-queue-suffix-index-start",
        description: "Inclusive start for task queue suffix range");

    private static readonly Option<uint> taskQSuffixEndOption = new(
        name: "--task-queue-suffix-index-end",
        description: "Inclusive end for task queue suffix range");

    private static readonly Option<uint?> maxATPollersOption = new(
        name: "--max-concurrent-activity-pollers",
        description: "Max concurrent activity pollers");

    private static readonly Option<uint?> maxWFTPollersOption = new(
        name: "--max-concurrent-workflow-pollers",
        description: "Max concurrent workflow pollers");

    private static readonly Option<uint?> activityPollerAutoscaleMaxOption = new(
        name: "--activity-poller-autoscale-max",
        description: "Max for activity poller autoscaling (overrides max-concurrent-activity-pollers)");

    private static readonly Option<uint?> workflowPollerAutoscaleMaxOption = new(
        name: "--workflow-poller-autoscale-max",
        description: "Max for workflow poller autoscaling (overrides max-concurrent-workflow-pollers)");

    private static readonly Option<uint?> maxATOption = new(
        name: "--max-concurrent-activities",
        description: "Max concurrent activities");

    private static readonly Option<uint?> maxWFTOption = new(
        name: "--max-concurrent-workflow-tasks",
        description: "Max concurrent workflow tasks");

    private static readonly Option<double?> workerActivitiesPerSecond = new(
        name: "--activities-per-second",
        description: "Per-worker activity rate limit");

    private static readonly Option<string> logLevelOption = new Option<string>(
        name: "--log-level",
        description: "Log level",
        getDefaultValue: () => "info").FromAmong("trace", "debug", "info", "warn", "error");

    private static readonly Option<string> logEncodingOption = new Option<string>(
        name: "--log-encoding",
        description: "Log encoding",
        getDefaultValue: () => "console").FromAmong("console", "json");

    private static readonly Option<bool> useTLSOption = new(
        name: "--tls",
        description: "Enable TLS");

    private static readonly Option<FileInfo?> clientCertPathOption = new(
        name: "--tls-cert-path",
        description: "Path to a client certificate for TLS");

    private static readonly Option<FileInfo?> clientKeyPathOption = new(
        name: "--tls-key-path",
        description: "Path to a client key for TLS");

    private static readonly Option<string> promAddrOption = new Option<string>(
        name: "--prom-listen-address",
        description: "Prometheus listen address");

    private static readonly Option<string> authHeader = new Option<string>(
        name: "--auth-header",
        description: "Authorization header value");

    private static readonly Option<string> promHandlerPathOption = new Option<string>(
        name: "--prom-handler-path",
        description: "Prometheus handler path",
        getDefaultValue: () => "/metrics");

    private static readonly Option<string> buildIdOption = new Option<string>(
        name: "--build-id",
        description: "Build ID");

    private static readonly Option<bool> errOnUnimplementedOption = new(
        name: "--err-on-unimplemented",
        description: "Error when receiving unimplemented actions (currently only affects concurrent client actions)",
        getDefaultValue: () => false);

    /// <summary>
    /// Run Omes worker with the given args.
    /// </summary>
    /// <param name="args">CLI args.</param>
    /// <returns>Task for completion.</returns>
    public static Task<int> RunAsync(string[] args) => CreateCommand().InvokeAsync(args);

    private static Command CreateCommand()
    {
        var cmd = new RootCommand(".NET omes worker");
        cmd.Add(serverOption);
        cmd.Add(taskQueueOption);
        cmd.Add(namespaceOption);
        cmd.Add(taskQSuffixStartOption);
        cmd.Add(taskQSuffixEndOption);
        cmd.Add(maxATPollersOption);
        cmd.Add(maxWFTPollersOption);
        cmd.Add(activityPollerAutoscaleMaxOption);
        cmd.Add(workflowPollerAutoscaleMaxOption);
        cmd.Add(maxATOption);
        cmd.Add(maxWFTOption);
        cmd.Add(workerActivitiesPerSecond);
        cmd.Add(logLevelOption);
        cmd.Add(logEncodingOption);
        cmd.Add(useTLSOption);
        cmd.Add(clientCertPathOption);
        cmd.Add(clientKeyPathOption);
        cmd.Add(promAddrOption);
        cmd.Add(promHandlerPathOption);
        cmd.Add(authHeader);
        cmd.Add(buildIdOption);
        cmd.Add(errOnUnimplementedOption);
        cmd.SetHandler(RunCommandAsync);
        return cmd;
    }

    private static async Task RunCommandAsync(InvocationContext ctx)
    {
        // Create logger factory
        using var loggerFactory = LoggerFactory.Create(builder => builder.AddSimpleConsole(
            options =>
            {
                options.IncludeScopes = true;
                options.SingleLine = true;
                options.TimestampFormat = "HH:mm:ss ";
            }));
        var logger = loggerFactory.CreateLogger(typeof(App));

        var promAddr = ctx.ParseResult.GetValueForOption(promAddrOption);

        var runtime = new TemporalRuntime(new()
        {
            Telemetry = new TelemetryOptions
            {
                Logging = new() { Filter = new(TelemetryFilterOptions.Level.Info) },
                Metrics = promAddr != null ? new MetricsOptions
                {
                    Prometheus = new PrometheusOptions(promAddr)
                    {
                        UseSecondsForDuration = true
                    }
                } : null
            }
        });

        // Connect a client
        TlsOptions? tls = null;
        var certPath = ctx.ParseResult.GetValueForOption(clientCertPathOption);
        if (ctx.ParseResult.GetValueForOption(useTLSOption) || certPath != null)
        {
            tls = certPath is null ? new() : new()
            {
                ClientCert = await File.ReadAllBytesAsync(certPath.FullName),
                ClientPrivateKey = await File.ReadAllBytesAsync(
                        ctx.ParseResult.GetValueForOption(clientKeyPathOption)?.FullName ??
                        throw new ArgumentException("Missing key with cert"))
            };
        }

        // Configure API key
        var authHeaderValue = ctx.ParseResult.GetValueForOption(authHeader);
        var apiKey = authHeaderValue?.StartsWith("Bearer ") == true
            ? authHeaderValue.Substring("Bearer ".Length)
            : authHeaderValue;

        var serverAddr = ctx.ParseResult.GetValueForOption(serverOption)!;
        logger.LogInformation(".NET Omes will connect to server at {}", serverAddr);

        var client = await TemporalClient.ConnectAsync(
            new(serverAddr)
            {
                Runtime = runtime,
                Namespace = ctx.ParseResult.GetValueForOption(namespaceOption)!,
                Tls = tls,
                ApiKey = apiKey,
                LoggerFactory = loggerFactory
            });

        // Collect task queues to run workers for
        var taskQueues = new List<string>();
        var taskQueueBase = ctx.ParseResult.GetValueForOption(taskQueueOption)!;
        if (ctx.ParseResult.GetValueForOption(taskQSuffixStartOption) == 0)
        {
            taskQueues.Add(taskQueueBase);
        }
        else
        {
            var start = ctx.ParseResult.GetValueForOption(taskQSuffixStartOption);
            var end = ctx.ParseResult.GetValueForOption(taskQSuffixEndOption);
            for (var i = start; i <= end; i++)
            {
                taskQueues.Add($"{taskQueueBase}-{i}");
            }
        }

        logger.LogInformation("Running .NET workers for {Count} task queues", taskQueues.Count);

        // Start all workers, exiting early if any fail
        var workerTasks = new List<Task>();
        foreach (var taskQueue in taskQueues)
        {
            var workerOptions = new TemporalWorkerOptions(taskQueue);
            if (ctx.ParseResult.GetValueForOption(maxWFTOption) is { } maxWft)
            {
                workerOptions.MaxConcurrentWorkflowTasks = (int)maxWft;
            }

            if (ctx.ParseResult.GetValueForOption(maxATOption) is { } maxAt)
            {
                workerOptions.MaxConcurrentActivities = (int)maxAt;
            }

            if (ctx.ParseResult.GetValueForOption(workerActivitiesPerSecond) is { } rate)
            {
                workerOptions.MaxActivitiesPerSecond = rate;
            }

            // Configure poller behaviors with autoscaling support
            if (ctx.ParseResult.GetValueForOption(activityPollerAutoscaleMaxOption) is { } activityAutoscaleMax)
            {
                workerOptions.ActivityTaskPollerBehavior = new PollerBehavior.Autoscaling(maximum: (int)activityAutoscaleMax);
            }
            if (ctx.ParseResult.GetValueForOption(workflowPollerAutoscaleMaxOption) is { } workflowAutoscaleMax)
            {
                workerOptions.WorkflowTaskPollerBehavior = new PollerBehavior.Autoscaling(maximum: (int)workflowAutoscaleMax);
            }

            workerOptions.AddWorkflow<KitchenSinkWorkflow>();
            workerOptions.AddActivity(KitchenSinkWorkflow.Noop);
            workerOptions.AddActivity(KitchenSinkWorkflow.Delay);
            workerOptions.AddActivity(KitchenSinkWorkflow.Payload);
            var errOnUnimplemented = ctx.ParseResult.GetValueForOption(errOnUnimplementedOption);
            var clientActivities = new ClientActivitiesImpl(client, errOnUnimplemented);
            workerOptions.AddActivity(clientActivities.Client);
            var worker = new TemporalWorker(client, workerOptions);
            var workerTask = worker.ExecuteAsync(default);
            workerTasks.Add(workerTask);
        }

        var doneTask = await Task.WhenAny(workerTasks.ToArray());
        await doneTask;
        if (doneTask.IsFaulted)
        {
            throw doneTask.Exception!;
        }
        // Make sure every worker task is completed
        await Task.WhenAll(workerTasks.ToArray());
    }
}
