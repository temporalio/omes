using Temporalio.Client;
using Temporalio.Exceptions;
using Temporal.Omes.KitchenSink;

namespace Temporalio.Omes;

public class ClientActionsExecutor
{
    public ClientActionsExecutor(ITemporalClient client, string workflowId, string taskQueue)
    {
    }

    // Always unsupported in this branch
    public Task ExecuteClientSequence(ClientSequence _)
        => throw new ApplicationFailureException("client actions activity is not supported", "UnsupportedOperation", nonRetryable: true);
}
