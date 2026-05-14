using Temporalio.Worker;
using Temporalio.Worker.Tuning;

namespace Temporalio.Omes.Projects.Harness;

internal static class WorkerProfiles
{
    public const string EnvVarName = "OMES_WORKER_PROFILE";
    public const string ResourceBasedDefaultProfile = "resource-based-default";

    private static readonly IReadOnlyDictionary<string, TemporalWorkerOptions> Profiles =
        new Dictionary<string, TemporalWorkerOptions>
        {
            [ResourceBasedDefaultProfile] = new TemporalWorkerOptions
            {
                Tuner = WorkerTuner.CreateResourceBased(
                    targetMemoryUsage: 0.8,
                    targetCpuUsage: 0.8),
            },
        };

    public static TemporalWorkerOptions Lookup(string name)
    {
        if (!Profiles.TryGetValue(name, out var profile))
        {
            throw new ArgumentException($"Unknown worker profile \"{name}\"");
        }

        return (TemporalWorkerOptions)profile.Clone();
    }
}
