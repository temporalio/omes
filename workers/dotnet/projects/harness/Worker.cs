using System.CommandLine;
using System.CommandLine.Invocation;
using System.CommandLine.Parsing;
using System.Runtime.ExceptionServices;
using Microsoft.Extensions.Logging;
using Temporalio.Client;
using Temporalio.Worker;
using Temporalio.Worker.Tuning;

namespace Temporalio.Omes.Projects.Harness;

public sealed record WorkerContext(
    ILogger Logger,
    string TaskQueue,
    bool ErrOnUnimplemented,
    TemporalWorkerOptions WorkerOptions);

public delegate TemporalWorker WorkerFactory(ITemporalClient client, WorkerContext context);

// This small generic seam exists only to let tests substitute fake worker objects, which is the
// closest .NET analogue to Python patching `_run_workers`.
internal delegate TWorker WorkerFactoryCore<out TWorker>(ITemporalClient client, WorkerContext context);

internal static class WorkerHarness
{
    private static readonly Option<string> ServerAddressOption = new(
        aliases: new[] { "-a", "--server-address" },
        description: "The host:port of the server",
        getDefaultValue: () => "localhost:7233");

    private static readonly Option<string> TaskQueueOption = new(
        aliases: new[] { "-q", "--task-queue" },
        description: "The task queue to use",
        getDefaultValue: () => "omes");

    private static readonly Option<string> NamespaceOption = new(
        aliases: new[] { "-n", "--namespace" },
        description: "The namespace to use",
        getDefaultValue: () => "default");

    private static readonly Option<int> TaskQueueSuffixStartOption = new(
        name: "--task-queue-suffix-index-start",
        description: "Inclusive start for task queue suffix range");

    private static readonly Option<int> TaskQueueSuffixEndOption = new(
        name: "--task-queue-suffix-index-end",
        description: "Inclusive end for task queue suffix range");

    private static readonly Option<int?> MaxConcurrentActivityPollersOption = new(
        name: "--max-concurrent-activity-pollers",
        description: "Max concurrent activity pollers");

    private static readonly Option<int?> MaxConcurrentWorkflowPollersOption = new(
        name: "--max-concurrent-workflow-pollers",
        description: "Max concurrent workflow pollers");

    private static readonly Option<int?> ActivityPollerAutoscaleMaxOption = new(
        name: "--activity-poller-autoscale-max",
        description: "Max for activity poller autoscaling (overrides max-concurrent-activity-pollers)");

    private static readonly Option<int?> WorkflowPollerAutoscaleMaxOption = new(
        name: "--workflow-poller-autoscale-max",
        description: "Max for workflow poller autoscaling (overrides max-concurrent-workflow-pollers)");

    private static readonly Option<int?> MaxConcurrentActivitiesOption = new(
        name: "--max-concurrent-activities",
        description: "Max concurrent activities");

    private static readonly Option<int?> MaxConcurrentWorkflowTasksOption = new(
        name: "--max-concurrent-workflow-tasks",
        description: "Max concurrent workflow tasks");

    private static readonly Option<double?> ActivitiesPerSecondOption = new(
        name: "--activities-per-second",
        description: "Per-worker activity rate limit");

    private static readonly Option<string> LogLevelOption = new Option<string>(
        name: "--log-level",
        description: "Log level",
        getDefaultValue: () => "info");

    private static readonly Option<string> LogEncodingOption = new Option<string>(
        name: "--log-encoding",
        description: "Log encoding",
        getDefaultValue: () => "console").FromAmong("console", "json");

    private static readonly Option<bool> TlsOption = new(
        name: "--tls",
        description: "Enable TLS");

    private static readonly Option<string> TlsCertPathOption = new(
        name: "--tls-cert-path",
        description: "Path to a client certificate for TLS",
        getDefaultValue: () => string.Empty);

    private static readonly Option<string> TlsKeyPathOption = new(
        name: "--tls-key-path",
        description: "Path to a client key for TLS",
        getDefaultValue: () => string.Empty);

    private static readonly Option<string?> PromListenAddressOption = new(
        name: "--prom-listen-address",
        description: "Prometheus listen address");

    private static readonly Option<string> AuthHeaderOption = new(
        name: "--auth-header",
        description: "Authorization header value",
        getDefaultValue: () => string.Empty);

