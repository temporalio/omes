using Temporalio.Worker;
using Temporalio.Worker.Tuning;
using WorkerProfile = Temporalio.Worker.TemporalWorkerOptions;

namespace Temporalio.Omes.Projects.Harness;

internal static class WorkerProfiles
{
    public const string EnvVarName = "OMES_WORKER_PROFILE";
    public const string ResourceBasedDefaultProfile = "resource-based-default";
    public const string ThroughputStressBaselineProfile = "throughput-stress-baseline";

    private static readonly IReadOnlyDictionary<string, WorkerProfile> Profiles = BuildProfiles();

    private static Dictionary<string, WorkerProfile> BuildProfiles()
    {
        var profiles = new Dictionary<string, WorkerProfile>
        {
            [ResourceBasedDefaultProfile] = new TemporalWorkerOptions
            {
                Tuner = WorkerTuner.CreateResourceBased(
                    targetMemoryUsage: 0.8,
                    targetCpuUsage: 0.8),
            },
            [ThroughputStressBaselineProfile] = ThroughputStressProfile(1),
        };
        // The baseline profile with every limit scaled, e.g. throughput-stress-baseline-x4.
        foreach (var scale in new[] { 2, 4, 8, 16 })
        {
            profiles[$"{ThroughputStressBaselineProfile}-x{scale}"] = ThroughputStressProfile(scale);
        }
        return profiles;
    }

    private static WorkerProfile ThroughputStressProfile(int scale) => new TemporalWorkerOptions
    {
        MaxCachedWorkflows = 50 * scale,
        MaxConcurrentWorkflowTasks = 8 * scale,
        MaxConcurrentActivities = 32 * scale,
        MaxConcurrentLocalActivities = 32 * scale,
        MaxConcurrentWorkflowTaskPolls = 2 * scale,
        MaxConcurrentActivityTaskPolls = 4 * scale,
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
