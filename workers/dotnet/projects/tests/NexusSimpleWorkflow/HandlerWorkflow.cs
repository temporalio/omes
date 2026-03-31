using Temporalio.Workflows;

namespace NexusSimpleWorkflowProject;

[Workflow]
public class HandlerWorkflow
{
    [WorkflowRun]
    public Task<string> RunAsync(string name) => Task.FromResult($"Hello from workflow, {name}");
}