    private static readonly Option<string?> PromHandlerPathOption = new(
        name: "--prom-handler-path",
        description: "Prometheus handler path",
        getDefaultValue: () => "/metrics");

    private static readonly Option<string?> BuildIdOption = new(
        name: "--build-id",
        description: "Build ID");

    private static readonly Option<bool> ErrOnUnimplementedOption = new(
        name: "--err-on-unimplemented",
        description: "Error when receiving unimplemented actions (currently only affects concurrent client actions)",
        getDefaultValue: () => false);

    public static async Task<int> RunAsync(App app, string[] args)
    {
        var command = CreateWorkerCommand();
        command.SetHandler(
            async (InvocationContext context) =>
            {
                try
                {
                    if (context.ParseResult.Errors.Count > 0)
                    {
                        throw new ArgumentException(context.ParseResult.Errors[0].Message);
                    }

                    var options = ParseWorkerOptions(context.ParseResult);
                    await RunCoreAsync(
                        workerFactory: (client, workerContext) => app.Worker(client, workerContext),
                        clientFactory: app.ClientFactory,
                        options: options,
                        runWorkersAsync: RunWorkersAsync);
                    context.ExitCode = 0;
                }
                catch (ArgumentException err)
                {
                    Console.Error.WriteLine(err.Message);
                    context.ExitCode = 1;
                }
            });
        return await command.InvokeAsync(args);
    }

    internal static async Task RunCoreAsync<TWorker>(
        WorkerFactoryCore<TWorker> workerFactory,
        ClientFactory clientFactory,
        WorkerCliOptions options,
        Func<IReadOnlyList<TWorker>, Task> runWorkersAsync)
    {
        if (options.TaskQueueSuffixIndexStart > options.TaskQueueSuffixIndexEnd)
        {
            throw new ArgumentException("Task queue suffix start after end");
        }

        var loggerFactory = HarnessHelpers.ConfigureLogger(options.LogLevel, options.LogEncoding);
        var logger = loggerFactory.CreateLogger("Temporalio.Omes.Projects.Harness");
        var clientConfig = ClientHelpers.BuildClientConfig(
            serverAddress: options.ServerAddress,
            @namespace: options.Namespace,
            authHeader: options.AuthHeader,
            tls: options.Tls,
            tlsCertPath: options.TlsCertPath,
            tlsKeyPath: options.TlsKeyPath,
            promListenAddress: options.PromListenAddress,
            loggerFactory: loggerFactory);

        var workerSettings = BuildWorkerSettings(options);
        var client = await clientFactory(clientConfig);
        var taskQueues = BuildTaskQueues(
            logger,
            options.TaskQueue,
            options.TaskQueueSuffixIndexStart,
            options.TaskQueueSuffixIndexEnd);

        var workers = taskQueues
            .Select(taskQueue =>
                workerFactory(
                    client,
                    new WorkerContext(
                        Logger: logger,
                        TaskQueue: taskQueue,
                        ErrOnUnimplemented: options.ErrOnUnimplemented,
                        WorkerOptions: BuildWorkerOptions(taskQueue, workerSettings))))
            .ToList();

        await runWorkersAsync(workers);
    }

    internal static Task RunWorkersAsync(IReadOnlyList<TemporalWorker> workers) =>
        RunWorkersAsync(
            workers,
            static (worker, cancellationToken) => worker.ExecuteAsync(cancellationToken),
            static worker => worker.Dispose());

    internal static async Task RunWorkersAsync<TWorker>(
        IReadOnlyList<TWorker> workers,
        Func<TWorker, CancellationToken, Task> executeWorkerAsync,
        Action<TWorker> disposeWorker)
    {
        using var cancellationTokenSource = new CancellationTokenSource();
        ConsoleCancelEventHandler? cancelHandler = (_, eventArgs) =>
        {
            eventArgs.Cancel = true;
            cancellationTokenSource.Cancel();
        };

        Console.CancelKeyPress += cancelHandler;
        try
        {
            var workerTasks = workers
                .Select(worker => executeWorkerAsync(worker, cancellationTokenSource.Token))
                .ToList();

            var completedTask = await Task.WhenAny(workerTasks);
            cancellationTokenSource.Cancel();

            Exception? firstFailure = null;
            if (completedTask.IsFaulted)
            {
                firstFailure = completedTask.Exception?.GetBaseException();
            }
            try
            {
                await Task.WhenAll(workerTasks);
            }
            catch
            {
                firstFailure ??= workerTasks
                    .Where(task => task.IsFaulted)
                    .Select(task => task.Exception?.GetBaseException())
                    .FirstOrDefault(exception => exception is not null);
            }

            if (firstFailure is not null)
            {
                ExceptionDispatchInfo.Capture(firstFailure).Throw();
            }
        }
        finally
        {
            Console.CancelKeyPress -= cancelHandler;
            foreach (var worker in workers)
            {
                disposeWorker(worker);
            }
        }
    }

