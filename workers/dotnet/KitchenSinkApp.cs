using Temporalio.Client;
using Temporalio.Omes.Projects.Harness;
using Temporalio.Worker;

namespace Temporalio.Omes;

public static class KitchenSinkApp
{
    public static Temporalio.Omes.Projects.Harness.App Create() =>
        new(
            Worker: CreateWorker,
            ClientFactory: ClientHelpers.DefaultClientFactory);

    private static TemporalWorker CreateWorker(ITemporalClient client, WorkerContext context)
    {
        var workerOptions = context.WorkerOptions;
        workerOptions.AddWorkflow<KitchenSinkWorkflow>();
        workerOptions.AddActivity(KitchenSinkWorkflow.Noop);
        workerOptions.AddActivity(KitchenSinkWorkflow.Delay);
        workerOptions.AddActivity(KitchenSinkWorkflow.Payload);

        var clientActivities = new ClientActivitiesImpl(client, context.ErrOnUnimplemented);
        workerOptions.AddActivity(clientActivities.Client);
        return new TemporalWorker(client, workerOptions);
    }
}
