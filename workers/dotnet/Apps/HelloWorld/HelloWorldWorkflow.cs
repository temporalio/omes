using Temporalio.Workflows;

namespace Temporalio.Omes.Apps.HelloWorld;

[Workflow("HelloWorldWorkflow")]
public class HelloWorldWorkflow
{
    [WorkflowRun]
    public Task<string> RunAsync(string name) => Task.FromResult($"Hello {name}");
}