    private static List<string> BuildTaskQueues(
        ILogger logger,
        string taskQueue,
        int suffixStart,
        int suffixEnd)
    {
        if (suffixEnd == 0)
        {
            logger.LogInformation(".NET worker will run on task queue {TaskQueue}", taskQueue);
            return new List<string> { taskQueue };
        }

        var taskQueues = Enumerable.Range(suffixStart, suffixEnd - suffixStart + 1)
            .Select(index => $"{taskQueue}-{index}")
            .ToList();
        logger.LogInformation(".NET worker will run on {Count} task queue(s)", taskQueues.Count);
        return taskQueues;
    }

    private static WorkerSettings BuildWorkerSettings(WorkerCliOptions options) =>
        new(
            MaxConcurrentActivities: options.MaxConcurrentActivities,
            MaxConcurrentWorkflowTasks: options.MaxConcurrentWorkflowTasks,
            ActivitiesPerSecond: options.ActivitiesPerSecond,
            MaxConcurrentActivityPollers: options.MaxConcurrentActivityPollers,
            MaxConcurrentWorkflowPollers: options.MaxConcurrentWorkflowPollers,
            ActivityPollerAutoscaleMax: options.ActivityPollerAutoscaleMax,
            WorkflowPollerAutoscaleMax: options.WorkflowPollerAutoscaleMax);

    private static TemporalWorkerOptions BuildWorkerOptions(string taskQueue, WorkerSettings settings)
    {
        var workerOptions = new TemporalWorkerOptions(taskQueue);
        if (settings.MaxConcurrentWorkflowTasks is { } maxConcurrentWorkflowTasks)
        {
            workerOptions.MaxConcurrentWorkflowTasks = maxConcurrentWorkflowTasks;
        }

        if (settings.MaxConcurrentActivities is { } maxConcurrentActivities)
        {
            workerOptions.MaxConcurrentActivities = maxConcurrentActivities;
        }

        if (settings.ActivitiesPerSecond is { } activitiesPerSecond)
        {
            workerOptions.MaxActivitiesPerSecond = activitiesPerSecond;
        }

        if (settings.ActivityPollerAutoscaleMax is { } activityAutoscaleMax)
        {
            workerOptions.ActivityTaskPollerBehavior = new PollerBehavior.Autoscaling(maximum: activityAutoscaleMax);
        }
        else if (settings.MaxConcurrentActivityPollers is { } maxConcurrentActivityPollers)
        {
            workerOptions.MaxConcurrentActivityTaskPolls = maxConcurrentActivityPollers;
        }

        if (settings.WorkflowPollerAutoscaleMax is { } workflowAutoscaleMax)
        {
            workerOptions.WorkflowTaskPollerBehavior = new PollerBehavior.Autoscaling(maximum: workflowAutoscaleMax);
        }
        else if (settings.MaxConcurrentWorkflowPollers is { } maxConcurrentWorkflowPollers)
        {
            workerOptions.MaxConcurrentWorkflowTaskPolls = maxConcurrentWorkflowPollers;
        }

        return workerOptions;
    }

