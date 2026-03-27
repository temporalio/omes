using Temporalio.Workflows;

[Workflow]
public class HelloWorldWorkflow
{
    [WorkflowRun]
    public Task<string> RunAsync(string name)
    {
        return Task.FromResult($"Hello {name}");
    }
}
