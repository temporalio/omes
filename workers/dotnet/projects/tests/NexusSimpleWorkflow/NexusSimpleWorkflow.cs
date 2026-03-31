using Temporalio.Client;
using Temporalio.Common;
using Temporalio.Omes.Projects.Harness;
using NexusSimpleWorkflowProject;

public static class NexusSimpleWorkflow
{
    public static Task<int> RunAsync(string[] args)
    {
        string? nexusEndpointName = null;

        var harness = new ProjectHarness();

        harness.RegisterClient(async (opts, config) =>
        {
            return await TemporalClient.ConnectAsync(opts);
        });

        harness.OnInit(async (client, config) =>
        {
            nexusEndpointName = $"nexus-endpoint-{config.TaskQueue}";
            await client.Connection.OperatorService.CreateNexusEndpointAsync(
                new Temporalio.Api.OperatorService.V1.CreateNexusEndpointRequest
                {
                    Spec = new Temporalio.Api.Nexus.V1.EndpointSpec
                    {
                        Name = nexusEndpointName,
                        Target = new Temporalio.Api.Nexus.V1.EndpointTarget
                        {
                            Worker = new Temporalio.Api.Nexus.V1.EndpointTarget.Types.Worker
                            {
                                Namespace = client.Options.Namespace,
                                TaskQueue = config.TaskQueue,
                            },
                        },
                    },
                });
            Console.WriteLine($"Created Nexus endpoint: {nexusEndpointName}");
        });

        harness.RegisterWorker(async (client, config) =>
        {
            using var worker = new Temporalio.Worker.TemporalWorker(client,
                new Temporalio.Worker.TemporalWorkerOptions(config.TaskQueue)
                    .AddWorkflow<CallerWorkflow>()
                    .AddWorkflow<HandlerWorkflow>()
                    .AddNexusService(new StringServiceHandler()));
            using var cts = new CancellationTokenSource();
            Console.CancelKeyPress += (_, e) => { e.Cancel = true; cts.Cancel(); };
            await worker.ExecuteAsync(cts.Token);
        });

        harness.OnExecute(async (client, info) =>
        {
            var handle = await client.StartWorkflowAsync(
                (CallerWorkflow wf) => wf.RunAsync(nexusEndpointName!, "some-name"),
                new WorkflowOptions(id: $"nexus-caller-{info.Iteration}", taskQueue: info.TaskQueue)
                {
                    TypedSearchAttributes = new SearchAttributeCollection.Builder()
                        .Set(SearchAttributeKey.CreateKeyword(ProjectHarness.OmesSearchAttributeKey), info.ExecutionId)
                        .ToSearchAttributeCollection()
                });

            var result = await handle.GetResultAsync();
            Console.WriteLine($"Nexus workflow result: {result}");

            if (result != "Hello from workflow, some-name")
            {
                throw new Exception($"unexpected result: {result}");
            }
        });

        return harness.RunAsync(args);
    }
}
