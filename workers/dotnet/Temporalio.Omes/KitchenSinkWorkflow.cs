using Temporalio.Workflows;

namespace Temporalio.Omes;

[Workflow("kitchenSink")]
public class KitchenSinkWorkflow
{
    [WorkflowRun]
    public async Task RunAsync()
    {
        await Workflow.DelayAsync(1);
    }
}