using Temporalio.Worker;
using Temporalio.Worker.Tuning;
using WorkerProfile = Temporalio.Worker.TemporalWorkerOptions;

namespace Temporalio.Omes.Projects.Harness;

internal static class WorkerProfiles
{
    public const string EnvVarName = "OMES_WORKER_PROFILE";
    public const string ResourceBasedDefaultProfile = "resource-based-default";
    public const string ThroughputStressBaselineProfile = "throughput-stress-baseline";

    private static readonly IReadOnlyDictionary<string, WorkerProfile> Profiles =
        new Dictionary<string, WorkerProfile>
        {
            [ResourceBasedDefaultProfile] = new TemporalWorkerOptions
            {
                Tuner = WorkerTuner.CreateResourceBased(
                    targetMemoryUsage: 0.8,
                    targetCpuUsage: 0.8),
            },
            [ThroughputStressBaselineProfile] = new TemporalWorkerOptions
            {
                MaxCachedWorkflows = 50,
                MaxConcurrentWorkflowTasks = 8,
                MaxConcurrentActivities = 32,
                MaxConcurrentLocalActivities = 32,
                MaxConcurrentWorkflowTaskPolls = 2,
                MaxConcurrentActivityTaskPolls = 4,
            },
        };

    public static WorkerProfile LookupWorkerProfile(string name)
    {
        if (!Profiles.TryGetValue(name, out var profile))
        {
            throw new ArgumentException($"Unknown worker profile \"{name}\"");
        }

        return (WorkerProfile)profile.Clone();
    }

}
