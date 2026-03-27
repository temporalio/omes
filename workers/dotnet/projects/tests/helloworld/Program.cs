using Temporalio.Client;
using Temporalio.Common;
using Temporalio.Omes.Projects.Harness;

var harness = new ProjectHarness();
harness.RegisterClient(async (opts, config) =>
{
    return await TemporalClient.ConnectAsync(opts);
});

harness.RegisterWorker(async (client, config) =>
{
    using var worker = new Temporalio.Worker.TemporalWorker(client, new Temporalio.Worker.TemporalWorkerOptions(config.TaskQueue)
        .AddWorkflow<HelloWorldWorkflow>());
    using var cts = new CancellationTokenSource();
    Console.CancelKeyPress += (_, e) => { e.Cancel = true; cts.Cancel(); };
    Console.WriteLine($"Worker starting on task queue: {config.TaskQueue}");
    await worker.ExecuteAsync(cts.Token);
});

harness.OnExecute(async (client, info) =>
{
    var handle = await client.StartWorkflowAsync(
        (HelloWorldWorkflow wf) => wf.RunAsync("World"),
        new WorkflowOptions(id: $"helloworld-{info.Iteration}", taskQueue: info.TaskQueue));

    var result = await handle.GetResultAsync();
    Console.WriteLine($"Workflow result: {result}");
});

return await harness.RunAsync(args);