    internal static WorkerCliOptions ParseWorkerOptions(ParseResult parseResult) =>
        new(
            ServerAddress: parseResult.GetValueForOption(ServerAddressOption)!,
            TaskQueue: parseResult.GetValueForOption(TaskQueueOption)!,
            Namespace: parseResult.GetValueForOption(NamespaceOption)!,
            TaskQueueSuffixIndexStart: parseResult.GetValueForOption(TaskQueueSuffixStartOption),
            TaskQueueSuffixIndexEnd: parseResult.GetValueForOption(TaskQueueSuffixEndOption),
            MaxConcurrentActivityPollers: parseResult.GetValueForOption(MaxConcurrentActivityPollersOption),
            MaxConcurrentWorkflowPollers: parseResult.GetValueForOption(MaxConcurrentWorkflowPollersOption),
            ActivityPollerAutoscaleMax: parseResult.GetValueForOption(ActivityPollerAutoscaleMaxOption),
            WorkflowPollerAutoscaleMax: parseResult.GetValueForOption(WorkflowPollerAutoscaleMaxOption),
            MaxConcurrentActivities: parseResult.GetValueForOption(MaxConcurrentActivitiesOption),
            MaxConcurrentWorkflowTasks: parseResult.GetValueForOption(MaxConcurrentWorkflowTasksOption),
            ActivitiesPerSecond: parseResult.GetValueForOption(ActivitiesPerSecondOption),
            LogLevel: parseResult.GetValueForOption(LogLevelOption)!,
            LogEncoding: parseResult.GetValueForOption(LogEncodingOption)!,
            Tls: parseResult.GetValueForOption(TlsOption),
            TlsCertPath: parseResult.GetValueForOption(TlsCertPathOption)!,
            TlsKeyPath: parseResult.GetValueForOption(TlsKeyPathOption)!,
            PromListenAddress: parseResult.GetValueForOption(PromListenAddressOption),
            AuthHeader: parseResult.GetValueForOption(AuthHeaderOption)!,
            PromHandlerPath: parseResult.GetValueForOption(PromHandlerPathOption),
            BuildId: parseResult.GetValueForOption(BuildIdOption),
            ErrOnUnimplemented: parseResult.GetValueForOption(ErrOnUnimplementedOption));

    internal static RootCommand CreateWorkerCommand()
    {
        var command = new RootCommand(".NET project harness worker");
        command.AddOption(ServerAddressOption);
        command.AddOption(TaskQueueOption);
        command.AddOption(NamespaceOption);
        command.AddOption(TaskQueueSuffixStartOption);
        command.AddOption(TaskQueueSuffixEndOption);
        command.AddOption(MaxConcurrentActivityPollersOption);
        command.AddOption(MaxConcurrentWorkflowPollersOption);
        command.AddOption(ActivityPollerAutoscaleMaxOption);
        command.AddOption(WorkflowPollerAutoscaleMaxOption);
        command.AddOption(MaxConcurrentActivitiesOption);
        command.AddOption(MaxConcurrentWorkflowTasksOption);
        command.AddOption(ActivitiesPerSecondOption);
        command.AddOption(LogLevelOption);
        command.AddOption(LogEncodingOption);
        command.AddOption(TlsOption);
        command.AddOption(TlsCertPathOption);
        command.AddOption(TlsKeyPathOption);
        command.AddOption(PromListenAddressOption);
        command.AddOption(AuthHeaderOption);
        command.AddOption(PromHandlerPathOption);
        command.AddOption(BuildIdOption);
        command.AddOption(ErrOnUnimplementedOption);
        return command;
    }

    private sealed record WorkerSettings(
        int? MaxConcurrentActivities,
        int? MaxConcurrentWorkflowTasks,
        double? ActivitiesPerSecond,
        int? MaxConcurrentActivityPollers,
        int? MaxConcurrentWorkflowPollers,
        int? ActivityPollerAutoscaleMax,
        int? WorkflowPollerAutoscaleMax);

    internal sealed record WorkerCliOptions(
        string ServerAddress,
        string TaskQueue,
        string Namespace,
        int TaskQueueSuffixIndexStart,
        int TaskQueueSuffixIndexEnd,
        int? MaxConcurrentActivityPollers,
        int? MaxConcurrentWorkflowPollers,
        int? ActivityPollerAutoscaleMax,
        int? WorkflowPollerAutoscaleMax,
        int? MaxConcurrentActivities,
        int? MaxConcurrentWorkflowTasks,
        double? ActivitiesPerSecond,
        string LogLevel,
        string LogEncoding,
        bool Tls,
        string TlsCertPath,
        string TlsKeyPath,
        string? PromListenAddress,
        string AuthHeader,
        string? PromHandlerPath,
        string? BuildId,
        bool ErrOnUnimplemented);
}
