using Temporalio.Nexus;
using Temporalio.Workflows;

namespace NexusSimpleWorkflow;

[Workflow]
public class CallerWorkflow
{
    [WorkflowRun]
    public async Task<string> RunAsync(string endpointName, string input)
    {
        return await Workflow.CreateNexusWorkflowClient<IStringService>(endpointName)
            .ExecuteNexusOperationAsync(svc => svc.DoSomething(input));
    }
}
