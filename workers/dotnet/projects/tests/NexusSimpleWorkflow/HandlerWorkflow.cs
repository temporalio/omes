using Temporalio.Workflows;

namespace NexusSimpleWorkflow;

[Workflow]
public class HandlerWorkflow
{
    [WorkflowRun]
    public Task<string> RunAsync(string name) => Task.FromResult($"Hello from workflow, {name}");
}
