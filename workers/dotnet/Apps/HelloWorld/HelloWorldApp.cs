using Temporalio.Client;
using Temporalio.Omes.Projects.Harness;
using Temporalio.Worker;

namespace Temporalio.Omes.Apps.HelloWorld;

public static class HelloWorldApp
{
    public static Temporalio.Omes.Projects.Harness.App Create() =>
        new(
            Worker: CreateWorker,
            ClientFactory: ClientHelpers.DefaultClientFactory,
            Project: new ProjectHandlers(Execute: ExecuteProjectIterationAsync));

    private static TemporalWorker CreateWorker(ITemporalClient client, WorkerContext context)
    {
        var workerOptions = context.WorkerOptions;
        workerOptions.AddWorkflow<HelloWorldWorkflow>();
        return new TemporalWorker(client, workerOptions);
    }

    private static async Task ExecuteProjectIterationAsync(ITemporalClient client, ProjectExecuteContext context)
    {
        var handle = await client.StartWorkflowAsync(
            (HelloWorldWorkflow workflow) => workflow.RunAsync("World"),
            new WorkflowOptions(
                id: $"{context.Run.ExecutionId}-{context.Iteration}",
                taskQueue: context.TaskQueue));
        var result = await handle.GetResultAsync<string>();
        Console.WriteLine(result);
    }
}
